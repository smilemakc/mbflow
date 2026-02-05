package serviceapi

import (
	"context"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/engine"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ListExecutionsParams contains parameters for listing executions.
type ListExecutionsParams struct {
	Limit      int
	Offset     int
	WorkflowID *uuid.UUID
	Status     *string
}

// ListExecutionsResult contains the result of listing executions.
type ListExecutionsResult struct {
	Executions []*models.Execution
	Total      int
}

func (o *Operations) ListExecutions(ctx context.Context, params ListExecutionsParams) (*ListExecutionsResult, error) {
	var execModels []*storagemodels.ExecutionModel
	var err error

	if params.WorkflowID != nil {
		execModels, err = o.ExecutionRepo.FindByWorkflowID(ctx, *params.WorkflowID, params.Limit, params.Offset)
	} else if params.Status != nil {
		execModels, err = o.ExecutionRepo.FindByStatus(ctx, *params.Status, params.Limit, params.Offset)
	} else {
		execModels, err = o.ExecutionRepo.FindAll(ctx, params.Limit, params.Offset)
	}

	if err != nil {
		o.Logger.Error("Failed to list executions", "error", err, "limit", params.Limit, "offset", params.Offset)
		return nil, err
	}

	executions := make([]*models.Execution, len(execModels))
	for i, em := range execModels {
		executions[i] = engine.ExecutionModelToDomain(em)
	}

	return &ListExecutionsResult{
		Executions: executions,
		Total:      len(executions),
	}, nil
}

// GetExecutionParams contains parameters for getting an execution.
type GetExecutionParams struct {
	ExecutionID uuid.UUID
}

func (o *Operations) GetExecution(ctx context.Context, params GetExecutionParams) (*models.Execution, error) {
	execModel, err := o.ExecutionRepo.FindByIDWithRelations(ctx, params.ExecutionID)
	if err != nil {
		o.Logger.Error("Failed to find execution", "error", err, "execution_id", params.ExecutionID)
		return nil, err
	}

	execution := engine.ExecutionModelToDomain(execModel)

	workflowModel, err := o.WorkflowRepo.FindByIDWithRelations(ctx, execModel.WorkflowID)
	if err == nil && workflowModel != nil {
		nodeIDMap := make(map[string]string)
		nodeNameMap := make(map[string]string)
		nodeTypeMap := make(map[string]string)
		for _, node := range workflowModel.Nodes {
			nodeIDMap[node.ID.String()] = node.NodeID
			nodeNameMap[node.ID.String()] = node.Name
			nodeTypeMap[node.ID.String()] = node.Type
		}

		for _, ne := range execution.NodeExecutions {
			if logicalID, found := nodeIDMap[ne.NodeID]; found {
				ne.NodeID = logicalID
			}
			if nodeName, found := nodeNameMap[ne.NodeID]; found {
				ne.NodeName = nodeName
			} else if ne.NodeID != "" {
				for _, node := range workflowModel.Nodes {
					if node.NodeID == ne.NodeID {
						ne.NodeName = node.Name
						ne.NodeType = node.Type
						break
					}
				}
			}
			if nodeType, found := nodeTypeMap[ne.NodeID]; found {
				ne.NodeType = nodeType
			}
		}
	}

	return execution, nil
}

// StartExecutionParams contains parameters for starting an execution.
type StartExecutionParams struct {
	WorkflowID string
	Input      map[string]any
}

func (o *Operations) StartExecution(ctx context.Context, params StartExecutionParams) (*models.Execution, error) {
	opts := engine.DefaultExecutionOptions()
	execution, err := o.ExecutionMgr.ExecuteAsync(ctx, params.WorkflowID, params.Input, opts)
	if err != nil {
		o.Logger.Error("Failed to start workflow execution", "error", err, "workflow_id", params.WorkflowID)
		return nil, err
	}

	o.Logger.Info("Workflow execution started via service API", "execution_id", execution.ID, "workflow_id", params.WorkflowID)
	return execution, nil
}

// CancelExecutionParams contains parameters for cancelling an execution.
type CancelExecutionParams struct {
	ExecutionID uuid.UUID
}

func (o *Operations) CancelExecution(ctx context.Context, params CancelExecutionParams) error {
	return NewNotImplementedError("execution cancellation not yet implemented")
}

// RetryExecutionParams contains parameters for retrying an execution.
type RetryExecutionParams struct {
	ExecutionID uuid.UUID
}

func (o *Operations) RetryExecution(ctx context.Context, params RetryExecutionParams) error {
	return NewNotImplementedError("execution retry not yet implemented")
}
