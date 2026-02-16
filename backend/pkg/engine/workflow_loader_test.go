package engine

import (
	"context"
	"testing"

	"github.com/smilemakc/mbflow/pkg/models"
)

func TestMockWorkflowLoader_LoadWorkflow(t *testing.T) {
	t.Parallel()

	childWF := &models.Workflow{
		ID:   "child-wf-1",
		Name: "Child Workflow",
		Nodes: []*models.Node{
			{ID: "step1", Name: "Step 1", Type: "transform"},
		},
	}

	loader := NewMockWorkflowLoader(map[string]*models.Workflow{
		"child-wf-1": childWF,
	})

	wf, err := loader.LoadWorkflow(context.Background(), "child-wf-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if wf.ID != "child-wf-1" {
		t.Fatalf("expected workflow ID child-wf-1, got: %s", wf.ID)
	}

	_, err = loader.LoadWorkflow(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestNilWorkflowLoader_AlwaysErrors(t *testing.T) {
	t.Parallel()

	loader := NewNilWorkflowLoader()
	_, err := loader.LoadWorkflow(context.Background(), "any-id")
	if err == nil {
		t.Fatal("expected error from NilWorkflowLoader")
	}
}
