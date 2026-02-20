package mbflow

import (
	"context"
	"strconv"

	"github.com/smilemakc/mbflow/go/sdk/internal"
	"github.com/smilemakc/mbflow/go/sdk/models"
)

// convertError checks HTTP status and returns *APIError for non-2xx responses.
func convertError(resp *internal.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	ierr := internal.ParseErrorResponse(resp.StatusCode, resp.Body)
	return &APIError{
		StatusCode: ierr.StatusCode,
		Code:       ierr.Code,
		Message:    ierr.Message,
		Details:    ierr.Details,
	}
}

func listOptsToQuery(lo *models.ListOptions) map[string]string {
	if lo == nil {
		return nil
	}
	q := make(map[string]string)
	if lo.Limit > 0 {
		q["limit"] = strconv.Itoa(lo.Limit)
	}
	if lo.Offset > 0 {
		q["offset"] = strconv.Itoa(lo.Offset)
	}
	if lo.Sort != "" {
		q["sort"] = lo.Sort
	}
	if lo.Order != "" {
		q["order"] = lo.Order
	}
	if lo.Search != "" {
		q["search"] = lo.Search
	}
	if lo.WorkflowID != "" {
		q["workflow_id"] = lo.WorkflowID
	}
	return q
}

// --- WorkflowService ---

type workflowClient struct{ tr internal.Transport }

func newWorkflowClient(tr internal.Transport) WorkflowService { return &workflowClient{tr: tr} }

func (w *workflowClient) Create(ctx context.Context, wf *models.Workflow, opts ...RequestOption) (*models.Workflow, error) {
	resp, err := w.tr.Do(ctx, &internal.Request{Method: internal.MethodPost, Path: "/workflows", Body: wf})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Workflow](resp.Body)
}

func (w *workflowClient) Get(ctx context.Context, id string, opts ...RequestOption) (*models.Workflow, error) {
	resp, err := w.tr.Do(ctx, &internal.Request{Method: internal.MethodGet, Path: "/workflows/" + id})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Workflow](resp.Body)
}

func (w *workflowClient) Update(ctx context.Context, id string, wf *models.Workflow, opts ...RequestOption) (*models.Workflow, error) {
	resp, err := w.tr.Do(ctx, &internal.Request{Method: internal.MethodPut, Path: "/workflows/" + id, Body: wf})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Workflow](resp.Body)
}

func (w *workflowClient) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	resp, err := w.tr.Do(ctx, &internal.Request{Method: internal.MethodDelete, Path: "/workflows/" + id})
	if err != nil {
		return err
	}
	if resp.StatusCode == 204 {
		return nil
	}
	return convertError(resp)
}

func (w *workflowClient) List(ctx context.Context, lo *models.ListOptions, opts ...RequestOption) (*models.Page[models.Workflow], error) {
	resp, err := w.tr.Do(ctx, &internal.Request{Method: internal.MethodGet, Path: "/workflows", Query: listOptsToQuery(lo)})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	items, total, err := internal.DecodeListResponse[models.Workflow](resp.Body, "workflows")
	if err != nil {
		return nil, err
	}
	return &models.Page[models.Workflow]{Items: items, Total: total}, nil
}

// --- ExecutionService ---

type executionClient struct{ tr internal.Transport }

func newExecutionClient(tr internal.Transport) ExecutionService { return &executionClient{tr: tr} }

func (e *executionClient) Run(ctx context.Context, wfID string, input map[string]any, opts ...RequestOption) (*models.Execution, error) {
	body := map[string]any{"workflow_id": wfID, "input": input}
	resp, err := e.tr.Do(ctx, &internal.Request{Method: internal.MethodPost, Path: "/executions", Body: body})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Execution](resp.Body)
}

func (e *executionClient) Get(ctx context.Context, id string, opts ...RequestOption) (*models.Execution, error) {
	resp, err := e.tr.Do(ctx, &internal.Request{Method: internal.MethodGet, Path: "/executions/" + id})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Execution](resp.Body)
}

func (e *executionClient) List(ctx context.Context, lo *models.ListOptions, opts ...RequestOption) (*models.Page[models.Execution], error) {
	resp, err := e.tr.Do(ctx, &internal.Request{Method: internal.MethodGet, Path: "/executions", Query: listOptsToQuery(lo)})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	items, total, err := internal.DecodeListResponse[models.Execution](resp.Body, "executions")
	if err != nil {
		return nil, err
	}
	return &models.Page[models.Execution]{Items: items, Total: total}, nil
}

func (e *executionClient) Cancel(ctx context.Context, id string, opts ...RequestOption) (*models.Execution, error) {
	resp, err := e.tr.Do(ctx, &internal.Request{Method: internal.MethodPost, Path: "/executions/" + id + "/cancel"})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Execution](resp.Body)
}

func (e *executionClient) Retry(ctx context.Context, id string, opts ...RequestOption) (*models.Execution, error) {
	resp, err := e.tr.Do(ctx, &internal.Request{Method: internal.MethodPost, Path: "/executions/" + id + "/retry"})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Execution](resp.Body)
}

// --- TriggerService ---

type triggerClient struct{ tr internal.Transport }

func newTriggerClient(tr internal.Transport) TriggerService { return &triggerClient{tr: tr} }

func (t *triggerClient) Create(ctx context.Context, trigger *models.Trigger, opts ...RequestOption) (*models.Trigger, error) {
	resp, err := t.tr.Do(ctx, &internal.Request{Method: internal.MethodPost, Path: "/triggers", Body: trigger})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Trigger](resp.Body)
}

func (t *triggerClient) Get(ctx context.Context, id string, opts ...RequestOption) (*models.Trigger, error) {
	resp, err := t.tr.Do(ctx, &internal.Request{Method: internal.MethodGet, Path: "/triggers/" + id})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Trigger](resp.Body)
}

func (t *triggerClient) Update(ctx context.Context, id string, trigger *models.Trigger, opts ...RequestOption) (*models.Trigger, error) {
	resp, err := t.tr.Do(ctx, &internal.Request{Method: internal.MethodPut, Path: "/triggers/" + id, Body: trigger})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Trigger](resp.Body)
}

func (t *triggerClient) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	resp, err := t.tr.Do(ctx, &internal.Request{Method: internal.MethodDelete, Path: "/triggers/" + id})
	if err != nil {
		return err
	}
	if resp.StatusCode == 204 {
		return nil
	}
	return convertError(resp)
}

func (t *triggerClient) List(ctx context.Context, lo *models.ListOptions, opts ...RequestOption) (*models.Page[models.Trigger], error) {
	resp, err := t.tr.Do(ctx, &internal.Request{Method: internal.MethodGet, Path: "/triggers", Query: listOptsToQuery(lo)})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	items, total, err := internal.DecodeListResponse[models.Trigger](resp.Body, "triggers")
	if err != nil {
		return nil, err
	}
	return &models.Page[models.Trigger]{Items: items, Total: total}, nil
}

// --- CredentialService ---

type credentialClient struct{ tr internal.Transport }

func newCredentialClient(tr internal.Transport) CredentialService { return &credentialClient{tr: tr} }

func (c *credentialClient) Create(ctx context.Context, cred *models.Credential, opts ...RequestOption) (*models.Credential, error) {
	resp, err := c.tr.Do(ctx, &internal.Request{Method: internal.MethodPost, Path: "/credentials", Body: cred})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Credential](resp.Body)
}

func (c *credentialClient) Get(ctx context.Context, id string, opts ...RequestOption) (*models.Credential, error) {
	resp, err := c.tr.Do(ctx, &internal.Request{Method: internal.MethodGet, Path: "/credentials/" + id})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Credential](resp.Body)
}

func (c *credentialClient) Update(ctx context.Context, id string, cred *models.Credential, opts ...RequestOption) (*models.Credential, error) {
	resp, err := c.tr.Do(ctx, &internal.Request{Method: internal.MethodPut, Path: "/credentials/" + id, Body: cred})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	return internal.DecodeResponse[models.Credential](resp.Body)
}

func (c *credentialClient) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	resp, err := c.tr.Do(ctx, &internal.Request{Method: internal.MethodDelete, Path: "/credentials/" + id})
	if err != nil {
		return err
	}
	if resp.StatusCode == 204 {
		return nil
	}
	return convertError(resp)
}

func (c *credentialClient) List(ctx context.Context, lo *models.ListOptions, opts ...RequestOption) (*models.Page[models.Credential], error) {
	resp, err := c.tr.Do(ctx, &internal.Request{Method: internal.MethodGet, Path: "/credentials", Query: listOptsToQuery(lo)})
	if err != nil {
		return nil, err
	}
	if err := convertError(resp); err != nil {
		return nil, err
	}
	items, total, err := internal.DecodeListResponse[models.Credential](resp.Body, "credentials")
	if err != nil {
		return nil, err
	}
	return &models.Page[models.Credential]{Items: items, Total: total}, nil
}
