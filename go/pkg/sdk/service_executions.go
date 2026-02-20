package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ServiceExecutionsAPI provides access to execution operations via Service API.
type ServiceExecutionsAPI struct {
	client *ServiceClient
}

// List returns all executions, optionally filtered.
func (a *ServiceExecutionsAPI) List(ctx context.Context, opts *ServiceExecutionListOptions, callOpts ...CallOption) ([]*models.Execution, int, error) {
	path := "/executions"
	if opts != nil {
		path += fmt.Sprintf("?limit=%d&offset=%d", opts.Limit, opts.Offset)
		if opts.WorkflowID != "" {
			path += "&workflow_id=" + opts.WorkflowID
		}
		if opts.Status != "" {
			path += "&status=" + opts.Status
		}
	}

	resp, err := a.client.doRequest(ctx, http.MethodGet, path, nil, callOpts...)
	if err != nil {
		return nil, 0, err
	}

	return decodeListResponse[*models.Execution](resp, "executions")
}

// Get returns an execution by ID.
func (a *ServiceExecutionsAPI) Get(ctx context.Context, executionID string, callOpts ...CallOption) (*models.Execution, error) {
	resp, err := a.client.doRequest(ctx, http.MethodGet, "/executions/"+executionID, nil, callOpts...)
	if err != nil {
		return nil, err
	}
	return decodeResponse[models.Execution](resp)
}

// Start starts a workflow execution.
func (a *ServiceExecutionsAPI) Start(ctx context.Context, workflowID string, input map[string]any, callOpts ...CallOption) (*models.Execution, error) {
	body := map[string]any{"input": input}
	resp, err := a.client.doRequest(ctx, http.MethodPost, "/workflows/"+workflowID+"/execute", body, callOpts...)
	if err != nil {
		return nil, err
	}
	return decodeResponse[models.Execution](resp)
}

// Cancel cancels an execution.
func (a *ServiceExecutionsAPI) Cancel(ctx context.Context, executionID string, callOpts ...CallOption) error {
	resp, err := a.client.doRequest(ctx, http.MethodPost, "/executions/"+executionID+"/cancel", nil, callOpts...)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}

// Retry retries an execution.
func (a *ServiceExecutionsAPI) Retry(ctx context.Context, executionID string, callOpts ...CallOption) error {
	resp, err := a.client.doRequest(ctx, http.MethodPost, "/executions/"+executionID+"/retry", nil, callOpts...)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}

// ServiceExecutionListOptions defines filtering for listing executions.
type ServiceExecutionListOptions struct {
	Limit      int
	Offset     int
	WorkflowID string
	Status     string
}
