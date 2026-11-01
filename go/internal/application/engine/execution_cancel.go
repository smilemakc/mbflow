package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// CancelledExecutionError is stored on executions cancelled by CancelExecution.
const CancelledExecutionError = "execution cancelled"

// ShutdownCancelledError is stored on executions that were still running when
// the shutdown deadline expired.
const ShutdownCancelledError = "execution cancelled: server shut down before it finished"

// errCancelRequested is the cancellation cause of executions cancelled by
// CancelExecution.
var errCancelRequested = models.ErrExecutionCancelled

// ErrExecutionNotCancellable is returned by CancelExecution when the execution
// does not exist or is no longer pending or running.
var ErrExecutionNotCancellable = errors.New("execution is not pending or running")

// ErrCancelUnsupported is returned by CancelExecution for an execution this
// process does not run when the execution repository does not implement
// repository.ExecutionLeaseRepository, so it cannot be cancelled elsewhere.
var ErrCancelUnsupported = errors.New("execution is not running in this process and the execution repository does not support cross-replica cancellation")

// CancelOutcome tells how far a cancellation got when CancelExecution returned.
type CancelOutcome int

const (
	// CancelFinished means the execution ran in this process, stopped and
	// has been finalized. Its stored status is cancelled, unless it finished
	// on its own before the cancellation took effect.
	CancelFinished CancelOutcome = iota + 1
	// CancelPending means the execution is stored as cancelled, but it may
	// still be stopping: it runs on another replica (which stops it within
	// one heartbeat interval) or it did not stop within the wait.
	CancelPending
)

// cancelWait bounds how long CancelExecution waits for a local execution to
// stop. Tests shorten it.
var cancelWait = 10 * time.Second

// CancelExecution cancels a pending or running persisted execution. It is
// safe with several replicas sharing a database:
//
//   - if this process runs (or queued) the execution, its context is
//     cancelled and CancelExecution waits (up to ctx and a few seconds) for it
//     to be finalized with status cancelled, its node executions recorded and
//     an execution.cancelled event emitted;
//   - otherwise the execution is marked cancelled in the database with a
//     conditional update, which fails with ErrExecutionNotCancellable if it
//     is not pending or running any more. The replica that runs it notices on
//     its next lease heartbeat (see StartRecovery), so it stops within one
//     heartbeat interval, and then records its node executions. Without
//     recovery running the owner keeps running it, but cannot store it with
//     any status other than cancelled.
func (em *ExecutionManager) CancelExecution(ctx context.Context, executionID string) (CancelOutcome, error) {
	id, err := uuid.Parse(executionID)
	if err != nil {
		return 0, fmt.Errorf("invalid execution ID: %w", err)
	}
	if done := em.active.cancel(id); done != nil {
		return waitCancelled(ctx, done), nil
	}

	leases, ok := em.executionRepo.(repository.ExecutionLeaseRepository)
	if !ok {
		return 0, ErrCancelUnsupported
	}
	changed, err := leases.CancelExecution(ctx, id, CancelledExecutionError)
	if err != nil {
		return 0, fmt.Errorf("failed to cancel execution: %w", err)
	}
	if !changed {
		return 0, ErrExecutionNotCancellable
	}
	// The execution may have started tracking here after the check above.
	if done := em.active.cancel(id); done != nil {
		return waitCancelled(ctx, done), nil
	}
	return CancelPending, nil
}

func waitCancelled(ctx context.Context, done <-chan struct{}) CancelOutcome {
	timer := time.NewTimer(cancelWait)
	defer timer.Stop()
	select {
	case <-done:
		return CancelFinished
	case <-ctx.Done():
	case <-timer.C:
	}
	return CancelPending
}

// cancellationReason reports whether an execution whose run context is ctx
// was cancelled, by CancelExecution or by the shutdown deadline, rather than
// failing, and the error to store on it.
func cancellationReason(ctx context.Context) (bool, string) {
	if ctx == nil || ctx.Err() == nil {
		return false, ""
	}
	switch cause := context.Cause(ctx); {
	case errors.Is(cause, errCancelRequested):
		return true, CancelledExecutionError
	case errors.Is(cause, ErrShuttingDown):
		return true, ShutdownCancelledError
	}
	return false, ""
}

// queuedCancelCause returns why a queued execution never got a slot.
func queuedCancelCause(ctx context.Context, err error) error {
	if errors.Is(context.Cause(ctx), errCancelRequested) {
		return errCancelRequested
	}
	return err
}

// persistFinalState stores an execution's final state. If the execution was
// cancelled through another replica meanwhile, it is recorded as cancelled
// instead, keeping its node executions.
func (em *ExecutionManager) persistFinalState(ctx context.Context, execution *models.Execution) error {
	err := em.executionRepo.Update(ctx, storagemodels.ExecutionDomainToModel(execution))
	if !errors.Is(err, repository.ErrExecutionAlreadyCancelled) {
		return err
	}
	execution.Status = models.ExecutionStatusCancelled
	execution.Error = CancelledExecutionError
	execution.Output = nil
	return em.executionRepo.Update(ctx, storagemodels.ExecutionDomainToModel(execution))
}
