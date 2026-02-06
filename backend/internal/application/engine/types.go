package engine

import (
	"time"

	"github.com/smilemakc/mbflow/internal/application/observer"
)

// ExecutionOptions configures execution behavior for the internal engine.
type ExecutionOptions struct {
	StrictMode       bool
	MaxParallelism   int
	Timeout          time.Duration
	NodeTimeout      time.Duration
	Variables        map[string]interface{}
	ObserverManager  *observer.ObserverManager
	RetryPolicy      *RetryPolicy
	ContinueOnError  bool
	MaxOutputSize    int64
	MaxTotalMemory   int64
	EnableMemoryOpts bool
}

// RetryPolicy defines the retry behavior for node execution.
type RetryPolicy struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffStrategy BackoffStrategy
	RetryableErrors []string
	OnRetry         func(attempt int, err error)
}

// BackoffStrategy defines how retry delays are calculated.
type BackoffStrategy string

const (
	BackoffConstant    BackoffStrategy = "constant"
	BackoffLinear      BackoffStrategy = "linear"
	BackoffExponential BackoffStrategy = "exponential"
)

// DefaultExecutionOptions returns default execution options.
func DefaultExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		StrictMode:       false,
		MaxParallelism:   10,
		Timeout:          5 * time.Minute,
		NodeTimeout:      1 * time.Minute,
		Variables:        make(map[string]interface{}),
		RetryPolicy:      NoRetryPolicy(),
		ContinueOnError:  false,
		MaxOutputSize:    0,
		MaxTotalMemory:   0,
		EnableMemoryOpts: false,
	}
}

// NoRetryPolicy returns a policy that doesn't retry.
func NoRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts: 1,
	}
}
