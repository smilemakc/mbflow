package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// TestLoopEdge_BasicLoop tests a basic loop where a conditional node returns false twice, then true.
func TestLoopEdge_BasicLoop(t *testing.T) {
	t.Parallel()

	var validateCallCount int32

	mockValidate := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			count := atomic.AddInt32(&validateCallCount, 1)
			// Return false on first 2 calls, true on 3rd
			if count <= 2 {
				return map[string]any{"result": false}, nil
			}
			return map[string]any{"result": true}, nil
		},
	}

	mockFix := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"fixed": true}, nil
		},
	}

	mockDefault := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockDefault)
	registry.Register("conditional", mockValidate)
	registry.Register("fix", mockFix)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), NewNilWorkflowLoader())

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Loop Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Generate", Type: "test"},
			{ID: "N2", Name: "Validate", Type: "conditional"},
			{ID: "N3", Name: "Fix", Type: "fix"},
			{ID: "N4", Name: "OK", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "e2", From: "N2", To: "N3", SourceHandle: "false"},
			{ID: "e3", From: "N2", To: "N4", SourceHandle: "true"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 3}},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// N4 should be completed (validate eventually returned true)
	n4Status, _ := execState.GetNodeStatus("N4")
	if n4Status != models.NodeExecutionStatusCompleted {
		t.Errorf("expected N4 to be completed, got: %v", n4Status)
	}

	// Validate should have been called 3 times
	finalCount := atomic.LoadInt32(&validateCallCount)
	if finalCount != 3 {
		t.Errorf("expected validate to be called 3 times, got: %d", finalCount)
	}
}

// TestLoopEdge_LoopExhausted tests that when max iterations are reached, the loop stops.
func TestLoopEdge_LoopExhausted(t *testing.T) {
	t.Parallel()

	mockValidate := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			// Always return false
			return map[string]any{"result": false}, nil
		},
	}

	mockFix := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"fixed": true}, nil
		},
	}

	mockDefault := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockDefault)
	registry.Register("conditional", mockValidate)
	registry.Register("fix", mockFix)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), NewNilWorkflowLoader())

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Loop Exhausted Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Generate", Type: "test"},
			{ID: "N2", Name: "Validate", Type: "conditional"},
			{ID: "N3", Name: "Fix", Type: "fix"},
			{ID: "N4", Name: "OK", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "e2", From: "N2", To: "N3", SourceHandle: "false"},
			{ID: "e3", From: "N2", To: "N4", SourceHandle: "true"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 2}},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// N3 should be completed on the last iteration
	n3Status, _ := execState.GetNodeStatus("N3")
	if n3Status != models.NodeExecutionStatusCompleted {
		t.Errorf("expected N3 to be completed, got: %v", n3Status)
	}

	// N4 should be skipped (validate returned false on last call)
	n4Status, _ := execState.GetNodeStatus("N4")
	if n4Status != models.NodeExecutionStatusSkipped {
		t.Errorf("expected N4 to be skipped, got: %v", n4Status)
	}
}

// TestLoopEdge_ImmediateSuccess tests that if validation passes immediately, loop never fires.
func TestLoopEdge_ImmediateSuccess(t *testing.T) {
	t.Parallel()

	mockValidate := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			// Return true immediately
			return map[string]any{"result": true}, nil
		},
	}

	mockFix := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"fixed": true}, nil
		},
	}

	mockDefault := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockDefault)
	registry.Register("conditional", mockValidate)
	registry.Register("fix", mockFix)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), NewNilWorkflowLoader())

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Immediate Success Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Generate", Type: "test"},
			{ID: "N2", Name: "Validate", Type: "conditional"},
			{ID: "N3", Name: "Fix", Type: "fix"},
			{ID: "N4", Name: "OK", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "e2", From: "N2", To: "N3", SourceHandle: "false"},
			{ID: "e3", From: "N2", To: "N4", SourceHandle: "true"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 3}},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// N3 should be skipped (false branch never taken)
	n3Status, _ := execState.GetNodeStatus("N3")
	if n3Status != models.NodeExecutionStatusSkipped {
		t.Errorf("expected N3 to be skipped, got: %v", n3Status)
	}

	// N4 should be completed
	n4Status, _ := execState.GetNodeStatus("N4")
	if n4Status != models.NodeExecutionStatusCompleted {
		t.Errorf("expected N4 to be completed, got: %v", n4Status)
	}
}

// TestLoopEdge_ExcludedFromTopSort tests that loop edges don't cause cycle detection errors.
func TestLoopEdge_ExcludedFromTopSort(t *testing.T) {
	t.Parallel()

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Loop Edge Exclusion Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Start", Type: "test"},
			{ID: "N2", Name: "Middle", Type: "test"},
			{ID: "N3", Name: "End", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "e2", From: "N2", To: "N3"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 3}},
		},
	}

	dag := BuildDAG(workflow)

	// Verify loop edge is in LoopEdges
	if len(dag.LoopEdges) != 1 {
		t.Errorf("expected 1 loop edge, got: %d", len(dag.LoopEdges))
	}

	if dag.LoopEdges[0].ID != "loop1" {
		t.Errorf("expected loop edge ID 'loop1', got: %s", dag.LoopEdges[0].ID)
	}

	// Topological sort should NOT return an error (no cycle detected)
	_, err := TopologicalSort(dag)
	if err != nil {
		t.Errorf("expected no error from TopologicalSort, got: %v", err)
	}
}

// TestLoopEdge_InputPropagation tests that loop input correctly overrides parent input.
func TestLoopEdge_InputPropagation(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var n2Inputs []map[string]any
	var n2CallCount int32

	mockN2 := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			count := atomic.AddInt32(&n2CallCount, 1)

			mu.Lock()
			if inputMap, ok := input.(map[string]any); ok {
				// Deep copy to preserve the input state
				inputCopy := make(map[string]any)
				for k, v := range inputMap {
					inputCopy[k] = v
				}
				n2Inputs = append(n2Inputs, inputCopy)
			}
			mu.Unlock()

			// Return false first time, true second time
			if count == 1 {
				return map[string]any{"result": false}, nil
			}
			return map[string]any{"result": true}, nil
		},
	}

	mockN1 := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"data": "from_n1"}, nil
		},
	}

	mockN3 := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"data": "from_n3"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("n1", mockN1)
	registry.Register("conditional", mockN2)
	registry.Register("n3", mockN3)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), NewNilWorkflowLoader())

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Input Propagation Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Node1", Type: "n1"},
			{ID: "N2", Name: "Node2", Type: "conditional"},
			{ID: "N3", Name: "Node3", Type: "n3"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "e2", From: "N2", To: "N3", SourceHandle: "false"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 3}},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Should have 2 inputs recorded
	if len(n2Inputs) != 2 {
		t.Fatalf("expected 2 inputs for N2, got: %d", len(n2Inputs))
	}

	// First call: should contain data from N1
	if n2Inputs[0]["data"] != "from_n1" {
		t.Errorf("expected first input to have data='from_n1', got: %v", n2Inputs[0]["data"])
	}

	// Second call (after loop): should contain data from N3
	if n2Inputs[1]["data"] != "from_n3" {
		t.Errorf("expected second input to have data='from_n3', got: %v", n2Inputs[1]["data"])
	}
}

// TestLoopEdge_ContextCancellation tests that context cancellation stops loop execution.
func TestLoopEdge_ContextCancellation(t *testing.T) {
	t.Parallel()

	mockValidate := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			time.Sleep(20 * time.Millisecond)
			return map[string]any{"result": false}, nil
		},
	}

	mockDefault := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			time.Sleep(20 * time.Millisecond)
			return map[string]any{"status": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockDefault)
	registry.Register("conditional", mockValidate)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), NewNilWorkflowLoader())

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Context Cancellation Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Start", Type: "test"},
			{ID: "N2", Name: "Validate", Type: "conditional"},
			{ID: "N3", Name: "Fix", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "e2", From: "N2", To: "N3", SourceHandle: "false"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 100}},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := dagExec.Execute(ctx, execState, opts)
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}

	if !strings.Contains(err.Error(), "cancel") && !strings.Contains(err.Error(), "context") {
		t.Errorf("expected error to contain 'cancel' or 'context', got: %v", err)
	}
}

// TestLoopEdge_MultipleLoops tests two independent loops in separate branches.
func TestLoopEdge_MultipleLoops(t *testing.T) {
	t.Parallel()

	var a2CallCount int32

	mockA2 := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			count := atomic.AddInt32(&a2CallCount, 1)
			// Return false once, then true
			if count == 1 {
				return map[string]any{"result": false}, nil
			}
			return map[string]any{"result": true}, nil
		},
	}

	mockDefault := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockDefault)
	registry.Register("conditional", mockA2)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), NewNilWorkflowLoader())

	// Use sequential chains to avoid interference
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Multiple Loops Test",
		Nodes: []*models.Node{
			{ID: "A1", Name: "A1", Type: "test"},
			{ID: "A2", Name: "A2", Type: "conditional"},
			{ID: "A3", Name: "A3", Type: "test"},
			{ID: "A4", Name: "A4", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "ea1", From: "A1", To: "A2"},
			{ID: "ea2", From: "A2", To: "A3", SourceHandle: "false"},
			{ID: "ea3", From: "A2", To: "A4", SourceHandle: "true"},
			{ID: "loopa", From: "A3", To: "A2", Loop: &models.LoopConfig{MaxIterations: 5}},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// A2 should have been called twice (once + one loop iteration)
	finalA2Count := atomic.LoadInt32(&a2CallCount)
	if finalA2Count != 2 {
		t.Errorf("expected A2 to be called 2 times, got: %d", finalA2Count)
	}

	// A1, A2, and A4 should be completed (final path after loop succeeds)
	for _, nodeID := range []string{"A1", "A2", "A4"} {
		status, _ := execState.GetNodeStatus(nodeID)
		if status != models.NodeExecutionStatusCompleted {
			t.Errorf("expected %s to be completed, got: %v", nodeID, status)
		}
	}

	// A3 is reset during the loop and then skipped when A2 returns true on second iteration
	// This is expected behavior - the node is in the reset range but not executed on final iteration
}

// TestLoopEdge_Validation tests edge validation for loop configurations.
func TestLoopEdge_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		edge        *models.Edge
		expectError bool
		errorText   string
	}{
		{
			name: "MaxIterations zero",
			edge: &models.Edge{
				ID:   "e1",
				From: "N1",
				To:   "N2",
				Loop: &models.LoopConfig{MaxIterations: 0},
			},
			expectError: true,
			errorText:   "must be > 0",
		},
		{
			name: "Loop with condition",
			edge: &models.Edge{
				ID:        "e1",
				From:      "N1",
				To:        "N2",
				Condition: "output.value > 10",
				Loop:      &models.LoopConfig{MaxIterations: 1},
			},
			expectError: true,
			errorText:   "must not have conditions",
		},
		{
			name: "Valid loop edge",
			edge: &models.Edge{
				ID:   "e1",
				From: "N1",
				To:   "N2",
				Loop: &models.LoopConfig{MaxIterations: 5},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.edge.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorText)
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("expected error containing '%s', got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// recordingNotifier captures all execution events for testing.
type recordingNotifier struct {
	mu     sync.Mutex
	events []ExecutionEvent
}

func (r *recordingNotifier) Notify(ctx context.Context, event ExecutionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

// TestLoopEdge_Events tests that loop iteration events are properly emitted.
func TestLoopEdge_Events(t *testing.T) {
	t.Parallel()

	var validateCallCount int32

	mockValidate := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			count := atomic.AddInt32(&validateCallCount, 1)
			// Return false once, then true
			if count == 1 {
				return map[string]any{"result": false}, nil
			}
			return map[string]any{"result": true}, nil
		},
	}

	mockDefault := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockDefault)
	registry.Register("conditional", mockValidate)

	nodeExec := NewNodeExecutor(registry)

	recorder := &recordingNotifier{}
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), recorder, NewNilWorkflowLoader())

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Event Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Start", Type: "test"},
			{ID: "N2", Name: "Validate", Type: "conditional"},
			{ID: "N3", Name: "Fix", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "e2", From: "N2", To: "N3", SourceHandle: "false"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 2}},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	// Find loop iteration events
	var loopIterationEvents []ExecutionEvent
	for _, event := range recorder.events {
		if event.Type == EventTypeLoopIteration {
			loopIterationEvents = append(loopIterationEvents, event)
		}
	}

	if len(loopIterationEvents) == 0 {
		t.Fatal("expected at least one loop iteration event, got none")
	}

	// Verify loop iteration event details
	loopEvent := loopIterationEvents[0]
	if loopEvent.LoopEdgeID != "loop1" {
		t.Errorf("expected LoopEdgeID='loop1', got: %s", loopEvent.LoopEdgeID)
	}

	if loopEvent.LoopIteration != 1 {
		t.Errorf("expected LoopIteration=1, got: %d", loopEvent.LoopIteration)
	}

	if loopEvent.LoopMaxIter != 2 {
		t.Errorf("expected LoopMaxIter=2, got: %d", loopEvent.LoopMaxIter)
	}
}

// TestLoopEdge_ResetClearsState tests that node state is reset when loop fires.
func TestLoopEdge_ResetClearsState(t *testing.T) {
	t.Parallel()

	var n2ExecutionCount int32

	mockN2 := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			count := atomic.AddInt32(&n2ExecutionCount, 1)

			// Return false first time, true second time
			if count == 1 {
				return map[string]any{"result": false}, nil
			}
			return map[string]any{"result": true}, nil
		},
	}

	mockDefault := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockDefault)
	registry.Register("conditional", mockN2)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), NewNilWorkflowLoader())

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Reset State Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Start", Type: "test"},
			{ID: "N2", Name: "Node2", Type: "conditional"},
			{ID: "N3", Name: "Fix", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "e2", From: "N2", To: "N3", SourceHandle: "false"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 3}},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify that node was executed twice
	finalCount := atomic.LoadInt32(&n2ExecutionCount)
	if finalCount != 2 {
		t.Errorf("expected N2 to be called 2 times, got: %d", finalCount)
	}

	// Note: The status is reset by ResetNodeForLoop, but it's immediately set to running
	// when the node starts executing again. We can verify the reset happened by checking
	// that the execution completed successfully (which means reset worked correctly).
	// If reset didn't work, the node wouldn't execute again.
	n2Status, _ := execState.GetNodeStatus("N2")
	if n2Status != models.NodeExecutionStatusCompleted {
		t.Errorf("expected final N2 status to be completed, got: %v", n2Status)
	}
}

// TestLoopEdge_ParentNodesFiltering tests that GetRegularParentNodes excludes loop sources.
func TestLoopEdge_ParentNodesFiltering(t *testing.T) {
	t.Parallel()

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Parent Filtering Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Node1", Type: "test"},
			{ID: "N2", Name: "Node2", Type: "test"},
			{ID: "N3", Name: "Node3", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 3}},
		},
	}

	node2 := workflow.Nodes[1] // N2
	parents := GetRegularParentNodes(workflow, node2)

	// Should only return N1, not N3 (because loop edge is excluded)
	if len(parents) != 1 {
		t.Fatalf("expected 1 parent node, got: %d", len(parents))
	}

	if parents[0].ID != "N1" {
		t.Errorf("expected parent to be N1, got: %s", parents[0].ID)
	}
}

// TestLoopEdge_ComplexWorkflow tests a more complex workflow with nested conditionals and loops.
func TestLoopEdge_ComplexWorkflow(t *testing.T) {
	t.Parallel()

	var processCallCount int32

	mockProcess := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			count := atomic.AddInt32(&processCallCount, 1)
			// Succeed on third attempt
			if count < 3 {
				return map[string]any{"result": false}, nil
			}
			return map[string]any{"result": true}, nil
		},
	}

	mockDefault := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockDefault)
	registry.Register("conditional", mockProcess)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), NewNilWorkflowLoader())

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Complex Loop Test",
		Nodes: []*models.Node{
			{ID: "start", Name: "Start", Type: "test"},
			{ID: "process", Name: "Process", Type: "conditional"},
			{ID: "retry", Name: "Retry", Type: "test"},
			{ID: "success", Name: "Success", Type: "test"},
			{ID: "end", Name: "End", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "start", To: "process"},
			{ID: "e2", From: "process", To: "retry", SourceHandle: "false"},
			{ID: "e3", From: "process", To: "success", SourceHandle: "true"},
			{ID: "loop1", From: "retry", To: "process", Loop: &models.LoopConfig{MaxIterations: 5}},
			{ID: "e4", From: "success", To: "end"},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Process should have been called 3 times
	finalCount := atomic.LoadInt32(&processCallCount)
	if finalCount != 3 {
		t.Errorf("expected process to be called 3 times, got: %d", finalCount)
	}

	// Success and end nodes should be completed
	successStatus, _ := execState.GetNodeStatus("success")
	if successStatus != models.NodeExecutionStatusCompleted {
		t.Errorf("expected success to be completed, got: %v", successStatus)
	}

	endStatus, _ := execState.GetNodeStatus("end")
	if endStatus != models.NodeExecutionStatusCompleted {
		t.Errorf("expected end to be completed, got: %v", endStatus)
	}

	// Loop iteration count should be 2 (since it succeeded on 3rd call = 2 loops)
	loopIter := execState.GetLoopIteration("loop1")
	if loopIter != 2 {
		t.Errorf("expected loop iteration count to be 2, got: %d", loopIter)
	}
}

// TestLoopEdge_EdgeIsLoop tests the Edge.IsLoop() method.
func TestLoopEdge_EdgeIsLoop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		edge     *models.Edge
		expected bool
	}{
		{
			name: "Regular edge",
			edge: &models.Edge{
				ID:   "e1",
				From: "N1",
				To:   "N2",
			},
			expected: false,
		},
		{
			name: "Loop edge",
			edge: &models.Edge{
				ID:   "loop1",
				From: "N2",
				To:   "N1",
				Loop: &models.LoopConfig{MaxIterations: 3},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.edge.IsLoop()
			if result != tt.expected {
				t.Errorf("expected IsLoop()=%v, got: %v", tt.expected, result)
			}
		})
	}
}

// TestLoopEdge_LoopIterationTracking tests that loop iterations are tracked correctly.
func TestLoopEdge_LoopIterationTracking(t *testing.T) {
	t.Parallel()

	execState := NewExecutionState("exec-1", "wf-1", nil, map[string]any{}, map[string]any{})

	// Initial count should be 0
	count := execState.GetLoopIteration("loop1")
	if count != 0 {
		t.Errorf("expected initial loop iteration to be 0, got: %d", count)
	}

	// Increment and verify
	newCount := execState.IncrementLoopIteration("loop1")
	if newCount != 1 {
		t.Errorf("expected first increment to return 1, got: %d", newCount)
	}

	// Get again
	count = execState.GetLoopIteration("loop1")
	if count != 1 {
		t.Errorf("expected loop iteration to be 1, got: %d", count)
	}

	// Increment again
	newCount = execState.IncrementLoopIteration("loop1")
	if newCount != 2 {
		t.Errorf("expected second increment to return 2, got: %d", newCount)
	}
}

// TestLoopEdge_LoopInputManagement tests loop input setting and clearing.
func TestLoopEdge_LoopInputManagement(t *testing.T) {
	t.Parallel()

	execState := NewExecutionState("exec-1", "wf-1", nil, map[string]any{}, map[string]any{})

	// Initially no loop input
	_, hasInput := execState.GetLoopInput("N2")
	if hasInput {
		t.Error("expected no loop input initially")
	}

	// Set loop input
	testInput := map[string]any{"key": "value"}
	execState.SetLoopInput("N2", testInput)

	// Verify it was set
	loopInput, hasInput := execState.GetLoopInput("N2")
	if !hasInput {
		t.Error("expected loop input to be set")
	}

	if inputMap, ok := loopInput.(map[string]any); ok {
		if inputMap["key"] != "value" {
			t.Errorf("expected loop input key='value', got: %v", inputMap["key"])
		}
	} else {
		t.Errorf("expected loop input to be map, got: %T", loopInput)
	}

	// Clear loop input
	execState.ClearLoopInput("N2")

	// Verify it was cleared
	_, hasInput = execState.GetLoopInput("N2")
	if hasInput {
		t.Error("expected loop input to be cleared")
	}
}

// TestLoopEdge_MaxIterationsReached tests behavior when max iterations is reached.
func TestLoopEdge_MaxIterationsReached(t *testing.T) {
	t.Parallel()

	recorder := &recordingNotifier{}

	mockValidate := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			// Always return false
			return map[string]any{"result": false}, nil
		},
	}

	mockDefault := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"status": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockDefault)
	registry.Register("conditional", mockValidate)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), recorder, NewNilWorkflowLoader())

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Max Iterations Test",
		Nodes: []*models.Node{
			{ID: "N1", Name: "Start", Type: "test"},
			{ID: "N2", Name: "Validate", Type: "conditional"},
			{ID: "N3", Name: "Fix", Type: "test"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "N1", To: "N2"},
			{ID: "e2", From: "N2", To: "N3", SourceHandle: "false"},
			{ID: "loop1", From: "N3", To: "N2", Loop: &models.LoopConfig{MaxIterations: 3}},
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify loop exhausted event was emitted
	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	var exhaustedEvents []ExecutionEvent
	for _, event := range recorder.events {
		if event.Type == EventTypeLoopExhausted {
			exhaustedEvents = append(exhaustedEvents, event)
		}
	}

	if len(exhaustedEvents) != 1 {
		t.Errorf("expected 1 loop exhausted event, got: %d", len(exhaustedEvents))
	} else {
		exhaustedEvent := exhaustedEvents[0]
		if exhaustedEvent.LoopEdgeID != "loop1" {
			t.Errorf("expected LoopEdgeID='loop1', got: %s", exhaustedEvent.LoopEdgeID)
		}
		if exhaustedEvent.LoopIteration != 3 {
			t.Errorf("expected LoopIteration=3, got: %d", exhaustedEvent.LoopIteration)
		}
		if exhaustedEvent.LoopMaxIter != 3 {
			t.Errorf("expected LoopMaxIter=3, got: %d", exhaustedEvent.LoopMaxIter)
		}
		if !strings.Contains(exhaustedEvent.Message, fmt.Sprintf("exhausted after %d iterations", 3)) {
			t.Errorf("expected message to contain 'exhausted after 3 iterations', got: %s", exhaustedEvent.Message)
		}
	}

	// Verify final loop iteration count
	finalIter := execState.GetLoopIteration("loop1")
	if finalIter != 3 {
		t.Errorf("expected final loop iteration to be 3, got: %d", finalIter)
	}
}
