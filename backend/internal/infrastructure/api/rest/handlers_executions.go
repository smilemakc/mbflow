package rest

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecutionHandlers provides HTTP handlers for execution-related endpoints
type ExecutionHandlers struct {
	executionRepo    repository.ExecutionRepository
	workflowRepo     repository.WorkflowRepository
	executionManager *engine.ExecutionManager
	logger           *logger.Logger
}

// NewExecutionHandlers creates a new ExecutionHandlers instance
func NewExecutionHandlers(
	executionRepo repository.ExecutionRepository,
	workflowRepo repository.WorkflowRepository,
	executionManager *engine.ExecutionManager,
	log *logger.Logger,
) *ExecutionHandlers {
	return &ExecutionHandlers{
		executionRepo:    executionRepo,
		workflowRepo:     workflowRepo,
		executionManager: executionManager,
		logger:           log,
	}
}

// HandleRunExecution handles POST /api/v1/executions
func (h *ExecutionHandlers) HandleRunExecution(c *gin.Context) {
	var req struct {
		WorkflowID string                 `json:"workflow_id"`
		Input      map[string]interface{} `json:"input"`
		Async      bool                   `json:"async"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	if workflowID := c.Param("workflow_id"); workflowID != "" {
		req.WorkflowID = workflowID
	}

	if req.WorkflowID == "" {
		respondAPIError(c, NewAPIError("WORKFLOW_ID_REQUIRED", "Workflow ID is required", http.StatusBadRequest))
		return
	}

	opts := engine.DefaultExecutionOptions()
	execution, err := h.executionManager.ExecuteAsync(c.Request.Context(), req.WorkflowID, req.Input, opts)
	if err != nil {
		h.logger.Error("Failed to start workflow execution", "error", err, "workflow_id", req.WorkflowID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Workflow execution started", "execution_id", execution.ID, "workflow_id", req.WorkflowID, "request_id", GetRequestID(c))
	respondJSON(c, http.StatusAccepted, execution)
}

// HandleGetExecution handles GET /api/v1/executions/{id}
func (h *ExecutionHandlers) HandleGetExecution(c *gin.Context) {
	executionID := c.Param("id")
	if executionID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		h.logger.Error("Invalid execution ID", "error", err, "execution_id", executionID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	execModel, err := h.executionRepo.FindByIDWithRelations(c.Request.Context(), execUUID)
	if err != nil {
		h.logger.Error("Failed to find execution", "error", err, "execution_id", execUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	execution := engine.ExecutionModelToDomain(execModel)

	// Map node UUIDs to logical IDs for frontend compatibility
	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), execModel.WorkflowID)
	if err == nil && workflowModel != nil {
		// Build node UUID -> logical ID mapping
		nodeIDMap := make(map[string]string)
		nodeNameMap := make(map[string]string)
		nodeTypeMap := make(map[string]string)
		for _, node := range workflowModel.Nodes {
			nodeIDMap[node.ID.String()] = node.NodeID
			nodeNameMap[node.ID.String()] = node.Name
			nodeTypeMap[node.ID.String()] = node.Type
		}

		// Replace UUIDs with logical IDs in node executions
		for _, ne := range execution.NodeExecutions {
			if logicalID, ok := nodeIDMap[ne.NodeID]; ok {
				ne.NodeID = logicalID
			}
			if nodeName, ok := nodeNameMap[ne.NodeID]; ok {
				ne.NodeName = nodeName
			} else if ne.NodeID != "" {
				// If we already have logical ID, try to find name by it
				for _, node := range workflowModel.Nodes {
					if node.NodeID == ne.NodeID {
						ne.NodeName = node.Name
						ne.NodeType = node.Type
						break
					}
				}
			}
			if nodeType, ok := nodeTypeMap[ne.NodeID]; ok {
				ne.NodeType = nodeType
			}
		}
	}

	respondJSON(c, http.StatusOK, execution)
}

// HandleListExecutions handles GET /api/v1/executions
func (h *ExecutionHandlers) HandleListExecutions(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)
	workflowID := c.Query("workflow_id")
	status := c.Query("status")

	var execModels []*storagemodels.ExecutionModel
	var err error

	if workflowID != "" {
		wfUUID, parseErr := uuid.Parse(workflowID)
		if parseErr != nil {
			h.logger.Error("Invalid workflow ID in ListExecutions", "error", parseErr, "workflow_id", workflowID, "request_id", GetRequestID(c))
			respondAPIError(c, ErrInvalidID)
			return
		}
		execModels, err = h.executionRepo.FindByWorkflowID(c.Request.Context(), wfUUID, limit, offset)
	} else if status != "" {
		execModels, err = h.executionRepo.FindByStatus(c.Request.Context(), status, limit, offset)
	} else {
		execModels, err = h.executionRepo.FindAll(c.Request.Context(), limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to list executions", "error", err, "workflow_id", workflowID, "status", status, "limit", limit, "offset", offset, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	// Convert to domain models
	executions := make([]*models.Execution, len(execModels))
	for i, em := range execModels {
		executions[i] = engine.ExecutionModelToDomain(em)
	}

	respondList(c, http.StatusOK, executions, len(executions), limit, offset)
}

// HandleGetLogs handles GET /api/v1/executions/{id}/logs
func (h *ExecutionHandlers) HandleGetLogs(c *gin.Context) {
	executionID := c.Param("id")
	if executionID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		h.logger.Error("Invalid execution ID in GetLogs", "error", err, "execution_id", executionID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	// Get events from repository
	events, err := h.executionRepo.GetEvents(c.Request.Context(), execUUID)
	if err != nil {
		h.logger.Error("Failed to get execution events", "error", err, "execution_id", execUUID, "request_id", GetRequestID(c))
		// Return empty array instead of error for better UX
		respondJSON(c, http.StatusOK, gin.H{
			"logs":  []interface{}{},
			"total": 0,
		})
		return
	}

	// Convert events to log format
	logs := make([]gin.H, 0, len(events))
	for _, event := range events {
		logEntry := gin.H{
			"timestamp":  event.CreatedAt,
			"event_type": event.EventType,
			"level":      getLogLevel(event.EventType),
			"message":    formatLogMessage(event.EventType, event.Payload),
			"data":       event.Payload,
		}
		logs = append(logs, logEntry)
	}

	respondJSON(c, http.StatusOK, gin.H{
		"logs":  logs,
		"total": len(logs),
	})
}

// getLogLevel determines log level based on event type
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

// formatLogMessage creates a human-readable message from event
func formatLogMessage(eventType string, payload storagemodels.JSONBMap) string {
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

// HandleGetNodeResult handles GET /api/v1/executions/{id}/nodes/{nodeId}
func (h *ExecutionHandlers) HandleGetNodeResult(c *gin.Context) {
	executionID := c.Param("id")
	nodeID := c.Param("nodeId")

	if executionID == "" || nodeID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		h.logger.Error("Invalid execution ID in GetNodeResult", "error", err, "execution_id", executionID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	execModel, err := h.executionRepo.FindByIDWithRelations(c.Request.Context(), execUUID)
	if err != nil {
		h.logger.Error("Failed to find execution in GetNodeResult", "error", err, "execution_id", execUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), execModel.WorkflowID)
	if err != nil {
		h.logger.Error("Failed to find workflow in GetNodeResult", "error", err, "workflow_id", execModel.WorkflowID, "execution_id", execUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	// Build node ID mapping (UUID -> logical ID)
	nodeIDMap := make(map[uuid.UUID]string)
	for _, node := range workflowModel.Nodes {
		nodeIDMap[node.ID] = node.NodeID
	}

	// Find matching node execution
	for _, ne := range execModel.NodeExecutions {
		if logicalID, ok := nodeIDMap[ne.NodeID]; ok && logicalID == nodeID {
			nodeExec := engine.NodeExecutionModelToDomain(ne)
			nodeExec.NodeID = nodeID // Replace UUID with logical ID
			respondJSON(c, http.StatusOK, nodeExec)
			return
		}
	}

	respondAPIError(c, NewAPIError("NODE_EXECUTION_NOT_FOUND", "Node execution not found", http.StatusNotFound))
}

// HandleCancelExecution handles POST /api/v1/executions/{id}/cancel (deferred)
func (h *ExecutionHandlers) HandleCancelExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution cancellation not yet implemented", http.StatusNotImplemented))
}

// HandleRetryExecution handles POST /api/v1/executions/{id}/retry (deferred)
func (h *ExecutionHandlers) HandleRetryExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution retry not yet implemented", http.StatusNotImplemented))
}

// HandleWatchExecution handles GET /api/v1/executions/{id}/watch (deferred)
func (h *ExecutionHandlers) HandleWatchExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "real-time execution watching not yet implemented", http.StatusNotImplemented))
}

// HandleStreamLogs handles GET /api/v1/executions/{id}/logs/stream (deferred)
func (h *ExecutionHandlers) HandleStreamLogs(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "log streaming not yet implemented", http.StatusNotImplemented))
}
