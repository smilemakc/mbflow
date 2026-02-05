package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

type ServiceAPITriggerHandlers struct {
	triggerRepo  repository.TriggerRepository
	workflowRepo repository.WorkflowRepository
	logger       *logger.Logger
}

func NewServiceAPITriggerHandlers(
	triggerRepo repository.TriggerRepository,
	workflowRepo repository.WorkflowRepository,
	log *logger.Logger,
) *ServiceAPITriggerHandlers {
	return &ServiceAPITriggerHandlers{
		triggerRepo:  triggerRepo,
		workflowRepo: workflowRepo,
		logger:       log,
	}
}

func (h *ServiceAPITriggerHandlers) ListTriggers(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)
	workflowID := c.Query("workflow_id")
	triggerType := c.Query("type")

	var triggerModels []*storagemodels.TriggerModel
	var err error

	if workflowID != "" {
		wfUUID, parseErr := uuid.Parse(workflowID)
		if parseErr != nil {
			respondAPIError(c, ErrInvalidID)
			return
		}
		triggerModels, err = h.triggerRepo.FindByWorkflowID(c.Request.Context(), wfUUID)
	} else if triggerType != "" {
		triggerModels, err = h.triggerRepo.FindByType(c.Request.Context(), triggerType, limit, offset)
	} else {
		triggerModels, err = h.triggerRepo.FindAll(c.Request.Context(), limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to list triggers", "error", err, "workflow_id", workflowID, "type", triggerType, "limit", limit, "offset", offset)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	triggers := make([]*models.Trigger, len(triggerModels))
	for i, tm := range triggerModels {
		triggers[i] = triggerModelToDomain(tm, "", "")
	}

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

func (h *ServiceAPITriggerHandlers) CreateTrigger(c *gin.Context) {
	var req struct {
		WorkflowID  string                 `json:"workflow_id"`
		Name        string                 `json:"name"`
		Description string                 `json:"description,omitempty"`
		Type        string                 `json:"type"`
		Config      map[string]any `json:"config"`
		Enabled     bool                   `json:"enabled"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	if req.WorkflowID == "" {
		respondAPIError(c, NewAPIError("WORKFLOW_ID_REQUIRED", "workflow_id is required", http.StatusBadRequest))
		return
	}

	if req.Name == "" {
		respondAPIError(c, NewAPIError("NAME_REQUIRED", "name is required", http.StatusBadRequest))
		return
	}

	if req.Type == "" {
		respondAPIError(c, NewAPIError("TYPE_REQUIRED", "type is required", http.StatusBadRequest))
		return
	}

	workflowUUID, err := uuid.Parse(req.WorkflowID)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	if _, err := h.workflowRepo.FindByID(c.Request.Context(), workflowUUID); err != nil {
		h.logger.Error("Workflow not found in CreateTrigger", "error", err, "workflow_id", workflowUUID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	validTypes := map[string]bool{
		"manual":   true,
		"cron":     true,
		"webhook":  true,
		"event":    true,
		"interval": true,
	}

	if !validTypes[req.Type] {
		respondAPIError(c, NewAPIError("INVALID_TRIGGER_TYPE", "invalid trigger type", http.StatusBadRequest))
		return
	}

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
		h.logger.Error("Failed to create trigger", "error", err, "workflow_id", workflowUUID, "trigger_type", req.Type)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	trigger := triggerModelToDomain(triggerModel, req.Name, req.Description)
	respondJSON(c, http.StatusCreated, trigger)
}

func (h *ServiceAPITriggerHandlers) UpdateTrigger(c *gin.Context) {
	triggerID, ok := getParam(c, "id")
	if !ok {
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req struct {
		Name        string                 `json:"name,omitempty"`
		Description string                 `json:"description,omitempty"`
		Type        string                 `json:"type,omitempty"`
		Config      map[string]any `json:"config,omitempty"`
		Enabled     *bool                  `json:"enabled,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	triggerModel, err := h.triggerRepo.FindByID(c.Request.Context(), triggerUUID)
	if err != nil || triggerModel == nil {
		h.logger.Error("Failed to find trigger for update", "error", err, "trigger_id", triggerUUID)
		respondAPIErrorWithRequestID(c, TranslateError(models.ErrTriggerNotFound))
		return
	}

	if req.Type != "" {
		validTypes := map[string]bool{
			"manual":   true,
			"cron":     true,
			"webhook":  true,
			"event":    true,
			"interval": true,
		}

		if !validTypes[req.Type] {
			respondAPIError(c, NewAPIError("INVALID_TRIGGER_TYPE", "invalid trigger type", http.StatusBadRequest))
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
		h.logger.Error("Failed to update trigger", "error", err, "trigger_id", triggerUUID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	trigger := triggerModelToDomain(triggerModel, req.Name, req.Description)
	respondJSON(c, http.StatusOK, trigger)
}

func (h *ServiceAPITriggerHandlers) DeleteTrigger(c *gin.Context) {
	triggerID, ok := getParam(c, "id")
	if !ok {
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	if err := h.triggerRepo.Delete(c.Request.Context(), triggerUUID); err != nil {
		h.logger.Error("Failed to delete trigger", "error", err, "trigger_id", triggerUUID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "trigger deleted successfully"})
}
