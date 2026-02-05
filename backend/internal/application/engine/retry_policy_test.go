package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryPolicy(t *testing.T) {
	t.Parallel()
	policy := DefaultRetryPolicy()

	if policy.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", policy.MaxAttempts)
	}

	if policy.InitialDelay != 1*time.Second {
		t.Errorf("expected InitialDelay 1s, got %v", policy.InitialDelay)
	}

	if policy.BackoffStrategy != BackoffExponential {
		t.Errorf("expected BackoffExponential, got %v", policy.BackoffStrategy)
	}
}

func TestNoRetryPolicy(t *testing.T) {
	t.Parallel()
	policy := NoRetryPolicy()

	if policy.MaxAttempts != 1 {
		t.Errorf("expected MaxAttempts 1, got %d", policy.MaxAttempts)
	}
}

func TestRetryPolicy_ShouldRetry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		retryableErrors []string
		err             error
		expected        bool
	}{
		{
			name:            "nil error",
			retryableErrors: []string{},
			err:             nil,
			expected:        false,
		},
		{
			name:            "empty retryable list - all errors retryable",
			retryableErrors: []string{},
			err:             errors.New("any error"),
			expected:        true,
		},
		{
			name:            "matching error",
			retryableErrors: []string{"timeout", "connection"},
			err:             errors.New("connection refused"),
			expected:        true,
		},
		{
			name:            "non-matching error",
			retryableErrors: []string{"timeout", "connection"},
			err:             errors.New("invalid input"),
			expected:        false,
		},
		{
			name:            "exact match",
			retryableErrors: []string{"timeout"},
			err:             errors.New("timeout"),
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			policy := &RetryPolicy{
				RetryableErrors: tt.retryableErrors,
			}

			result := policy.ShouldRetry(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRetryPolicy_GetDelay_Constant(t *testing.T) {
	t.Parallel()
	policy := &RetryPolicy{
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        1 * time.Second,
		BackoffStrategy: BackoffConstant,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{attempt: 1, expected: 100 * time.Millisecond},
		{attempt: 2, expected: 100 * time.Millisecond},
		{attempt: 3, expected: 100 * time.Millisecond},
		{attempt: 10, expected: 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			delay := policy.GetDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, delay)
			}
		})
	}
}

func TestRetryPolicy_GetDelay_Linear(t *testing.T) {
	policy := &RetryPolicy{
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        1 * time.Second,
		BackoffStrategy: BackoffLinear,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{attempt: 1, expected: 100 * time.Millisecond},
		{attempt: 2, expected: 200 * time.Millisecond},
		{attempt: 3, expected: 300 * time.Millisecond},
		{attempt: 5, expected: 500 * time.Millisecond},
		{attempt: 10, expected: 1 * time.Second}, // Capped at MaxDelay
		{attempt: 20, expected: 1 * time.Second}, // Capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			delay := policy.GetDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, delay)
			}
		})
	}
}

func TestRetryPolicy_GetDelay_Exponential(t *testing.T) {
	policy := &RetryPolicy{
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        2 * time.Second,
		BackoffStrategy: BackoffExponential,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{attempt: 1, expected: 100 * time.Millisecond},  // 100 * 2^0
		{attempt: 2, expected: 200 * time.Millisecond},  // 100 * 2^1
		{attempt: 3, expected: 400 * time.Millisecond},  // 100 * 2^2
		{attempt: 4, expected: 800 * time.Millisecond},  // 100 * 2^3
		{attempt: 5, expected: 1600 * time.Millisecond}, // 100 * 2^4
		{attempt: 6, expected: 2 * time.Second},         // Capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			delay := policy.GetDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, delay)
			}
		})
	}
}

func TestRetryPolicy_Execute_Success(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    10 * time.Millisecond,
		BackoffStrategy: BackoffConstant,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return nil // Success on first attempt
	}

	ctx := context.Background()
	err := policy.Execute(ctx, fn)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryPolicy_Execute_SuccessAfterRetry(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    10 * time.Millisecond,
		BackoffStrategy: BackoffConstant,
	}

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil // Success on third attempt
	}

	ctx := context.Background()
	err := policy.Execute(ctx, fn)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryPolicy_Execute_MaxAttemptsExceeded(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    10 * time.Millisecond,
		BackoffStrategy: BackoffConstant,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("persistent error")
	}

	ctx := context.Background()
	err := policy.Execute(ctx, fn)

	if err == nil {
		t.Error("expected error after max attempts")
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryPolicy_Execute_NonRetryableError(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    10 * time.Millisecond,
		BackoffStrategy: BackoffConstant,
		RetryableErrors: []string{"timeout"},
	}

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("invalid input") // Not retryable
	}

	ctx := context.Background()
	err := policy.Execute(ctx, fn)

	if err == nil {
		t.Error("expected error")
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry for non-retryable error), got %d", attempts)
	}
}

func TestRetryPolicy_Execute_ContextCancellation(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:     5,
		InitialDelay:    50 * time.Millisecond,
		BackoffStrategy: BackoffConstant,
	}

	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	fn := func() error {
		attempts++
		if attempts == 2 {
			cancel() // Cancel after second attempt
		}
		return errors.New("error")
	}

	err := policy.Execute(ctx, fn)

	if err == nil {
		t.Error("expected error due to context cancellation")
	}

	if attempts >= 5 {
		t.Errorf("expected fewer than 5 attempts due to cancellation, got %d", attempts)
	}
}

func TestRetryPolicy_Execute_OnRetryCallback(t *testing.T) {
	callbackCalls := 0

	policy := &RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    10 * time.Millisecond,
		BackoffStrategy: BackoffConstant,
		OnRetry: func(attempt int, err error) {
			callbackCalls++
			if attempt < 1 || attempt > 2 {
				t.Errorf("unexpected attempt number in callback: %d", attempt)
			}
		},
	}

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("error")
		}
		return nil
	}

	ctx := context.Background()
	policy.Execute(ctx, fn)

	// Callback should be called before each retry (not before first attempt)
	if callbackCalls != 2 {
		t.Errorf("expected 2 callback calls, got %d", callbackCalls)
	}
}

func TestRetryPolicy_Execute_ZeroMaxAttempts(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:     0, // Should default to 1
		InitialDelay:    10 * time.Millisecond,
		BackoffStrategy: BackoffConstant,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return nil
	}

	ctx := context.Background()
	policy.Execute(ctx, fn)

	if attempts != 1 {
		t.Errorf("expected 1 attempt with MaxAttempts=0, got %d", attempts)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("some error"),
			expected: true,
		},
		{
			name:     "context cancelled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRetryPolicy_GetDelay_ZeroAttempt(t *testing.T) {
	policy := &RetryPolicy{
		InitialDelay:    100 * time.Millisecond,
		BackoffStrategy: BackoffExponential,
	}

	delay := policy.GetDelay(0)
	if delay != 0 {
		t.Errorf("expected 0 delay for attempt 0, got %v", delay)
	}
}
