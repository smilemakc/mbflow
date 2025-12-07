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
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/visualization"
)

// WorkflowHandlers provides HTTP handlers for workflow-related endpoints
type WorkflowHandlers struct {
	workflowRepo repository.WorkflowRepository
	logger       *logger.Logger
}

// NewWorkflowHandlers creates a new WorkflowHandlers instance
func NewWorkflowHandlers(workflowRepo repository.WorkflowRepository, log *logger.Logger) *WorkflowHandlers {
	return &WorkflowHandlers{
		workflowRepo: workflowRepo,
		logger:       log,
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

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind JSON in CreateWorkflow", "error", err)
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(c, http.StatusBadRequest, "name is required")
		return
	}

	// Create workflow model
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

	if err := h.workflowRepo.Create(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to create workflow", "error", err, "workflow_name", req.Name)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain model
	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusCreated, workflow)
}

// HandleGetWorkflow handles GET /api/v1/workflows/{id}
func (h *WorkflowHandlers) HandleGetWorkflow(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format in GetWorkflow", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusOK, workflow)
}

// HandleListWorkflows handles GET /api/v1/workflows
func (h *WorkflowHandlers) HandleListWorkflows(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)
	status := c.Query("status")

	var workflowModels []*storagemodels.WorkflowModel
	var err error

	if status != "" {
		workflowModels, err = h.workflowRepo.FindByStatus(c.Request.Context(), status, limit, offset)
	} else {
		workflowModels, err = h.workflowRepo.FindAll(c.Request.Context(), limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to list workflows", "error", err, "status", status, "limit", limit, "offset", offset)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain models
	workflows := make([]*models.Workflow, len(workflowModels))
	for i, wm := range workflowModels {
		workflows[i] = engine.WorkflowModelToDomain(wm)
	}

	// Get total count
	var total int
	if status != "" {
		total, err = h.workflowRepo.CountByStatus(c.Request.Context(), status)
	} else {
		total, err = h.workflowRepo.Count(c.Request.Context())
	}
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
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	var req UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind JSON in UpdateWorkflow", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate nodes if provided
	if err := h.validateNodes(req.Nodes); err != nil {
		h.logger.Error("Node validation failed in UpdateWorkflow", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate edges if provided
	if err := h.validateEdges(req.Edges, req.Nodes); err != nil {
		h.logger.Error("Edge validation failed in UpdateWorkflow", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Fetch existing workflow
	workflowModel, err := h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow for update", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusNotFound, "workflow not found")
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

	// Update workflow (repository handles smart merge of nodes and edges)
	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to update workflow", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch updated workflow with relations to return complete data
	updatedWorkflow, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to fetch updated workflow", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, "failed to fetch updated workflow")
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

	nodeIDs := make(map[string]bool)
	validTypes := map[string]bool{
		"http":         true,
		"transform":    true,
		"llm":          true,
		"conditional":  true,
		"merge":        true,
		"split":        true,
		"delay":        true,
		"webhook":      true,
		"telegram":     true,
		"file_storage": true,
	}

	for i, node := range nodes {
		// Check required fields
		if node.ID == "" {
			return fmt.Errorf("node at index %d: id is required", i)
		}
		if node.Name == "" {
			return fmt.Errorf("node at index %d: name is required", i)
		}
		if node.Type == "" {
			return fmt.Errorf("node at index %d: type is required", i)
		}

		// Check for duplicate node IDs
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}
		nodeIDs[node.ID] = true

		// Validate node type
		if !validTypes[node.Type] {
			return fmt.Errorf("node %s: invalid type '%s'", node.ID, node.Type)
		}

		// Validate field lengths
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
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Soft delete
	if err := h.workflowRepo.Delete(c.Request.Context(), workflowUUID); err != nil {
		h.logger.Error("Failed to delete workflow", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "workflow deleted successfully",
	})
}

// HandlePublishWorkflow handles POST /api/v1/workflows/{id}/publish
func (h *WorkflowHandlers) HandlePublishWorkflow(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Fetch workflow
	workflowModel, err := h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow for publish", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Change status to active
	workflowModel.Status = "active"

	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to publish workflow", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusOK, workflow)
}

// HandleUnpublishWorkflow handles POST /api/v1/workflows/{id}/unpublish
func (h *WorkflowHandlers) HandleUnpublishWorkflow(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Fetch workflow
	workflowModel, err := h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow for unpublish", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Change status to draft
	workflowModel.Status = "draft"

	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to unpublish workflow", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
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
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Fetch workflow with relations (nodes and edges)
	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow for diagram", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusNotFound, "workflow not found")
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
		h.logger.Error("Failed to render workflow diagram", "error", err, "workflow_id", workflowUUID, "format", format)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Return as plain text
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, diagram)
}
