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
)

type ServiceAPIWorkflowHandlers struct {
	workflowRepo    repository.WorkflowRepository
	logger          *logger.Logger
	executorManager executor.Manager
}

func NewServiceAPIWorkflowHandlers(
	workflowRepo repository.WorkflowRepository,
	log *logger.Logger,
	executorManager executor.Manager,
) *ServiceAPIWorkflowHandlers {
	return &ServiceAPIWorkflowHandlers{
		workflowRepo:    workflowRepo,
		logger:          log,
		executorManager: executorManager,
	}
}

func (h *ServiceAPIWorkflowHandlers) ListWorkflows(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)
	status := c.Query("status")
	userIDParam := c.Query("user_id")

	filters := repository.WorkflowFilters{
		IncludeUnowned: true,
	}

	if status != "" {
		filters.Status = &status
	}

	if userIDParam != "" {
		requestedUserID, err := uuid.Parse(userIDParam)
		if err != nil {
			respondAPIError(c, NewAPIError("INVALID_USER_ID", "Invalid user_id format", http.StatusBadRequest))
			return
		}
		filters.CreatedBy = &requestedUserID
		filters.IncludeUnowned = false
	}

	workflowModels, err := h.workflowRepo.FindAllWithFilters(c.Request.Context(), filters, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list workflows", "error", err, "limit", limit, "offset", offset)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflows := make([]*models.Workflow, len(workflowModels))
	for i, wm := range workflowModels {
		workflows[i] = engine.WorkflowModelToDomain(wm)
	}

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

	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow", "error", err, "workflow_id", workflowUUID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusOK, workflow)
}

func (h *ServiceAPIWorkflowHandlers) CreateWorkflow(c *gin.Context) {
	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description,omitempty"`
		Variables   map[string]any `json:"variables,omitempty"`
		Metadata    map[string]any `json:"metadata,omitempty"`
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

	if userID, ok := GetUserIDAsUUID(c); ok {
		workflowModel.CreatedBy = &userID
	}

	if err := h.workflowRepo.Create(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to create workflow", "error", err, "workflow_name", req.Name)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
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

	if err := h.validateServiceAPINodes(req.Nodes); err != nil {
		respondAPIError(c, NewAPIError("NODE_VALIDATION_FAILED", err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.validateServiceAPIEdges(req.Edges, req.Nodes); err != nil {
		respondAPIError(c, NewAPIError("EDGE_VALIDATION_FAILED", err.Error(), http.StatusBadRequest))
		return
	}

	workflowModel, err := h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow for update", "error", err, "workflow_id", workflowUUID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

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

	if req.Resources != nil {
		workflowModel.Resources = make([]*storagemodels.WorkflowResourceModel, len(req.Resources))
		for i, resReq := range req.Resources {
			resourceUUID, parseErr := uuid.Parse(resReq.ResourceID)
			if parseErr != nil {
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

	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to update workflow", "error", err, "workflow_id", workflowUUID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	updatedWorkflow, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to fetch updated workflow", "error", err, "workflow_id", workflowUUID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	workflow := engine.WorkflowModelToDomain(updatedWorkflow)
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

	if err := h.workflowRepo.Delete(c.Request.Context(), workflowUUID); err != nil {
		h.logger.Error("Failed to delete workflow", "error", err, "workflow_id", workflowUUID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "workflow deleted successfully"})
}

func (h *ServiceAPIWorkflowHandlers) validateServiceAPINodes(nodes []NodeRequest) error {
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

func (h *ServiceAPIWorkflowHandlers) validateServiceAPIEdges(edges []EdgeRequest, nodes []NodeRequest) error {
	if edges == nil {
		return nil
	}

	nodeIDSet := make(map[string]bool)
	for _, node := range nodes {
		nodeIDSet[node.ID] = true
	}

	edgeIDs := make(map[string]bool)

	for i, edge := range edges {
		if edge.ID == "" {
			return fmt.Errorf("edge at index %d: id is required", i)
		}
		if edge.From == "" {
			return fmt.Errorf("edge at index %d: from is required", i)
		}
		if edge.To == "" {
			return fmt.Errorf("edge at index %d: to is required", i)
		}

		if edgeIDs[edge.ID] {
			return fmt.Errorf("duplicate edge id: %s", edge.ID)
		}
		edgeIDs[edge.ID] = true

		if edge.From == edge.To {
			return fmt.Errorf("edge %s: self-reference not allowed (from=%s, to=%s)", edge.ID, edge.From, edge.To)
		}

		if len(nodes) > 0 {
			if !nodeIDSet[edge.From] {
				return fmt.Errorf("edge %s: from node '%s' not found in nodes", edge.ID, edge.From)
			}
			if !nodeIDSet[edge.To] {
				return fmt.Errorf("edge %s: to node '%s' not found in nodes", edge.ID, edge.To)
			}
		}

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
