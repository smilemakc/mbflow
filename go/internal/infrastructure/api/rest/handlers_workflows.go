package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/visualization"
)

type WorkflowHandlers struct {
	ops    *serviceapi.Operations
	logger *logger.Logger
}

func NewWorkflowHandlers(ops *serviceapi.Operations, log *logger.Logger) *WorkflowHandlers {
	return &WorkflowHandlers{ops: ops, logger: log}
}

// HandleCreateWorkflow creates a new workflow
//
//	@Summary		Create a new workflow
//	@Description	Creates a new workflow with the specified name and optional description, variables, and metadata
//	@Tags			workflows
//	@Accept			json
//	@Produce		json
//	@Param			request	body		object{name=string,description=string,variables=object,metadata=object}	true	"Workflow creation request"
//	@Success		201		{object}	models.Workflow											"Created workflow"
//	@Failure		400		{object}	APIError												"Invalid request"
//	@Failure		401		{object}	APIError												"Unauthorized"
//	@Failure		500		{object}	APIError												"Internal server error"
//	@Security		BearerAuth
//	@Router			/workflows [post]
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

// HandleGetWorkflow retrieves a workflow by ID
//
//	@Summary		Get workflow by ID
//	@Description	Retrieves a specific workflow by its unique identifier
//	@Tags			workflows
//	@Accept			json
//	@Produce		json
//	@Param			workflow_id	path		string			true	"Workflow ID"	format(uuid)
//	@Success		200			{object}	models.Workflow	"Workflow details"
//	@Failure		400			{object}	APIError		"Invalid workflow ID"
//	@Failure		404			{object}	APIError		"Workflow not found"
//	@Failure		500			{object}	APIError		"Internal server error"
//	@Security		BearerAuth
//	@Router			/workflows/{workflow_id} [get]
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

// HandleListWorkflows lists all workflows with optional filtering
//
//	@Summary		List workflows
//	@Description	Lists all workflows with optional filtering by status and user ID
//	@Tags			workflows
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int		false	"Maximum number of results"	default(50)
//	@Param			offset	query		int		false	"Offset for pagination"		default(0)
//	@Param			status	query		string	false	"Filter by status"
//	@Param			user_id	query		string	false	"Filter by user ID"			format(uuid)
//	@Success		200		{object}	object{data=[]models.Workflow,total=int,limit=int,offset=int}	"List of workflows"
//	@Failure		400		{object}	APIError													"Invalid request"
//	@Failure		401		{object}	APIError													"Unauthorized"
//	@Failure		500		{object}	APIError													"Internal server error"
//	@Security		BearerAuth
//	@Router			/workflows [get]
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
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Variables   map[string]any    `json:"variables,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	Nodes       []NodeRequest     `json:"nodes,omitempty"`
	Edges       []EdgeRequest     `json:"edges,omitempty"`
	Resources   []ResourceRequest `json:"resources,omitempty"`
}

type ResourceRequest struct {
	ResourceID string `json:"resource_id" binding:"required"`
	Alias      string `json:"alias" binding:"required,min=1,max=100"`
	AccessType string `json:"access_type" binding:"omitempty,oneof=read write admin"`
}

type NodeRequest struct {
	ID       string         `json:"id" binding:"required,max=100"`
	Name     string         `json:"name" binding:"required,max=255"`
	Type     string         `json:"type" binding:"required"`
	Config   map[string]any `json:"config,omitempty"`
	Position map[string]any `json:"position,omitempty"`
}

type EdgeRequest struct {
	ID           string         `json:"id" binding:"required,max=100"`
	From         string         `json:"from" binding:"required,max=100"`
	To           string         `json:"to" binding:"required,max=100"`
	SourceHandle string         `json:"source_handle,omitempty"`
	Condition    map[string]any `json:"condition,omitempty"`
	Loop         *struct {
		MaxIterations int `json:"max_iterations"`
	} `json:"loop,omitempty"`
}

// HandleUpdateWorkflow updates an existing workflow
//
//	@Summary		Update workflow
//	@Description	Updates a workflow's name, description, variables, metadata, nodes, edges, and resources
//	@Tags			workflows
//	@Accept			json
//	@Produce		json
//	@Param			workflow_id	path		string					true	"Workflow ID"	format(uuid)
//	@Param			request		body		UpdateWorkflowRequest	true	"Workflow update request"
//	@Success		200			{object}	models.Workflow			"Updated workflow"
//	@Failure		400			{object}	APIError				"Invalid request"
//	@Failure		404			{object}	APIError				"Workflow not found"
//	@Failure		500			{object}	APIError				"Internal server error"
//	@Security		BearerAuth
//	@Router			/workflows/{workflow_id} [put]
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
			ei := serviceapi.EdgeInput{
				ID:           e.ID,
				From:         e.From,
				To:           e.To,
				SourceHandle: e.SourceHandle,
				Condition:    e.Condition,
			}
			if e.Loop != nil {
				ei.Loop = &serviceapi.LoopInput{MaxIterations: e.Loop.MaxIterations}
			}
			params.Edges[i] = ei
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

// HandleDeleteWorkflow deletes a workflow by ID
//
//	@Summary		Delete workflow
//	@Description	Deletes a specific workflow by its unique identifier
//	@Tags			workflows
//	@Accept			json
//	@Produce		json
//	@Param			workflow_id	path		string					true	"Workflow ID"	format(uuid)
//	@Success		200			{object}	object{message=string}	"Success message"
//	@Failure		400			{object}	APIError				"Invalid workflow ID"
//	@Failure		404			{object}	APIError				"Workflow not found"
//	@Failure		500			{object}	APIError				"Internal server error"
//	@Security		BearerAuth
//	@Router			/workflows/{workflow_id} [delete]
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

// HandlePublishWorkflow publishes a workflow
//
//	@Summary		Publish workflow
//	@Description	Publishes a workflow, making it available for execution
//	@Tags			workflows
//	@Accept			json
//	@Produce		json
//	@Param			workflow_id	path		string			true	"Workflow ID"	format(uuid)
//	@Success		200			{object}	models.Workflow	"Published workflow"
//	@Failure		400			{object}	APIError		"Invalid workflow ID or workflow cannot be published"
//	@Failure		404			{object}	APIError		"Workflow not found"
//	@Failure		500			{object}	APIError		"Internal server error"
//	@Security		BearerAuth
//	@Router			/workflows/{workflow_id}/publish [post]
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

// HandleUnpublishWorkflow unpublishes a workflow
//
//	@Summary		Unpublish workflow
//	@Description	Unpublishes a workflow, making it unavailable for new executions
//	@Tags			workflows
//	@Accept			json
//	@Produce		json
//	@Param			workflow_id	path		string			true	"Workflow ID"	format(uuid)
//	@Success		200			{object}	models.Workflow	"Unpublished workflow"
//	@Failure		400			{object}	APIError		"Invalid workflow ID"
//	@Failure		404			{object}	APIError		"Workflow not found"
//	@Failure		500			{object}	APIError		"Internal server error"
//	@Security		BearerAuth
//	@Router			/workflows/{workflow_id}/unpublish [post]
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

// HandleGetWorkflowDiagram generates a diagram representation of a workflow
//
//	@Summary		Get workflow diagram
//	@Description	Generates a visual diagram representation of a workflow in the specified format
//	@Tags			workflows
//	@Accept			json
//	@Produce		text/plain
//	@Param			workflow_id		path		string	true	"Workflow ID"								format(uuid)
//	@Param			format			query		string	false	"Diagram format (mermaid, dot, ascii)"		default(mermaid)
//	@Param			direction		query		string	false	"Diagram direction (TB, LR, BT, RL)"		default(TB)
//	@Param			show_config		query		bool	false	"Show node configuration in diagram"		default(true)
//	@Param			show_conditions	query		bool	false	"Show edge conditions in diagram"			default(true)
//	@Param			compact			query		bool	false	"Use compact mode"							default(false)
//	@Success		200				{string}	string	"Diagram representation"
//	@Failure		400				{object}	APIError	"Invalid workflow ID or format"
//	@Failure		404				{object}	APIError	"Workflow not found"
//	@Failure		500				{object}	APIError	"Internal server error"
//	@Security		BearerAuth
//	@Router			/workflows/{workflow_id}/diagram [get]
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
