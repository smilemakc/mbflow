package builder_test

import (
	"testing"

	"github.com/smilemakc/mbflow/sdk/go/builder"
	"github.com/smilemakc/mbflow/sdk/go/models"
)

func TestWorkflowBuilder_Simple(t *testing.T) {
	wf, err := builder.NewWorkflow("Test Workflow",
		builder.WithDescription("A simple test"),
		builder.WithVariable("key", "value"),
		builder.WithTag("test"),
	).
		AddNode("n1", "First", "http",
			builder.WithConfig("url", "https://example.com"),
			builder.WithConfig("method", "GET"),
		).
		AddNode("n2", "Second", "transform",
			builder.WithConfig("type", "passthrough"),
		).
		Connect("n1", "n2").
		Build()

	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if wf.Name != "Test Workflow" {
		t.Errorf("Name = %q", wf.Name)
	}
	if wf.Description != "A simple test" {
		t.Errorf("Description = %q", wf.Description)
	}
	if len(wf.Nodes) != 2 {
		t.Fatalf("Nodes len = %d, want 2", len(wf.Nodes))
	}
	if len(wf.Edges) != 1 {
		t.Fatalf("Edges len = %d, want 1", len(wf.Edges))
	}
	if wf.Edges[0].From != "n1" || wf.Edges[0].To != "n2" {
		t.Errorf("Edge = %s -> %s", wf.Edges[0].From, wf.Edges[0].To)
	}
	if wf.Variables["key"] != "value" {
		t.Errorf("Variables[key] = %v", wf.Variables["key"])
	}
	if wf.Status != models.WorkflowStatusDraft {
		t.Errorf("Status = %q, want draft", wf.Status)
	}
}

func TestWorkflowBuilder_InvalidConnect(t *testing.T) {
	_, err := builder.NewWorkflow("Bad").
		AddNode("n1", "First", "http").
		Connect("n1", "nonexistent").
		Build()

	if err == nil {
		t.Error("expected error for invalid Connect target")
	}
}

func TestWorkflowBuilder_MustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from MustBuild")
		}
	}()

	builder.NewWorkflow("Bad").
		Connect("a", "b").
		MustBuild()
}

func TestWorkflowBuilder_AutoLayout(t *testing.T) {
	wf, err := builder.NewWorkflow("Layout Test").
		AddNode("n1", "A", "http").
		AddNode("n2", "B", "http").
		Connect("n1", "n2").
		WithAutoLayout().
		Build()

	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, n := range wf.Nodes {
		if n.Position == nil {
			t.Errorf("node %s has no position after auto layout", n.ID)
		}
	}
}
