package serviceapi

import (
	"context"
	"fmt"
	"time"

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

type GetExecutionLogsParams struct {
	ExecutionID uuid.UUID
}

type ExecutionLogEntry struct {
	Timestamp time.Time
	EventType string
	Level     string
	Message   string
	Data      map[string]any
}

type GetExecutionLogsResult struct {
	Logs  []ExecutionLogEntry
	Total int
}

func (o *Operations) GetExecutionLogs(ctx context.Context, params GetExecutionLogsParams) (*GetExecutionLogsResult, error) {
	events, err := o.ExecutionRepo.GetEvents(ctx, params.ExecutionID)
	if err != nil {
		o.Logger.Error("Failed to get execution events", "error", err, "execution_id", params.ExecutionID)
		return &GetExecutionLogsResult{Logs: []ExecutionLogEntry{}, Total: 0}, nil
	}

	logs := make([]ExecutionLogEntry, 0, len(events))
	for _, event := range events {
		logs = append(logs, ExecutionLogEntry{
			Timestamp: event.CreatedAt,
			EventType: event.EventType,
			Level:     getLogLevel(event.EventType),
			Message:   formatLogMessage(event.EventType, map[string]any(event.Payload)),
			Data:      map[string]any(event.Payload),
		})
	}

	return &GetExecutionLogsResult{Logs: logs, Total: len(logs)}, nil
}

type GetNodeResultParams struct {
	ExecutionID uuid.UUID
	NodeID      string
}

func (o *Operations) GetNodeResult(ctx context.Context, params GetNodeResultParams) (*models.NodeExecution, error) {
	execModel, err := o.ExecutionRepo.FindByIDWithRelations(ctx, params.ExecutionID)
	if err != nil {
		o.Logger.Error("Failed to find execution in GetNodeResult", "error", err, "execution_id", params.ExecutionID)
		return nil, err
	}

	workflowModel, err := o.WorkflowRepo.FindByIDWithRelations(ctx, execModel.WorkflowID)
	if err != nil {
		o.Logger.Error("Failed to find workflow in GetNodeResult", "error", err, "workflow_id", execModel.WorkflowID)
		return nil, err
	}

	nodeIDMap := make(map[uuid.UUID]string)
	for _, node := range workflowModel.Nodes {
		nodeIDMap[node.ID] = node.NodeID
	}

	for _, ne := range execModel.NodeExecutions {
		if logicalID, ok := nodeIDMap[ne.NodeID]; ok && logicalID == params.NodeID {
			nodeExec := engine.NodeExecutionModelToDomain(ne)
			nodeExec.NodeID = params.NodeID
			return nodeExec, nil
		}
	}

	return nil, NewValidationError("NODE_EXECUTION_NOT_FOUND", "Node execution not found")
}

func getLogLevel(eventType string) string {
	switch eventType {
	case "execution.failed", "node.failed":
		return "error"
	case "execution.completed", "node.completed", "wave.completed":
		return "success"
	case "execution.started", "node.started", "wave.started":
		return "info"
	case "node.retrying":
		return "warning"
	default:
		return "info"
	}
}

func formatLogMessage(eventType string, payload map[string]any) string {
	switch eventType {
	case "execution.started":
		return "Execution started"
	case "execution.completed":
		if duration, ok := payload["duration_ms"].(float64); ok {
			return fmt.Sprintf("Execution completed in %dms", int64(duration))
		}
		return "Execution completed"
	case "execution.failed":
		if errMsg, ok := payload["error"].(string); ok {
			return fmt.Sprintf("Execution failed: %s", errMsg)
		}
		return "Execution failed"
	case "wave.started":
		if waveIdx, ok := payload["wave_index"].(float64); ok {
			if nodeCount, ok := payload["node_count"].(float64); ok {
				return fmt.Sprintf("Wave %d started with %d nodes", int(waveIdx), int(nodeCount))
			}
			return fmt.Sprintf("Wave %d started", int(waveIdx))
		}
		return "Wave started"
	case "wave.completed":
		if waveIdx, ok := payload["wave_index"].(float64); ok {
			return fmt.Sprintf("Wave %d completed", int(waveIdx))
		}
		return "Wave completed"
	case "node.started":
		if nodeName, ok := payload["node_name"].(string); ok {
			return fmt.Sprintf("Node '%s' started", nodeName)
		}
		return "Node started"
	case "node.completed":
		if nodeName, ok := payload["node_name"].(string); ok {
			if duration, ok := payload["duration_ms"].(float64); ok {
				return fmt.Sprintf("Node '%s' completed in %dms", nodeName, int64(duration))
			}
			return fmt.Sprintf("Node '%s' completed", nodeName)
		}
		return "Node completed"
	case "node.failed":
		if nodeName, ok := payload["node_name"].(string); ok {
			if errMsg, ok := payload["error"].(string); ok {
				return fmt.Sprintf("Node '%s' failed: %s", nodeName, errMsg)
			}
			return fmt.Sprintf("Node '%s' failed", nodeName)
		}
		return "Node failed"
	case "node.retrying":
		if nodeName, ok := payload["node_name"].(string); ok {
			return fmt.Sprintf("Node '%s' retrying", nodeName)
		}
		return "Node retrying"
	default:
		return eventType
	}
}
