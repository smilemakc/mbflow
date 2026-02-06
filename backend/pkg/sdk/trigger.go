package sdk

import (
	"context"
	"fmt"

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

// Embedded mode implementations (standalone mode - no database persistence)
// For full persistence support, use pkg/server.Server directly.

var errTriggersNotAvailable = fmt.Errorf("trigger operations not available in standalone mode; use remote mode or pkg/server.Server for persistence")

func (t *TriggerAPI) createEmbedded(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	return nil, errTriggersNotAvailable
}

func (t *TriggerAPI) getEmbedded(ctx context.Context, triggerID string) (*models.Trigger, error) {
	return nil, errTriggersNotAvailable
}

func (t *TriggerAPI) listEmbedded(ctx context.Context, opts *TriggerListOptions) ([]*models.Trigger, error) {
	return nil, errTriggersNotAvailable
}

func (t *TriggerAPI) updateEmbedded(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	return nil, errTriggersNotAvailable
}

func (t *TriggerAPI) deleteEmbedded(ctx context.Context, triggerID string) error {
	return errTriggersNotAvailable
}

func (t *TriggerAPI) enableEmbedded(ctx context.Context, triggerID string) error {
	return errTriggersNotAvailable
}

func (t *TriggerAPI) disableEmbedded(ctx context.Context, triggerID string) error {
	return errTriggersNotAvailable
}

func (t *TriggerAPI) triggerEmbedded(ctx context.Context, triggerID string, input map[string]interface{}) (*models.Execution, error) {
	return nil, errTriggersNotAvailable
}

func (t *TriggerAPI) getHistoryEmbedded(ctx context.Context, triggerID string, opts *TriggerHistoryOptions) ([]*models.Execution, error) {
	return nil, errTriggersNotAvailable
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
