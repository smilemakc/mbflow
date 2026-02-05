package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
)

type ExecutionHandlers struct {
	ops    *serviceapi.Operations
	logger *logger.Logger
}

func NewExecutionHandlers(ops *serviceapi.Operations, log *logger.Logger) *ExecutionHandlers {
	return &ExecutionHandlers{ops: ops, logger: log}
}

func (h *ExecutionHandlers) HandleRunExecution(c *gin.Context) {
	var req struct {
		WorkflowID string         `json:"workflow_id"`
		Input      map[string]any `json:"input"`
		Async      bool           `json:"async"`
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

	execution, err := h.ops.StartExecution(c.Request.Context(), serviceapi.StartExecutionParams{
		WorkflowID: req.WorkflowID,
		Input:      req.Input,
	})
	if err != nil {
		h.logger.Error("Failed to start workflow execution", "error", err, "workflow_id", req.WorkflowID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Workflow execution started", "execution_id", execution.ID, "workflow_id", req.WorkflowID, "request_id", GetRequestID(c))
	respondJSON(c, http.StatusAccepted, execution)
}

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

	execution, err := h.ops.GetExecution(c.Request.Context(), serviceapi.GetExecutionParams{
		ExecutionID: execUUID,
	})
	if err != nil {
		h.logger.Error("Failed to find execution", "error", err, "execution_id", execUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, execution)
}

func (h *ExecutionHandlers) HandleListExecutions(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	params := serviceapi.ListExecutionsParams{
		Limit:  limit,
		Offset: offset,
	}

	if workflowID := c.Query("workflow_id"); workflowID != "" {
		wfUUID, err := uuid.Parse(workflowID)
		if err != nil {
			h.logger.Error("Invalid workflow ID in ListExecutions", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
			respondAPIError(c, ErrInvalidID)
			return
		}
		params.WorkflowID = &wfUUID
	}
	if status := c.Query("status"); status != "" {
		params.Status = &status
	}

	result, err := h.ops.ListExecutions(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to list executions", "error", err, "limit", limit, "offset", offset, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondList(c, http.StatusOK, result.Executions, result.Total, limit, offset)
}

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

	result, err := h.ops.GetExecutionLogs(c.Request.Context(), serviceapi.GetExecutionLogsParams{
		ExecutionID: execUUID,
	})
	if err != nil {
		respondJSON(c, http.StatusOK, gin.H{"logs": []any{}, "total": 0})
		return
	}

	logs := make([]gin.H, 0, len(result.Logs))
	for _, log := range result.Logs {
		logs = append(logs, gin.H{
			"timestamp":  log.Timestamp,
			"event_type": log.EventType,
			"level":      log.Level,
			"message":    log.Message,
			"data":       log.Data,
		})
	}

	respondJSON(c, http.StatusOK, gin.H{"logs": logs, "total": result.Total})
}

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

	nodeExec, err := h.ops.GetNodeResult(c.Request.Context(), serviceapi.GetNodeResultParams{
		ExecutionID: execUUID,
		NodeID:      nodeID,
	})
	if err != nil {
		h.logger.Error("Failed to get node result", "error", err, "execution_id", execUUID, "node_id", nodeID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, nodeExec)
}

func (h *ExecutionHandlers) HandleCancelExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution cancellation not yet implemented", http.StatusNotImplemented))
}

func (h *ExecutionHandlers) HandleRetryExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution retry not yet implemented", http.StatusNotImplemented))
}

func (h *ExecutionHandlers) HandleWatchExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "real-time execution watching not yet implemented", http.StatusNotImplemented))
}

func (h *ExecutionHandlers) HandleStreamLogs(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "log streaming not yet implemented", http.StatusNotImplemented))
}
