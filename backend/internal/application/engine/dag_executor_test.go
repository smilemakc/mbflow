package engine

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// TestTopologicalSort_SimpleDAG tests topological sort on a simple DAG
func TestTopologicalSort_SimpleDAG(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	dagExec := NewDAGExecutor(nodeExec, nil) // no observer for tests

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
	t.Parallel()
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
	dagExec := NewDAGExecutor(nodeExec, nil) // no observer for tests

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
	t.Parallel()
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

// TestDAGExecutor_ConditionalEdge_TrueBranch tests that true branch is executed when conditional node returns true
func TestDAGExecutor_ConditionalEdge_TrueBranch(t *testing.T) {
	t.Parallel()
	var executedNodes []string
	var mu sync.Mutex

	// Mock executor that returns true for conditional node
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID := config["nodeID"].(string)
			mu.Lock()
			executedNodes = append(executedNodes, nodeID)
			mu.Unlock()

			// Conditional node returns true
			if nodeID == "conditional" {
				return true, nil
			}
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("conditional", mockExec)
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	// Workflow: start -> conditional -> true-branch, false-branch
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Conditional Test",
		Nodes: []*models.Node{
			{ID: "start", Name: "Start", Type: "test", Config: map[string]interface{}{"nodeID": "start"}},
			{ID: "conditional", Name: "Check", Type: "conditional", Config: map[string]interface{}{"nodeID": "conditional"}},
			{ID: "true-branch", Name: "True Branch", Type: "test", Config: map[string]interface{}{"nodeID": "true-branch"}},
			{ID: "false-branch", Name: "False Branch", Type: "test", Config: map[string]interface{}{"nodeID": "false-branch"}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "start", To: "conditional"},
			{ID: "e2", From: "conditional", To: "true-branch", SourceHandle: "true"},
			{ID: "e3", From: "conditional", To: "false-branch", SourceHandle: "false"},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Verify true-branch was executed (conditional returned true)
	trueBranchStatus, _ := execState.GetNodeStatus("true-branch")
	if trueBranchStatus != models.NodeExecutionStatusCompleted {
		t.Errorf("expected true-branch to be completed, got %v", trueBranchStatus)
	}

	// Verify false-branch was skipped
	falseBranchStatus, _ := execState.GetNodeStatus("false-branch")
	if falseBranchStatus != models.NodeExecutionStatusSkipped {
		t.Errorf("expected false-branch to be skipped, got %v", falseBranchStatus)
	}
}

// TestDAGExecutor_ConditionalEdge_FalseBranch tests that false branch is executed when conditional node returns false
func TestDAGExecutor_ConditionalEdge_FalseBranch(t *testing.T) {
	t.Parallel()
	var executedNodes []string
	var mu sync.Mutex

	// Mock executor that returns false for conditional node
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID := config["nodeID"].(string)
			mu.Lock()
			executedNodes = append(executedNodes, nodeID)
			mu.Unlock()

			// Conditional node returns false
			if nodeID == "conditional" {
				return false, nil
			}
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("conditional", mockExec)
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	// Workflow: start -> conditional -> true-branch, false-branch
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Conditional Test",
		Nodes: []*models.Node{
			{ID: "start", Name: "Start", Type: "test", Config: map[string]interface{}{"nodeID": "start"}},
			{ID: "conditional", Name: "Check", Type: "conditional", Config: map[string]interface{}{"nodeID": "conditional"}},
			{ID: "true-branch", Name: "True Branch", Type: "test", Config: map[string]interface{}{"nodeID": "true-branch"}},
			{ID: "false-branch", Name: "False Branch", Type: "test", Config: map[string]interface{}{"nodeID": "false-branch"}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "start", To: "conditional"},
			{ID: "e2", From: "conditional", To: "true-branch", SourceHandle: "true"},
			{ID: "e3", From: "conditional", To: "false-branch", SourceHandle: "false"},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Verify true-branch was skipped (conditional returned false)
	trueBranchStatus, _ := execState.GetNodeStatus("true-branch")
	if trueBranchStatus != models.NodeExecutionStatusSkipped {
		t.Errorf("expected true-branch to be skipped, got %v", trueBranchStatus)
	}

	// Verify false-branch was executed
	falseBranchStatus, _ := execState.GetNodeStatus("false-branch")
	if falseBranchStatus != models.NodeExecutionStatusCompleted {
		t.Errorf("expected false-branch to be completed, got %v", falseBranchStatus)
	}
}

// TestDAGExecutor_MultiParentWithConditionalEdges tests OR semantics for multi-parent nodes
// A node with multiple incoming edges should execute if at least one edge passes its condition
func TestDAGExecutor_MultiParentWithConditionalEdges(t *testing.T) {
	t.Parallel()
	var executedNodes []string
	var mu sync.Mutex

	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID := config["nodeID"].(string)
			mu.Lock()
			executedNodes = append(executedNodes, nodeID)
			mu.Unlock()

			// Analyze node returns score=50
			if nodeID == "analyze" {
				return map[string]interface{}{"score": 50}, nil
			}
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	// Workflow simulating content_generation pattern:
	// generate -> analyze -> merge (conditional: score >= 80)
	// generate -> merge (unconditional)
	// Since score=50, analyze->merge condition fails, but generate->merge succeeds
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Multi-Parent Test",
		Nodes: []*models.Node{
			{ID: "generate", Name: "Generate", Type: "test", Config: map[string]interface{}{"nodeID": "generate"}},
			{ID: "analyze", Name: "Analyze", Type: "test", Config: map[string]interface{}{"nodeID": "analyze"}},
			{ID: "merge", Name: "Merge", Type: "test", Config: map[string]interface{}{"nodeID": "merge"}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "generate", To: "analyze"},
			{ID: "e2", From: "analyze", To: "merge", Condition: "output.score >= 80"}, // Will fail (score=50)
			{ID: "e3", From: "generate", To: "merge"},                                 // No condition - should succeed
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Verify all nodes executed (none skipped)
	for _, node := range workflow.Nodes {
		status, _ := execState.GetNodeStatus(node.ID)
		if status != models.NodeExecutionStatusCompleted {
			t.Errorf("node %s should be completed, got %v", node.ID, status)
		}
	}

	// Verify execution order and count
	if len(executedNodes) != 3 {
		t.Errorf("expected 3 nodes executed, got %d", len(executedNodes))
	}
}

// TestDAGExecutor_MultiParentAllConditionsFail tests that node is skipped when all incoming edges fail
func TestDAGExecutor_MultiParentAllConditionsFail(t *testing.T) {
	t.Parallel()
	var executedNodes []string
	var mu sync.Mutex

	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID := config["nodeID"].(string)
			mu.Lock()
			executedNodes = append(executedNodes, nodeID)
			mu.Unlock()

			// Both parent nodes return score that fails merge conditions
			if nodeID == "parent1" || nodeID == "parent2" {
				return map[string]interface{}{"score": 10}, nil
			}
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	// Workflow where merge has two incoming edges, both with failing conditions
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "All Conditions Fail Test",
		Nodes: []*models.Node{
			{ID: "parent1", Name: "Parent1", Type: "test", Config: map[string]interface{}{"nodeID": "parent1"}},
			{ID: "parent2", Name: "Parent2", Type: "test", Config: map[string]interface{}{"nodeID": "parent2"}},
			{ID: "merge", Name: "Merge", Type: "test", Config: map[string]interface{}{"nodeID": "merge"}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "parent1", To: "merge", Condition: "output.score >= 80"}, // Will fail (score=10)
			{ID: "e2", From: "parent2", To: "merge", Condition: "output.score >= 80"}, // Will fail (score=10)
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Verify parent nodes executed
	parent1Status, _ := execState.GetNodeStatus("parent1")
	if parent1Status != models.NodeExecutionStatusCompleted {
		t.Errorf("parent1 should be completed, got %v", parent1Status)
	}

	parent2Status, _ := execState.GetNodeStatus("parent2")
	if parent2Status != models.NodeExecutionStatusCompleted {
		t.Errorf("parent2 should be completed, got %v", parent2Status)
	}

	// Verify merge was skipped (all conditions failed)
	mergeStatus, _ := execState.GetNodeStatus("merge")
	if mergeStatus != models.NodeExecutionStatusSkipped {
		t.Errorf("merge should be skipped, got %v", mergeStatus)
	}

	// Verify only 2 nodes executed (not merge)
	if len(executedNodes) != 2 {
		t.Errorf("expected 2 nodes executed, got %d", len(executedNodes))
	}
}

// TestDAGExecutor_ConditionalEdge_MapOutputWithResult tests conditional with map output containing "result" key
func TestDAGExecutor_ConditionalEdge_MapOutputWithResult(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		result        bool
		sourceHandle  string
		shouldExecute bool
		targetBranch  string
	}{
		{
			name:          "map output result=true with sourceHandle=true",
			result:        true,
			sourceHandle:  "true",
			shouldExecute: true,
			targetBranch:  "true-branch",
		},
		{
			name:          "map output result=true with sourceHandle=false",
			result:        true,
			sourceHandle:  "false",
			shouldExecute: false,
			targetBranch:  "false-branch",
		},
		{
			name:          "map output result=false with sourceHandle=true",
			result:        false,
			sourceHandle:  "true",
			shouldExecute: false,
			targetBranch:  "true-branch",
		},
		{
			name:          "map output result=false with sourceHandle=false",
			result:        false,
			sourceHandle:  "false",
			shouldExecute: true,
			targetBranch:  "false-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockExec := &mockExecutor{
				executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
					nodeID := config["nodeID"].(string)
					// Conditional node returns map with "result" key
					if nodeID == "conditional" {
						return map[string]interface{}{
							"result":  tt.result,
							"message": "some metadata",
						}, nil
					}
					return map[string]interface{}{"result": "ok"}, nil
				},
			}

			registry := executor.NewManager()
			registry.Register("conditional", mockExec)
			registry.Register("test", mockExec)

			nodeExec := NewNodeExecutor(registry)
			dagExec := NewDAGExecutor(nodeExec, nil)

			workflow := &models.Workflow{
				ID:   "wf-1",
				Name: "Conditional Map Output Test",
				Nodes: []*models.Node{
					{ID: "start", Name: "Start", Type: "test", Config: map[string]interface{}{"nodeID": "start"}},
					{ID: "conditional", Name: "Check", Type: "conditional", Config: map[string]interface{}{"nodeID": "conditional"}},
					{ID: "true-branch", Name: "True Branch", Type: "test", Config: map[string]interface{}{"nodeID": "true-branch"}},
					{ID: "false-branch", Name: "False Branch", Type: "test", Config: map[string]interface{}{"nodeID": "false-branch"}},
				},
				Edges: []*models.Edge{
					{ID: "e1", From: "start", To: "conditional"},
					{ID: "e2", From: "conditional", To: "true-branch", SourceHandle: "true"},
					{ID: "e3", From: "conditional", To: "false-branch", SourceHandle: "false"},
				},
			}

			execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
			opts := DefaultExecutionOptions()

			err := dagExec.Execute(context.Background(), execState, opts)
			if err != nil {
				t.Fatalf("DAG execution failed: %v", err)
			}

			// Check target branch status
			targetStatus, _ := execState.GetNodeStatus(tt.targetBranch)
			if tt.shouldExecute {
				if targetStatus != models.NodeExecutionStatusCompleted {
					t.Errorf("expected %s to be completed, got %v", tt.targetBranch, targetStatus)
				}
			} else {
				if targetStatus != models.NodeExecutionStatusSkipped {
					t.Errorf("expected %s to be skipped, got %v", tt.targetBranch, targetStatus)
				}
			}
		})
	}
}

// TestDAGExecutor_ConditionalEdge_UnknownSourceHandle tests conditional with unknown sourceHandle
func TestDAGExecutor_ConditionalEdge_UnknownSourceHandle(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID := config["nodeID"].(string)
			if nodeID == "conditional" {
				return false, nil // Returns false but sourceHandle is unknown
			}
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("conditional", mockExec)
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Unknown SourceHandle Test",
		Nodes: []*models.Node{
			{ID: "start", Name: "Start", Type: "test", Config: map[string]interface{}{"nodeID": "start"}},
			{ID: "conditional", Name: "Check", Type: "conditional", Config: map[string]interface{}{"nodeID": "conditional"}},
			{ID: "branch", Name: "Branch", Type: "test", Config: map[string]interface{}{"nodeID": "branch"}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "start", To: "conditional"},
			{ID: "e2", From: "conditional", To: "branch", SourceHandle: "unknown-handle"}, // Unknown handle
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Unknown handle should default to pass (line 721)
	branchStatus, _ := execState.GetNodeStatus("branch")
	if branchStatus != models.NodeExecutionStatusCompleted {
		t.Errorf("expected branch to be completed (unknown handle defaults to pass), got %v", branchStatus)
	}
}

// TestDAGExecutor_ConditionalEdge_MapOutputWithoutResult tests map output without "result" key
func TestDAGExecutor_ConditionalEdge_MapOutputWithoutResult(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID := config["nodeID"].(string)
			if nodeID == "conditional" {
				// Returns map without "result" key
				return map[string]interface{}{
					"data":   "some data",
					"status": "ok",
				}, nil
			}
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("conditional", mockExec)
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Map Without Result Test",
		Nodes: []*models.Node{
			{ID: "start", Name: "Start", Type: "test", Config: map[string]interface{}{"nodeID": "start"}},
			{ID: "conditional", Name: "Check", Type: "conditional", Config: map[string]interface{}{"nodeID": "conditional"}},
			{ID: "branch", Name: "Branch", Type: "test", Config: map[string]interface{}{"nodeID": "branch"}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "start", To: "conditional"},
			{ID: "e2", From: "conditional", To: "branch", SourceHandle: "true"},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Map without "result" key defaults to pass (line 739)
	branchStatus, _ := execState.GetNodeStatus("branch")
	if branchStatus != models.NodeExecutionStatusCompleted {
		t.Errorf("expected branch to be completed (map without result defaults to pass), got %v", branchStatus)
	}
}

// TestDAGExecutor_ConditionalEdge_MapOutputNonBooleanResult tests map with non-boolean "result"
func TestDAGExecutor_ConditionalEdge_MapOutputNonBooleanResult(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID := config["nodeID"].(string)
			if nodeID == "conditional" {
				// Returns map with non-boolean "result"
				return map[string]interface{}{
					"result": "success", // String instead of bool
				}, nil
			}
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("conditional", mockExec)
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Non-Boolean Result Test",
		Nodes: []*models.Node{
			{ID: "start", Name: "Start", Type: "test", Config: map[string]interface{}{"nodeID": "start"}},
			{ID: "conditional", Name: "Check", Type: "conditional", Config: map[string]interface{}{"nodeID": "conditional"}},
			{ID: "branch", Name: "Branch", Type: "test", Config: map[string]interface{}{"nodeID": "branch"}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "start", To: "conditional"},
			{ID: "e2", From: "conditional", To: "branch", SourceHandle: "true"},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Non-boolean result defaults to pass (line 739)
	branchStatus, _ := execState.GetNodeStatus("branch")
	if branchStatus != models.NodeExecutionStatusCompleted {
		t.Errorf("expected branch to be completed (non-boolean result defaults to pass), got %v", branchStatus)
	}
}

// TestDAGExecutor_EdgeCondition_CompilationError tests edge with invalid condition syntax
func TestDAGExecutor_EdgeCondition_CompilationError(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return map[string]interface{}{"score": 50}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Invalid Condition Syntax Test",
		Nodes: []*models.Node{
			{ID: "source", Name: "Source", Type: "test", Config: map[string]interface{}{}},
			{ID: "target", Name: "Target", Type: "test", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "source", To: "target", Condition: "output.score >= && 80"}, // Invalid syntax
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Target should be skipped due to condition compilation error
	targetStatus, _ := execState.GetNodeStatus("target")
	if targetStatus != models.NodeExecutionStatusSkipped {
		t.Errorf("expected target to be skipped (invalid condition syntax), got %v", targetStatus)
	}
}

// TestDAGExecutor_EdgeCondition_RuntimeError tests edge condition with runtime error
func TestDAGExecutor_EdgeCondition_RuntimeError(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return map[string]interface{}{"data": "value"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Condition Runtime Error Test",
		Nodes: []*models.Node{
			{ID: "source", Name: "Source", Type: "test", Config: map[string]interface{}{}},
			{ID: "target", Name: "Target", Type: "test", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "source", To: "target", Condition: "output.score >= 80"}, // score doesn't exist
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Target should be skipped due to runtime error (missing field)
	targetStatus, _ := execState.GetNodeStatus("target")
	if targetStatus != models.NodeExecutionStatusSkipped {
		t.Errorf("expected target to be skipped (condition runtime error), got %v", targetStatus)
	}
}

// TestDAGExecutor_EdgeCondition_NonBooleanResult tests condition returning non-boolean
func TestDAGExecutor_EdgeCondition_NonBooleanResult(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return map[string]interface{}{"score": 50}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Non-Boolean Condition Result Test",
		Nodes: []*models.Node{
			{ID: "source", Name: "Source", Type: "test", Config: map[string]interface{}{}},
			{ID: "target", Name: "Target", Type: "test", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "source", To: "target", Condition: "output.score"}, // Returns number, not bool
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Target should be skipped because condition returns non-boolean (number)
	targetStatus, _ := execState.GetNodeStatus("target")
	if targetStatus != models.NodeExecutionStatusSkipped {
		t.Errorf("expected target to be skipped (condition returns non-boolean), got %v", targetStatus)
	}
}

// TestDAGExecutor_EdgeCondition_EmptyCondition tests edge with empty condition string
func TestDAGExecutor_EdgeCondition_EmptyCondition(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Empty Condition Test",
		Nodes: []*models.Node{
			{ID: "source", Name: "Source", Type: "test", Config: map[string]interface{}{}},
			{ID: "target", Name: "Target", Type: "test", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "source", To: "target", Condition: ""}, // Empty condition = always pass
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Target should be executed (empty condition = always pass)
	targetStatus, _ := execState.GetNodeStatus("target")
	if targetStatus != models.NodeExecutionStatusCompleted {
		t.Errorf("expected target to be completed (empty condition always passes), got %v", targetStatus)
	}
}

// TestDAGExecutor_shouldExecuteNode_InvalidEdge tests behavior when edge references non-existent source node
func TestDAGExecutor_shouldExecuteNode_InvalidEdge(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Invalid Edge Test",
		Nodes: []*models.Node{
			{ID: "node1", Name: "Node 1", Type: "test", Config: map[string]interface{}{}},
			{ID: "node2", Name: "Node 2", Type: "test", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "nonexistent", To: "node2"}, // Invalid source node
			{ID: "e2", From: "node1", To: "node2"},       // Also connect node1 to node2
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})

	// Mark node1 as completed
	execState.SetNodeStatus("node1", models.NodeExecutionStatusCompleted)
	execState.SetNodeOutput("node1", map[string]interface{}{"result": "ok"})

	// Check if node2 should execute (it should, because node1 edge is valid even though nonexistent edge is invalid)
	shouldExecute, _ := dagExec.shouldExecuteNode(execState, workflow.Nodes[1])

	if !shouldExecute {
		t.Error("expected node2 to execute because it has one valid incoming edge from node1")
	}

	// Now test with only invalid edge by marking node1 as skipped
	execState.SetNodeStatus("node1", models.NodeExecutionStatusSkipped)

	// Now node2 should not execute because the only valid source (node1) is skipped
	shouldExecute2, skipReason := dagExec.shouldExecuteNode(execState, workflow.Nodes[1])

	if shouldExecute2 {
		t.Error("expected node2 to not execute when only valid source is skipped")
	}

	if skipReason == "" {
		t.Error("expected skip reason to be set")
	}
}

// TestDAGExecutor_shouldExecuteNode_SourceNotCompleted tests when source node is in pending/running state
func TestDAGExecutor_shouldExecuteNode_SourceNotCompleted(t *testing.T) {
	t.Parallel()
	// This tests the edge case where shouldExecuteNode encounters a source node that's not completed yet
	// In normal wave execution this shouldn't happen, but we test the defensive check

	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Source Not Completed Test",
		Nodes: []*models.Node{
			{ID: "source", Name: "Source", Type: "test", Config: map[string]interface{}{}},
			{ID: "target", Name: "Target", Type: "test", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "source", To: "target"},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})

	// Manually set source node to "running" status to simulate the edge case
	execState.SetNodeStatus("source", models.NodeExecutionStatusRunning)

	// Now check if target should execute
	shouldExecute, skipReason := dagExec.shouldExecuteNode(execState, workflow.Nodes[1])

	if shouldExecute {
		t.Error("expected target to not execute when source is running")
	}

	if skipReason == "" {
		t.Error("expected skip reason to be set")
	}

	if !stringContains(skipReason, "not completed") {
		t.Errorf("expected skip reason to mention 'not completed', got: %s", skipReason)
	}
}

// TestDAGExecutor_shouldExecuteNode_SourceFailed tests when source node failed
func TestDAGExecutor_shouldExecuteNode_SourceFailed(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Source Failed Test",
		Nodes: []*models.Node{
			{ID: "source", Name: "Source", Type: "test", Config: map[string]interface{}{}},
			{ID: "target", Name: "Target", Type: "test", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "source", To: "target"},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})

	// Manually set source node to "failed" status
	execState.SetNodeStatus("source", models.NodeExecutionStatusFailed)

	// Check if target should execute
	shouldExecute, skipReason := dagExec.shouldExecuteNode(execState, workflow.Nodes[1])

	if shouldExecute {
		t.Error("expected target to not execute when source failed")
	}

	if !stringContains(skipReason, "not completed") {
		t.Errorf("expected skip reason to mention 'not completed', got: %s", skipReason)
	}
}

// Helper function to check if string contains substring
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPtrString(t *testing.T) {
	t.Parallel()
	str := "test string"
	ptr := ptrString(str)

	if ptr == nil {
		t.Fatal("ptrString returned nil")
	}

	if *ptr != str {
		t.Errorf("expected %s, got %s", str, *ptr)
	}
}

func TestContainsError(t *testing.T) {
	t.Parallel()
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	errors := []error{err1, err2}

	// Test found
	if !containsError(errors, err1) {
		t.Error("should find err1")
	}

	// Test not found
	if containsError(errors, err3) {
		t.Error("should not find err3")
	}

	// Test empty slice
	emptyErrors := []error{}
	if containsError(emptyErrors, err1) {
		t.Error("empty slice should not contain any error")
	}
}
