package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/serviceapi"
)

type ServiceAPITriggerHandlers struct {
	ops *serviceapi.Operations
}

func NewServiceAPITriggerHandlers(ops *serviceapi.Operations) *ServiceAPITriggerHandlers {
	return &ServiceAPITriggerHandlers{ops: ops}
}

func (h *ServiceAPITriggerHandlers) ListTriggers(c *gin.Context) {
	params := serviceapi.ListTriggersParams{
		Limit:  getQueryInt(c, "limit", 50),
		Offset: getQueryInt(c, "offset", 0),
	}
	if wfID := c.Query("workflow_id"); wfID != "" {
		parsed, err := uuid.Parse(wfID)
		if err != nil {
			respondAPIError(c, ErrInvalidID)
			return
		}
		params.WorkflowID = &parsed
	}
	if t := c.Query("type"); t != "" {
		params.Type = &t
	}

	result, err := h.ops.ListTriggers(c.Request.Context(), params)
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondList(c, http.StatusOK, result.Triggers, result.Total, params.Limit, params.Offset)
}

func (h *ServiceAPITriggerHandlers) CreateTrigger(c *gin.Context) {
	var req struct {
		WorkflowID  string         `json:"workflow_id" binding:"required,uuid"`
		Name        string         `json:"name" binding:"required"`
		Description string         `json:"description,omitempty"`
		Type        string         `json:"type" binding:"required"`
		Config      map[string]any `json:"config"`
		Enabled     bool           `json:"enabled"`
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
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

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
		Name        string         `json:"name,omitempty"`
		Description string         `json:"description,omitempty"`
		Type        string         `json:"type,omitempty"`
		Config      map[string]any `json:"config,omitempty"`
		Enabled     *bool          `json:"enabled,omitempty"`
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
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

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

	if err := h.ops.DeleteTrigger(c.Request.Context(), serviceapi.DeleteTriggerParams{
		TriggerID: triggerUUID,
	}); err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "trigger deleted successfully"})
}
