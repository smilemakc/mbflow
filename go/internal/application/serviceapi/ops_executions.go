package serviceapi

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
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
		executions[i] = storagemodels.ExecutionModelToDomain(em)
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

	execution := storagemodels.ExecutionModelToDomain(execModel)

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

// WebhookSubscription defines a per-execution webhook callback configuration.
type WebhookSubscription struct {
	URL     string            `json:"url"`
	Events  []string          `json:"events,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	NodeIDs []string          `json:"node_ids,omitempty"`
}

// StartExecutionParams contains parameters for starting an execution.
type StartExecutionParams struct {
	WorkflowID string
	Input      map[string]any
	Webhooks   []WebhookSubscription
}

func (o *Operations) StartExecution(ctx context.Context, params StartExecutionParams) (*models.Execution, error) {
	// Validate webhook subscriptions
	if err := validateWebhooks(params.Webhooks); err != nil {
		return nil, err
	}

	opts := engine.DefaultExecutionOptions()

	// Convert serviceapi webhooks to engine webhooks
	if len(params.Webhooks) > 0 {
		opts.Webhooks = make([]engine.WebhookSubscription, len(params.Webhooks))
		for i, wh := range params.Webhooks {
			opts.Webhooks[i] = engine.WebhookSubscription{
				URL:     wh.URL,
				Events:  wh.Events,
				Headers: wh.Headers,
				NodeIDs: wh.NodeIDs,
			}
		}
	}

	execution, err := o.ExecutionMgr.ExecuteAsync(ctx, params.WorkflowID, params.Input, opts)
	if err != nil {
		o.Logger.Error("Failed to start workflow execution", "error", err, "workflow_id", params.WorkflowID)
		return nil, err
	}

	o.Logger.Info("Workflow execution started via service API", "execution_id", execution.ID, "workflow_id", params.WorkflowID)
	return execution, nil
}

// validateWebhooks validates webhook subscription configurations.
func validateWebhooks(webhooks []WebhookSubscription) error {
	for i, wh := range webhooks {
		if wh.URL == "" {
			return NewValidationError("INVALID_WEBHOOK", fmt.Sprintf("webhook[%d]: url is required", i))
		}
		if _, err := url.Parse(wh.URL); err != nil {
			return NewValidationError("INVALID_WEBHOOK", fmt.Sprintf("webhook[%d]: invalid url: %s", i, err))
		}
		for _, evt := range wh.Events {
			if !isValidEventType(evt) {
				return NewValidationError("INVALID_WEBHOOK", fmt.Sprintf("webhook[%d]: unknown event type %q", i, evt))
			}
		}
	}
	return nil
}

var validEventTypes = map[string]bool{
	"execution.started":   true,
	"execution.completed": true,
	"execution.failed":    true,
	"wave.started":        true,
	"wave.completed":      true,
	"node.started":        true,
	"node.completed":      true,
	"node.failed":         true,
	"node.skipped":        true,
	"node.retrying":       true,
}

func isValidEventType(s string) bool {
	return validEventTypes[s]
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
			nodeExec := storagemodels.NodeExecutionModelToDomain(ne)
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
