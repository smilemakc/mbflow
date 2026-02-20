package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
)

type ExecutionHandlers struct {
	ops    *serviceapi.Operations
	logger *logger.Logger
}

func NewExecutionHandlers(ops *serviceapi.Operations, log *logger.Logger) *ExecutionHandlers {
	return &ExecutionHandlers{ops: ops, logger: log}
}

// HandleRunExecution starts a new workflow execution
//
//	@Summary		Start workflow execution
//	@Description	Starts a new execution of the specified workflow with optional input parameters
//	@Tags			executions
//	@Accept			json
//	@Produce		json
//	@Param			workflow_id	path		string												false	"Workflow ID (can also be provided in body)"	format(uuid)
//	@Param			request		body		object{workflow_id=string,input=object,async=bool}	true	"Execution request"
//	@Success		202			{object}	models.Execution									"Started execution"
//	@Failure		400			{object}	APIError											"Invalid request"
//	@Failure		404			{object}	APIError											"Workflow not found"
//	@Failure		500			{object}	APIError											"Internal server error"
//	@Security		BearerAuth
//	@Router			/executions [post]
//	@Router			/workflows/{workflow_id}/execute [post]
func (h *ExecutionHandlers) HandleRunExecution(c *gin.Context) {
	var req struct {
		WorkflowID string `json:"workflow_id"`
		Input      map[string]any `json:"input"`
		Async      bool   `json:"async"`
		Webhooks   []struct {
			URL     string            `json:"url"`
			Events  []string          `json:"events,omitempty"`
			Headers map[string]string `json:"headers,omitempty"`
			NodeIDs []string          `json:"node_ids,omitempty"`
		} `json:"webhooks,omitempty"`
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

	params := serviceapi.StartExecutionParams{
		WorkflowID: req.WorkflowID,
		Input:      req.Input,
	}

	if len(req.Webhooks) > 0 {
		params.Webhooks = make([]serviceapi.WebhookSubscription, len(req.Webhooks))
		for i, wh := range req.Webhooks {
			params.Webhooks[i] = serviceapi.WebhookSubscription{
				URL:     wh.URL,
				Events:  wh.Events,
				Headers: wh.Headers,
				NodeIDs: wh.NodeIDs,
			}
		}
	}

	execution, err := h.ops.StartExecution(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to start workflow execution", "error", err, "workflow_id", req.WorkflowID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Workflow execution started", "execution_id", execution.ID, "workflow_id", req.WorkflowID, "request_id", GetRequestID(c))
	respondJSON(c, http.StatusAccepted, execution)
}

// HandleGetExecution retrieves an execution by ID
//
//	@Summary		Get execution by ID
//	@Description	Retrieves a specific workflow execution by its unique identifier
//	@Tags			executions
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string				true	"Execution ID"	format(uuid)
//	@Success		200	{object}	models.Execution	"Execution details"
//	@Failure		400	{object}	APIError			"Invalid execution ID"
//	@Failure		404	{object}	APIError			"Execution not found"
//	@Failure		500	{object}	APIError			"Internal server error"
//	@Security		BearerAuth
//	@Router			/executions/{id} [get]
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

// HandleListExecutions lists executions with optional filtering
//
//	@Summary		List executions
//	@Description	Lists all executions with optional filtering by workflow ID and status
//	@Tags			executions
//	@Accept			json
//	@Produce		json
//	@Param			limit		query		int		false	"Maximum number of results"	default(50)
//	@Param			offset		query		int		false	"Offset for pagination"		default(0)
//	@Param			workflow_id	query		string	false	"Filter by workflow ID"		format(uuid)
//	@Param			status		query		string	false	"Filter by status"
//	@Success		200			{object}	object{data=[]models.Execution,total=int,limit=int,offset=int}	"List of executions"
//	@Failure		400			{object}	APIError													"Invalid request"
//	@Failure		500			{object}	APIError													"Internal server error"
//	@Security		BearerAuth
//	@Router			/executions [get]
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

// HandleGetLogs retrieves logs for an execution
//
//	@Summary		Get execution logs
//	@Description	Retrieves all log entries for a specific workflow execution
//	@Tags			executions
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string										true	"Execution ID"	format(uuid)
//	@Success		200	{object}	object{logs=[]object,total=int}				"Execution logs"
//	@Failure		400	{object}	APIError									"Invalid execution ID"
//	@Failure		404	{object}	APIError									"Execution not found"
//	@Failure		500	{object}	APIError									"Internal server error"
//	@Security		BearerAuth
//	@Router			/executions/{id}/logs [get]
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
