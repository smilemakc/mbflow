package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

// BackoffStrategy defines how retry delays are calculated
type BackoffStrategy string

const (
	// BackoffConstant uses a constant delay between retries
	BackoffConstant BackoffStrategy = "constant"

	// BackoffLinear increases delay linearly with each attempt
	BackoffLinear BackoffStrategy = "linear"

	// BackoffExponential doubles delay with each attempt
	BackoffExponential BackoffStrategy = "exponential"
)

// RetryPolicy defines the retry behavior for node execution
type RetryPolicy struct {
	// MaxAttempts is the maximum number of attempts (including the first one)
	// 0 or 1 means no retries
	MaxAttempts int

	// InitialDelay is the initial delay before the first retry
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// BackoffStrategy determines how delays increase
	BackoffStrategy BackoffStrategy

	// RetryableErrors is a list of error messages/patterns that should trigger a retry
	// If empty, all errors are retryable
	RetryableErrors []string

	// OnRetry is an optional callback called before each retry
	OnRetry func(attempt int, err error)
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffStrategy: BackoffExponential,
		RetryableErrors: []string{}, // All errors are retryable by default
	}
}

// NoRetryPolicy returns a policy that doesn't retry
func NoRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts: 1,
	}
}

// ShouldRetry determines if an error is retryable according to the policy
func (rp *RetryPolicy) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// If no specific retryable errors are configured, all errors are retryable
	if len(rp.RetryableErrors) == 0 {
		return true
	}

	// Check if error message matches any retryable error pattern
	errorMsg := err.Error()
	for _, pattern := range rp.RetryableErrors {
		if contains(errorMsg, pattern) {
			return true
		}
	}

	return false
}

// GetDelay calculates the delay before the next retry based on the attempt number
func (rp *RetryPolicy) GetDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	var delay time.Duration

	switch rp.BackoffStrategy {
	case BackoffConstant:
		delay = rp.InitialDelay

	case BackoffLinear:
		delay = rp.InitialDelay * time.Duration(attempt)

	case BackoffExponential:
		// delay = initialDelay * 2^(attempt-1)
		multiplier := math.Pow(2, float64(attempt-1))
		delay = time.Duration(float64(rp.InitialDelay) * multiplier)

	default:
		delay = rp.InitialDelay
	}

	// Cap at max delay
	if delay > rp.MaxDelay {
		delay = rp.MaxDelay
	}

	return delay
}

// Execute executes a function with retry logic
func (rp *RetryPolicy) Execute(ctx context.Context, fn func() error) error {
	if rp.MaxAttempts <= 0 {
		rp.MaxAttempts = 1
	}

	var lastErr error

	for attempt := 1; attempt <= rp.MaxAttempts; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("execution cancelled: %w", ctx.Err())
		default:
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if attempt >= rp.MaxAttempts {
			break // No more attempts left
		}

		if !rp.ShouldRetry(err) {
			break // Error is not retryable
		}

		// Call retry callback if provided
		if rp.OnRetry != nil {
			rp.OnRetry(attempt, err)
		}

		// Wait before next attempt
		delay := rp.GetDelay(attempt)
		if delay > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("execution cancelled during retry delay: %w", ctx.Err())
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	return fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// contains checks if a string contains a substring (case-sensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfString(s, substr) >= 0)
}

// indexOfString returns the index of substr in s, or -1 if not found
func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// IsRetryableError checks if an error is temporary and should be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context deadline exceeded or cancelled (not retryable)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	// Check for common retryable error types
	var temporaryErr interface{ Temporary() bool }
	if errors.As(err, &temporaryErr) {
		return temporaryErr.Temporary()
	}

	// Check for timeout errors
	var timeoutErr interface{ Timeout() bool }
	if errors.As(err, &timeoutErr) {
		return timeoutErr.Timeout()
	}

	return true
}
