package engine

import (
	"time"

	"github.com/smilemakc/mbflow/go/internal/application/observer"
)

// WebhookSubscription defines a per-execution webhook callback configuration.
type WebhookSubscription struct {
	URL     string            // Endpoint URL to send events to
	Events  []string          // Event type filter (empty = all events)
	Headers map[string]string // Custom HTTP headers (e.g. Authorization)
	NodeIDs []string          // Optional node ID filter (empty = all nodes)
}

// ExecutionOptions configures execution behavior for the internal engine.
type ExecutionOptions struct {
	StrictMode       bool
	MaxParallelism   int
	Timeout          time.Duration
	NodeTimeout      time.Duration
	Variables        map[string]any
	ObserverManager  *observer.ObserverManager
	Webhooks         []WebhookSubscription // Per-execution webhook subscriptions
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
		Variables:        make(map[string]any),
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
