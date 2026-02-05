package serviceapi

import (
	"context"
	"time"

	"github.com/google/uuid"

	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ListTriggersParams contains parameters for listing triggers.
type ListTriggersParams struct {
	Limit      int
	Offset     int
	WorkflowID *uuid.UUID
	Type       *string
}

// ListTriggersResult contains the result of listing triggers.
type ListTriggersResult struct {
	Triggers []*models.Trigger
	Total    int
}

func (o *Operations) ListTriggers(ctx context.Context, params ListTriggersParams) (*ListTriggersResult, error) {
	var triggerModels []*storagemodels.TriggerModel
	var err error

	if params.WorkflowID != nil {
		triggerModels, err = o.TriggerRepo.FindByWorkflowID(ctx, *params.WorkflowID)
	} else if params.Type != nil {
		triggerModels, err = o.TriggerRepo.FindByType(ctx, *params.Type, params.Limit, params.Offset)
	} else {
		triggerModels, err = o.TriggerRepo.FindAll(ctx, params.Limit, params.Offset)
	}

	if err != nil {
		o.Logger.Error("Failed to list triggers", "error", err, "limit", params.Limit, "offset", params.Offset)
		return nil, err
	}

	triggers := make([]*models.Trigger, len(triggerModels))
	for i, tm := range triggerModels {
		triggers[i] = triggerModelToDomain(tm, "", "")
	}

	var total int
	if params.WorkflowID != nil {
		total, err = o.TriggerRepo.CountByWorkflowID(ctx, *params.WorkflowID)
	} else if params.Type != nil {
		total, err = o.TriggerRepo.CountByType(ctx, *params.Type)
	} else {
		total, err = o.TriggerRepo.Count(ctx)
	}
	if err != nil {
		total = len(triggers)
	}

	return &ListTriggersResult{
		Triggers: triggers,
		Total:    total,
	}, nil
}

// CreateTriggerParams contains parameters for creating a trigger.
type CreateTriggerParams struct {
	WorkflowID  string
	Name        string
	Description string
	Type        string
	Config      map[string]any
	Enabled     bool
}

func (o *Operations) CreateTrigger(ctx context.Context, params CreateTriggerParams) (*models.Trigger, error) {
	if params.WorkflowID == "" {
		return nil, NewValidationError("WORKFLOW_ID_REQUIRED", "workflow_id is required")
	}
	if params.Name == "" {
		return nil, NewValidationError("NAME_REQUIRED", "name is required")
	}
	if params.Type == "" {
		return nil, NewValidationError("TYPE_REQUIRED", "type is required")
	}

	workflowUUID, err := uuid.Parse(params.WorkflowID)
	if err != nil {
		return nil, NewValidationError("INVALID_ID", "Invalid ID format")
	}

	if _, err := o.WorkflowRepo.FindByID(ctx, workflowUUID); err != nil {
		o.Logger.Error("Workflow not found in CreateTrigger", "error", err, "workflow_id", workflowUUID)
		return nil, err
	}

	if !isValidTriggerType(params.Type) {
		return nil, NewValidationError("INVALID_TRIGGER_TYPE", "invalid trigger type")
	}

	triggerModel := &storagemodels.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflowUUID,
		Type:       params.Type,
		Config:     storagemodels.JSONBMap(params.Config),
		Enabled:    params.Enabled,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := o.TriggerRepo.Create(ctx, triggerModel); err != nil {
		o.Logger.Error("Failed to create trigger", "error", err, "workflow_id", workflowUUID, "trigger_type", params.Type)
		return nil, err
	}

	return triggerModelToDomain(triggerModel, params.Name, params.Description), nil
}

// UpdateTriggerParams contains parameters for updating a trigger.
type UpdateTriggerParams struct {
	TriggerID   uuid.UUID
	Name        string
	Description string
	Type        string
	Config      map[string]any
	Enabled     *bool
}

func (o *Operations) UpdateTrigger(ctx context.Context, params UpdateTriggerParams) (*models.Trigger, error) {
	triggerModel, err := o.TriggerRepo.FindByID(ctx, params.TriggerID)
	if err != nil || triggerModel == nil {
		o.Logger.Error("Failed to find trigger for update", "error", err, "trigger_id", params.TriggerID)
		return nil, models.ErrTriggerNotFound
	}

	if params.Type != "" {
		if !isValidTriggerType(params.Type) {
			return nil, NewValidationError("INVALID_TRIGGER_TYPE", "invalid trigger type")
		}
		triggerModel.Type = params.Type
	}

	if params.Config != nil {
		triggerModel.Config = storagemodels.JSONBMap(params.Config)
	}

	if params.Enabled != nil {
		triggerModel.Enabled = *params.Enabled
	}

	if err := o.TriggerRepo.Update(ctx, triggerModel); err != nil {
		o.Logger.Error("Failed to update trigger", "error", err, "trigger_id", params.TriggerID)
		return nil, err
	}

	return triggerModelToDomain(triggerModel, params.Name, params.Description), nil
}

// DeleteTriggerParams contains parameters for deleting a trigger.
type DeleteTriggerParams struct {
	TriggerID uuid.UUID
}

func (o *Operations) DeleteTrigger(ctx context.Context, params DeleteTriggerParams) error {
	if err := o.TriggerRepo.Delete(ctx, params.TriggerID); err != nil {
		o.Logger.Error("Failed to delete trigger", "error", err, "trigger_id", params.TriggerID)
		return err
	}
	return nil
}

func isValidTriggerType(t string) bool {
	validTypes := map[string]bool{
		"manual":   true,
		"cron":     true,
		"webhook":  true,
		"event":    true,
		"interval": true,
	}
	return validTypes[t]
}

type GetTriggerParams struct {
	TriggerID uuid.UUID
}

func (o *Operations) GetTrigger(ctx context.Context, params GetTriggerParams) (*models.Trigger, error) {
	triggerModel, err := o.TriggerRepo.FindByID(ctx, params.TriggerID)
	if err != nil || triggerModel == nil {
		o.Logger.Error("Failed to find trigger", "error", err, "trigger_id", params.TriggerID)
		return nil, models.ErrTriggerNotFound
	}
	return triggerModelToDomain(triggerModel, "", ""), nil
}

type EnableTriggerParams struct {
	TriggerID uuid.UUID
}

func (o *Operations) EnableTrigger(ctx context.Context, params EnableTriggerParams) (*models.Trigger, error) {
	if err := o.TriggerRepo.Enable(ctx, params.TriggerID); err != nil {
		o.Logger.Error("Failed to enable trigger", "error", err, "trigger_id", params.TriggerID)
		return nil, err
	}
	triggerModel, err := o.TriggerRepo.FindByID(ctx, params.TriggerID)
	if err != nil || triggerModel == nil {
		o.Logger.Error("Failed to find trigger after enable", "error", err, "trigger_id", params.TriggerID)
		return nil, models.ErrTriggerNotFound
	}
	return triggerModelToDomain(triggerModel, "", ""), nil
}

type DisableTriggerParams struct {
	TriggerID uuid.UUID
}

func (o *Operations) DisableTrigger(ctx context.Context, params DisableTriggerParams) (*models.Trigger, error) {
	if err := o.TriggerRepo.Disable(ctx, params.TriggerID); err != nil {
		o.Logger.Error("Failed to disable trigger", "error", err, "trigger_id", params.TriggerID)
		return nil, err
	}
	triggerModel, err := o.TriggerRepo.FindByID(ctx, params.TriggerID)
	if err != nil || triggerModel == nil {
		o.Logger.Error("Failed to find trigger after disable", "error", err, "trigger_id", params.TriggerID)
		return nil, models.ErrTriggerNotFound
	}
	return triggerModelToDomain(triggerModel, "", ""), nil
}

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

	if tm.Config != nil {
		trigger.Config = map[string]interface{}(tm.Config)
	}

	if tm.LastTriggeredAt != nil {
		trigger.LastRun = tm.LastTriggeredAt
	}

	return trigger
}
