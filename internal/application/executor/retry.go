package executor

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/smilemakc/mbflow/internal/domain"
)

// RetryPolicy defines the retry behavior for node execution failures
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts (0 = no retries)
	MaxAttempts int

	// InitialDelay is the delay before the first retry
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases (exponential backoff)
	Multiplier float64

	// Jitter adds randomness to the delay to avoid thundering herd
	Jitter bool

	// RetryableErrors defines which errors should trigger a retry
	// If nil, all errors are retryable
	RetryableErrors []string
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// NoRetryPolicy returns a policy that disables retries
func NoRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts: 0,
	}
}

// RetryExecutor wraps a node executor with retry logic
type RetryExecutor struct {
	executor NodeExecutor
	policy   *RetryPolicy
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(executor NodeExecutor, policy *RetryPolicy) *RetryExecutor {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}

	return &RetryExecutor{
		executor: executor,
		policy:   policy,
	}
}

// Execute executes the node with retry logic
func (r *RetryExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	var lastErr error

	for attempt := 0; attempt <= r.policy.MaxAttempts; attempt++ {
		// First attempt is not a retry
		if attempt > 0 {
			// Calculate delay
			delay := r.calculateDelay(attempt)

			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}
		}

		// Execute node
		output, err := r.executor.Execute(ctx, node, inputs)
		if err == nil {
			// Success
			return output, nil
		}

		// Check if error is retryable
		if !r.isRetryable(err) {
			return nil, err
		}

		lastErr = err

		// Check if we should retry
		if attempt < r.policy.MaxAttempts {
			// Will retry
			continue
		}

		// Max attempts reached
		break
	}

	// All retries exhausted
	return nil, fmt.Errorf("max retry attempts (%d) exhausted: %w", r.policy.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay before the next retry using exponential backoff
func (r *RetryExecutor) calculateDelay(attempt int) time.Duration {
	// Calculate exponential delay
	delay := float64(r.policy.InitialDelay) * math.Pow(r.policy.Multiplier, float64(attempt-1))

	// Apply max delay cap
	if delay > float64(r.policy.MaxDelay) {
		delay = float64(r.policy.MaxDelay)
	}

	// Add jitter if enabled
	if r.policy.Jitter {
		jitterAmount := delay * 0.1 // 10% jitter
		jitter := (2*float64(time.Now().UnixNano()%1000)/1000 - 1) * jitterAmount
		delay += jitter
	}

	return time.Duration(delay)
}

// isRetryable checks if an error should trigger a retry
func (r *RetryExecutor) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// If no specific retryable errors defined, retry all errors
	if len(r.policy.RetryableErrors) == 0 {
		return true
	}

	// Check if error matches any retryable error pattern
	errMsg := err.Error()
	for _, retryableErr := range r.policy.RetryableErrors {
		if contains(errMsg, retryableErr) {
			return true
		}
	}

	return false
}

// RetryConfig holds per-node retry configuration
type RetryConfig struct {
	Enabled      bool
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// GetRetryConfig extracts retry configuration from node config
func GetRetryConfig(node domain.Node) *RetryConfig {
	config := node.Config()

	retryConfig := &RetryConfig{
		Enabled:      false,
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}

	// Check if retry is enabled
	if enabled, ok := config["retry_enabled"].(bool); ok {
		retryConfig.Enabled = enabled
	}

	// Get max attempts
	if maxAttempts, ok := config["retry_max_attempts"].(int); ok {
		retryConfig.MaxAttempts = maxAttempts
	} else if maxAttempts, ok := config["retry_max_attempts"].(float64); ok {
		retryConfig.MaxAttempts = int(maxAttempts)
	}

	// Get initial delay
	if initialDelay, ok := config["retry_initial_delay"].(string); ok {
		if d, err := time.ParseDuration(initialDelay); err == nil {
			retryConfig.InitialDelay = d
		}
	} else if initialDelayMs, ok := config["retry_initial_delay_ms"].(float64); ok {
		retryConfig.InitialDelay = time.Duration(initialDelayMs) * time.Millisecond
	}

	// Get max delay
	if maxDelay, ok := config["retry_max_delay"].(string); ok {
		if d, err := time.ParseDuration(maxDelay); err == nil {
			retryConfig.MaxDelay = d
		}
	} else if maxDelayMs, ok := config["retry_max_delay_ms"].(float64); ok {
		retryConfig.MaxDelay = time.Duration(maxDelayMs) * time.Millisecond
	}

	// Get multiplier
	if multiplier, ok := config["retry_multiplier"].(float64); ok {
		retryConfig.Multiplier = multiplier
	}

	return retryConfig
}

// CreateRetryPolicy creates a retry policy from retry config
func CreateRetryPolicy(config *RetryConfig) *RetryPolicy {
	if !config.Enabled {
		return NoRetryPolicy()
	}

	return &RetryPolicy{
		MaxAttempts:  config.MaxAttempts,
		InitialDelay: config.InitialDelay,
		MaxDelay:     config.MaxDelay,
		Multiplier:   config.Multiplier,
		Jitter:       true,
	}
}

// RetryableExecutor is an interface for executors that support retry configuration
type RetryableExecutor interface {
	NodeExecutor
	SupportsRetry() bool
}

// // WithRetry wraps an executor with retry logic based on node configuration
// func WithRetry(executor NodeExecutor, node domain.Node) NodeExecutor {
// 	retryConfig := GetRetryConfig(node)
// 	if !retryConfig.Enabled {
// 		return executor
// 	}
//
// 	policy := CreateRetryPolicy(retryConfig)
// 	return NewRetryExecutor(executor, policy)
// }

// RetryBudget tracks the number of retries to prevent infinite loops
type RetryBudget struct {
	maxRetries int
	used       int
}

// NewRetryBudget creates a new retry budget
func NewRetryBudget(maxRetries int) *RetryBudget {
	return &RetryBudget{
		maxRetries: maxRetries,
		used:       0,
	}
}

// CanRetry checks if there are retries left in the budget
func (rb *RetryBudget) CanRetry() bool {
	return rb.used < rb.maxRetries
}

// UseRetry consumes one retry from the budget
func (rb *RetryBudget) UseRetry() bool {
	if !rb.CanRetry() {
		return false
	}
	rb.used++
	return true
}

// Remaining returns the number of retries left
func (rb *RetryBudget) Remaining() int {
	return rb.maxRetries - rb.used
}

// Used returns the number of retries used
func (rb *RetryBudget) Used() int {
	return rb.used
}

// Reset resets the retry budget
func (rb *RetryBudget) Reset() {
	rb.used = 0
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
