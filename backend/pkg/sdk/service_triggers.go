package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/smilemakc/mbflow/pkg/models"
)

// ServiceTriggersAPI provides access to trigger operations via Service API.
type ServiceTriggersAPI struct {
	client *ServiceClient
}

// List returns all triggers, optionally filtered.
func (a *ServiceTriggersAPI) List(ctx context.Context, opts *ServiceTriggerListOptions, callOpts ...CallOption) ([]*models.Trigger, int, error) {
	path := "/triggers"
	if opts != nil {
		path += fmt.Sprintf("?limit=%d&offset=%d", opts.Limit, opts.Offset)
		if opts.WorkflowID != "" {
			path += "&workflow_id=" + opts.WorkflowID
		}
		if opts.Type != "" {
			path += "&type=" + opts.Type
		}
	}

	resp, err := a.client.doRequest(ctx, http.MethodGet, path, nil, callOpts...)
	if err != nil {
		return nil, 0, err
	}

	return decodeListResponse[*models.Trigger](resp, "triggers")
}

// Create creates a new trigger.
func (a *ServiceTriggersAPI) Create(ctx context.Context, req *ServiceCreateTriggerRequest, callOpts ...CallOption) (*models.Trigger, error) {
	resp, err := a.client.doRequest(ctx, http.MethodPost, "/triggers", req, callOpts...)
	if err != nil {
		return nil, err
	}
	return decodeResponse[models.Trigger](resp)
}

// Update updates a trigger.
func (a *ServiceTriggersAPI) Update(ctx context.Context, triggerID string, req *ServiceUpdateTriggerRequest, callOpts ...CallOption) (*models.Trigger, error) {
	resp, err := a.client.doRequest(ctx, http.MethodPut, "/triggers/"+triggerID, req, callOpts...)
	if err != nil {
		return nil, err
	}
	return decodeResponse[models.Trigger](resp)
}

// Delete deletes a trigger.
func (a *ServiceTriggersAPI) Delete(ctx context.Context, triggerID string, callOpts ...CallOption) error {
	resp, err := a.client.doRequest(ctx, http.MethodDelete, "/triggers/"+triggerID, nil, callOpts...)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}

// ServiceTriggerListOptions defines filtering for listing triggers.
type ServiceTriggerListOptions struct {
	Limit      int
	Offset     int
	WorkflowID string
	Type       string
}

// ServiceCreateTriggerRequest defines the request for creating a trigger.
type ServiceCreateTriggerRequest struct {
	WorkflowID  string         `json:"workflow_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	Enabled     bool           `json:"enabled"`
}

// ServiceUpdateTriggerRequest defines the request for updating a trigger.
type ServiceUpdateTriggerRequest struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
	Enabled     *bool          `json:"enabled,omitempty"`
}
