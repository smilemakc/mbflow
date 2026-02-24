package rest

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/pkg/models"
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

	respondList(c, http.StatusOK, result.Executions, result.Total, params.Limit, params.Offset)
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
		Input     map[string]any `json:"input"`
		Variables map[string]any `json:"variables,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	execution, err := h.ops.StartExecution(c.Request.Context(), serviceapi.StartExecutionParams{
		WorkflowID: workflowID,
		Input:      req.Input,
		Variables:  req.Variables,
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusAccepted, execution)
}

func (h *ServiceAPIExecutionHandlers) StartEphemeralExecution(c *gin.Context) {
	var req struct {
		Workflow         json.RawMessage   `json:"workflow"`
		Input            map[string]any    `json:"input"`
		Mode             string            `json:"mode"`
		CredentialIDs    []string          `json:"credential_ids"`
		Variables        map[string]any    `json:"variables"`
		PersistExecution bool              `json:"persist_execution"`
		Webhooks         []struct {
			URL     string            `json:"url"`
			Events  []string          `json:"events,omitempty"`
			Headers map[string]string `json:"headers,omitempty"`
			NodeIDs []string          `json:"node_ids,omitempty"`
		} `json:"webhooks,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	if req.Mode == "" {
		respondAPIError(c, NewAPIError("MODE_REQUIRED", "mode is required (sync or async)", http.StatusBadRequest))
		return
	}

	if req.Mode != "sync" && req.Mode != "async" {
		respondAPIError(c, NewAPIError("INVALID_MODE", "mode must be sync or async", http.StatusBadRequest))
		return
	}

	if len(req.Workflow) > maxWorkflowSnapshotSize {
		respondAPIError(c, NewAPIError("WORKFLOW_TOO_LARGE", "workflow snapshot exceeds 1 MB limit", http.StatusRequestEntityTooLarge))
		return
	}

	var workflow models.Workflow
	if err := json.Unmarshal(req.Workflow, &workflow); err != nil {
		respondAPIError(c, NewAPIError("INVALID_WORKFLOW", "failed to parse workflow: "+err.Error(), http.StatusBadRequest))
		return
	}

	for i, edge := range workflow.Edges {
		if edge.ID == "" {
			workflow.Edges[i].ID = uuid.New().String()
		}
	}

	params := serviceapi.EphemeralExecutionParams{
		Workflow:         &workflow,
		Input:            req.Input,
		Mode:             req.Mode,
		CredentialIDs:    req.CredentialIDs,
		Variables:        req.Variables,
		PersistExecution: req.PersistExecution,
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

	execution, err := h.ops.StartEphemeralExecution(c.Request.Context(), params)
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	status := http.StatusAccepted
	if req.Mode == "sync" {
		status = http.StatusOK
	}

	respondJSON(c, status, execution)
}

func (h *ServiceAPIExecutionHandlers) CancelExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution cancellation not yet implemented", http.StatusNotImplemented))
}

func (h *ServiceAPIExecutionHandlers) RetryExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution retry not yet implemented", http.StatusNotImplemented))
}
