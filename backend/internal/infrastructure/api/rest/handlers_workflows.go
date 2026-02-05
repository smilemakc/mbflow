package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/visualization"
)

type WorkflowHandlers struct {
	ops    *serviceapi.Operations
	logger *logger.Logger
}

func NewWorkflowHandlers(ops *serviceapi.Operations, log *logger.Logger) *WorkflowHandlers {
	return &WorkflowHandlers{ops: ops, logger: log}
}

func (h *WorkflowHandlers) HandleCreateWorkflow(c *gin.Context) {
	var req struct {
		Name        string         `json:"name" binding:"required"`
		Description string         `json:"description,omitempty"`
		Variables   map[string]any `json:"variables,omitempty"`
		Metadata    map[string]any `json:"metadata,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	params := serviceapi.CreateWorkflowParams{
		Name:        req.Name,
		Description: req.Description,
		Variables:   req.Variables,
		Metadata:    req.Metadata,
	}

	if userID, ok := GetUserIDAsUUID(c); ok {
		params.CreatedBy = &userID
	}

	workflow, err := h.ops.CreateWorkflow(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to create workflow", "error", err, "workflow_name", req.Name, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusCreated, workflow)
}

func (h *WorkflowHandlers) HandleGetWorkflow(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	workflow, err := h.ops.GetWorkflow(c.Request.Context(), serviceapi.GetWorkflowParams{
		WorkflowID: workflowUUID,
	})
	if err != nil {
		h.logger.Error("Failed to find workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, workflow)
}

func (h *WorkflowHandlers) HandleListWorkflows(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)
	status := c.Query("status")
	userIDParam := c.Query("user_id")

	currentUserID, isAuthenticated := GetUserIDAsUUID(c)
	isAdmin := IsAdmin(c)

	params := serviceapi.ListWorkflowsParams{
		Limit:  limit,
		Offset: offset,
	}

	if status != "" {
		params.Status = &status
	}

	if userIDParam != "" {
		requestedUserID, err := uuid.Parse(userIDParam)
		if err != nil {
			respondAPIError(c, NewAPIError("INVALID_USER_ID", "Invalid user_id format", http.StatusBadRequest))
			return
		}

		if !isAdmin && isAuthenticated && requestedUserID != currentUserID {
			respondAPIError(c, NewAPIError("FORBIDDEN", "You can only view your own workflows", http.StatusForbidden))
			return
		}

		params.UserID = &requestedUserID
	}

	result, err := h.ops.ListWorkflows(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to list workflows", "error", err, "limit", limit, "offset", offset, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondList(c, http.StatusOK, result.Workflows, result.Total, limit, offset)
}

type UpdateWorkflowRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Nodes       []NodeRequest          `json:"nodes,omitempty"`
	Edges       []EdgeRequest          `json:"edges,omitempty"`
	Resources   []ResourceRequest      `json:"resources,omitempty"`
}

type ResourceRequest struct {
	ResourceID string `json:"resource_id" binding:"required"`
	Alias      string `json:"alias" binding:"required,min=1,max=100"`
	AccessType string `json:"access_type" binding:"omitempty,oneof=read write admin"`
}

type NodeRequest struct {
	ID       string                 `json:"id" binding:"required,max=100"`
	Name     string                 `json:"name" binding:"required,max=255"`
	Type     string                 `json:"type" binding:"required"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Position map[string]interface{} `json:"position,omitempty"`
}

type EdgeRequest struct {
	ID        string                 `json:"id" binding:"required,max=100"`
	From      string                 `json:"from" binding:"required,max=100"`
	To        string                 `json:"to" binding:"required,max=100"`
	Condition map[string]interface{} `json:"condition,omitempty"`
}

func (h *WorkflowHandlers) HandleUpdateWorkflow(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req UpdateWorkflowRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	params := serviceapi.UpdateWorkflowParams{
		WorkflowID:  workflowUUID,
		Name:        req.Name,
		Description: req.Description,
		Variables:   req.Variables,
		Metadata:    req.Metadata,
	}

	if req.Nodes != nil {
		params.Nodes = make([]serviceapi.NodeInput, len(req.Nodes))
		for i, n := range req.Nodes {
			params.Nodes[i] = serviceapi.NodeInput{
				ID:       n.ID,
				Name:     n.Name,
				Type:     n.Type,
				Config:   n.Config,
				Position: n.Position,
			}
		}
	}

	if req.Edges != nil {
		params.Edges = make([]serviceapi.EdgeInput, len(req.Edges))
		for i, e := range req.Edges {
			params.Edges[i] = serviceapi.EdgeInput{
				ID:        e.ID,
				From:      e.From,
				To:        e.To,
				Condition: e.Condition,
			}
		}
	}

	if req.Resources != nil {
		params.Resources = make([]serviceapi.ResourceInput, len(req.Resources))
		for i, r := range req.Resources {
			params.Resources[i] = serviceapi.ResourceInput{
				ResourceID: r.ResourceID,
				Alias:      r.Alias,
				AccessType: r.AccessType,
			}
		}
	}

	workflow, err := h.ops.UpdateWorkflow(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to update workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, workflow)
}

func (h *WorkflowHandlers) HandleDeleteWorkflow(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	if err := h.ops.DeleteWorkflow(c.Request.Context(), serviceapi.DeleteWorkflowParams{
		WorkflowID: workflowUUID,
	}); err != nil {
		h.logger.Error("Failed to delete workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "workflow deleted successfully"})
}

func (h *WorkflowHandlers) HandlePublishWorkflow(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	workflow, err := h.ops.PublishWorkflow(c.Request.Context(), serviceapi.PublishWorkflowParams{
		WorkflowID: workflowUUID,
	})
	if err != nil {
		h.logger.Error("Failed to publish workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, workflow)
}

func (h *WorkflowHandlers) HandleUnpublishWorkflow(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	workflow, err := h.ops.UnpublishWorkflow(c.Request.Context(), serviceapi.UnpublishWorkflowParams{
		WorkflowID: workflowUUID,
	})
	if err != nil {
		h.logger.Error("Failed to unpublish workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, workflow)
}

func (h *WorkflowHandlers) HandleGetWorkflowDiagram(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	workflow, err := h.ops.GetWorkflow(c.Request.Context(), serviceapi.GetWorkflowParams{
		WorkflowID: workflowUUID,
	})
	if err != nil {
		h.logger.Error("Failed to find workflow for diagram", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	format := c.DefaultQuery("format", "mermaid")
	direction := c.DefaultQuery("direction", "TB")
	showConfig := c.DefaultQuery("show_config", "true") == "true"
	showConditions := c.DefaultQuery("show_conditions", "true") == "true"
	compact := c.DefaultQuery("compact", "false") == "true"

	opts := &visualization.RenderOptions{
		ShowConfig:     showConfig,
		ShowConditions: showConditions,
		CompactMode:    compact,
		Direction:      direction,
		UseColor:       false,
	}

	diagram, err := visualization.RenderWorkflow(workflow, format, opts)
	if err != nil {
		h.logger.Error("Failed to render workflow diagram", "error", err, "workflow_id", workflowUUID, "format", format, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, diagram)
}

type AttachResourceRequest struct {
	ResourceID string `json:"resource_id" binding:"required,uuid"`
	Alias      string `json:"alias" binding:"required,min=1,max=100"`
	AccessType string `json:"access_type" binding:"omitempty,oneof=read write admin"`
}

func (h *WorkflowHandlers) AttachWorkflowResource(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondAPIError(c, ErrUnauthorized)
		return
	}

	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req AttachResourceRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	resourceUUID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		respondAPIError(c, NewAPIError("INVALID_RESOURCE_ID", "Invalid resource ID format", http.StatusBadRequest))
		return
	}

	workflowResource := &models.WorkflowResource{
		ResourceID: req.ResourceID,
		Alias:      req.Alias,
		AccessType: req.AccessType,
	}
	if workflowResource.AccessType == "" {
		workflowResource.AccessType = "read"
	}
	if err := workflowResource.Validate(); err != nil {
		respondAPIError(c, NewAPIError("VALIDATION_FAILED", err.Error(), http.StatusBadRequest))
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		userUUID = uuid.Nil
	}

	var assignedBy *uuid.UUID
	if userUUID != uuid.Nil {
		assignedBy = &userUUID
	}

	accessType := req.AccessType
	if accessType == "" {
		accessType = "read"
	}

	if err := h.ops.AttachWorkflowResource(c.Request.Context(), serviceapi.AttachWorkflowResourceParams{
		WorkflowID: workflowUUID,
		ResourceID: resourceUUID,
		Alias:      req.Alias,
		AccessType: accessType,
		AssignedBy: assignedBy,
	}); err != nil {
		h.logger.Error("Failed to attach resource", "error", err, "workflow_id", workflowUUID, "resource_id", req.ResourceID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Resource attached to workflow", "workflow_id", workflowUUID, "resource_id", req.ResourceID, "alias", req.Alias, "request_id", GetRequestID(c))
	respondJSON(c, http.StatusCreated, gin.H{
		"resource_id": req.ResourceID,
		"alias":       req.Alias,
		"access_type": accessType,
	})
}

func (h *WorkflowHandlers) DetachWorkflowResource(c *gin.Context) {
	_, ok := GetUserID(c)
	if !ok {
		respondAPIError(c, ErrUnauthorized)
		return
	}

	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	resourceID := c.Param("resource_id")
	if resourceID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	resourceUUID, err := uuid.Parse(resourceID)
	if err != nil {
		h.logger.Error("Invalid resource ID format", "error", err, "resource_id", resourceID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	if err := h.ops.DetachWorkflowResource(c.Request.Context(), serviceapi.DetachWorkflowResourceParams{
		WorkflowID: workflowUUID,
		ResourceID: resourceUUID,
	}); err != nil {
		h.logger.Error("Failed to detach resource", "error", err, "workflow_id", workflowUUID, "resource_id", resourceID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Resource detached from workflow", "workflow_id", workflowUUID, "resource_id", resourceID, "request_id", GetRequestID(c))
	respondJSON(c, http.StatusOK, gin.H{"message": "resource detached successfully"})
}

func (h *WorkflowHandlers) GetWorkflowResources(c *gin.Context) {
	_, ok := GetUserID(c)
	if !ok {
		respondAPIError(c, ErrUnauthorized)
		return
	}

	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	result, err := h.ops.GetWorkflowResources(c.Request.Context(), serviceapi.GetWorkflowResourcesParams{
		WorkflowID: workflowUUID,
	})
	if err != nil {
		h.logger.Error("Failed to get workflow resources", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	response := make([]gin.H, len(result.Resources))
	for i, r := range result.Resources {
		response[i] = gin.H{
			"resource_id": r.ResourceID,
			"alias":       r.Alias,
			"access_type": r.AccessType,
		}
	}

	respondJSON(c, http.StatusOK, gin.H{"resources": response})
}

type UpdateResourceAliasRequest struct {
	Alias string `json:"alias" binding:"required,min=1,max=100"`
}

func (h *WorkflowHandlers) UpdateWorkflowResourceAlias(c *gin.Context) {
	_, ok := GetUserID(c)
	if !ok {
		respondAPIError(c, ErrUnauthorized)
		return
	}

	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	resourceID := c.Param("resource_id")
	if resourceID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	resourceUUID, err := uuid.Parse(resourceID)
	if err != nil {
		h.logger.Error("Invalid resource ID format", "error", err, "resource_id", resourceID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req UpdateResourceAliasRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	tempResource := &models.WorkflowResource{ResourceID: resourceID, Alias: req.Alias, AccessType: "read"}
	if err := tempResource.Validate(); err != nil {
		respondAPIError(c, NewAPIError("VALIDATION_FAILED", err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.ops.UpdateWorkflowResourceAlias(c.Request.Context(), serviceapi.UpdateWorkflowResourceAliasParams{
		WorkflowID: workflowUUID,
		ResourceID: resourceUUID,
		Alias:      req.Alias,
	}); err != nil {
		h.logger.Error("Failed to update resource alias", "error", err, "workflow_id", workflowUUID, "resource_id", resourceID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Resource alias updated", "workflow_id", workflowUUID, "resource_id", resourceID, "new_alias", req.Alias, "request_id", GetRequestID(c))
	respondJSON(c, http.StatusOK, gin.H{
		"resource_id": resourceID,
		"alias":       req.Alias,
	})
}
