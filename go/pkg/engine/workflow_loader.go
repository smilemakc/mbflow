package engine

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// WorkflowLoader loads workflow definitions for sub-workflow execution.
type WorkflowLoader interface {
	LoadWorkflow(ctx context.Context, workflowID string) (*models.Workflow, error)
}

// MockWorkflowLoader is a test implementation of WorkflowLoader.
type MockWorkflowLoader struct {
	workflows map[string]*models.Workflow
}

func NewMockWorkflowLoader(workflows map[string]*models.Workflow) *MockWorkflowLoader {
	return &MockWorkflowLoader{workflows: workflows}
}

func (m *MockWorkflowLoader) LoadWorkflow(_ context.Context, workflowID string) (*models.Workflow, error) {
	wf, ok := m.workflows[workflowID]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}
	return wf, nil
}

// NilWorkflowLoader returns errors for all loads. Used when sub-workflow is not needed.
type NilWorkflowLoader struct{}

func NewNilWorkflowLoader() *NilWorkflowLoader {
	return &NilWorkflowLoader{}
}

func (n *NilWorkflowLoader) LoadWorkflow(_ context.Context, workflowID string) (*models.Workflow, error) {
	return nil, fmt.Errorf("workflow loading not configured (requested: %s)", workflowID)
}
