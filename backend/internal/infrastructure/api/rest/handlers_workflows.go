package rest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/visualization"
)

// WorkflowHandlers provides HTTP handlers for workflow-related endpoints
type WorkflowHandlers struct {
	workflowRepo    repository.WorkflowRepository
	logger          *logger.Logger
	executorManager executor.Manager
}

// NewWorkflowHandlers creates a new WorkflowHandlers instance
func NewWorkflowHandlers(workflowRepo repository.WorkflowRepository, log *logger.Logger, executorManager executor.Manager) *WorkflowHandlers {
	return &WorkflowHandlers{
		workflowRepo:    workflowRepo,
		logger:          log,
		executorManager: executorManager,
	}
}

// HandleCreateWorkflow handles POST /api/v1/workflows
func (h *WorkflowHandlers) HandleCreateWorkflow(c *gin.Context) {
	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description,omitempty"`
		Variables   map[string]interface{} `json:"variables,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	if req.Name == "" {
		respondAPIError(c, NewAPIError("NAME_REQUIRED", "Workflow name is required", http.StatusBadRequest))
		return
	}

	workflowModel := &storagemodels.WorkflowModel{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Status:      "draft",
		Version:     1,
		Variables:   storagemodels.JSONBMap(req.Variables),
		Metadata:    storagemodels.JSONBMap(req.Metadata),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set created_by if user is authenticated (optional auth)
	if userID, ok := GetUserIDAsUUID(c); ok {
		workflowModel.CreatedBy = &userID
	}

	if err := h.workflowRepo.Create(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to create workflow", "error", err, "workflow_name", req.Name, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusCreated, workflow)
}

// HandleGetWorkflow handles GET /api/v1/workflows/{id}
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

	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusOK, workflow)
}

// HandleListWorkflows handles GET /api/v1/workflows
// Query parameters:
//   - limit: int (default 50)
//   - offset: int (default 0)
//   - status: string (optional)
//   - user_id: uuid (optional, filter by creator)
func (h *WorkflowHandlers) HandleListWorkflows(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)
	status := c.Query("status")
	userIDParam := c.Query("user_id")

	// Get current user info (may be empty if not authenticated)
	currentUserID, isAuthenticated := GetUserIDAsUUID(c)
	isAdmin := IsAdmin(c)

	// Build filters
	filters := repository.WorkflowFilters{
		IncludeUnowned: true, // Include legacy workflows without owner by default
	}

	// Apply status filter if provided
	if status != "" {
		filters.Status = &status
	}

	// Handle user_id filter with authorization
	if userIDParam != "" {
		requestedUserID, err := uuid.Parse(userIDParam)
		if err != nil {
			respondAPIError(c, NewAPIError("INVALID_USER_ID", "Invalid user_id format", http.StatusBadRequest))
			return
		}

		// Authorization check:
		// - Admins can query any user's workflows
		// - Non-admins can only query their own workflows
		if !isAdmin && isAuthenticated && requestedUserID != currentUserID {
			respondAPIError(c, NewAPIError("FORBIDDEN", "You can only view your own workflows", http.StatusForbidden))
			return
		}

		filters.CreatedBy = &requestedUserID
		filters.IncludeUnowned = false // When filtering by specific user, don't include unowned
	} else if isAuthenticated && !isAdmin {
		// Non-admin authenticated user without user_id filter:
		// Show only their own workflows + unowned (legacy) workflows
		filters.CreatedBy = &currentUserID
		filters.IncludeUnowned = true
	}
	// Admins and unauthenticated users without user_id filter see all workflows

	// Execute query
	workflowModels, err := h.workflowRepo.FindAllWithFilters(c.Request.Context(), filters, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list workflows", "error", err, "filters", filters, "limit", limit, "offset", offset, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflows := make([]*models.Workflow, len(workflowModels))
	for i, wm := range workflowModels {
		workflows[i] = engine.WorkflowModelToDomain(wm)
	}

	// Get total count with same filters
	total, err := h.workflowRepo.CountWithFilters(c.Request.Context(), filters)
	if err != nil {
		total = len(workflows)
	}

	c.JSON(http.StatusOK, gin.H{
		"workflows": workflows,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// UpdateWorkflowRequest represents the request body for updating a workflow
type UpdateWorkflowRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Nodes       []NodeRequest          `json:"nodes,omitempty"`
	Edges       []EdgeRequest          `json:"edges,omitempty"`
	Resources   []ResourceRequest      `json:"resources,omitempty"`
}

// ResourceRequest represents a resource attachment in the request body
type ResourceRequest struct {
	ResourceID string `json:"resource_id" validate:"required"`
	Alias      string `json:"alias" validate:"required,min=1,max=100"`
	AccessType string `json:"access_type" validate:"omitempty,oneof=read write admin"`
}

// NodeRequest represents a node in the request body
type NodeRequest struct {
	ID       string                 `json:"id" validate:"required,max=100"`
	Name     string                 `json:"name" validate:"required,max=255"`
	Type     string                 `json:"type" validate:"required,oneof=http transform llm conditional merge split delay webhook"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Position map[string]interface{} `json:"position,omitempty"`
}

// EdgeRequest represents an edge in the request body
type EdgeRequest struct {
	ID        string                 `json:"id" validate:"required,max=100"`
	From      string                 `json:"from" validate:"required,max=100"`
	To        string                 `json:"to" validate:"required,max=100"`
	Condition map[string]interface{} `json:"condition,omitempty"`
}

// HandleUpdateWorkflow handles PUT /api/v1/workflows/{id}
// Updates a workflow including its metadata, nodes, and edges.
// The repository performs smart merge:
// - Existing nodes/edges (by ID): preserved UUID, updated fields
// - New nodes/edges: created with new UUID
// - Missing nodes/edges: deleted from database
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

	// Validate nodes if provided
	if err := h.validateNodes(req.Nodes); err != nil {
		h.logger.Error("Node validation failed in UpdateWorkflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIError(c, NewAPIError("NODE_VALIDATION_FAILED", err.Error(), http.StatusBadRequest))
		return
	}

	// Validate edges if provided
	if err := h.validateEdges(req.Edges, req.Nodes); err != nil {
		h.logger.Error("Edge validation failed in UpdateWorkflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIError(c, NewAPIError("EDGE_VALIDATION_FAILED", err.Error(), http.StatusBadRequest))
		return
	}

	// Fetch existing workflow
	workflowModel, err := h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow for update", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	// Update workflow metadata fields
	if req.Name != "" {
		workflowModel.Name = req.Name
	}
	if req.Description != "" {
		workflowModel.Description = req.Description
	}
	if req.Variables != nil {
		workflowModel.Variables = storagemodels.JSONBMap(req.Variables)
	}
	if req.Metadata != nil {
		workflowModel.Metadata = storagemodels.JSONBMap(req.Metadata)
	}

	// Update nodes if provided
	if req.Nodes != nil {
		workflowModel.Nodes = make([]*storagemodels.NodeModel, len(req.Nodes))
		for i, nodeReq := range req.Nodes {
			workflowModel.Nodes[i] = &storagemodels.NodeModel{
				NodeID:     nodeReq.ID,
				WorkflowID: workflowUUID,
				Name:       nodeReq.Name,
				Type:       nodeReq.Type,
				Config:     storagemodels.JSONBMap(nodeReq.Config),
				Position:   storagemodels.JSONBMap(nodeReq.Position),
			}
		}
	}

	// Update edges if provided
	if req.Edges != nil {
		workflowModel.Edges = make([]*storagemodels.EdgeModel, len(req.Edges))
		for i, edgeReq := range req.Edges {
			workflowModel.Edges[i] = &storagemodels.EdgeModel{
				EdgeID:     edgeReq.ID,
				WorkflowID: workflowUUID,
				FromNodeID: edgeReq.From,
				ToNodeID:   edgeReq.To,
				Condition:  storagemodels.JSONBMap(edgeReq.Condition),
			}
		}
	}

	// Update resources if provided
	if req.Resources != nil {
		workflowModel.Resources = make([]*storagemodels.WorkflowResourceModel, len(req.Resources))
		for i, resReq := range req.Resources {
			resourceUUID, err := uuid.Parse(resReq.ResourceID)
			if err != nil {
				respondAPIError(c, NewAPIError("INVALID_RESOURCE_ID", fmt.Sprintf("invalid resource_id: %s", resReq.ResourceID), http.StatusBadRequest))
				return
			}

			accessType := resReq.AccessType
			if accessType == "" {
				accessType = "read"
			}

			workflowModel.Resources[i] = &storagemodels.WorkflowResourceModel{
				WorkflowID: workflowUUID,
				ResourceID: resourceUUID,
				Alias:      resReq.Alias,
				AccessType: accessType,
			}
		}
	}

	// Update workflow (repository handles smart merge of nodes and edges)
	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to update workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	// Fetch updated workflow with relations to return complete data
	updatedWorkflow, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to fetch updated workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflow := engine.WorkflowModelToDomain(updatedWorkflow)
	respondJSON(c, http.StatusOK, workflow)
}

// validateNodes validates node data in the request
func (h *WorkflowHandlers) validateNodes(nodes []NodeRequest) error {
	if nodes == nil {
		return nil
	}

	uiOnlyTypes := map[string]bool{
		"comment": true,
	}

	nodeIDs := make(map[string]bool)

	for i, node := range nodes {
		if node.ID == "" {
			return fmt.Errorf("node at index %d: id is required", i)
		}
		if node.Name == "" {
			return fmt.Errorf("node at index %d: name is required", i)
		}
		if node.Type == "" {
			return fmt.Errorf("node at index %d: type is required", i)
		}

		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}
		nodeIDs[node.ID] = true

		if !uiOnlyTypes[node.Type] && !h.executorManager.Has(node.Type) {
			return fmt.Errorf("node %s: invalid type '%s'", node.ID, node.Type)
		}

		if len(node.ID) > 100 {
			return fmt.Errorf("node id too long (max 100 chars): %s", node.ID)
		}
		if len(node.Name) > 255 {
			return fmt.Errorf("node %s: name too long (max 255 chars)", node.ID)
		}
	}

	return nil
}

// validateEdges validates edge data in the request
func (h *WorkflowHandlers) validateEdges(edges []EdgeRequest, nodes []NodeRequest) error {
	if edges == nil {
		return nil
	}

	// Build node ID set for validation
	nodeIDSet := make(map[string]bool)
	for _, node := range nodes {
		nodeIDSet[node.ID] = true
	}

	edgeIDs := make(map[string]bool)

	for i, edge := range edges {
		// Check required fields
		if edge.ID == "" {
			return fmt.Errorf("edge at index %d: id is required", i)
		}
		if edge.From == "" {
			return fmt.Errorf("edge at index %d: from is required", i)
		}
		if edge.To == "" {
			return fmt.Errorf("edge at index %d: to is required", i)
		}

		// Check for duplicate edge IDs
		if edgeIDs[edge.ID] {
			return fmt.Errorf("duplicate edge id: %s", edge.ID)
		}
		edgeIDs[edge.ID] = true

		// Validate no self-reference
		if edge.From == edge.To {
			return fmt.Errorf("edge %s: self-reference not allowed (from=%s, to=%s)", edge.ID, edge.From, edge.To)
		}

		// If nodes are provided in the request, validate edge references
		if len(nodes) > 0 {
			if !nodeIDSet[edge.From] {
				return fmt.Errorf("edge %s: from node '%s' not found in nodes", edge.ID, edge.From)
			}
			if !nodeIDSet[edge.To] {
				return fmt.Errorf("edge %s: to node '%s' not found in nodes", edge.ID, edge.To)
			}
		}

		// Validate field lengths
		if len(edge.ID) > 100 {
			return fmt.Errorf("edge id too long (max 100 chars): %s", edge.ID)
		}
		if len(edge.From) > 100 {
			return fmt.Errorf("edge %s: from node id too long (max 100 chars)", edge.ID)
		}
		if len(edge.To) > 100 {
			return fmt.Errorf("edge %s: to node id too long (max 100 chars)", edge.ID)
		}
	}

	return nil
}

// HandleDeleteWorkflow handles DELETE /api/v1/workflows/{id}
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

	// Soft delete
	if err := h.workflowRepo.Delete(c.Request.Context(), workflowUUID); err != nil {
		h.logger.Error("Failed to delete workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, gin.H{
		"message": "workflow deleted successfully",
	})
}

// HandlePublishWorkflow handles POST /api/v1/workflows/{id}/publish
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

	// Fetch workflow
	workflowModel, err := h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow for publish", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	// Change status to active
	workflowModel.Status = "active"

	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to publish workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusOK, workflow)
}

// HandleUnpublishWorkflow handles POST /api/v1/workflows/{id}/unpublish
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

	// Fetch workflow
	workflowModel, err := h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow for unpublish", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	// Change status to draft
	workflowModel.Status = "draft"

	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to unpublish workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusOK, workflow)
}

// HandleGetWorkflowDiagram handles GET /api/v1/workflows/{id}/diagram
// Returns workflow visualization in the specified format (mermaid or ascii).
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

	// Fetch workflow with relations (nodes and edges)
	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow for diagram", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	// Convert to domain model
	workflow := engine.WorkflowModelToDomain(workflowModel)

	// Parse query parameters
	format := c.DefaultQuery("format", "mermaid")
	direction := c.DefaultQuery("direction", "TB")
	showConfig := c.DefaultQuery("show_config", "true") == "true"
	showConditions := c.DefaultQuery("show_conditions", "true") == "true"
	compact := c.DefaultQuery("compact", "false") == "true"

	// Import visualization package
	// Note: This import is done at file level at the top
	opts := &visualization.RenderOptions{
		ShowConfig:     showConfig,
		ShowConditions: showConditions,
		CompactMode:    compact,
		Direction:      direction,
		UseColor:       false, // No ANSI colors in HTTP response
	}

	// Render diagram
	diagram, err := visualization.RenderWorkflow(workflow, format, opts)
	if err != nil {
		h.logger.Error("Failed to render workflow diagram", "error", err, "workflow_id", workflowUUID, "format", format, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	// Return as plain text
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, diagram)
}

// AttachResourceRequest represents request to attach a resource to workflow
type AttachResourceRequest struct {
	ResourceID string `json:"resource_id" binding:"required,uuid"`
	Alias      string `json:"alias" binding:"required,min=1,max=100"`
	AccessType string `json:"access_type" binding:"omitempty,oneof=read write admin"`
}

// AttachWorkflowResource attaches a resource to a workflow
// POST /api/v1/workflows/:workflow_id/resources
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

	// Verify workflow exists and user has access
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	_ = userID // suppress unused variable (for future auth)

	accessType := req.AccessType
	if accessType == "" {
		accessType = "read"
	}

	workflowResource := &models.WorkflowResource{
		ResourceID: req.ResourceID,
		Alias:      req.Alias,
		AccessType: accessType,
	}

	if err := workflowResource.Validate(); err != nil {
		respondAPIError(c, NewAPIError("VALIDATION_FAILED", err.Error(), http.StatusBadRequest))
		return
	}

	resourceUUID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		respondAPIError(c, NewAPIError("INVALID_RESOURCE_ID", "Invalid resource ID format", http.StatusBadRequest))
		return
	}

	// Parse userID as UUID for assignedBy field
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		userUUID = uuid.Nil
	}

	// Create workflow resource model
	workflowResourceModel := &storagemodels.WorkflowResourceModel{
		WorkflowID: workflowUUID,
		ResourceID: resourceUUID,
		Alias:      req.Alias,
		AccessType: accessType,
	}

	var assignedBy *uuid.UUID
	if userUUID != uuid.Nil {
		assignedBy = &userUUID
	}

	if err := h.workflowRepo.AssignResource(c.Request.Context(), workflowUUID, workflowResourceModel, assignedBy); err != nil {
		h.logger.Error("Failed to attach resource", "error", err, "workflow_id", workflowUUID, "resource_id", req.ResourceID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Resource attached to workflow",
		"workflow_id", workflowUUID,
		"resource_id", req.ResourceID,
		"alias", req.Alias,
		"request_id", GetRequestID(c),
	)

	respondJSON(c, http.StatusCreated, gin.H{
		"resource_id": req.ResourceID,
		"alias":       req.Alias,
		"access_type": accessType,
	})
}

// DetachWorkflowResource removes a resource from a workflow
// DELETE /api/v1/workflows/:workflow_id/resources/:resource_id
func (h *WorkflowHandlers) DetachWorkflowResource(c *gin.Context) {
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

	// Verify workflow exists
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	_ = userID // suppress unused variable (for future auth)

	if err := h.workflowRepo.UnassignResource(c.Request.Context(), workflowUUID, resourceUUID); err != nil {
		h.logger.Error("Failed to detach resource", "error", err, "workflow_id", workflowUUID, "resource_id", resourceID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Resource detached from workflow",
		"workflow_id", workflowUUID,
		"resource_id", resourceID,
		"request_id", GetRequestID(c),
	)

	respondJSON(c, http.StatusOK, gin.H{"message": "resource detached successfully"})
}

// GetWorkflowResources returns all resources attached to a workflow
// GET /api/v1/workflows/:workflow_id/resources
func (h *WorkflowHandlers) GetWorkflowResources(c *gin.Context) {
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

	// Verify workflow exists
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	_ = userID // suppress unused variable (for future auth)

	resources, err := h.workflowRepo.GetWorkflowResources(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to get workflow resources", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	response := make([]gin.H, len(resources))
	for i, r := range resources {
		response[i] = gin.H{
			"resource_id": r.ResourceID.String(),
			"alias":       r.Alias,
			"access_type": r.AccessType,
		}
	}

	respondJSON(c, http.StatusOK, gin.H{"resources": response})
}

// UpdateResourceAliasRequest represents request to update resource alias
type UpdateResourceAliasRequest struct {
	Alias string `json:"alias" binding:"required,min=1,max=100"`
}

// UpdateWorkflowResourceAlias updates the alias of a workflow resource
// PUT /api/v1/workflows/:workflow_id/resources/:resource_id
func (h *WorkflowHandlers) UpdateWorkflowResourceAlias(c *gin.Context) {
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

	// Verify workflow exists
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	_ = userID // suppress unused variable (for future auth)

	// Validate alias format
	tempResource := &models.WorkflowResource{ResourceID: resourceID, Alias: req.Alias, AccessType: "read"}
	if err := tempResource.Validate(); err != nil {
		respondAPIError(c, NewAPIError("VALIDATION_FAILED", err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.workflowRepo.UpdateResourceAlias(c.Request.Context(), workflowUUID, resourceUUID, req.Alias); err != nil {
		h.logger.Error("Failed to update resource alias", "error", err, "workflow_id", workflowUUID, "resource_id", resourceID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Resource alias updated",
		"workflow_id", workflowUUID,
		"resource_id", resourceID,
		"new_alias", req.Alias,
		"request_id", GetRequestID(c),
	)

	respondJSON(c, http.StatusOK, gin.H{
		"resource_id": resourceID,
		"alias":       req.Alias,
	})
}
