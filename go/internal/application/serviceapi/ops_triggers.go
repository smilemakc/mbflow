package serviceapi

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/trigger"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ListTriggersParams contains parameters for listing triggers.
type ListTriggersParams struct {
	Limit      int
	Offset     int
	WorkflowID *uuid.UUID
	Type       *string
	// OwnerID restricts results to triggers of workflows created by this user.
	OwnerID *uuid.UUID
}

// ListTriggersResult contains the result of listing triggers.
type ListTriggersResult struct {
	Triggers []*models.Trigger
	Total    int
}

func (o *Operations) ListTriggers(ctx context.Context, params ListTriggersParams) (*ListTriggersResult, error) {
	var triggerModels []*storagemodels.TriggerModel
	var err error

	ownerFilters := repository.TriggerFilters{
		WorkflowID: params.WorkflowID,
		Type:       params.Type,
		OwnerID:    params.OwnerID,
	}

	if params.OwnerID != nil {
		triggerModels, err = o.TriggerRepo.FindWithFilters(ctx, ownerFilters, params.Limit, params.Offset)
	} else if params.WorkflowID != nil {
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
		triggers[i] = triggerModelToDomain(tm)
	}

	var total int
	if params.OwnerID != nil {
		total, err = o.TriggerRepo.CountWithFilters(ctx, ownerFilters)
	} else if params.WorkflowID != nil {
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

	if err := trigger.ValidateScheduleConfig(params.Type, params.Config); err != nil {
		return nil, NewValidationError("INVALID_TRIGGER_CONFIG", err.Error())
	}

	triggerModel := &storagemodels.TriggerModel{
		ID:          uuid.New(),
		WorkflowID:  workflowUUID,
		Name:        params.Name,
		Description: params.Description,
		Type:        params.Type,
		Config:      storagemodels.JSONBMap(params.Config),
		Enabled:     params.Enabled,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := o.TriggerRepo.Create(ctx, triggerModel); err != nil {
		o.Logger.Error("Failed to create trigger", "error", err, "workflow_id", workflowUUID, "trigger_type", params.Type)
		return nil, err
	}
	o.syncTrigger(ctx, triggerModel.ID.String())

	return triggerModelToDomain(triggerModel), nil
}

// UpdateTriggerParams contains parameters for updating a trigger.
type UpdateTriggerParams struct {
	TriggerID uuid.UUID
	// Name replaces the stored name when non-empty.
	Name string
	// Description replaces the stored description when non-nil (an empty
	// string clears it).
	Description *string
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

	if params.Name != "" {
		triggerModel.Name = params.Name
	}

	if params.Description != nil {
		triggerModel.Description = *params.Description
	}

	if params.Config != nil {
		triggerModel.Config = storagemodels.JSONBMap(params.Config)
	}

	if params.Enabled != nil {
		triggerModel.Enabled = *params.Enabled
	}

	if err := trigger.ValidateScheduleConfig(triggerModel.Type, triggerModel.Config); err != nil {
		return nil, NewValidationError("INVALID_TRIGGER_CONFIG", err.Error())
	}

	if err := o.TriggerRepo.Update(ctx, triggerModel); err != nil {
		o.Logger.Error("Failed to update trigger", "error", err, "trigger_id", params.TriggerID)
		return nil, err
	}
	o.syncTrigger(ctx, params.TriggerID.String())

	return triggerModelToDomain(triggerModel), nil
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
	o.syncTrigger(ctx, params.TriggerID.String())
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
	return triggerModelToDomain(triggerModel), nil
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
	o.syncTrigger(ctx, params.TriggerID.String())
	return triggerModelToDomain(triggerModel), nil
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
	o.syncTrigger(ctx, params.TriggerID.String())
	return triggerModelToDomain(triggerModel), nil
}

// timeNow is the clock used to compute next_run; tests replace it.
var timeNow = time.Now

// triggerModelToDomain converts a stored trigger to its API representation.
// Name and description come from their columns, falling back to the legacy
// config.name / config.description keys for triggers created before they
// were persisted. next_run is computed for enabled cron and interval
// triggers from the schedule, exactly as the scheduler computes it.
func triggerModelToDomain(tm *storagemodels.TriggerModel) *models.Trigger {
	if tm == nil {
		return nil
	}

	t := &models.Trigger{
		ID:          tm.ID.String(),
		WorkflowID:  tm.WorkflowID.String(),
		Name:        tm.DisplayName(),
		Description: tm.DisplayDescription(),
		Type:        models.TriggerType(tm.Type),
		Config:      make(map[string]any),
		Enabled:     tm.Enabled,
		CreatedAt:   tm.CreatedAt,
		UpdatedAt:   tm.UpdatedAt,
		LastRun:     tm.LastTriggeredAt,
	}

	if tm.Config != nil {
		t.Config = map[string]any(tm.Config)
	}

	t.NextRun = trigger.NextRun(tm.Type, tm.Enabled, t.Config, timeNow())

	return t
}

// RunTriggerParams contains parameters for running a trigger on demand.
type RunTriggerParams struct {
	TriggerID uuid.UUID
	// Input is merged over the trigger's default config.input (request keys
	// win).
	Input map[string]any
}

// RunTrigger starts the trigger's workflow now ("Run now") and returns the
// queued execution without waiting for it to finish.
//
// The execution is started through ExecutionMgr directly rather than the
// trigger manager: the manager only exists when Redis is available and its
// TriggerManual runs the workflow synchronously, while a manual run needs
// neither scheduler state nor Redis and must not hold the request open for
// the whole workflow. The execution is tagged with the trigger ID and the
// trigger's last_run is updated, the same as for scheduled runs.
func (o *Operations) RunTrigger(ctx context.Context, params RunTriggerParams) (*models.Execution, error) {
	triggerModel, err := o.TriggerRepo.FindByID(ctx, params.TriggerID)
	if err != nil || triggerModel == nil {
		o.Logger.Error("Failed to find trigger for manual run", "error", err, "trigger_id", params.TriggerID)
		return nil, models.ErrTriggerNotFound
	}
	if !triggerModel.Enabled {
		return nil, models.ErrTriggerDisabled
	}
	if o.ExecutionMgr == nil {
		return nil, &OperationError{
			Code:       "EXECUTION_UNAVAILABLE",
			Message:    "workflow execution is not available on this server",
			HTTPStatus: http.StatusServiceUnavailable,
		}
	}

	opts := engine.DefaultExecutionOptions()
	opts.TriggerID = triggerModel.ID.String()

	input := mergeTriggerInput(triggerModel.Config, params.Input)
	execution, err := o.ExecutionMgr.ExecuteAsync(ctx, triggerModel.WorkflowID.String(), input, opts)
	if err != nil {
		o.Logger.Error("Failed to start manual trigger run", "error", err, "trigger_id", params.TriggerID, "workflow_id", triggerModel.WorkflowID)
		return nil, err
	}

	if err := o.TriggerRepo.MarkTriggered(ctx, triggerModel.ID); err != nil {
		// The execution already started; last_run is informational.
		o.Logger.Error("Failed to record trigger run", "error", err, "trigger_id", params.TriggerID)
	}

	o.Logger.Info("Trigger run started", "trigger_id", params.TriggerID, "execution_id", execution.ID, "workflow_id", triggerModel.WorkflowID)
	return execution, nil
}

// mergeTriggerInput returns the trigger's default config.input overlaid with
// the request input. Neither source map is modified.
func mergeTriggerInput(config, input map[string]any) map[string]any {
	merged := make(map[string]any)
	if defaults, ok := config["input"].(map[string]any); ok {
		for k, v := range defaults {
			merged[k] = v
		}
	}
	for k, v := range input {
		merged[k] = v
	}
	return merged
}
