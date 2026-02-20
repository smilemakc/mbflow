package mock_test

import (
	"context"
	"testing"

	"github.com/smilemakc/mbflow/go/sdk/mock"
	"github.com/smilemakc/mbflow/go/sdk/models"
)

func TestMockClient_WorkflowGet(t *testing.T) {
	m := mock.NewClient()
	m.Workflows().(*mock.WorkflowServiceMock).OnGet("wf-1", &models.Workflow{
		ID: "wf-1", Name: "Test",
	}, nil)

	wf, err := m.Workflows().Get(context.Background(), "wf-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if wf.Name != "Test" {
		t.Errorf("Name = %q", wf.Name)
	}
}

func TestMockClient_ExecutionRun(t *testing.T) {
	m := mock.NewClient()
	m.Executions().(*mock.ExecutionServiceMock).OnRun("wf-1", &models.Execution{
		ID: "exec-1", Status: models.ExecutionStatusCompleted,
	}, nil)

	exec, err := m.Executions().Run(context.Background(), "wf-1", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if exec.Status != models.ExecutionStatusCompleted {
		t.Errorf("Status = %q", exec.Status)
	}
}

func TestMockClient_UnexpectedCall(t *testing.T) {
	m := mock.NewClient()
	_, err := m.Workflows().Get(context.Background(), "unknown")
	if err == nil {
		t.Error("expected error for unexpected call")
	}
}
