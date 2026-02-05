package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/serviceapi"
)

type ServiceAPIExecutionHandlers struct {
	ops *serviceapi.Operations
}

func NewServiceAPIExecutionHandlers(ops *serviceapi.Operations) *ServiceAPIExecutionHandlers {
	return &ServiceAPIExecutionHandlers{ops: ops}
}

func (h *ServiceAPIExecutionHandlers) ListExecutions(c *gin.Context) {
	params := serviceapi.ListExecutionsParams{
		Limit:  getQueryInt(c, "limit", 50),
		Offset: getQueryInt(c, "offset", 0),
	}
	if wfID := c.Query("workflow_id"); wfID != "" {
		parsed, err := uuid.Parse(wfID)
		if err != nil {
			respondAPIError(c, ErrInvalidID)
			return
		}
		params.WorkflowID = &parsed
	}
	if s := c.Query("status"); s != "" {
		params.Status = &s
	}

	result, err := h.ops.ListExecutions(c.Request.Context(), params)
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": result.Executions,
		"total":      result.Total,
		"limit":      params.Limit,
		"offset":     params.Offset,
	})
}

func (h *ServiceAPIExecutionHandlers) GetExecution(c *gin.Context) {
	executionID, ok := getParam(c, "id")
	if !ok {
		return
	}

	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	execution, err := h.ops.GetExecution(c.Request.Context(), serviceapi.GetExecutionParams{
		ExecutionID: execUUID,
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, execution)
}

func (h *ServiceAPIExecutionHandlers) StartExecution(c *gin.Context) {
	workflowID, ok := getParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		Input map[string]any `json:"input"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	execution, err := h.ops.StartExecution(c.Request.Context(), serviceapi.StartExecutionParams{
		WorkflowID: workflowID,
		Input:      req.Input,
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusAccepted, execution)
}

func (h *ServiceAPIExecutionHandlers) CancelExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution cancellation not yet implemented", http.StatusNotImplemented))
}

func (h *ServiceAPIExecutionHandlers) RetryExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution retry not yet implemented", http.StatusNotImplemented))
}
