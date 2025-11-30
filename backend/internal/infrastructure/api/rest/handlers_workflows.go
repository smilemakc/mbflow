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

// WorkflowHandlers provides HTTP handlers for workflow-related endpoints
type WorkflowHandlers struct {
	workflowRepo repository.WorkflowRepository
}

// NewWorkflowHandlers creates a new WorkflowHandlers instance
func NewWorkflowHandlers(workflowRepo repository.WorkflowRepository) *WorkflowHandlers {
	return &WorkflowHandlers{
		workflowRepo: workflowRepo,
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
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain model
	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusCreated, workflow)
}

// HandleGetWorkflow handles GET /api/v1/workflows/{id}
func (h *WorkflowHandlers) HandleGetWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
	if workflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow ID is required")
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
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

// HandleUpdateWorkflow handles PUT /api/v1/workflows/{id}
func (h *WorkflowHandlers) HandleUpdateWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
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
		Name        string                 `json:"name,omitempty"`
		Description string                 `json:"description,omitempty"`
		Variables   map[string]interface{} `json:"variables,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Fetch existing workflow
	workflowModel, err := h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Update fields
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

	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusOK, workflow)
}

// HandleDeleteWorkflow handles DELETE /api/v1/workflows/{id}
func (h *WorkflowHandlers) HandleDeleteWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
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
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "workflow deleted successfully",
	})
}

// HandlePublishWorkflow handles POST /api/v1/workflows/{id}/publish
func (h *WorkflowHandlers) HandlePublishWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
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
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Change status to active
	workflowModel.Status = "active"

	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusOK, workflow)
}

// HandleUnpublishWorkflow handles POST /api/v1/workflows/{id}/unpublish
func (h *WorkflowHandlers) HandleUnpublishWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
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
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Change status to draft
	workflowModel.Status = "draft"

	if err := h.workflowRepo.Update(c.Request.Context(), workflowModel); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	workflow := engine.WorkflowModelToDomain(workflowModel)
	respondJSON(c, http.StatusOK, workflow)
}
