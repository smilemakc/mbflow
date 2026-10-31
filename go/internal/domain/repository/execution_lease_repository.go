package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ExecutionLeaseRepository supports lease-based crash recovery of executions.
//
// A process that runs executions periodically renews their lease by touching
// updated_at; any replica may then fail executions whose lease expired, which
// means the process that owned them died. It is a separate interface so that
// implementations of ExecutionRepository are not required to support it.
type ExecutionLeaseRepository interface {
	// TouchExecutions sets updated_at to the current database time for the
	// given executions that are still pending or running, and returns the
	// number of rows renewed.
	TouchExecutions(ctx context.Context, ids []uuid.UUID) (int, error)

	// FailStaleExecutions marks every pending or running execution whose
	// updated_at is older than staleAfter (by database time) as failed with
	// errMsg, and returns the IDs of the executions it changed.
	FailStaleExecutions(ctx context.Context, staleAfter time.Duration, errMsg string) ([]uuid.UUID, error)
}
