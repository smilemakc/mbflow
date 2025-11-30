package engine

import (
	"context"
	"sync"
	"testing"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// TestTopologicalSort_SimpleDAG tests topological sort on a simple DAG
func TestTopologicalSort_SimpleDAG(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node-1", Name: "Start", Type: "test"},
			{ID: "node-2", Name: "Middle", Type: "test"},
			{ID: "node-3", Name: "End", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "edge-1", From: "node-1", To: "node-2"},
			{ID: "edge-2", From: "node-2", To: "node-3"},
		},
	}

	dag := buildDAG(workflow)
	waves, err := topologicalSort(dag)

	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}

	// Should have 3 waves (linear chain)
	if len(waves) != 3 {
		t.Errorf("expected 3 waves, got %d", len(waves))
	}

	// Wave 0 should have node-1
	if len(waves[0]) != 1 || waves[0][0].ID != "node-1" {
		t.Errorf("wave 0 should have node-1")
	}

	// Wave 1 should have node-2
	if len(waves[1]) != 1 || waves[1][0].ID != "node-2" {
		t.Errorf("wave 1 should have node-2")
	}

	// Wave 2 should have node-3
	if len(waves[2]) != 1 || waves[2][0].ID != "node-3" {
		t.Errorf("wave 2 should have node-3")
	}
}

// TestTopologicalSort_ParallelDAG tests topological sort with parallel branches
func TestTopologicalSort_ParallelDAG(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node-1", Name: "Start", Type: "test"},
			{ID: "node-2", Name: "Branch A", Type: "test"},
			{ID: "node-3", Name: "Branch B", Type: "test"},
			{ID: "node-4", Name: "Join", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "edge-1", From: "node-1", To: "node-2"},
			{ID: "edge-2", From: "node-1", To: "node-3"},
			{ID: "edge-3", From: "node-2", To: "node-4"},
			{ID: "edge-4", From: "node-3", To: "node-4"},
		},
	}

	dag := buildDAG(workflow)
	waves, err := topologicalSort(dag)

	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}

	// Should have 3 waves
	if len(waves) != 3 {
		t.Errorf("expected 3 waves, got %d", len(waves))
	}

	// Wave 0: node-1
	if len(waves[0]) != 1 {
		t.Errorf("wave 0 should have 1 node, got %d", len(waves[0]))
	}

	// Wave 1: node-2 and node-3 (parallel)
	if len(waves[1]) != 2 {
		t.Errorf("wave 1 should have 2 nodes, got %d", len(waves[1]))
	}

	// Wave 2: node-4
	if len(waves[2]) != 1 {
		t.Errorf("wave 2 should have 1 node, got %d", len(waves[2]))
	}
}

// TestTopologicalSort_CycleDetection tests cycle detection
func TestTopologicalSort_CycleDetection(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node-1", Name: "A", Type: "test"},
			{ID: "node-2", Name: "B", Type: "test"},
			{ID: "node-3", Name: "C", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "edge-1", From: "node-1", To: "node-2"},
			{ID: "edge-2", From: "node-2", To: "node-3"},
			{ID: "edge-3", From: "node-3", To: "node-1"}, // Cycle!
		},
	}

	dag := buildDAG(workflow)
	_, err := topologicalSort(dag)

	if err == nil {
		t.Error("expected error for cyclic graph, got nil")
	}

	if err.Error() != "cycle detected in workflow graph" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestDAGExecutor_Execute_Success tests successful DAG execution
func TestDAGExecutor_Execute_Success(t *testing.T) {
	// Track execution order
	var executionOrder []string
	var mu sync.Mutex

	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID := config["nodeID"].(string)
			mu.Lock()
			executionOrder = append(executionOrder, nodeID)
			mu.Unlock()
			return map[string]interface{}{"result": "ok", "from": nodeID}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Variables: map[string]interface{}{
			"key": "value",
		},
		Nodes: []*models.Node{
			{ID: "node-1", Name: "Start", Type: "test", Config: map[string]interface{}{"nodeID": "node-1"}},
			{ID: "node-2", Name: "End", Type: "test", Config: map[string]interface{}{"nodeID": "node-2"}},
		},
		Edges: []*models.Edge{
			{ID: "edge-1", From: "node-1", To: "node-2"},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, workflow.Variables)
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Verify execution order
	if len(executionOrder) != 2 {
		t.Errorf("expected 2 executions, got %d", len(executionOrder))
	}

	if executionOrder[0] != "node-1" {
		t.Errorf("expected first execution to be node-1, got %s", executionOrder[0])
	}

	if executionOrder[1] != "node-2" {
		t.Errorf("expected second execution to be node-2, got %s", executionOrder[1])
	}

	// Verify node statuses
	status1, _ := execState.GetNodeStatus("node-1")
	if status1 != models.NodeExecutionStatusCompleted {
		t.Errorf("expected node-1 status completed, got %v", status1)
	}

	status2, _ := execState.GetNodeStatus("node-2")
	if status2 != models.NodeExecutionStatusCompleted {
		t.Errorf("expected node-2 status completed, got %v", status2)
	}
}

// TestDAGExecutor_Execute_ParallelExecution tests parallel execution within waves
func TestDAGExecutor_Execute_ParallelExecution(t *testing.T) {
	// Track concurrent executions
	var activeConcurrent int
	var maxConcurrent int
	var mu sync.Mutex

	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			mu.Lock()
			activeConcurrent++
			if activeConcurrent > maxConcurrent {
				maxConcurrent = activeConcurrent
			}
			mu.Unlock()

			// Simulate work
			// time.Sleep(10 * time.Millisecond)

			mu.Lock()
			activeConcurrent--
			mu.Unlock()

			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec)

	// Create workflow with parallel branches
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node-1", Name: "Start", Type: "test", Config: map[string]interface{}{}},
			{ID: "node-2", Name: "Parallel A", Type: "test", Config: map[string]interface{}{}},
			{ID: "node-3", Name: "Parallel B", Type: "test", Config: map[string]interface{}{}},
			{ID: "node-4", Name: "Parallel C", Type: "test", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "edge-1", From: "node-1", To: "node-2"},
			{ID: "edge-2", From: "node-1", To: "node-3"},
			{ID: "edge-3", From: "node-1", To: "node-4"},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Verify all nodes completed
	for _, node := range workflow.Nodes {
		status, _ := execState.GetNodeStatus(node.ID)
		if status != models.NodeExecutionStatusCompleted {
			t.Errorf("node %s not completed, status: %v", node.ID, status)
		}
	}
}

// TestGetParentNodes tests getting parent nodes
func TestGetParentNodes(t *testing.T) {
	workflow := &models.Workflow{
		Nodes: []*models.Node{
			{ID: "node-1", Name: "A"},
			{ID: "node-2", Name: "B"},
			{ID: "node-3", Name: "C"},
		},
		Edges: []*models.Edge{
			{From: "node-1", To: "node-3"},
			{From: "node-2", To: "node-3"},
		},
	}

	node3 := workflow.Nodes[2]
	parents := getParentNodes(workflow, node3)

	if len(parents) != 2 {
		t.Errorf("expected 2 parents, got %d", len(parents))
	}

	// Verify parent IDs
	parentIDs := make(map[string]bool)
	for _, p := range parents {
		parentIDs[p.ID] = true
	}

	if !parentIDs["node-1"] || !parentIDs["node-2"] {
		t.Error("expected parents to be node-1 and node-2")
	}
}
