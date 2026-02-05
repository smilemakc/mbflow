// Package engine provides public types and interfaces for workflow execution.
// This package exposes the execution capabilities of MBFlow without
// requiring direct imports from internal packages.
package engine

import (
	"time"
)

// ExecutionOptions configures workflow execution behavior.
type ExecutionOptions struct {
	// RetryPolicy configures retry behavior for node execution
	RetryPolicy *RetryPolicy

	// Timeout is the maximum duration for the entire workflow execution
	Timeout time.Duration

	// NodeTimeout is the default timeout for individual node execution
	NodeTimeout time.Duration

	// ContinueOnError determines if execution continues after node failures
	ContinueOnError bool

	// StrictMode enables strict validation during execution
	StrictMode bool

	// MaxConcurrency limits the number of nodes executing in parallel
	MaxConcurrency int

	// MaxParallelism is an alias for MaxConcurrency (for backward compatibility)
	MaxParallelism int

	// MaxOutputSize limits the size of node outputs in bytes (0 = unlimited)
	MaxOutputSize int64

	// MaxTotalMemory limits total memory usage across all nodes (0 = unlimited)
	MaxTotalMemory int64

	// EnableMemoryOpts enables memory optimization features
	EnableMemoryOpts bool

	// Variables are workflow-level variables available to all nodes
	Variables map[string]interface{}

	// ObserverManager handles execution events (optional).
	// Can be either engine.ObserverManager interface or *observer.ObserverManager from internal.
	ObserverManager interface{}
}

// RetryPolicy configures retry behavior for node execution.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts (including first attempt)
	MaxAttempts int

	// InitialDelay is the delay before the first retry
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// BackoffStrategy determines how delay increases between retries
	BackoffStrategy BackoffStrategy

	// RetryOn specifies which errors should trigger a retry
	RetryOn []string
}

// BackoffStrategy defines how retry delays increase.
type BackoffStrategy int

const (
	// BackoffConstant uses the same delay for all retries
	BackoffConstant BackoffStrategy = iota

	// BackoffLinear increases delay linearly
	BackoffLinear

	// BackoffExponential doubles the delay with each retry
	BackoffExponential
)

// DefaultExecutionOptions returns execution options with sensible defaults.
func DefaultExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		Timeout:         5 * time.Minute,
		NodeTimeout:     2 * time.Minute,
		ContinueOnError: false,
		StrictMode:      false,
		MaxConcurrency:  10,
		MaxParallelism:  10,
		MaxOutputSize:   10 * 1024 * 1024, // 10MB
		MaxTotalMemory:  0,                 // unlimited
		EnableMemoryOpts: false,
		Variables:       make(map[string]interface{}),
	}
}
