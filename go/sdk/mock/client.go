package mock

import (
	"context"
	"fmt"
	"sync"

	mbflow "github.com/smilemakc/mbflow/go/sdk"
	"github.com/smilemakc/mbflow/go/sdk/models"
)

// Client is a mock implementation for testing user code that depends on MBFlow SDK.
type Client struct {
	workflows   *WorkflowServiceMock
	executions  *ExecutionServiceMock
	triggers    *TriggerServiceMock
	credentials *CredentialServiceMock
}

func NewClient() *Client {
	return &Client{
		workflows:   &WorkflowServiceMock{getResults: make(map[string]mockResult[models.Workflow])},
		executions:  &ExecutionServiceMock{runResults: make(map[string]mockResult[models.Execution]), getResults: make(map[string]mockResult[models.Execution])},
		triggers:    &TriggerServiceMock{getResults: make(map[string]mockResult[models.Trigger])},
		credentials: &CredentialServiceMock{getResults: make(map[string]mockResult[models.Credential])},
	}
}

func (c *Client) Workflows() mbflow.WorkflowService    { return c.workflows }
func (c *Client) Executions() mbflow.ExecutionService  { return c.executions }
func (c *Client) Triggers() mbflow.TriggerService      { return c.triggers }
func (c *Client) Credentials() mbflow.CredentialService { return c.credentials }
func (c *Client) Close() error                          { return nil }

type mockResult[T any] struct {
	value *T
	err   error
}

// --- WorkflowServiceMock ---

type WorkflowServiceMock struct {
	mu           sync.RWMutex
	getResults   map[string]mockResult[models.Workflow]
	createResult *mockResult[models.Workflow]
	listResult   *struct {
		page *models.Page[models.Workflow]
		err  error
	}
}

func (m *WorkflowServiceMock) OnGet(id string, wf *models.Workflow, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getResults[id] = mockResult[models.Workflow]{value: wf, err: err}
}

func (m *WorkflowServiceMock) OnCreate(wf *models.Workflow, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createResult = &mockResult[models.Workflow]{value: wf, err: err}
}

func (m *WorkflowServiceMock) OnList(page *models.Page[models.Workflow], err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listResult = &struct {
		page *models.Page[models.Workflow]
		err  error
	}{page: page, err: err}
}

func (m *WorkflowServiceMock) Create(_ context.Context, _ *models.Workflow, _ ...mbflow.RequestOption) (*models.Workflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.createResult != nil {
		return m.createResult.value, m.createResult.err
	}
	return nil, fmt.Errorf("mock: unexpected Create call")
}

func (m *WorkflowServiceMock) Get(_ context.Context, id string, _ ...mbflow.RequestOption) (*models.Workflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.getResults[id]; ok {
		return r.value, r.err
	}
	return nil, fmt.Errorf("mock: unexpected Get(%q) call", id)
}

func (m *WorkflowServiceMock) Update(_ context.Context, id string, _ *models.Workflow, _ ...mbflow.RequestOption) (*models.Workflow, error) {
	return nil, fmt.Errorf("mock: unexpected Update(%q) call", id)
}

func (m *WorkflowServiceMock) Delete(_ context.Context, id string, _ ...mbflow.RequestOption) error {
	return fmt.Errorf("mock: unexpected Delete(%q) call", id)
}

func (m *WorkflowServiceMock) List(_ context.Context, _ *models.ListOptions, _ ...mbflow.RequestOption) (*models.Page[models.Workflow], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listResult != nil {
		return m.listResult.page, m.listResult.err
	}
	return nil, fmt.Errorf("mock: unexpected List call")
}

// --- ExecutionServiceMock ---

type ExecutionServiceMock struct {
	mu         sync.RWMutex
	runResults map[string]mockResult[models.Execution]
	getResults map[string]mockResult[models.Execution]
}

func (m *ExecutionServiceMock) OnRun(workflowID string, exec *models.Execution, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.runResults == nil {
		m.runResults = make(map[string]mockResult[models.Execution])
	}
	m.runResults[workflowID] = mockResult[models.Execution]{value: exec, err: err}
}

func (m *ExecutionServiceMock) OnGet(id string, exec *models.Execution, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getResults == nil {
		m.getResults = make(map[string]mockResult[models.Execution])
	}
	m.getResults[id] = mockResult[models.Execution]{value: exec, err: err}
}

func (m *ExecutionServiceMock) Run(_ context.Context, wfID string, _ map[string]any, _ ...mbflow.RequestOption) (*models.Execution, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.runResults[wfID]; ok {
		return r.value, r.err
	}
	return nil, fmt.Errorf("mock: unexpected Run(%q) call", wfID)
}

func (m *ExecutionServiceMock) Get(_ context.Context, id string, _ ...mbflow.RequestOption) (*models.Execution, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.getResults[id]; ok {
		return r.value, r.err
	}
	return nil, fmt.Errorf("mock: unexpected Get(%q) call", id)
}

func (m *ExecutionServiceMock) List(_ context.Context, _ *models.ListOptions, _ ...mbflow.RequestOption) (*models.Page[models.Execution], error) {
	return nil, fmt.Errorf("mock: unexpected List call")
}

func (m *ExecutionServiceMock) Cancel(_ context.Context, id string, _ ...mbflow.RequestOption) (*models.Execution, error) {
	return nil, fmt.Errorf("mock: unexpected Cancel(%q) call", id)
}

func (m *ExecutionServiceMock) Retry(_ context.Context, id string, _ ...mbflow.RequestOption) (*models.Execution, error) {
	return nil, fmt.Errorf("mock: unexpected Retry(%q) call", id)
}

// --- TriggerServiceMock ---

type TriggerServiceMock struct {
	mu         sync.RWMutex
	getResults map[string]mockResult[models.Trigger]
}

func (m *TriggerServiceMock) OnGet(id string, trigger *models.Trigger, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getResults[id] = mockResult[models.Trigger]{value: trigger, err: err}
}

func (m *TriggerServiceMock) Create(_ context.Context, _ *models.Trigger, _ ...mbflow.RequestOption) (*models.Trigger, error) {
	return nil, fmt.Errorf("mock: unexpected Create call")
}

func (m *TriggerServiceMock) Get(_ context.Context, id string, _ ...mbflow.RequestOption) (*models.Trigger, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.getResults[id]; ok {
		return r.value, r.err
	}
	return nil, fmt.Errorf("mock: unexpected Get(%q) call", id)
}

func (m *TriggerServiceMock) Update(_ context.Context, id string, _ *models.Trigger, _ ...mbflow.RequestOption) (*models.Trigger, error) {
	return nil, fmt.Errorf("mock: unexpected Update(%q) call", id)
}

func (m *TriggerServiceMock) Delete(_ context.Context, id string, _ ...mbflow.RequestOption) error {
	return fmt.Errorf("mock: unexpected Delete(%q) call", id)
}

func (m *TriggerServiceMock) List(_ context.Context, _ *models.ListOptions, _ ...mbflow.RequestOption) (*models.Page[models.Trigger], error) {
	return nil, fmt.Errorf("mock: unexpected List call")
}

// --- CredentialServiceMock ---

type CredentialServiceMock struct {
	mu         sync.RWMutex
	getResults map[string]mockResult[models.Credential]
}

func (m *CredentialServiceMock) OnGet(id string, cred *models.Credential, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getResults[id] = mockResult[models.Credential]{value: cred, err: err}
}

func (m *CredentialServiceMock) Create(_ context.Context, _ *models.Credential, _ ...mbflow.RequestOption) (*models.Credential, error) {
	return nil, fmt.Errorf("mock: unexpected Create call")
}

func (m *CredentialServiceMock) Get(_ context.Context, id string, _ ...mbflow.RequestOption) (*models.Credential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.getResults[id]; ok {
		return r.value, r.err
	}
	return nil, fmt.Errorf("mock: unexpected Get(%q) call", id)
}

func (m *CredentialServiceMock) Update(_ context.Context, id string, _ *models.Credential, _ ...mbflow.RequestOption) (*models.Credential, error) {
	return nil, fmt.Errorf("mock: unexpected Update(%q) call", id)
}

func (m *CredentialServiceMock) Delete(_ context.Context, id string, _ ...mbflow.RequestOption) error {
	return fmt.Errorf("mock: unexpected Delete(%q) call", id)
}

func (m *CredentialServiceMock) List(_ context.Context, _ *models.ListOptions, _ ...mbflow.RequestOption) (*models.Page[models.Credential], error) {
	return nil, fmt.Errorf("mock: unexpected List call")
}
