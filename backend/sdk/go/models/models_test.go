package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/sdk/go/models"
)

func TestWorkflowJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := &models.Workflow{
		ID:          "wf-001",
		Name:        "Test Workflow",
		Description: "A test workflow",
		Version:     1,
		Status:      models.WorkflowStatusActive,
		Tags:        []string{"test", "demo"},
		Nodes: []*models.Node{
			{
				ID:   "node-1",
				Name: "Start",
				Type: "start",
				Config: map[string]any{
					"timeout": 30,
				},
				Position: &models.Position{X: 100, Y: 200},
			},
			{
				ID:   "node-2",
				Name: "End",
				Type: "end",
				Config: map[string]any{},
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "edge-1",
				From: "node-1",
				To:   "node-2",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored models.Workflow
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if restored.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", restored.ID, original.ID)
	}
	if restored.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", restored.Name, original.Name)
	}
	if restored.Status != original.Status {
		t.Errorf("Status mismatch: got %q, want %q", restored.Status, original.Status)
	}
	if len(restored.Nodes) != len(original.Nodes) {
		t.Errorf("Nodes count mismatch: got %d, want %d", len(restored.Nodes), len(original.Nodes))
	}
	if len(restored.Edges) != len(original.Edges) {
		t.Errorf("Edges count mismatch: got %d, want %d", len(restored.Edges), len(original.Edges))
	}
	if restored.Nodes[0].Position == nil {
		t.Fatal("expected Position to be non-nil after round-trip")
	}
	if restored.Nodes[0].Position.X != 100 || restored.Nodes[0].Position.Y != 200 {
		t.Errorf("Position mismatch: got %+v", restored.Nodes[0].Position)
	}
}

func TestExecutionStatusConstants(t *testing.T) {
	statuses := []models.ExecutionStatus{
		models.ExecutionStatusPending,
		models.ExecutionStatusRunning,
		models.ExecutionStatusCompleted,
		models.ExecutionStatusFailed,
		models.ExecutionStatusCancelled,
		models.ExecutionStatusTimeout,
	}
	for _, s := range statuses {
		if s == "" {
			t.Errorf("ExecutionStatus constant must not be empty")
		}
	}

	nodeStatuses := []models.NodeExecutionStatus{
		models.NodeExecutionStatusPending,
		models.NodeExecutionStatusRunning,
		models.NodeExecutionStatusCompleted,
		models.NodeExecutionStatusFailed,
		models.NodeExecutionStatusSkipped,
		models.NodeExecutionStatusCancelled,
	}
	for _, s := range nodeStatuses {
		if s == "" {
			t.Errorf("NodeExecutionStatus constant must not be empty")
		}
	}
}

func TestPageGeneric(t *testing.T) {
	page := &models.Page[models.Workflow]{
		Items: []*models.Workflow{
			{ID: "wf-1", Name: "First"},
			{ID: "wf-2", Name: "Second"},
		},
		Total: 2,
	}

	if page.Total != 2 {
		t.Errorf("Total mismatch: got %d, want 2", page.Total)
	}
	if len(page.Items) != 2 {
		t.Errorf("Items count mismatch: got %d, want 2", len(page.Items))
	}
	if page.Items[0].ID != "wf-1" {
		t.Errorf("First item ID mismatch: got %q, want %q", page.Items[0].ID, "wf-1")
	}

	data, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("marshal Page[Workflow] failed: %v", err)
	}

	var restored models.Page[models.Workflow]
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal Page[Workflow] failed: %v", err)
	}

	if restored.Total != page.Total {
		t.Errorf("Total mismatch after round-trip: got %d, want %d", restored.Total, page.Total)
	}
	if len(restored.Items) != len(page.Items) {
		t.Errorf("Items count mismatch after round-trip: got %d, want %d", len(restored.Items), len(page.Items))
	}
}
