package executor

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/smilemakc/mbflow/internal/domain/errors"
)

// RetryPolicy defines the retry behavior for failed nodes.
// It supports configurable retry strategies with exponential backoff.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of attempts (including the initial attempt)
	MaxAttempts int
	// InitialDelay is the delay before the first retry
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// RetryableErrors is a function that determines if an error is retryable
	RetryableErrors func(error) bool
}

// DefaultRetryPolicy returns a default retry policy.
// Default: 3 attempts, 1s initial delay, 30s max delay, 2x backoff multiplier.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:       3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableErrors:   errors.IsRetryable,
	}
}

// NoRetryPolicy returns a policy that never retries.
func NoRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:     1,
		RetryableErrors: func(error) bool { return false },
	}
}

// AggressiveRetryPolicy returns a policy with more aggressive retries.
// 5 attempts, 500ms initial delay, 60s max delay, 2x backoff.
func AggressiveRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:       5,
		InitialDelay:      500 * time.Millisecond,
		MaxDelay:          60 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableErrors:   errors.IsRetryable,
	}
}

// ShouldRetry determines if an error should be retried based on the policy.
func (p *RetryPolicy) ShouldRetry(err error, attemptNumber int) bool {
	if attemptNumber >= p.MaxAttempts {
		return false
	}
	if p.RetryableErrors == nil {
		return false
	}
	return p.RetryableErrors(err)
}

// GetDelay calculates the delay before the next retry using exponential backoff.
func (p *RetryPolicy) GetDelay(attemptNumber int) time.Duration {
	if attemptNumber <= 1 {
		return p.InitialDelay
	}

	// Calculate exponential backoff: initialDelay * (multiplier ^ (attempt - 1))
	delay := float64(p.InitialDelay) * math.Pow(p.BackoffMultiplier, float64(attemptNumber-1))

	// Cap at max delay
	if delay > float64(p.MaxDelay) {
		return p.MaxDelay
	}

	return time.Duration(delay)
}

// Wait waits for the appropriate delay before retrying.
// It respects context cancellation.
func (p *RetryPolicy) Wait(ctx context.Context, attemptNumber int) error {
	delay := p.GetDelay(attemptNumber)

	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RetryExecutor wraps a function with retry logic.
type RetryExecutor struct {
	policy *RetryPolicy
}

// NewRetryExecutor creates a new RetryExecutor with the given policy.
func NewRetryExecutor(policy *RetryPolicy) *RetryExecutor {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}
	return &RetryExecutor{
		policy: policy,
	}
}

// Execute executes a function with retry logic.
// The function is called with the attempt number (starting from 1).
func (r *RetryExecutor) Execute(ctx context.Context, fn func(attemptNumber int) error) error {
	var lastErr error

	for attempt := 1; attempt <= r.policy.MaxAttempts; attempt++ {
		// Execute the function
		err := fn(attempt)

		// Success
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !r.policy.ShouldRetry(err, attempt) {
			return fmt.Errorf("non-retryable error after %d attempts: %w", attempt, err)
		}

		// Check if we've exhausted attempts
		if attempt >= r.policy.MaxAttempts {
			return fmt.Errorf("max retry attempts (%d) exceeded: %w", r.policy.MaxAttempts, err)
		}

		// Wait before retrying
		if waitErr := r.policy.Wait(ctx, attempt); waitErr != nil {
			return fmt.Errorf("retry cancelled: %w", waitErr)
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", r.policy.MaxAttempts, lastErr)
}

// ExecuteWithCallback executes a function with retry logic and callbacks.
// The onRetry callback is called before each retry attempt.
func (r *RetryExecutor) ExecuteWithCallback(
	ctx context.Context,
	fn func(attemptNumber int) error,
	onRetry func(attemptNumber int, err error, delay time.Duration),
) error {
	var lastErr error

	for attempt := 1; attempt <= r.policy.MaxAttempts; attempt++ {
		// Execute the function
		err := fn(attempt)

		// Success
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !r.policy.ShouldRetry(err, attempt) {
			return fmt.Errorf("non-retryable error after %d attempts: %w", attempt, err)
		}

		// Check if we've exhausted attempts
		if attempt >= r.policy.MaxAttempts {
			return fmt.Errorf("max retry attempts (%d) exceeded: %w", r.policy.MaxAttempts, err)
		}

		// Calculate delay and call callback
		delay := r.policy.GetDelay(attempt)
		if onRetry != nil {
			onRetry(attempt, err, delay)
		}

		// Wait before retrying
		if waitErr := r.policy.Wait(ctx, attempt); waitErr != nil {
			return fmt.Errorf("retry cancelled: %w", waitErr)
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", r.policy.MaxAttempts, lastErr)
}
