package serviceapi

import (
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
}
