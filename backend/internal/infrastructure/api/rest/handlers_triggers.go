package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// TriggerHandlers provides HTTP handlers for trigger-related endpoints
type TriggerHandlers struct {
	triggerRepo  repository.TriggerRepository
	workflowRepo repository.WorkflowRepository
}

// NewTriggerHandlers creates a new TriggerHandlers instance
func NewTriggerHandlers(
	triggerRepo repository.TriggerRepository,
	workflowRepo repository.WorkflowRepository,
) *TriggerHandlers {
	return &TriggerHandlers{
		triggerRepo:  triggerRepo,
		workflowRepo: workflowRepo,
	}
}

// HandleCreateTrigger handles POST /api/v1/triggers
func (h *TriggerHandlers) HandleCreateTrigger(c *gin.Context) {
	var req struct {
		WorkflowID  string                 `json:"workflow_id"`
		Name        string                 `json:"name"`
		Description string                 `json:"description,omitempty"`
		Type        string                 `json:"type"`
		Config      map[string]interface{} `json:"config"`
		Enabled     bool                   `json:"enabled"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow_id is required")
		return
	}

	if req.Name == "" {
		respondError(c, http.StatusBadRequest, "name is required")
		return
	}

	if req.Type == "" {
		respondError(c, http.StatusBadRequest, "type is required")
		return
	}

	workflowUUID, err := uuid.Parse(req.WorkflowID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid workflow_id")
		return
	}

	// Verify workflow exists
	_, err = h.workflowRepo.FindByID(c.Request.Context(), workflowUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Validate trigger type
	validTypes := map[string]bool{
		"manual":   true,
		"cron":     true,
		"webhook":  true,
		"event":    true,
		"interval": true,
	}

	if !validTypes[req.Type] {
		respondError(c, http.StatusBadRequest, "invalid trigger type")
		return
	}

	// Create trigger model
	triggerModel := &storagemodels.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflowUUID,
		Type:       req.Type,
		Config:     storagemodels.JSONBMap(req.Config),
		Enabled:    req.Enabled,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.triggerRepo.Create(c.Request.Context(), triggerModel); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain model
	trigger := triggerModelToDomain(triggerModel, req.Name, req.Description)
	respondJSON(c, http.StatusCreated, trigger)
}

// HandleGetTrigger handles GET /api/v1/triggers/{id}
func (h *TriggerHandlers) HandleGetTrigger(c *gin.Context) {
	triggerID := c.Param("id")
	if triggerID == "" {
		respondError(c, http.StatusBadRequest, "trigger ID is required")
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid trigger ID")
		return
	}

	triggerModel, err := h.triggerRepo.FindByID(c.Request.Context(), triggerUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "trigger not found")
		return
	}

	trigger := triggerModelToDomain(triggerModel, "", "")
	respondJSON(c, http.StatusOK, trigger)
}

// HandleListTriggers handles GET /api/v1/triggers
func (h *TriggerHandlers) HandleListTriggers(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)
	workflowID := c.Query("workflow_id")
	triggerType := c.Query("type")

	var triggerModels []*storagemodels.TriggerModel
	var err error

	if workflowID != "" {
		wfUUID, parseErr := uuid.Parse(workflowID)
		if parseErr != nil {
			respondError(c, http.StatusBadRequest, "invalid workflow_id")
			return
		}
		triggerModels, err = h.triggerRepo.FindByWorkflowID(c.Request.Context(), wfUUID)
	} else if triggerType != "" {
		triggerModels, err = h.triggerRepo.FindByType(c.Request.Context(), triggerType, limit, offset)
	} else {
		triggerModels, err = h.triggerRepo.FindAll(c.Request.Context(), limit, offset)
	}

	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain models
	triggers := make([]*models.Trigger, len(triggerModels))
	for i, tm := range triggerModels {
		triggers[i] = triggerModelToDomain(tm, "", "")
	}

	// Get total count
	var total int
	if workflowID != "" {
		wfUUID, _ := uuid.Parse(workflowID)
		total, err = h.triggerRepo.CountByWorkflowID(c.Request.Context(), wfUUID)
	} else if triggerType != "" {
		total, err = h.triggerRepo.CountByType(c.Request.Context(), triggerType)
	} else {
		total, err = h.triggerRepo.Count(c.Request.Context())
	}
	if err != nil {
		total = len(triggers)
	}

	c.JSON(http.StatusOK, gin.H{
		"triggers": triggers,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// HandleUpdateTrigger handles PUT /api/v1/triggers/{id}
func (h *TriggerHandlers) HandleUpdateTrigger(c *gin.Context) {
	triggerID := c.Param("id")
	if triggerID == "" {
		respondError(c, http.StatusBadRequest, "trigger ID is required")
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid trigger ID")
		return
	}

	var req struct {
		Name        string                 `json:"name,omitempty"`
		Description string                 `json:"description,omitempty"`
		Type        string                 `json:"type,omitempty"`
		Config      map[string]interface{} `json:"config,omitempty"`
		Enabled     *bool                  `json:"enabled,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Fetch existing trigger
	triggerModel, err := h.triggerRepo.FindByID(c.Request.Context(), triggerUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "trigger not found")
		return
	}

	// Update fields
	if req.Type != "" {
		// Validate trigger type
		validTypes := map[string]bool{
			"manual":   true,
			"cron":     true,
			"webhook":  true,
			"event":    true,
			"interval": true,
		}

		if !validTypes[req.Type] {
			respondError(c, http.StatusBadRequest, "invalid trigger type")
			return
		}

		triggerModel.Type = req.Type
	}

	if req.Config != nil {
		triggerModel.Config = storagemodels.JSONBMap(req.Config)
	}

	if req.Enabled != nil {
		triggerModel.Enabled = *req.Enabled
	}

	if err := h.triggerRepo.Update(c.Request.Context(), triggerModel); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	trigger := triggerModelToDomain(triggerModel, req.Name, req.Description)
	respondJSON(c, http.StatusOK, trigger)
}

// HandleDeleteTrigger handles DELETE /api/v1/triggers/{id}
func (h *TriggerHandlers) HandleDeleteTrigger(c *gin.Context) {
	triggerID := c.Param("id")
	if triggerID == "" {
		respondError(c, http.StatusBadRequest, "trigger ID is required")
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid trigger ID")
		return
	}

	if err := h.triggerRepo.Delete(c.Request.Context(), triggerUUID); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "trigger deleted successfully",
	})
}

// HandleEnableTrigger handles POST /api/v1/triggers/{id}/enable
func (h *TriggerHandlers) HandleEnableTrigger(c *gin.Context) {
	triggerID := c.Param("id")
	if triggerID == "" {
		respondError(c, http.StatusBadRequest, "trigger ID is required")
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid trigger ID")
		return
	}

	if err := h.triggerRepo.Enable(c.Request.Context(), triggerUUID); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch updated trigger
	triggerModel, err := h.triggerRepo.FindByID(c.Request.Context(), triggerUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "trigger not found")
		return
	}

	trigger := triggerModelToDomain(triggerModel, "", "")
	respondJSON(c, http.StatusOK, trigger)
}

// HandleDisableTrigger handles POST /api/v1/triggers/{id}/disable
func (h *TriggerHandlers) HandleDisableTrigger(c *gin.Context) {
	triggerID := c.Param("id")
	if triggerID == "" {
		respondError(c, http.StatusBadRequest, "trigger ID is required")
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid trigger ID")
		return
	}

	if err := h.triggerRepo.Disable(c.Request.Context(), triggerUUID); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch updated trigger
	triggerModel, err := h.triggerRepo.FindByID(c.Request.Context(), triggerUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "trigger not found")
		return
	}

	trigger := triggerModelToDomain(triggerModel, "", "")
	respondJSON(c, http.StatusOK, trigger)
}

// HandleTriggerManual handles POST /api/v1/triggers/{id}/execute
// Manually executes a trigger (primarily for manual trigger types)
func (h *TriggerHandlers) HandleTriggerManual(c *gin.Context) {
	triggerID := c.Param("id")
	if triggerID == "" {
		respondError(c, http.StatusBadRequest, "trigger ID is required")
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid trigger ID")
		return
	}

	// Parse input from request body
	var req struct {
		Input map[string]interface{} `json:"input"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is acceptable for manual triggers
		req.Input = make(map[string]interface{})
	}

	// Fetch trigger
	triggerModel, err := h.triggerRepo.FindByID(c.Request.Context(), triggerUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "trigger not found")
		return
	}

	if !triggerModel.Enabled {
		respondError(c, http.StatusForbidden, "trigger is disabled")
		return
	}

	// This endpoint will be called by the trigger manager
	// For now, return 501 Not Implemented with a message
	// The actual implementation will be done when integrating with trigger manager
	respondError(c, http.StatusNotImplemented, "trigger execution requires trigger manager integration")
}

// triggerModelToDomain converts storage TriggerModel to domain Trigger
func triggerModelToDomain(tm *storagemodels.TriggerModel, name, description string) *models.Trigger {
	if tm == nil {
		return nil
	}

	trigger := &models.Trigger{
		ID:         tm.ID.String(),
		WorkflowID: tm.WorkflowID.String(),
		Type:       models.TriggerType(tm.Type),
		Config:     make(map[string]interface{}),
		Enabled:    tm.Enabled,
		CreatedAt:  tm.CreatedAt,
		UpdatedAt:  tm.UpdatedAt,
	}

	// Use provided name/description if available, otherwise try to get from config
	if name != "" {
		trigger.Name = name
	} else if n, ok := tm.Config["name"].(string); ok {
		trigger.Name = n
	}

	if description != "" {
		trigger.Description = description
	} else if d, ok := tm.Config["description"].(string); ok {
		trigger.Description = d
	}

	// Convert config
	if tm.Config != nil {
		trigger.Config = map[string]interface{}(tm.Config)
	}

	// Set last run and next run
	if tm.LastTriggeredAt != nil {
		trigger.LastRun = tm.LastTriggeredAt
	}

	return trigger
}
