package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// InternalBackoffStrategy defines how retry delays are calculated.
type InternalBackoffStrategy string

const (
	// InternalBackoffConstant uses a constant delay between retries.
	InternalBackoffConstant InternalBackoffStrategy = "constant"

	// InternalBackoffLinear increases delay linearly with each attempt.
	InternalBackoffLinear InternalBackoffStrategy = "linear"

	// InternalBackoffExponential doubles delay with each attempt.
	InternalBackoffExponential InternalBackoffStrategy = "exponential"
)

// InternalRetryPolicy defines the retry behavior for node execution.
type InternalRetryPolicy struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffStrategy InternalBackoffStrategy
	RetryableErrors []string
	OnRetry         func(attempt int, err error)
}

// DefaultInternalRetryPolicy returns a sensible default retry policy.
func DefaultInternalRetryPolicy() *InternalRetryPolicy {
	return &InternalRetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffStrategy: InternalBackoffExponential,
		RetryableErrors: []string{},
	}
}

// NoInternalRetryPolicy returns a policy that doesn't retry.
func NoInternalRetryPolicy() *InternalRetryPolicy {
	return &InternalRetryPolicy{
		MaxAttempts: 1,
	}
}

// ShouldRetry determines if an error is retryable according to the policy.
func (rp *InternalRetryPolicy) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	if len(rp.RetryableErrors) == 0 {
		return true
	}

	errorMsg := err.Error()
	for _, pattern := range rp.RetryableErrors {
		if strings.Contains(errorMsg, pattern) {
			return true
		}
	}

	return false
}

// GetDelay calculates the delay before the next retry based on the attempt number.
func (rp *InternalRetryPolicy) GetDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	var delay time.Duration

	switch rp.BackoffStrategy {
	case InternalBackoffConstant:
		delay = rp.InitialDelay
	case InternalBackoffLinear:
		delay = rp.InitialDelay * time.Duration(attempt)
	case InternalBackoffExponential:
		multiplier := math.Pow(2, float64(attempt-1))
		delay = time.Duration(float64(rp.InitialDelay) * multiplier)
	default:
		delay = rp.InitialDelay
	}

	if delay > rp.MaxDelay {
		delay = rp.MaxDelay
	}

	return delay
}

// Execute executes a function with retry logic.
func (rp *InternalRetryPolicy) Execute(ctx context.Context, fn func() error) error {
	if rp.MaxAttempts <= 0 {
		rp.MaxAttempts = 1
	}

	var lastErr error

	for attempt := 1; attempt <= rp.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("execution cancelled: %w", ctx.Err())
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt >= rp.MaxAttempts {
			break
		}

		if !rp.ShouldRetry(err) {
			break
		}

		if rp.OnRetry != nil {
			rp.OnRetry(attempt, err)
		}

		delay := rp.GetDelay(attempt)
		if delay > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("execution cancelled during retry delay: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	return fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// IsRetryableError checks if an error is temporary and should be retried.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	var temporaryErr interface{ Temporary() bool }
	if errors.As(err, &temporaryErr) {
		return temporaryErr.Temporary()
	}

	var timeoutErr interface{ Timeout() bool }
	if errors.As(err, &timeoutErr) {
		return timeoutErr.Timeout()
	}

	return true
}
