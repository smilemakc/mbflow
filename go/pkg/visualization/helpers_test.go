package visualization

import (
	"os"
	"strings"
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func TestRenderWorkflow_Mermaid(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node1", Name: "Node 1", Type: "http"},
			{ID: "node2", Name: "Node 2", Type: "transform"},
		},
		Edges: []*models.Edge{
			{ID: "edge1", From: "node1", To: "node2"},
		},
	}

	diagram, err := RenderWorkflow(workflow, "mermaid", nil)
	if err != nil {
		t.Fatalf("RenderWorkflow failed: %v", err)
	}

	if diagram == "" {
		t.Error("expected non-empty diagram")
	}

	// Should contain flowchart declaration
	if !strings.Contains(diagram, "flowchart") {
		t.Error("expected diagram to contain 'flowchart'")
	}
}

func TestRenderWorkflow_ASCII(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node1", Name: "Node 1", Type: "http"},
		},
	}

	diagram, err := RenderWorkflow(workflow, "ascii", nil)
	if err != nil {
		t.Fatalf("RenderWorkflow failed: %v", err)
	}

	if diagram == "" {
		t.Error("expected non-empty diagram")
	}

	// Should contain node name
	if !strings.Contains(diagram, "Node 1") {
		t.Error("expected diagram to contain node name")
	}
}

func TestRenderWorkflow_WithOptions(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node1", Name: "Node 1", Type: "http"},
		},
	}

	opts := &RenderOptions{
		Direction:  "LR",
		ShowConfig: true,
	}

	diagram, err := RenderWorkflow(workflow, "mermaid", opts)
	if err != nil {
		t.Fatalf("RenderWorkflow failed: %v", err)
	}

	if diagram == "" {
		t.Error("expected non-empty diagram")
	}

	// Should use LR direction
	if !strings.Contains(diagram, "LR") {
		t.Error("expected diagram to use LR direction")
	}
}

func TestRenderWorkflow_UnsupportedFormat(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
	}

	_, err := RenderWorkflow(workflow, "invalid", nil)
	if err == nil {
		t.Error("expected error for unsupported format")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPrintWorkflow(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node1", Name: "Node 1", Type: "http"},
		},
	}

	// PrintWorkflow writes to stdout, just verify it doesn't error
	err := PrintWorkflow(workflow, "mermaid", nil)
	if err != nil {
		t.Errorf("PrintWorkflow failed: %v", err)
	}
}

func TestPrintWorkflow_Error(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
	}

	err := PrintWorkflow(workflow, "invalid", nil)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestSaveWorkflowToFile(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node1", Name: "Node 1", Type: "http"},
			{ID: "node2", Name: "Node 2", Type: "transform"},
		},
		Edges: []*models.Edge{
			{ID: "edge1", From: "node1", To: "node2"},
		},
	}

	// Create temp file
	tmpfile := "/tmp/test_workflow.mmd"
	defer os.Remove(tmpfile)

	err := SaveWorkflowToFile(workflow, "mermaid", tmpfile, nil)
	if err != nil {
		t.Fatalf("SaveWorkflowToFile failed: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if len(content) == 0 {
		t.Error("saved file is empty")
	}

	// Should contain flowchart
	if !strings.Contains(string(content), "flowchart") {
		t.Error("saved file doesn't contain flowchart")
	}
}

func TestSaveWorkflowToFile_Error(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
	}

	err := SaveWorkflowToFile(workflow, "invalid", "/tmp/test.txt", nil)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestSaveWorkflowToFile_ASCIIFormat(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node1", Name: "Node 1", Type: "http"},
		},
	}

	tmpfile := "/tmp/test_workflow.txt"
	defer os.Remove(tmpfile)

	err := SaveWorkflowToFile(workflow, "ascii", tmpfile, nil)
	if err != nil {
		t.Fatalf("SaveWorkflowToFile failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if !strings.Contains(string(content), "Node 1") {
		t.Error("saved file doesn't contain node name")
	}
}
