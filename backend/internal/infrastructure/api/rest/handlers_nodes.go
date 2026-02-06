package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// NodeHandlers provides HTTP handlers for node-related endpoints
type NodeHandlers struct {
	workflowRepo repository.WorkflowRepository
	logger       *logger.Logger
}

// NewNodeHandlers creates a new NodeHandlers instance
func NewNodeHandlers(workflowRepo repository.WorkflowRepository, log *logger.Logger) *NodeHandlers {
	return &NodeHandlers{
		workflowRepo: workflowRepo,
		logger:       log,
	}
}

// HandleAddNode handles POST /api/v1/workflows/{workflow_id}/nodes
func (h *NodeHandlers) HandleAddNode(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID in AddNode", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	var req struct {
		ID          string                 `json:"id" binding:"required"`
		Name        string                 `json:"name" binding:"required"`
		Type        string                 `json:"type" binding:"required"`
		Description string                 `json:"description,omitempty"`
		Config      map[string]interface{} `json:"config"`
		Position    *models.Position       `json:"position,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	// Verify workflow exists
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Workflow not found in AddNode", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Create node model
	nodeModel := &storagemodels.NodeModel{
		ID:         uuid.New(),
		NodeID:     req.ID,
		WorkflowID: workflowUUID,
		Name:       req.Name,
		Type:       req.Type,
		Config:     storagemodels.JSONBMap(req.Config),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Set position if provided
	if req.Position != nil {
		nodeModel.Position = storagemodels.JSONBMap{
			"x": req.Position.X,
			"y": req.Position.Y,
		}
	}

	if err := h.workflowRepo.CreateNode(c.Request.Context(), nodeModel); err != nil {
		h.logger.Error("Failed to create node", "error", err, "workflow_id", workflowUUID, "node_id", req.ID)
		// Check for duplicate node ID constraint violation
		if strings.Contains(err.Error(), "uq_nodes_workflow_node_id") {
			respondError(c, http.StatusBadRequest, "node with this ID already exists")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain model
	node := storagemodels.NodeModelToDomain(nodeModel)
	respondJSON(c, http.StatusCreated, node)
}

// HandleListNodes handles GET /api/v1/workflows/{workflow_id}/nodes
func (h *NodeHandlers) HandleListNodes(c *gin.Context) {
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

	// Verify workflow exists
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Workflow not found in ListNodes", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	nodeModels, err := h.workflowRepo.FindNodesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to list nodes", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain models
	nodes := make([]*models.Node, len(nodeModels))
	for i, nm := range nodeModels {
		nodes[i] = storagemodels.NodeModelToDomain(nm)
	}

	respondList(c, http.StatusOK, nodes, len(nodes), 0, 0)
}

// HandleGetNode handles GET /api/v1/workflows/{workflow_id}/nodes/{nodeId}
func (h *NodeHandlers) HandleGetNode(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	nodeID := c.Param("nodeId")

	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	if nodeID == "" {
		respondError(c, http.StatusBadRequest, "node ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID in GetNode", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Get all nodes for the workflow
	nodeModels, err := h.workflowRepo.FindNodesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find nodes in GetNode", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Find the specific node by logical ID
	var nodeModel *storagemodels.NodeModel
	for _, nm := range nodeModels {
		if nm.NodeID == nodeID {
			nodeModel = nm
			break
		}
	}

	if nodeModel == nil {
		h.logger.Error("Node not found", "workflow_id", workflowUUID, "node_id", nodeID)
		respondError(c, http.StatusNotFound, "node not found")
		return
	}

	node := storagemodels.NodeModelToDomain(nodeModel)
	respondJSON(c, http.StatusOK, node)
}

// HandleUpdateNode handles PUT /api/v1/workflows/{workflow_id}/nodes/{nodeId}
func (h *NodeHandlers) HandleUpdateNode(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	nodeID := c.Param("nodeId")

	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	if nodeID == "" {
		respondError(c, http.StatusBadRequest, "node ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID in UpdateNode", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	var req struct {
		Name        string                 `json:"name,omitempty"`
		Type        string                 `json:"type,omitempty"`
		Description string                 `json:"description,omitempty"`
		Config      map[string]interface{} `json:"config,omitempty"`
		Position    *models.Position       `json:"position,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	// Get all nodes for the workflow
	nodeModels, err := h.workflowRepo.FindNodesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find nodes in UpdateNode", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Find the specific node by logical ID
	var nodeModel *storagemodels.NodeModel
	for _, nm := range nodeModels {
		if nm.NodeID == nodeID {
			nodeModel = nm
			break
		}
	}

	if nodeModel == nil {
		h.logger.Error("Node not found in UpdateNode", "workflow_id", workflowUUID, "node_id", nodeID)
		respondError(c, http.StatusNotFound, "node not found")
		return
	}

	// Update fields
	if req.Name != "" {
		nodeModel.Name = req.Name
	}
	if req.Type != "" {
		nodeModel.Type = req.Type
	}
	if req.Config != nil {
		nodeModel.Config = storagemodels.JSONBMap(req.Config)
	}
	if req.Position != nil {
		nodeModel.Position = storagemodels.JSONBMap{
			"x": req.Position.X,
			"y": req.Position.Y,
		}
	}

	if err := h.workflowRepo.UpdateNode(c.Request.Context(), nodeModel); err != nil {
		h.logger.Error("Failed to update node", "error", err, "workflow_id", workflowUUID, "node_id", nodeID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	node := storagemodels.NodeModelToDomain(nodeModel)
	respondJSON(c, http.StatusOK, node)
}

// HandleDeleteNode handles DELETE /api/v1/workflows/{workflow_id}/nodes/{nodeId}
func (h *NodeHandlers) HandleDeleteNode(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	nodeID := c.Param("nodeId")

	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	if nodeID == "" {
		respondError(c, http.StatusBadRequest, "node ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID in DeleteNode", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Get all nodes for the workflow
	nodeModels, err := h.workflowRepo.FindNodesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find nodes in DeleteNode", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Find the specific node by logical ID
	var nodeUUID uuid.UUID
	found := false
	for _, nm := range nodeModels {
		if nm.NodeID == nodeID {
			nodeUUID = nm.ID
			found = true
			break
		}
	}

	if !found {
		h.logger.Error("Node not found in DeleteNode", "workflow_id", workflowUUID, "node_id", nodeID)
		respondError(c, http.StatusNotFound, "node not found")
		return
	}

	if err := h.workflowRepo.DeleteNode(c.Request.Context(), nodeUUID); err != nil {
		h.logger.Error("Failed to delete node", "error", err, "workflow_id", workflowUUID, "node_id", nodeID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "node deleted successfully"})
}
