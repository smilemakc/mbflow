package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/models"
)

type TriggerHandlers struct {
	ops    *serviceapi.Operations
	logger *logger.Logger
}

func NewTriggerHandlers(ops *serviceapi.Operations, log *logger.Logger) *TriggerHandlers {
	return &TriggerHandlers{ops: ops, logger: log}
}

func (h *TriggerHandlers) HandleCreateTrigger(c *gin.Context) {
	var req struct {
		WorkflowID  string         `json:"workflow_id" binding:"required,uuid"`
		Name        string         `json:"name" binding:"required"`
		Description string         `json:"description,omitempty"`
		Type        string         `json:"type" binding:"required"`
		Config      map[string]any `json:"config"`
		Enabled     bool           `json:"enabled"`
		Metadata    map[string]any `json:"metadata,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	trigger, err := h.ops.CreateTrigger(c.Request.Context(), serviceapi.CreateTriggerParams{
		WorkflowID:  req.WorkflowID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      req.Config,
		Enabled:     req.Enabled,
	})
	if err != nil {
		h.logger.Error("Failed to create trigger", "error", err, "workflow_id", req.WorkflowID, "trigger_type", req.Type, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusCreated, trigger)
}

func (h *TriggerHandlers) HandleGetTrigger(c *gin.Context) {
	triggerID, ok := getParam(c, "id")
	if !ok {
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		h.logger.Error("Invalid trigger ID format in GetTrigger", "error", err, "trigger_id", triggerID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	trigger, err := h.ops.GetTrigger(c.Request.Context(), serviceapi.GetTriggerParams{
		TriggerID: triggerUUID,
	})
	if err != nil {
		h.logger.Error("Failed to find trigger", "error", err, "trigger_id", triggerUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, trigger)
}

func (h *TriggerHandlers) HandleListTriggers(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	params := serviceapi.ListTriggersParams{
		Limit:  limit,
		Offset: offset,
	}

	if workflowID := c.Query("workflow_id"); workflowID != "" {
		wfUUID, err := uuid.Parse(workflowID)
		if err != nil {
			h.logger.Error("Invalid workflow ID in ListTriggers", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
			respondAPIError(c, ErrInvalidID)
			return
		}
		params.WorkflowID = &wfUUID
	}
	if triggerType := c.Query("type"); triggerType != "" {
		params.Type = &triggerType
	}

	result, err := h.ops.ListTriggers(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to list triggers", "error", err, "limit", limit, "offset", offset, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondList(c, http.StatusOK, result.Triggers, result.Total, limit, offset)
}

func (h *TriggerHandlers) HandleUpdateTrigger(c *gin.Context) {
	triggerID, ok := getParam(c, "id")
	if !ok {
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		h.logger.Error("Invalid trigger ID format in UpdateTrigger", "error", err, "trigger_id", triggerID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req struct {
		Name        string         `json:"name,omitempty"`
		Description string         `json:"description,omitempty"`
		Type        string         `json:"type,omitempty"`
		Config      map[string]any `json:"config,omitempty"`
		Enabled     *bool          `json:"enabled,omitempty"`
		Metadata    map[string]any `json:"metadata,omitempty"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	trigger, err := h.ops.UpdateTrigger(c.Request.Context(), serviceapi.UpdateTriggerParams{
		TriggerID:   triggerUUID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      req.Config,
		Enabled:     req.Enabled,
	})
	if err != nil {
		h.logger.Error("Failed to update trigger", "error", err, "trigger_id", triggerUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, trigger)
}

func (h *TriggerHandlers) HandleDeleteTrigger(c *gin.Context) {
	triggerID, ok := getParam(c, "id")
	if !ok {
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		h.logger.Error("Invalid trigger ID format in DeleteTrigger", "error", err, "trigger_id", triggerID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	if err := h.ops.DeleteTrigger(c.Request.Context(), serviceapi.DeleteTriggerParams{
		TriggerID: triggerUUID,
	}); err != nil {
		h.logger.Error("Failed to delete trigger", "error", err, "trigger_id", triggerUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "trigger deleted successfully"})
}

func (h *TriggerHandlers) HandleEnableTrigger(c *gin.Context) {
	triggerID, ok := getParam(c, "id")
	if !ok {
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		h.logger.Error("Invalid trigger ID format in EnableTrigger", "error", err, "trigger_id", triggerID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	trigger, err := h.ops.EnableTrigger(c.Request.Context(), serviceapi.EnableTriggerParams{
		TriggerID: triggerUUID,
	})
	if err != nil {
		h.logger.Error("Failed to enable trigger", "error", err, "trigger_id", triggerUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, trigger)
}

func (h *TriggerHandlers) HandleDisableTrigger(c *gin.Context) {
	triggerID, ok := getParam(c, "id")
	if !ok {
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		h.logger.Error("Invalid trigger ID format in DisableTrigger", "error", err, "trigger_id", triggerID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	trigger, err := h.ops.DisableTrigger(c.Request.Context(), serviceapi.DisableTriggerParams{
		TriggerID: triggerUUID,
	})
	if err != nil {
		h.logger.Error("Failed to disable trigger", "error", err, "trigger_id", triggerUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, trigger)
}

func (h *TriggerHandlers) HandleTriggerManual(c *gin.Context) {
	triggerID, ok := getParam(c, "id")
	if !ok {
		return
	}

	triggerUUID, err := uuid.Parse(triggerID)
	if err != nil {
		h.logger.Error("Invalid trigger ID format in TriggerManual", "error", err, "trigger_id", triggerID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req struct {
		Input map[string]any `json:"input"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Input = make(map[string]any)
	}

	trigger, err := h.ops.GetTrigger(c.Request.Context(), serviceapi.GetTriggerParams{
		TriggerID: triggerUUID,
	})
	if err != nil {
		h.logger.Error("Failed to find trigger for manual execution", "error", err, "trigger_id", triggerUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	if !trigger.Enabled {
		respondAPIError(c, TranslateError(models.ErrTriggerDisabled))
		return
	}

	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "trigger execution requires trigger manager integration", http.StatusNotImplemented))
}
