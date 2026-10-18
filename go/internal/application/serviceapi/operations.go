package serviceapi

import (
	"context"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/systemkey"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/pkg/executor"
)

// Operations provides transport-agnostic business logic for the Service API.
// Both REST and gRPC handlers delegate to these operations.
type Operations struct {
	WorkflowRepo    repository.WorkflowRepository
	ExecutionRepo   repository.ExecutionRepository
	TriggerRepo     repository.TriggerRepository
	CredentialsRepo repository.CredentialsRepository
	ExecutionMgr    *engine.ExecutionManager
	ExecutorManager executor.Manager
	EncryptionSvc   *crypto.EncryptionService
	AuditService    *systemkey.AuditService
	Logger          *logger.Logger
	// TriggerSync applies trigger changes to the running schedulers. Nil
	// when the trigger manager is disabled (e.g. Redis unavailable).
	TriggerSync TriggerSyncer
}

// TriggerSyncer reconciles running trigger handlers with stored triggers.
type TriggerSyncer interface {
	SyncTrigger(ctx context.Context, triggerID string) error
}

// syncTrigger applies a stored trigger change to the running schedulers.
// Failures are logged: the change is persisted and applies after a restart.
func (o *Operations) syncTrigger(ctx context.Context, triggerID string) {
	if o.TriggerSync == nil {
		return
	}
	if err := o.TriggerSync.SyncTrigger(ctx, triggerID); err != nil && o.Logger != nil {
		o.Logger.Error("Failed to apply trigger change to scheduler", "error", err, "trigger_id", triggerID)
	}
}
