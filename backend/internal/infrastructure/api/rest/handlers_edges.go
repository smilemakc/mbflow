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

// EdgeHandlers provides HTTP handlers for edge-related endpoints
type EdgeHandlers struct {
	workflowRepo repository.WorkflowRepository
	logger       *logger.Logger
}

// NewEdgeHandlers creates a new EdgeHandlers instance
func NewEdgeHandlers(workflowRepo repository.WorkflowRepository, log *logger.Logger) *EdgeHandlers {
	return &EdgeHandlers{
		workflowRepo: workflowRepo,
		logger:       log,
	}
}

// detectCycle checks if adding a new edge would create a cycle using DFS
func detectCycle(edges []*storagemodels.EdgeModel, newFrom, newTo string) bool {
	// Build adjacency list including the new edge
	adj := make(map[string][]string)
	for _, e := range edges {
		adj[e.FromNodeID] = append(adj[e.FromNodeID], e.ToNodeID)
	}
	// Add the new edge
	adj[newFrom] = append(adj[newFrom], newTo)

	// Use DFS to detect cycle starting from newTo node
	// If we can reach newFrom from newTo, there's a cycle
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	// Check from newTo if we can reach newFrom (would create cycle)
	return hasCycle(newTo)
}

// HandleAddEdge handles POST /api/v1/workflows/{workflow_id}/edges
func (h *EdgeHandlers) HandleAddEdge(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID in AddEdge", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	var req struct {
		ID           string `json:"id" binding:"required"`
		From         string `json:"from" binding:"required"`
		To           string `json:"to" binding:"required"`
		SourceHandle string `json:"source_handle,omitempty"`
		Condition    string `json:"condition,omitempty"`
		Loop         *struct {
			MaxIterations int `json:"max_iterations"`
		} `json:"loop,omitempty"`
		Metadata map[string]any `json:"metadata,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	if req.From == req.To {
		respondError(c, http.StatusBadRequest, "self-loop edges are not allowed")
		return
	}

	// Validate: loop edges must not have conditions
	if req.Loop != nil && req.Condition != "" {
		respondError(c, http.StatusBadRequest, "loop edges must not have conditions")
		return
	}

	// Validate loop config
	if req.Loop != nil && req.Loop.MaxIterations <= 0 {
		respondError(c, http.StatusBadRequest, "loop max_iterations must be > 0")
		return
	}

	// Verify workflow exists
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Workflow not found in AddEdge", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Verify source and target nodes exist
	nodes, err := h.workflowRepo.FindNodesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find nodes in AddEdge", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	fromExists := false
	toExists := false
	for _, node := range nodes {
		if node.NodeID == req.From {
			fromExists = true
		}
		if node.NodeID == req.To {
			toExists = true
		}
	}

	if !fromExists {
		respondError(c, http.StatusBadRequest, "source node does not exist")
		return
	}

	if !toExists {
		respondError(c, http.StatusBadRequest, "target node does not exist")
		return
	}

	// Get existing edges to check for cycles
	existingEdges, err := h.workflowRepo.FindEdgesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find edges for cycle detection", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Check if adding this edge would create a cycle
	// Skip cycle detection for loop edges — they are intentional back-edges
	if req.Loop == nil {
		if detectCycle(existingEdges, req.From, req.To) {
			respondError(c, http.StatusBadRequest, "adding this edge creates a cycle in the workflow")
			return
		}
	}

	// Create edge model
	edgeModel := &storagemodels.EdgeModel{
		ID:           uuid.New(),
		EdgeID:       req.ID,
		WorkflowID:   workflowUUID,
		FromNodeID:   req.From,
		ToNodeID:     req.To,
		SourceHandle: req.SourceHandle,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Set condition if provided
	if req.Condition != "" {
		edgeModel.Condition = storagemodels.JSONBMap{
			"expression": req.Condition,
		}
	}

	// Set loop config if provided
	if req.Loop != nil {
		edgeModel.Loop = storagemodels.JSONBMap{
			"max_iterations": req.Loop.MaxIterations,
		}
	}

	if err := h.workflowRepo.CreateEdge(c.Request.Context(), edgeModel); err != nil {
		h.logger.Error("Failed to create edge", "error", err, "workflow_id", workflowUUID, "edge_id", req.ID, "from", req.From, "to", req.To)

		errMsg := err.Error()
		// Check for duplicate edge ID constraint violation
		if strings.Contains(errMsg, "uq_edges_workflow_edge_id") {
			respondError(c, http.StatusBadRequest, "edge with this ID already exists")
			return
		}
		// Check for cycle detection errors
		if strings.Contains(errMsg, "cycle") || strings.Contains(errMsg, "creates a cycle") {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		// Check for node not found errors
		if strings.Contains(errMsg, "node not found") || strings.Contains(errMsg, "node does not exist") {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}

		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain model
	edge := storagemodels.EdgeModelToDomain(edgeModel)
	respondJSON(c, http.StatusCreated, edge)
}

// HandleListEdges handles GET /api/v1/workflows/{workflow_id}/edges
func (h *EdgeHandlers) HandleListEdges(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID in ListEdges", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Verify workflow exists
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Workflow not found in ListEdges", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	edgeModels, err := h.workflowRepo.FindEdgesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to list edges", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain models
	edges := make([]*models.Edge, len(edgeModels))
	for i, em := range edgeModels {
		edges[i] = storagemodels.EdgeModelToDomain(em)
	}

	respondList(c, http.StatusOK, edges, len(edges), 0, 0)
}

// HandleGetEdge handles GET /api/v1/workflows/{workflow_id}/edges/{edgeId}
func (h *EdgeHandlers) HandleGetEdge(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	edgeID := c.Param("edgeId")

	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	if edgeID == "" {
		respondError(c, http.StatusBadRequest, "edge ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID in GetEdge", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Get all edges for the workflow
	edgeModels, err := h.workflowRepo.FindEdgesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find edges in GetEdge", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Find the specific edge by logical ID
	var edgeModel *storagemodels.EdgeModel
	for _, em := range edgeModels {
		if em.EdgeID == edgeID {
			edgeModel = em
			break
		}
	}

	if edgeModel == nil {
		h.logger.Error("Edge not found", "workflow_id", workflowUUID, "edge_id", edgeID)
		respondError(c, http.StatusNotFound, "edge not found")
		return
	}

	edge := storagemodels.EdgeModelToDomain(edgeModel)
	respondJSON(c, http.StatusOK, edge)
}

// HandleUpdateEdge handles PUT /api/v1/workflows/{workflow_id}/edges/{edgeId}
func (h *EdgeHandlers) HandleUpdateEdge(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	edgeID := c.Param("edgeId")

	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	if edgeID == "" {
		respondError(c, http.StatusBadRequest, "edge ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID in UpdateEdge", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	var req struct {
		From         string  `json:"from,omitempty"`
		To           string  `json:"to,omitempty"`
		SourceHandle *string `json:"source_handle,omitempty"`
		Condition    string  `json:"condition,omitempty"`
		Loop         *struct {
			MaxIterations int `json:"max_iterations"`
		} `json:"loop,omitempty"`
		Metadata map[string]any `json:"metadata,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	// Get all edges for the workflow
	edgeModels, err := h.workflowRepo.FindEdgesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find edges in UpdateEdge", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Find the specific edge by logical ID
	var edgeModel *storagemodels.EdgeModel
	for _, em := range edgeModels {
		if em.EdgeID == edgeID {
			edgeModel = em
			break
		}
	}

	if edgeModel == nil {
		h.logger.Error("Edge not found in UpdateEdge", "workflow_id", workflowUUID, "edge_id", edgeID)
		respondError(c, http.StatusNotFound, "edge not found")
		return
	}

	// Update fields
	if req.From != "" {
		// Validate from node exists
		nodes, err := h.workflowRepo.FindNodesByWorkflowID(c.Request.Context(), workflowUUID)
		if err != nil {
			h.logger.Error("Failed to find nodes for from validation in UpdateEdge", "error", err, "workflow_id", workflowUUID, "edge_id", edgeID)
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}

		fromExists := false
		for _, node := range nodes {
			if node.NodeID == req.From {
				fromExists = true
				break
			}
		}

		if !fromExists {
			respondError(c, http.StatusBadRequest, "source node does not exist")
			return
		}

		edgeModel.FromNodeID = req.From
	}

	if req.To != "" {
		// Validate to node exists
		nodes, err := h.workflowRepo.FindNodesByWorkflowID(c.Request.Context(), workflowUUID)
		if err != nil {
			h.logger.Error("Failed to find nodes for to validation in UpdateEdge", "error", err, "workflow_id", workflowUUID, "edge_id", edgeID)
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}

		toExists := false
		for _, node := range nodes {
			if node.NodeID == req.To {
				toExists = true
				break
			}
		}

		if !toExists {
			respondError(c, http.StatusBadRequest, "target node does not exist")
			return
		}

		edgeModel.ToNodeID = req.To
	}

	// Validate no self-loop
	if edgeModel.FromNodeID == edgeModel.ToNodeID {
		respondError(c, http.StatusBadRequest, "self-loop edges are not allowed")
		return
	}

	// Update condition
	if req.Condition != "" {
		edgeModel.Condition = storagemodels.JSONBMap{
			"expression": req.Condition,
		}
	}

	// Update source handle
	if req.SourceHandle != nil {
		edgeModel.SourceHandle = *req.SourceHandle
	}

	// Update loop config
	if req.Loop != nil {
		if req.Loop.MaxIterations <= 0 {
			respondError(c, http.StatusBadRequest, "loop max_iterations must be > 0")
			return
		}
		edgeModel.Loop = storagemodels.JSONBMap{
			"max_iterations": req.Loop.MaxIterations,
		}
	}

	if err := h.workflowRepo.UpdateEdge(c.Request.Context(), edgeModel); err != nil {
		h.logger.Error("Failed to update edge", "error", err, "workflow_id", workflowUUID, "edge_id", edgeID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	edge := storagemodels.EdgeModelToDomain(edgeModel)
	respondJSON(c, http.StatusOK, edge)
}

// HandleDeleteEdge handles DELETE /api/v1/workflows/{workflow_id}/edges/{edgeId}
func (h *EdgeHandlers) HandleDeleteEdge(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	edgeID := c.Param("edgeId")

	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	if edgeID == "" {
		respondError(c, http.StatusBadRequest, "edge ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID in DeleteEdge", "error", err, "workflow_id", workflowID)
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Get all edges for the workflow
	edgeModels, err := h.workflowRepo.FindEdgesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find edges in DeleteEdge", "error", err, "workflow_id", workflowUUID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Find the specific edge by logical ID
	var edgeUUID uuid.UUID
	found := false
	for _, em := range edgeModels {
		if em.EdgeID == edgeID {
			edgeUUID = em.ID
			found = true
			break
		}
	}

	if !found {
		h.logger.Error("Edge not found in DeleteEdge", "workflow_id", workflowUUID, "edge_id", edgeID)
		respondError(c, http.StatusNotFound, "edge not found")
		return
	}

	if err := h.workflowRepo.DeleteEdge(c.Request.Context(), edgeUUID); err != nil {
		h.logger.Error("Failed to delete edge", "error", err, "workflow_id", workflowUUID, "edge_id", edgeID)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "edge deleted successfully"})
}
