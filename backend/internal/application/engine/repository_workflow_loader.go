package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgengine "github.com/smilemakc/mbflow/pkg/engine"
	"github.com/smilemakc/mbflow/pkg/models"
)

// RepositoryWorkflowLoader loads workflows from the database repository.
// Implements pkg/engine.WorkflowLoader interface for sub-workflow execution.
type RepositoryWorkflowLoader struct {
	workflowRepo repository.WorkflowRepository
}

// NewRepositoryWorkflowLoader creates a new loader backed by a workflow repository.
func NewRepositoryWorkflowLoader(workflowRepo repository.WorkflowRepository) pkgengine.WorkflowLoader {
	return &RepositoryWorkflowLoader{workflowRepo: workflowRepo}
}

// LoadWorkflow loads a workflow by ID string, converting from storage to domain model.
func (l *RepositoryWorkflowLoader) LoadWorkflow(ctx context.Context, workflowID string) (*models.Workflow, error) {
	wfUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow ID %q: %w", workflowID, err)
	}

	workflowModel, err := l.workflowRepo.FindByIDWithRelations(ctx, wfUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow %s: %w", workflowID, err)
	}

	workflow := storagemodels.WorkflowModelToDomain(workflowModel)
	if workflow == nil {
		return nil, fmt.Errorf("workflow %s conversion returned nil", workflowID)
	}

	return workflow, nil
}
