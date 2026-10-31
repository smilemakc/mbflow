package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrExecutionAlreadyCancelled is returned by ExecutionRepository.Update of a
// repository that implements ExecutionLeaseRepository when the execution was
// cancelled (see CancelExecution) and the update would give it another status.
var ErrExecutionAlreadyCancelled = errors.New("execution was cancelled")

// ExecutionLeaseRepository supports lease-based crash recovery and
// cross-replica cancellation of executions.
//
// A process that runs executions periodically renews their lease by touching
// updated_at; any replica may then fail executions whose lease expired, which
// means the process that owned them died. It is a separate interface so that
// implementations of ExecutionRepository are not required to support it.
//
// Implementations must make cancellation sticky: once an execution is
// cancelled, ExecutionRepository.Update may only store it as cancelled again
// (to record its final node executions) and returns
// ErrExecutionAlreadyCancelled, without changing anything, for any other
// status.
type ExecutionLeaseRepository interface {
	// TouchExecutions sets updated_at to the current database time for the
	// given executions that are still pending or running, and returns the
	// number of rows renewed.
	TouchExecutions(ctx context.Context, ids []uuid.UUID) (int, error)

	// FailStaleExecutions marks every pending or running execution whose
	// updated_at is older than staleAfter (by database time) as failed with
	// errMsg, and returns the IDs of the executions it changed.
	FailStaleExecutions(ctx context.Context, staleAfter time.Duration, errMsg string) ([]uuid.UUID, error)

	// CancelExecution atomically marks the execution as cancelled with errMsg
	// and a completion time if it is still pending or running. It reports
	// whether the execution was changed; false means it does not exist or
	// already reached a terminal status.
	CancelExecution(ctx context.Context, id uuid.UUID, errMsg string) (bool, error)

	// CancelledExecutions returns those of the given executions whose status
	// is cancelled.
	CancelledExecutions(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error)
}
