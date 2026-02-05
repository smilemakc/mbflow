package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/smilemakc/mbflow/pkg/models"
)

// ServiceWorkflowsAPI provides access to workflow operations via Service API.
type ServiceWorkflowsAPI struct {
	client *ServiceClient
}

// List returns all workflows, optionally filtered.
func (a *ServiceWorkflowsAPI) List(ctx context.Context, opts *ListOptions, callOpts ...CallOption) ([]*models.Workflow, int, error) {
	path := "/workflows"
	if opts != nil {
		path += fmt.Sprintf("?limit=%d&offset=%d", opts.Limit, opts.Offset)
		if opts.Status != "" {
			path += "&status=" + opts.Status
		}
	}

	resp, err := a.client.doRequest(ctx, http.MethodGet, path, nil, callOpts...)
	if err != nil {
		return nil, 0, err
	}

	return decodeListResponse[*models.Workflow](resp, "workflows")
}

// Get returns a workflow by ID.
func (a *ServiceWorkflowsAPI) Get(ctx context.Context, workflowID string, callOpts ...CallOption) (*models.Workflow, error) {
	resp, err := a.client.doRequest(ctx, http.MethodGet, "/workflows/"+workflowID, nil, callOpts...)
	if err != nil {
		return nil, err
	}
	return decodeResponse[models.Workflow](resp)
}

// Create creates a new workflow.
func (a *ServiceWorkflowsAPI) Create(ctx context.Context, req *ServiceCreateWorkflowRequest, callOpts ...CallOption) (*models.Workflow, error) {
	resp, err := a.client.doRequest(ctx, http.MethodPost, "/workflows", req, callOpts...)
	if err != nil {
		return nil, err
	}
	return decodeResponse[models.Workflow](resp)
}

// Update updates an existing workflow.
func (a *ServiceWorkflowsAPI) Update(ctx context.Context, workflowID string, req *ServiceUpdateWorkflowRequest, callOpts ...CallOption) (*models.Workflow, error) {
	resp, err := a.client.doRequest(ctx, http.MethodPut, "/workflows/"+workflowID, req, callOpts...)
	if err != nil {
		return nil, err
	}
	return decodeResponse[models.Workflow](resp)
}

// Delete deletes a workflow.
func (a *ServiceWorkflowsAPI) Delete(ctx context.Context, workflowID string, callOpts ...CallOption) error {
	resp, err := a.client.doRequest(ctx, http.MethodDelete, "/workflows/"+workflowID, nil, callOpts...)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}

// ServiceCreateWorkflowRequest defines the request for creating a workflow via Service API.
type ServiceCreateWorkflowRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ServiceUpdateWorkflowRequest defines the request for updating a workflow via Service API.
type ServiceUpdateWorkflowRequest struct {
	Name        string                        `json:"name,omitempty"`
	Description string                        `json:"description,omitempty"`
	Variables   map[string]any                `json:"variables,omitempty"`
	Metadata    map[string]any                `json:"metadata,omitempty"`
	Nodes       []ServiceNodeRequest          `json:"nodes,omitempty"`
	Edges       []ServiceEdgeRequest          `json:"edges,omitempty"`
	Resources   []ServiceResourceRequest      `json:"resources,omitempty"`
}

// ServiceNodeRequest represents a node in the request body for Service API.
type ServiceNodeRequest struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Config   map[string]any `json:"config,omitempty"`
	Position map[string]any `json:"position,omitempty"`
}

// ServiceEdgeRequest represents an edge in the request body for Service API.
type ServiceEdgeRequest struct {
	ID        string         `json:"id"`
	From      string         `json:"from"`
	To        string         `json:"to"`
	Condition map[string]any `json:"condition,omitempty"`
}

// ServiceResourceRequest represents a resource attachment in the request body for Service API.
type ServiceResourceRequest struct {
	ResourceID string `json:"resource_id"`
	Alias      string `json:"alias"`
	AccessType string `json:"access_type"`
}
