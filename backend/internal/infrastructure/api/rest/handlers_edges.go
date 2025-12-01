package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// EdgeHandlers provides HTTP handlers for edge-related endpoints
type EdgeHandlers struct {
	workflowRepo repository.WorkflowRepository
}

// NewEdgeHandlers creates a new EdgeHandlers instance
func NewEdgeHandlers(workflowRepo repository.WorkflowRepository) *EdgeHandlers {
	return &EdgeHandlers{
		workflowRepo: workflowRepo,
	}
}

// HandleAddEdge handles POST /api/v1/workflows/{workflowId}/edges
func (h *EdgeHandlers) HandleAddEdge(c *gin.Context) {
	workflowID := c.Param("workflowId")
	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	var req struct {
		ID        string                 `json:"id"`
		From      string                 `json:"from"`
		To        string                 `json:"to"`
		Condition string                 `json:"condition,omitempty"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ID == "" {
		respondError(c, http.StatusBadRequest, "edge ID is required")
		return
	}

	if req.From == "" {
		respondError(c, http.StatusBadRequest, "source node (from) is required")
		return
	}

	if req.To == "" {
		respondError(c, http.StatusBadRequest, "target node (to) is required")
		return
	}

	if req.From == req.To {
		respondError(c, http.StatusBadRequest, "self-loop edges are not allowed")
		return
	}

	// Verify workflow exists
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Verify source and target nodes exist
	nodes, err := h.workflowRepo.FindNodesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
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

	// Create edge model
	edgeModel := &storagemodels.EdgeModel{
		ID:         uuid.New(),
		EdgeID:     req.ID,
		WorkflowID: workflowUUID,
		FromNodeID: req.From,
		ToNodeID:   req.To,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Set condition if provided
	if req.Condition != "" {
		edgeModel.Condition = storagemodels.JSONBMap{
			"expression": req.Condition,
		}
	}

	if err := h.workflowRepo.CreateEdge(c.Request.Context(), edgeModel); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain model
	edge := engine.EdgeModelToDomain(edgeModel)
	respondJSON(c, http.StatusCreated, edge)
}

// HandleListEdges handles GET /api/v1/workflows/{workflowId}/edges
func (h *EdgeHandlers) HandleListEdges(c *gin.Context) {
	workflowID := c.Param("workflowId")
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
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	edgeModels, err := h.workflowRepo.FindEdgesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain models
	edges := make([]*models.Edge, len(edgeModels))
	for i, em := range edgeModels {
		edges[i] = engine.EdgeModelToDomain(em)
	}

	c.JSON(http.StatusOK, gin.H{
		"edges": edges,
		"total": len(edges),
	})
}

// HandleGetEdge handles GET /api/v1/workflows/{workflowId}/edges/{edgeId}
func (h *EdgeHandlers) HandleGetEdge(c *gin.Context) {
	workflowID := c.Param("workflowId")
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
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Get all edges for the workflow
	edgeModels, err := h.workflowRepo.FindEdgesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
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
		respondError(c, http.StatusNotFound, "edge not found")
		return
	}

	edge := engine.EdgeModelToDomain(edgeModel)
	respondJSON(c, http.StatusOK, edge)
}

// HandleUpdateEdge handles PUT /api/v1/workflows/{workflowId}/edges/{edgeId}
func (h *EdgeHandlers) HandleUpdateEdge(c *gin.Context) {
	workflowID := c.Param("workflowId")
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
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	var req struct {
		From      string                 `json:"from,omitempty"`
		To        string                 `json:"to,omitempty"`
		Condition string                 `json:"condition,omitempty"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get all edges for the workflow
	edgeModels, err := h.workflowRepo.FindEdgesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
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
		respondError(c, http.StatusNotFound, "edge not found")
		return
	}

	// Update fields
	if req.From != "" {
		// Validate from node exists
		nodes, err := h.workflowRepo.FindNodesByWorkflowID(c.Request.Context(), workflowUUID)
		if err != nil {
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

	if err := h.workflowRepo.UpdateEdge(c.Request.Context(), edgeModel); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	edge := engine.EdgeModelToDomain(edgeModel)
	respondJSON(c, http.StatusOK, edge)
}

// HandleDeleteEdge handles DELETE /api/v1/workflows/{workflowId}/edges/{edgeId}
func (h *EdgeHandlers) HandleDeleteEdge(c *gin.Context) {
	workflowID := c.Param("workflowId")
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
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Get all edges for the workflow
	edgeModels, err := h.workflowRepo.FindEdgesByWorkflowID(c.Request.Context(), workflowUUID)
	if err != nil {
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
		respondError(c, http.StatusNotFound, "edge not found")
		return
	}

	if err := h.workflowRepo.DeleteEdge(c.Request.Context(), edgeUUID); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "edge deleted successfully",
	})
}
