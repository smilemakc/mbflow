package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// TriggerAPI provides methods for managing workflow triggers.
// It supports various trigger types including time-based (cron), webhook,
// manual, and event-driven triggers.
type TriggerAPI struct {
	client *Client
}

// newTriggerAPI creates a new TriggerAPI instance.
func newTriggerAPI(client *Client) *TriggerAPI {
	return &TriggerAPI{
		client: client,
	}
}

// Create creates a new trigger for a workflow.
func (t *TriggerAPI) Create(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	if err := t.client.checkClosed(); err != nil {
		return nil, err
	}

	// Validate trigger
	if err := trigger.Validate(); err != nil {
		return nil, fmt.Errorf("trigger validation failed: %w", err)
	}

	if t.client.config.Mode == ModeRemote {
		return t.createRemote(ctx, trigger)
	}

	return t.createEmbedded(ctx, trigger)
}

// Get retrieves a trigger by ID.
func (t *TriggerAPI) Get(ctx context.Context, triggerID string) (*models.Trigger, error) {
	if err := t.client.checkClosed(); err != nil {
		return nil, err
	}

	if triggerID == "" {
		return nil, fmt.Errorf("trigger ID is required")
	}

	if t.client.config.Mode == ModeRemote {
		return t.getRemote(ctx, triggerID)
	}

	return t.getEmbedded(ctx, triggerID)
}

// List retrieves all triggers with optional filtering.
func (t *TriggerAPI) List(ctx context.Context, opts *TriggerListOptions) ([]*models.Trigger, error) {
	if err := t.client.checkClosed(); err != nil {
		return nil, err
	}

	if t.client.config.Mode == ModeRemote {
		return t.listRemote(ctx, opts)
	}

	return t.listEmbedded(ctx, opts)
}

// Update updates an existing trigger.
func (t *TriggerAPI) Update(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	if err := t.client.checkClosed(); err != nil {
		return nil, err
	}

	if trigger.ID == "" {
		return nil, fmt.Errorf("trigger ID is required")
	}

	// Validate trigger
	if err := trigger.Validate(); err != nil {
		return nil, fmt.Errorf("trigger validation failed: %w", err)
	}

	if t.client.config.Mode == ModeRemote {
		return t.updateRemote(ctx, trigger)
	}

	return t.updateEmbedded(ctx, trigger)
}

// Delete deletes a trigger by ID.
func (t *TriggerAPI) Delete(ctx context.Context, triggerID string) error {
	if err := t.client.checkClosed(); err != nil {
		return err
	}

	if triggerID == "" {
		return fmt.Errorf("trigger ID is required")
	}

	if t.client.config.Mode == ModeRemote {
		return t.deleteRemote(ctx, triggerID)
	}

	return t.deleteEmbedded(ctx, triggerID)
}

// Enable enables a trigger.
func (t *TriggerAPI) Enable(ctx context.Context, triggerID string) error {
	if err := t.client.checkClosed(); err != nil {
		return err
	}

	if triggerID == "" {
		return fmt.Errorf("trigger ID is required")
	}

	if t.client.config.Mode == ModeRemote {
		return t.enableRemote(ctx, triggerID)
	}

	return t.enableEmbedded(ctx, triggerID)
}

// Disable disables a trigger.
func (t *TriggerAPI) Disable(ctx context.Context, triggerID string) error {
	if err := t.client.checkClosed(); err != nil {
		return err
	}

	if triggerID == "" {
		return fmt.Errorf("trigger ID is required")
	}

	if t.client.config.Mode == ModeRemote {
		return t.disableRemote(ctx, triggerID)
	}

	return t.disableEmbedded(ctx, triggerID)
}

// Trigger manually triggers a workflow execution.
// This is typically used for manual or ad-hoc executions.
func (t *TriggerAPI) Trigger(ctx context.Context, triggerID string, input map[string]interface{}) (*models.Execution, error) {
	if err := t.client.checkClosed(); err != nil {
		return nil, err
	}

	if triggerID == "" {
		return nil, fmt.Errorf("trigger ID is required")
	}

	if t.client.config.Mode == ModeRemote {
		return t.triggerRemote(ctx, triggerID, input)
	}

	return t.triggerEmbedded(ctx, triggerID, input)
}

// GetWebhookURL returns the webhook URL for a webhook trigger.
func (t *TriggerAPI) GetWebhookURL(ctx context.Context, triggerID string) (string, error) {
	if err := t.client.checkClosed(); err != nil {
		return "", err
	}

	if triggerID == "" {
		return "", fmt.Errorf("trigger ID is required")
	}

	trigger, err := t.Get(ctx, triggerID)
	if err != nil {
		return "", err
	}

	if trigger.Type != models.TriggerTypeWebhook {
		return "", fmt.Errorf("trigger is not a webhook trigger")
	}

	if t.client.config.Mode == ModeRemote {
		return fmt.Sprintf("%s/api/v1/webhooks/%s", t.client.config.BaseURL, triggerID), nil
	}

	// For embedded mode, use configured WebhookBaseURL or default
	baseURL := t.client.config.WebhookBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8585"
	}
	return fmt.Sprintf("%s/api/v1/webhooks/%s", baseURL, triggerID), nil
}

// GetHistory retrieves the execution history for a trigger.
func (t *TriggerAPI) GetHistory(ctx context.Context, triggerID string, opts *TriggerHistoryOptions) ([]*models.Execution, error) {
	if err := t.client.checkClosed(); err != nil {
		return nil, err
	}

	if triggerID == "" {
		return nil, fmt.Errorf("trigger ID is required")
	}

	if t.client.config.Mode == ModeRemote {
		return t.getHistoryRemote(ctx, triggerID, opts)
	}

	return t.getHistoryEmbedded(ctx, triggerID, opts)
}

// TriggerListOptions provides filtering options for listing triggers.
type TriggerListOptions struct {
	WorkflowID string
	Type       string
	Enabled    *bool
	Limit      int
	Offset     int
}

// TriggerHistoryOptions provides filtering options for trigger history.
type TriggerHistoryOptions struct {
	Limit     int
	Offset    int
	StartTime *int64
	EndTime   *int64
	Status    string
}

// Embedded mode implementations

func (t *TriggerAPI) createEmbedded(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	if t.client.triggerRepo == nil {
		return nil, fmt.Errorf("embedded mode create not available: no repository configured")
	}

	// Generate ID if not provided
	if trigger.ID == "" {
		trigger.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	trigger.CreatedAt = now
	trigger.UpdatedAt = now

	// Convert to storage model
	storageTrigger, err := triggerDomainToStorage(trigger)
	if err != nil {
		return nil, fmt.Errorf("failed to convert trigger: %w", err)
	}

	// Create in database
	if err := t.client.triggerRepo.Create(ctx, storageTrigger); err != nil {
		return nil, fmt.Errorf("failed to create trigger: %w", err)
	}

	// Return the created trigger
	return triggerStorageToDomain(storageTrigger), nil
}

func (t *TriggerAPI) getEmbedded(ctx context.Context, triggerID string) (*models.Trigger, error) {
	if t.client.triggerRepo == nil {
		return nil, fmt.Errorf("embedded mode get not available: no repository configured")
	}

	id, err := uuid.Parse(triggerID)
	if err != nil {
		return nil, fmt.Errorf("invalid trigger ID: %w", err)
	}

	storageTrigger, err := t.client.triggerRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get trigger: %w", err)
	}

	if storageTrigger == nil {
		return nil, models.ErrTriggerNotFound
	}

	return triggerStorageToDomain(storageTrigger), nil
}

func (t *TriggerAPI) listEmbedded(ctx context.Context, opts *TriggerListOptions) ([]*models.Trigger, error) {
	if t.client.triggerRepo == nil {
		return nil, fmt.Errorf("embedded mode list not available: no repository configured")
	}

	if opts == nil {
		opts = &TriggerListOptions{Limit: 100, Offset: 0}
	}

	var storageTriggers []*storagemodels.TriggerModel
	var err error

	// Apply filters
	if opts.WorkflowID != "" {
		workflowID, parseErr := uuid.Parse(opts.WorkflowID)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid workflow ID: %w", parseErr)
		}
		storageTriggers, err = t.client.triggerRepo.FindByWorkflowID(ctx, workflowID)
	} else if opts.Type != "" {
		storageTriggers, err = t.client.triggerRepo.FindByType(ctx, opts.Type, opts.Limit, opts.Offset)
	} else if opts.Enabled != nil && *opts.Enabled {
		storageTriggers, err = t.client.triggerRepo.FindEnabled(ctx)
	} else {
		storageTriggers, err = t.client.triggerRepo.FindAll(ctx, opts.Limit, opts.Offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	triggers := make([]*models.Trigger, len(storageTriggers))
	for i, st := range storageTriggers {
		triggers[i] = triggerStorageToDomain(st)
	}

	return triggers, nil
}

func (t *TriggerAPI) updateEmbedded(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	if t.client.triggerRepo == nil {
		return nil, fmt.Errorf("embedded mode update not available: no repository configured")
	}

	// Update timestamp
	trigger.UpdatedAt = time.Now()

	// Convert to storage model
	storageTrigger, err := triggerDomainToStorage(trigger)
	if err != nil {
		return nil, fmt.Errorf("failed to convert trigger: %w", err)
	}

	// Update in database
	if err := t.client.triggerRepo.Update(ctx, storageTrigger); err != nil {
		return nil, fmt.Errorf("failed to update trigger: %w", err)
	}

	// Fetch updated trigger
	updated, err := t.client.triggerRepo.FindByID(ctx, storageTrigger.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated trigger: %w", err)
	}

	if updated == nil {
		return nil, models.ErrTriggerNotFound
	}

	return triggerStorageToDomain(updated), nil
}

func (t *TriggerAPI) deleteEmbedded(ctx context.Context, triggerID string) error {
	if t.client.triggerRepo == nil {
		return fmt.Errorf("embedded mode delete not available: no repository configured")
	}

	id, err := uuid.Parse(triggerID)
	if err != nil {
		return fmt.Errorf("invalid trigger ID: %w", err)
	}

	if err := t.client.triggerRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete trigger: %w", err)
	}

	return nil
}

func (t *TriggerAPI) enableEmbedded(ctx context.Context, triggerID string) error {
	if t.client.triggerRepo == nil {
		return fmt.Errorf("embedded mode enable not available: no repository configured")
	}

	id, err := uuid.Parse(triggerID)
	if err != nil {
		return fmt.Errorf("invalid trigger ID: %w", err)
	}

	if err := t.client.triggerRepo.Enable(ctx, id); err != nil {
		return fmt.Errorf("failed to enable trigger: %w", err)
	}

	return nil
}

func (t *TriggerAPI) disableEmbedded(ctx context.Context, triggerID string) error {
	if t.client.triggerRepo == nil {
		return fmt.Errorf("embedded mode disable not available: no repository configured")
	}

	id, err := uuid.Parse(triggerID)
	if err != nil {
		return fmt.Errorf("invalid trigger ID: %w", err)
	}

	if err := t.client.triggerRepo.Disable(ctx, id); err != nil {
		return fmt.Errorf("failed to disable trigger: %w", err)
	}

	return nil
}

func (t *TriggerAPI) triggerEmbedded(ctx context.Context, triggerID string, input map[string]interface{}) (*models.Execution, error) {
	if t.client.triggerRepo == nil {
		return nil, fmt.Errorf("embedded mode trigger not available: no repository configured")
	}

	// Get trigger
	id, err := uuid.Parse(triggerID)
	if err != nil {
		return nil, fmt.Errorf("invalid trigger ID: %w", err)
	}

	storageTrigger, err := t.client.triggerRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get trigger: %w", err)
	}

	if storageTrigger == nil {
		return nil, models.ErrTriggerNotFound
	}

	if !storageTrigger.Enabled {
		return nil, fmt.Errorf("trigger is disabled")
	}

	// Execute the workflow associated with this trigger
	execution, err := t.client.Executions().Run(ctx, storageTrigger.WorkflowID.String(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to execute workflow: %w", err)
	}

	// Mark trigger as triggered (non-fatal error, just update last triggered time)
	_ = t.client.triggerRepo.MarkTriggered(ctx, id)

	return execution, nil
}

func (t *TriggerAPI) getHistoryEmbedded(ctx context.Context, triggerID string, opts *TriggerHistoryOptions) ([]*models.Execution, error) {
	if t.client.triggerRepo == nil {
		return nil, fmt.Errorf("embedded mode history not available: no repository configured")
	}

	// Get trigger to find workflow ID
	id, err := uuid.Parse(triggerID)
	if err != nil {
		return nil, fmt.Errorf("invalid trigger ID: %w", err)
	}

	storageTrigger, err := t.client.triggerRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get trigger: %w", err)
	}

	if storageTrigger == nil {
		return nil, models.ErrTriggerNotFound
	}

	// Get executions for this workflow
	listOpts := &ExecutionListOptions{
		WorkflowID: storageTrigger.WorkflowID.String(),
	}

	if opts != nil {
		listOpts.Limit = opts.Limit
		listOpts.Offset = opts.Offset
		listOpts.Status = opts.Status
		listOpts.StartTime = opts.StartTime
		listOpts.EndTime = opts.EndTime
	}

	return t.client.Executions().List(ctx, listOpts)
}

// triggerStorageToDomain converts storage TriggerModel to domain Trigger
func triggerStorageToDomain(tm *storagemodels.TriggerModel) *models.Trigger {
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
		LastRun:    tm.LastTriggeredAt,
	}

	// Extract name from config if present
	if tm.Config != nil {
		trigger.Config = map[string]interface{}(tm.Config)
		if name, ok := tm.Config["name"].(string); ok {
			trigger.Name = name
		}
		if description, ok := tm.Config["description"].(string); ok {
			trigger.Description = description
		}
	}

	// Generate name if not set
	if trigger.Name == "" {
		trigger.Name = fmt.Sprintf("%s-trigger-%s", tm.Type, tm.ID.String()[:8])
	}

	return trigger
}

// triggerDomainToStorage converts domain Trigger to storage TriggerModel
func triggerDomainToStorage(trigger *models.Trigger) (*storagemodels.TriggerModel, error) {
	if trigger == nil {
		return nil, fmt.Errorf("trigger is nil")
	}

	// WorkflowID is mandatory
	if trigger.WorkflowID == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	workflowID, err := uuid.Parse(trigger.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}

	tm := &storagemodels.TriggerModel{
		Type:       string(trigger.Type),
		WorkflowID: workflowID,
		Enabled:    trigger.Enabled,
		Config:     storagemodels.JSONBMap(trigger.Config),
	}

	// Parse ID
	if trigger.ID != "" {
		id, err := uuid.Parse(trigger.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid trigger ID: %w", err)
		}
		tm.ID = id
	} else {
		tm.ID = uuid.New()
	}

	// Store name and description in config
	if tm.Config == nil {
		tm.Config = make(storagemodels.JSONBMap)
	}
	if trigger.Name != "" {
		tm.Config["name"] = trigger.Name
	}
	if trigger.Description != "" {
		tm.Config["description"] = trigger.Description
	}

	// Set timestamps
	if !trigger.CreatedAt.IsZero() {
		tm.CreatedAt = trigger.CreatedAt
	}
	if !trigger.UpdatedAt.IsZero() {
		tm.UpdatedAt = trigger.UpdatedAt
	}
	if trigger.LastRun != nil {
		tm.LastTriggeredAt = trigger.LastRun
	}

	return tm, nil
}

// Remote mode implementations
func (t *TriggerAPI) createRemote(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}

func (t *TriggerAPI) getRemote(ctx context.Context, triggerID string) (*models.Trigger, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}

func (t *TriggerAPI) listRemote(ctx context.Context, opts *TriggerListOptions) ([]*models.Trigger, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}

func (t *TriggerAPI) updateRemote(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}

func (t *TriggerAPI) deleteRemote(ctx context.Context, triggerID string) error {
	// TODO: Implement HTTP API call
	return fmt.Errorf("remote mode not implemented yet")
}

func (t *TriggerAPI) enableRemote(ctx context.Context, triggerID string) error {
	// TODO: Implement HTTP API call
	return fmt.Errorf("remote mode not implemented yet")
}

func (t *TriggerAPI) disableRemote(ctx context.Context, triggerID string) error {
	// TODO: Implement HTTP API call
	return fmt.Errorf("remote mode not implemented yet")
}

func (t *TriggerAPI) triggerRemote(ctx context.Context, triggerID string, input map[string]interface{}) (*models.Execution, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}

func (t *TriggerAPI) getHistoryRemote(ctx context.Context, triggerID string, opts *TriggerHistoryOptions) ([]*models.Execution, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}
