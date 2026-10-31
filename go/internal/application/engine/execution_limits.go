package engine

import (
	"context"
	"errors"
)

// DefaultMaxConcurrentExecutions is the global cap on concurrently running
// executions used by the server when no explicit limit is configured.
const DefaultMaxConcurrentExecutions = 100

// ErrShuttingDown is returned when an execution cannot start because the
// execution manager is shutting down.
var ErrShuttingDown = errors.New("execution manager is shutting down")

// executionSlots is a counting semaphore bounding concurrent executions.
type executionSlots chan struct{}

// SetMaxConcurrentExecutions caps how many executions (stored and inline,
// sync and async) run at the same time across this manager. Executions over
// the cap wait for a free slot: async ones stay pending, sync callers block
// until a slot frees up or their context ends. n <= 0 removes the limit.
//
// The new limit applies to executions that start waiting after the call;
// executions already holding or waiting for a slot of the previous limit are
// unaffected. Configure it before submitting work.
func (em *ExecutionManager) SetMaxConcurrentExecutions(n int) {
	if n <= 0 {
		em.slots.Store(nil)
		return
	}
	slots := make(executionSlots, n)
	em.slots.Store(&slots)
}

// MaxConcurrentExecutions returns the configured limit (0 means unlimited).
func (em *ExecutionManager) MaxConcurrentExecutions() int {
	slots := em.slots.Load()
	if slots == nil {
		return 0
	}
	return cap(*slots)
}

// RunningExecutions returns how many executions currently hold a slot. It is
// always 0 when the manager has no concurrency limit.
func (em *ExecutionManager) RunningExecutions() int {
	slots := em.slots.Load()
	if slots == nil {
		return 0
	}
	return len(*slots)
}

// acquireSlot blocks until an execution slot is free. It fails when ctx ends
// or the manager shuts down first. The returned release func must be called
// exactly once when the execution finishes.
func (em *ExecutionManager) acquireSlot(ctx context.Context) (func(), error) {
	slots := em.slots.Load()
	if slots == nil {
		return func() {}, nil
	}
	sem := *slots
	release := func() { <-sem }

	// Prefer a free slot over a concurrently cancelled context.
	select {
	case sem <- struct{}{}:
		return release, nil
	default:
	}

	select {
	case sem <- struct{}{}:
		return release, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-em.runContext().Done():
		return nil, ErrShuttingDown
	}
}
