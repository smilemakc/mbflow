package builder

import (
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func TestNewWorkflow(t *testing.T) {
	wf := NewWorkflow("Test Workflow").
		AddNode(NewPassthroughNode("node1", "Test Node")).
		MustBuild()

	if wf.Name != "Test Workflow" {
		t.Errorf("expected name 'Test Workflow', got '%s'", wf.Name)
	}

	if wf.Status != models.WorkflowStatusDraft {
		t.Errorf("expected status 'draft', got '%s'", wf.Status)
	}
}

func TestWorkflowWithDescription(t *testing.T) {
	wf := NewWorkflow("Test",
		WithDescription("A test workflow"),
	).AddNode(NewPassthroughNode("node1", "Test Node")).
		MustBuild()

	if wf.Description != "A test workflow" {
		t.Errorf("expected description 'A test workflow', got '%s'", wf.Description)
	}
}

func TestWorkflowWithStatus(t *testing.T) {
	wf := NewWorkflow("Test",
		WithStatus(models.WorkflowStatusActive),
	).AddNode(NewPassthroughNode("node1", "Test Node")).
		MustBuild()

	if wf.Status != models.WorkflowStatusActive {
		t.Errorf("expected status 'active', got '%s'", wf.Status)
	}
}

func TestWorkflowWithVariable(t *testing.T) {
	wf := NewWorkflow("Test",
		WithVariable("api_key", "secret123"),
		WithVariable("url", "https://api.example.com"),
	).AddNode(NewPassthroughNode("node1", "Test Node")).
		MustBuild()

	if wf.Variables["api_key"] != "secret123" {
		t.Errorf("expected api_key 'secret123', got '%v'", wf.Variables["api_key"])
	}

	if wf.Variables["url"] != "https://api.example.com" {
		t.Errorf("expected url 'https://api.example.com', got '%v'", wf.Variables["url"])
	}
}

func TestWorkflowWithVariables(t *testing.T) {
	vars := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	wf := NewWorkflow("Test",
		WithVariables(vars),
	).AddNode(NewPassthroughNode("node1", "Test Node")).
		MustBuild()

	if wf.Variables["key1"] != "value1" {
		t.Errorf("expected key1 'value1', got '%v'", wf.Variables["key1"])
	}

	if wf.Variables["key2"] != 42 {
		t.Errorf("expected key2 42, got '%v'", wf.Variables["key2"])
	}
}

func TestWorkflowWithTags(t *testing.T) {
	wf := NewWorkflow("Test",
		WithTags("tag1", "tag2", "tag3"),
	).AddNode(NewPassthroughNode("node1", "Test Node")).
		MustBuild()

	if len(wf.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(wf.Tags))
	}

	if wf.Tags[0] != "tag1" || wf.Tags[1] != "tag2" || wf.Tags[2] != "tag3" {
		t.Errorf("unexpected tags: %v", wf.Tags)
	}
}

func TestWorkflowWithMetadata(t *testing.T) {
	wf := NewWorkflow("Test",
		WithMetadata("author", "John Doe"),
		WithMetadata("version", "1.0"),
	).AddNode(NewPassthroughNode("node1", "Test Node")).
		MustBuild()

	if wf.Metadata["author"] != "John Doe" {
		t.Errorf("expected author 'John Doe', got '%v'", wf.Metadata["author"])
	}

	if wf.Metadata["version"] != "1.0" {
		t.Errorf("expected version '1.0', got '%v'", wf.Metadata["version"])
	}
}

func TestWorkflowAddNode(t *testing.T) {
	wf := NewWorkflow("Test").
		AddNode(NewHTTPGetNode("node1", "Get Data", "https://api.example.com")).
		MustBuild()

	if len(wf.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(wf.Nodes))
	}

	if wf.Nodes[0].ID != "node1" {
		t.Errorf("expected node ID 'node1', got '%s'", wf.Nodes[0].ID)
	}
}

func TestWorkflowConnect(t *testing.T) {
	wf := NewWorkflow("Test").
		AddNode(NewHTTPGetNode("node1", "Get Data", "https://api.example.com")).
		AddNode(NewPassthroughNode("node2", "Process")).
		Connect("node1", "node2").
		MustBuild()

	if len(wf.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(wf.Edges))
	}

	edge := wf.Edges[0]
	if edge.From != "node1" || edge.To != "node2" {
		t.Errorf("expected edge from 'node1' to 'node2', got from '%s' to '%s'", edge.From, edge.To)
	}

	if edge.ID != "edge_node1_node2" {
		t.Errorf("expected auto-generated edge ID 'edge_node1_node2', got '%s'", edge.ID)
	}
}

func TestWorkflowDuplicateNodeID(t *testing.T) {
	_, err := NewWorkflow("Test").
		AddNode(NewHTTPGetNode("node1", "Get Data", "https://api.example.com")).
		AddNode(NewHTTPGetNode("node1", "Get More Data", "https://api.example.com")).
		Build()

	if err == nil {
		t.Error("expected error for duplicate node ID, got nil")
	}
}

func TestWorkflowAutoLayout(t *testing.T) {
	wf := NewWorkflow("Test", WithAutoLayout()).
		AddNode(NewHTTPGetNode("node1", "Get Data", "https://api.example.com")).
		AddNode(NewPassthroughNode("node2", "Process")).
		AddNode(NewHTTPGetNode("node3", "Send Data", "https://api.example.com")).
		MustBuild()

	if wf.Nodes[0].Position == nil {
		t.Error("expected node1 to have position")
	}

	if wf.Nodes[0].Position.X != 0 || wf.Nodes[0].Position.Y != 100 {
		t.Errorf("expected node1 position (0, 100), got (%f, %f)", wf.Nodes[0].Position.X, wf.Nodes[0].Position.Y)
	}

	if wf.Nodes[1].Position.X != 200 || wf.Nodes[1].Position.Y != 100 {
		t.Errorf("expected node2 position (200, 100), got (%f, %f)", wf.Nodes[1].Position.X, wf.Nodes[1].Position.Y)
	}

	if wf.Nodes[2].Position.X != 400 || wf.Nodes[2].Position.Y != 100 {
		t.Errorf("expected node3 position (400, 100), got (%f, %f)", wf.Nodes[2].Position.X, wf.Nodes[2].Position.Y)
	}
}

func TestWorkflowMustBuildPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustBuild to panic, but it didn't")
		}
	}()

	// Create invalid workflow (no nodes)
	NewWorkflow("Test").
		Connect("nonexistent1", "nonexistent2").
		MustBuild()
}
