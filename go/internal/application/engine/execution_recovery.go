package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
)

// InterruptedExecutionError is stored on executions that were pending or
// running in a process that stopped without finishing them.
const InterruptedExecutionError = "execution interrupted: server stopped while it was running"

// Default lease settings for crash recovery.
const (
	DefaultHeartbeatInterval = 30 * time.Second
	// staleAfterHeartbeats is how many heartbeat intervals an execution may
	// miss before it is considered abandoned.
	staleAfterHeartbeats = 3
)

// ErrRecoveryUnsupported is returned by StartRecovery when the execution
// repository does not implement repository.ExecutionLeaseRepository.
var ErrRecoveryUnsupported = errors.New("execution repository does not support execution leases")

// RecoveryConfig configures lease-based crash recovery.
type RecoveryConfig struct {
	// HeartbeatInterval is how often the manager renews the lease of the
	// executions it owns and sweeps for abandoned ones. Defaults to
	// DefaultHeartbeatInterval.
	HeartbeatInterval time.Duration
	// StaleAfter is how old a lease may get before its execution is failed.
	// Defaults to three heartbeat intervals; it must exceed HeartbeatInterval.
	StaleAfter time.Duration
}

func (c RecoveryConfig) withDefaults() (RecoveryConfig, error) {
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if c.StaleAfter <= 0 {
		c.StaleAfter = staleAfterHeartbeats * c.HeartbeatInterval
	}
	if c.StaleAfter <= c.HeartbeatInterval {
		return c, fmt.Errorf("stale-after (%s) must be greater than the heartbeat interval (%s)", c.StaleAfter, c.HeartbeatInterval)
	}
	return c, nil
}

// activeExecutions tracks persisted executions owned by this process, i.e.
// whose lease it must keep renewing, and how to cancel each of them.
type activeExecutions struct {
	mu  sync.Mutex
	ids map[uuid.UUID]*activeExecution
}

// activeExecution is one execution owned by this process.
type activeExecution struct {
	cancel context.CancelCauseFunc
	done   chan struct{} // closed once the execution is finalized
}

// add starts tracking an execution; cancel (may be nil) cancels its context.
// The returned func stops tracking it and must be called once the execution
// has been finalized.
func (a *activeExecutions) add(id string, cancel context.CancelCauseFunc) func() {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return func() {}
	}
	entry := &activeExecution{cancel: cancel, done: make(chan struct{})}
	a.mu.Lock()
	if a.ids == nil {
		a.ids = make(map[uuid.UUID]*activeExecution)
	}
	a.ids[parsed] = entry
	a.mu.Unlock()
	return func() {
		a.mu.Lock()
		if a.ids[parsed] == entry {
			delete(a.ids, parsed)
		}
		a.mu.Unlock()
		close(entry.done)
	}
}

// cancel requests cancellation of an execution owned by this process. It
// returns a channel closed once the execution is finalized, or nil if this
// process does not own the execution.
func (a *activeExecutions) cancel(id uuid.UUID) <-chan struct{} {
	a.mu.Lock()
	entry, ok := a.ids[id]
	a.mu.Unlock()
	if !ok {
		return nil
	}
	if entry.cancel != nil {
		entry.cancel(errCancelRequested)
	}
	return entry.done
}

func (a *activeExecutions) list() []uuid.UUID {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]uuid.UUID, 0, len(a.ids))
	for id := range a.ids {
		ids = append(ids, id)
	}
	return ids
}

// StartRecovery enables lease-based crash recovery, which is safe to run in
// every replica sharing a database:
//
//   - every HeartbeatInterval the manager renews the lease (updated_at) of the
//     persisted executions it is running or has queued;
//   - every HeartbeatInterval, and once right away, it fails pending/running
//     executions whose lease is older than StaleAfter — they belonged to a
//     process that died — with InterruptedExecutionError.
//
// The loop stops when ctx ends or the manager shuts down. StartRecovery
// returns ErrRecoveryUnsupported if the execution repository does not
// implement repository.ExecutionLeaseRepository, and an error if recovery is
// already running.
func (em *ExecutionManager) StartRecovery(ctx context.Context, cfg RecoveryConfig) error {
	leases, ok := em.executionRepo.(repository.ExecutionLeaseRepository)
	if !ok {
		return ErrRecoveryUnsupported
	}
	cfg, err := cfg.withDefaults()
	if err != nil {
		return err
	}
	if !em.recoveryStarted.CompareAndSwap(false, true) {
		return errors.New("execution recovery already started")
	}

	runCtx := em.runContext()
	loopCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(runCtx, cancel)

	em.recoverySweep(loopCtx, leases, cfg)

	em.recoveryWG.Add(1)
	go func() {
		defer em.recoveryWG.Done()
		defer stop()
		defer cancel()

		ticker := time.NewTicker(cfg.HeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				em.heartbeat(loopCtx, leases)
				em.recoverySweep(loopCtx, leases, cfg)
			}
		}
	}()
	return nil
}

// heartbeat renews the lease of every execution owned by this process, and
// cancels those of them that were cancelled through another replica (see
// CancelExecution).
func (em *ExecutionManager) heartbeat(ctx context.Context, leases repository.ExecutionLeaseRepository) {
	ids := em.active.list()
	if len(ids) == 0 {
		return
	}
	touched, err := leases.TouchExecutions(ctx, ids)
	if err != nil {
		if ctx.Err() == nil {
			slog.Error("failed to renew execution leases", "count", len(ids), "error", err)
		}
		return
	}
	if touched == len(ids) {
		// Every execution is still pending or running: none was cancelled.
		return
	}
	cancelled, err := leases.CancelledExecutions(ctx, ids)
	if err != nil {
		if ctx.Err() == nil {
			slog.Error("failed to check for cancelled executions", "count", len(ids), "error", err)
		}
		return
	}
	for _, id := range cancelled {
		if em.active.cancel(id) != nil {
			slog.Info("canceling execution cancelled through another replica", "execution_id", id)
		}
	}
}

// recoverySweep fails executions abandoned by a stopped process.
func (em *ExecutionManager) recoverySweep(ctx context.Context, leases repository.ExecutionLeaseRepository, cfg RecoveryConfig) {
	ids, err := leases.FailStaleExecutions(ctx, cfg.StaleAfter, InterruptedExecutionError)
	if err != nil {
		if ctx.Err() == nil {
			slog.Error("failed to recover interrupted executions", "error", err)
		}
		return
	}
	if len(ids) > 0 {
		slog.Warn("marked interrupted executions as failed", "count", len(ids), "execution_ids", ids)
	}
}
