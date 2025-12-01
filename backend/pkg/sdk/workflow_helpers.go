package sdk

import (
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgModels "github.com/smilemakc/mbflow/pkg/models"
)

// workflowToStorage converts a domain workflow to storage model for creation
func workflowToStorageForCreate(w *pkgModels.Workflow) (*models.WorkflowModel, error) {
	workflowID := uuid.New()
	storageWorkflow := models.WorkflowToStorage(w, workflowID)
	return storageWorkflow, nil
}

// workflowToStorage converts a domain workflow to storage model for update
func workflowToStorageForUpdate(w *pkgModels.Workflow) (*models.WorkflowModel, error) {
	workflowID, err := uuid.Parse(w.ID)
	if err != nil {
		return nil, pkgModels.ErrInvalidWorkflowID
	}
	storageWorkflow := models.WorkflowToStorage(w, workflowID)
	return storageWorkflow, nil
}

// workflowFromStorage converts a storage workflow to domain model
func workflowFromStorage(sw *models.WorkflowModel) *pkgModels.Workflow {
	return models.WorkflowFromStorage(sw)
}
