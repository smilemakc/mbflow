package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/serviceapi"
)

type ServiceAPIWorkflowHandlers struct {
	ops *serviceapi.Operations
}

func NewServiceAPIWorkflowHandlers(ops *serviceapi.Operations) *ServiceAPIWorkflowHandlers {
	return &ServiceAPIWorkflowHandlers{ops: ops}
}

func (h *ServiceAPIWorkflowHandlers) ListWorkflows(c *gin.Context) {
	params := serviceapi.ListWorkflowsParams{
		Limit:  getQueryInt(c, "limit", 50),
		Offset: getQueryInt(c, "offset", 0),
	}
	if s := c.Query("status"); s != "" {
		params.Status = &s
	}
	if uid := c.Query("user_id"); uid != "" {
		parsed, err := uuid.Parse(uid)
		if err != nil {
			respondAPIError(c, NewAPIError("INVALID_USER_ID", "Invalid user_id format", http.StatusBadRequest))
			return
		}
		params.UserID = &parsed
	}

	result, err := h.ops.ListWorkflows(c.Request.Context(), params)
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workflows": result.Workflows,
		"total":     result.Total,
		"limit":     params.Limit,
		"offset":    params.Offset,
	})
}

func (h *ServiceAPIWorkflowHandlers) GetWorkflow(c *gin.Context) {
	workflowID, ok := getParam(c, "id")
	if !ok {
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	workflow, err := h.ops.GetWorkflow(c.Request.Context(), serviceapi.GetWorkflowParams{
		WorkflowID: workflowUUID,
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, workflow)
}

func (h *ServiceAPIWorkflowHandlers) CreateWorkflow(c *gin.Context) {
	var req struct {
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Variables   map[string]any `json:"variables,omitempty"`
		Metadata    map[string]any `json:"metadata,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	var createdBy *uuid.UUID
	if userID, ok := GetUserIDAsUUID(c); ok {
		createdBy = &userID
	}

	workflow, err := h.ops.CreateWorkflow(c.Request.Context(), serviceapi.CreateWorkflowParams{
		Name:        req.Name,
		Description: req.Description,
		Variables:   req.Variables,
		Metadata:    req.Metadata,
		CreatedBy:   createdBy,
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusCreated, workflow)
}

func (h *ServiceAPIWorkflowHandlers) UpdateWorkflow(c *gin.Context) {
	workflowID, ok := getParam(c, "id")
	if !ok {
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req UpdateWorkflowRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	var nodes []serviceapi.NodeInput
	if req.Nodes != nil {
		nodes = make([]serviceapi.NodeInput, len(req.Nodes))
		for i, n := range req.Nodes {
			nodes[i] = serviceapi.NodeInput{
				ID: n.ID, Name: n.Name, Type: n.Type,
				Config: n.Config, Position: n.Position,
			}
		}
	}

	var edges []serviceapi.EdgeInput
	if req.Edges != nil {
		edges = make([]serviceapi.EdgeInput, len(req.Edges))
		for i, e := range req.Edges {
			edges[i] = serviceapi.EdgeInput{
				ID: e.ID, From: e.From, To: e.To,
				Condition: e.Condition,
			}
		}
	}

	var resources []serviceapi.ResourceInput
	if req.Resources != nil {
		resources = make([]serviceapi.ResourceInput, len(req.Resources))
		for i, r := range req.Resources {
			resources[i] = serviceapi.ResourceInput{
				ResourceID: r.ResourceID, Alias: r.Alias, AccessType: r.AccessType,
			}
		}
	}

	workflow, err := h.ops.UpdateWorkflow(c.Request.Context(), serviceapi.UpdateWorkflowParams{
		WorkflowID:  workflowUUID,
		Name:        req.Name,
		Description: req.Description,
		Variables:   req.Variables,
		Metadata:    req.Metadata,
		Nodes:       nodes,
		Edges:       edges,
		Resources:   resources,
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, workflow)
}

func (h *ServiceAPIWorkflowHandlers) DeleteWorkflow(c *gin.Context) {
	workflowID, ok := getParam(c, "id")
	if !ok {
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	if err := h.ops.DeleteWorkflow(c.Request.Context(), serviceapi.DeleteWorkflowParams{
		WorkflowID: workflowUUID,
	}); err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "workflow deleted successfully"})
}
