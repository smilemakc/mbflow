package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/smilemakc/mbflow/go/pkg/models"
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
func (t *TriggerAPI) Trigger(ctx context.Context, triggerID string, input map[string]any) (*models.Execution, error) {
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

func (t *TriggerAPI) triggerEmbedded(ctx context.Context, triggerID string, input map[string]any) (*models.Execution, error) {
	return nil, errTriggersNotAvailable
}

func (t *TriggerAPI) getHistoryEmbedded(ctx context.Context, triggerID string, opts *TriggerHistoryOptions) ([]*models.Execution, error) {
	return nil, errTriggersNotAvailable
}

// Remote mode implementations.
//
// These call the MBFlow REST API (see pkg/server/routes.go):
//
//	POST   /api/v1/triggers
//	GET    /api/v1/triggers/{id}
//	GET    /api/v1/triggers?workflow_id=&type=&limit=&offset=
//	PUT    /api/v1/triggers/{id}
//	DELETE /api/v1/triggers/{id}
//	POST   /api/v1/triggers/{id}/enable
//	POST   /api/v1/triggers/{id}/disable
//	POST   /api/v1/triggers/{id}/execute
//
// The list endpoint has no "enabled" filter, so TriggerListOptions.Enabled is
// applied client-side. There is no trigger history endpoint: GetHistory
// returns an *UnsupportedOperationError (errors.Is(err, ErrNotSupportedByAPI)).

// remoteTriggerRequest is the body accepted by the trigger create/update endpoints.
type remoteTriggerRequest struct {
	WorkflowID  string         `json:"workflow_id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
	Enabled     *bool          `json:"enabled,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func triggerPath(triggerID string) string {
	return "/api/v1/triggers/" + pathEscape(triggerID)
}

func (t *TriggerAPI) createRemote(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	enabled := trigger.Enabled
	body := &remoteTriggerRequest{
		WorkflowID:  trigger.WorkflowID,
		Name:        trigger.Name,
		Description: trigger.Description,
		Type:        string(trigger.Type),
		Config:      trigger.Config,
		Enabled:     &enabled,
		Metadata:    trigger.Metadata,
	}
	var out models.Trigger
	if err := t.client.doRemote(ctx, http.MethodPost, "/api/v1/triggers", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TriggerAPI) getRemote(ctx context.Context, triggerID string) (*models.Trigger, error) {
	var out models.Trigger
	if err := t.client.doRemote(ctx, http.MethodGet, triggerPath(triggerID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TriggerAPI) listRemote(ctx context.Context, opts *TriggerListOptions) ([]*models.Trigger, error) {
	query := url.Values{}
	if opts != nil {
		setPagination(query, opts.Limit, opts.Offset)
		if opts.WorkflowID != "" {
			query.Set("workflow_id", opts.WorkflowID)
		}
		if opts.Type != "" {
			query.Set("type", opts.Type)
		}
	}

	var raw map[string]json.RawMessage
	if err := t.client.doRemote(ctx, http.MethodGet, "/api/v1/triggers", query, nil, &raw); err != nil {
		return nil, err
	}
	triggers, err := decodeListEnvelope[*models.Trigger](raw, "triggers")
	if err != nil {
		return nil, err
	}

	if opts == nil || opts.Enabled == nil {
		return triggers, nil
	}
	filtered := make([]*models.Trigger, 0, len(triggers))
	for _, tr := range triggers {
		if tr != nil && tr.Enabled == *opts.Enabled {
			filtered = append(filtered, tr)
		}
	}
	return filtered, nil
}

func (t *TriggerAPI) updateRemote(ctx context.Context, trigger *models.Trigger) (*models.Trigger, error) {
	enabled := trigger.Enabled
	body := &remoteTriggerRequest{
		Name:        trigger.Name,
		Description: trigger.Description,
		Type:        string(trigger.Type),
		Config:      trigger.Config,
		Enabled:     &enabled,
		Metadata:    trigger.Metadata,
	}
	var out models.Trigger
	if err := t.client.doRemote(ctx, http.MethodPut, triggerPath(trigger.ID), nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TriggerAPI) deleteRemote(ctx context.Context, triggerID string) error {
	return t.client.doRemote(ctx, http.MethodDelete, triggerPath(triggerID), nil, nil, nil)
}

func (t *TriggerAPI) enableRemote(ctx context.Context, triggerID string) error {
	return t.client.doRemote(ctx, http.MethodPost, triggerPath(triggerID)+"/enable", nil, nil, nil)
}

func (t *TriggerAPI) disableRemote(ctx context.Context, triggerID string) error {
	return t.client.doRemote(ctx, http.MethodPost, triggerPath(triggerID)+"/disable", nil, nil, nil)
}

func (t *TriggerAPI) triggerRemote(ctx context.Context, triggerID string, input map[string]any) (*models.Execution, error) {
	if input == nil {
		input = map[string]any{}
	}
	body := map[string]any{"input": input}
	var out models.Execution
	if err := t.client.doRemote(ctx, http.MethodPost, triggerPath(triggerID)+"/execute", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TriggerAPI) getHistoryRemote(ctx context.Context, triggerID string, opts *TriggerHistoryOptions) ([]*models.Execution, error) {
	return nil, errUnsupported("TriggerAPI.GetHistory", "the REST API has no trigger execution history endpoint")
}
