package engine

import (
	"testing"

	pkgengine "github.com/smilemakc/mbflow/pkg/engine"
	"github.com/smilemakc/mbflow/pkg/models"
)

func TestCreateCheckpoint(t *testing.T) {
	t.Parallel()
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node-1", Name: "Node 1"},
			{ID: "node-2", Name: "Node 2"},
		},
		Edges: []*models.Edge{},
	}

	execState := pkgengine.NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{"key": "value"})

	// Set some node statuses and outputs
	execState.SetNodeStatus("node-1", models.NodeExecutionStatusCompleted)
	execState.SetNodeOutput("node-1", map[string]any{"result": "ok"})
	execState.SetNodeStatus("node-2", models.NodeExecutionStatusRunning)

	checkpoint := CreateCheckpoint(execState, 1)

	if checkpoint.ExecutionID != "exec-1" {
		t.Errorf("expected ExecutionID exec-1, got %s", checkpoint.ExecutionID)
	}

	if checkpoint.WorkflowID != "wf-1" {
		t.Errorf("expected WorkflowID wf-1, got %s", checkpoint.WorkflowID)
	}

	if checkpoint.WaveIndex != 1 {
		t.Errorf("expected WaveIndex 1, got %d", checkpoint.WaveIndex)
	}

	// Only completed nodes should be in checkpoint
	if len(checkpoint.CompletedNodes) != 1 {
		t.Errorf("expected 1 completed node, got %d", len(checkpoint.CompletedNodes))
	}

	if checkpoint.CompletedNodes[0] != "node-1" {
		t.Errorf("expected node-1 in completed nodes, got %s", checkpoint.CompletedNodes[0])
	}

	// Check variables
	if checkpoint.Variables["key"] != "value" {
		t.Error("expected variables to be copied")
	}
}

func TestRestoreFromCheckpoint(t *testing.T) {
	t.Parallel()
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node-1", Name: "Node 1"},
			{ID: "node-2", Name: "Node 2"},
		},
		Edges: []*models.Edge{},
	}

	checkpoint := &ExecutionCheckpoint{
		ExecutionID:    "exec-1",
		WorkflowID:     "wf-1",
		WaveIndex:      2,
		CompletedNodes: []string{"node-1"},
		NodeOutputs:    map[string]any{"node-1": map[string]any{"result": "ok"}},
		NodeStatuses:   map[string]models.NodeExecutionStatus{"node-1": models.NodeExecutionStatusCompleted},
		Variables:      map[string]any{"restored": "true"},
	}

	execState := RestoreFromCheckpoint(checkpoint, workflow, map[string]any{})

	if execState.ExecutionID != "exec-1" {
		t.Errorf("expected ExecutionID exec-1, got %s", execState.ExecutionID)
	}

	status, ok := execState.GetNodeStatus("node-1")
	if !ok || status != models.NodeExecutionStatusCompleted {
		t.Error("expected node-1 status to be restored")
	}

	output, ok := execState.GetNodeOutput("node-1")
	if !ok {
		t.Error("expected node-1 output to be restored")
	}

	if outputMap, ok := output.(map[string]any); ok {
		if outputMap["result"] != "ok" {
			t.Error("expected output to match checkpoint")
		}
	} else {
		t.Error("output should be a map")
	}

	if execState.Variables["restored"] != "true" {
		t.Error("expected variables to be restored")
	}
}

func TestCheckpoint_Serialization(t *testing.T) {
	t.Parallel()
	checkpoint := &ExecutionCheckpoint{
		ExecutionID:    "exec-1",
		WorkflowID:     "wf-1",
		WaveIndex:      1,
		CompletedNodes: []string{"node-1", "node-2"},
		NodeOutputs:    map[string]any{"node-1": "output1"},
		NodeStatuses:   map[string]models.NodeExecutionStatus{"node-1": models.NodeExecutionStatusCompleted},
		Variables:      map[string]any{"key": "value"},
	}

	// Serialize
	data, err := checkpoint.Serialize()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Deserialize
	restored, err := DeserializeCheckpoint(data)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	if restored.ExecutionID != checkpoint.ExecutionID {
		t.Errorf("expected ExecutionID %s, got %s", checkpoint.ExecutionID, restored.ExecutionID)
	}

	if len(restored.CompletedNodes) != len(checkpoint.CompletedNodes) {
		t.Errorf("expected %d completed nodes, got %d", len(checkpoint.CompletedNodes), len(restored.CompletedNodes))
	}
}

func TestDeserializeCheckpoint_InvalidData(t *testing.T) {
	t.Parallel()
	_, err := DeserializeCheckpoint([]byte("invalid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestValidateCheckpoint(t *testing.T) {
	t.Parallel()
	workflow := &models.Workflow{
		ID:    "wf-1",
		Nodes: []*models.Node{{ID: "node-1"}, {ID: "node-2"}},
	}

	tests := []struct {
		name        string
		checkpoint  *ExecutionCheckpoint
		expectError bool
	}{
		{
			name: "valid checkpoint",
			checkpoint: &ExecutionCheckpoint{
				WorkflowID:     "wf-1",
				CompletedNodes: []string{"node-1"},
			},
			expectError: false,
		},
		{
			name: "workflow ID mismatch",
			checkpoint: &ExecutionCheckpoint{
				WorkflowID:     "wf-2",
				CompletedNodes: []string{"node-1"},
			},
			expectError: true,
		},
		{
			name: "non-existent node in checkpoint",
			checkpoint: &ExecutionCheckpoint{
				WorkflowID:     "wf-1",
				CompletedNodes: []string{"node-999"},
			},
			expectError: true,
		},
		{
			name: "empty completed nodes",
			checkpoint: &ExecutionCheckpoint{
				WorkflowID:     "wf-1",
				CompletedNodes: []string{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCheckpoint(tt.checkpoint, workflow)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error=%v, got error=%v", tt.expectError, err)
			}
		})
	}
}

func TestCheckpoint_GetNextWaveIndex(t *testing.T) {
	t.Parallel()
	checkpoint := &ExecutionCheckpoint{WaveIndex: 5}
	if checkpoint.GetNextWaveIndex() != 6 {
		t.Errorf("expected next wave 6, got %d", checkpoint.GetNextWaveIndex())
	}
}

func TestCheckpoint_IsNodeCompleted(t *testing.T) {
	t.Parallel()
	checkpoint := &ExecutionCheckpoint{
		CompletedNodes: []string{"node-1", "node-2"},
	}

	if !checkpoint.IsNodeCompleted("node-1") {
		t.Error("node-1 should be completed")
	}

	if !checkpoint.IsNodeCompleted("node-2") {
		t.Error("node-2 should be completed")
	}

	if checkpoint.IsNodeCompleted("node-3") {
		t.Error("node-3 should not be completed")
	}
}

func TestCheckpointManager(t *testing.T) {
	t.Parallel()
	manager := NewCheckpointManager()

	checkpoint := &ExecutionCheckpoint{
		ExecutionID: "exec-1",
		WorkflowID:  "wf-1",
		WaveIndex:   1,
	}

	// Save checkpoint
	manager.SaveCheckpoint(checkpoint)

	// Get checkpoint
	retrieved, ok := manager.GetCheckpoint("exec-1")
	if !ok {
		t.Error("expected to find checkpoint")
	}

	if retrieved.ExecutionID != checkpoint.ExecutionID {
		t.Error("retrieved checkpoint doesn't match")
	}

	// Get non-existent checkpoint
	_, ok = manager.GetCheckpoint("exec-999")
	if ok {
		t.Error("should not find non-existent checkpoint")
	}

	// Delete checkpoint
	manager.DeleteCheckpoint("exec-1")

	_, ok = manager.GetCheckpoint("exec-1")
	if ok {
		t.Error("checkpoint should be deleted")
	}
}

func TestCheckpointManager_ListCheckpoints(t *testing.T) {
	t.Parallel()
	manager := NewCheckpointManager()

	cp1 := &ExecutionCheckpoint{ExecutionID: "exec-1"}
	cp2 := &ExecutionCheckpoint{ExecutionID: "exec-2"}

	manager.SaveCheckpoint(cp1)
	manager.SaveCheckpoint(cp2)

	checkpoints := manager.ListCheckpoints()

	if len(checkpoints) != 2 {
		t.Errorf("expected 2 checkpoints, got %d", len(checkpoints))
	}
}
