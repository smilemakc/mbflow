package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// TestDAGExecutor_NodeTimeout tests per-node timeout functionality
func TestDAGExecutor_NodeTimeout(t *testing.T) {
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			// Simulate slow operation
			select {
			case <-time.After(200 * time.Millisecond):
				return map[string]interface{}{"result": "completed"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Timeout Test",
		Nodes: []*models.Node{
			{
				ID:   "node-1",
				Name: "Slow Node",
				Type: "test",
				Config: map[string]interface{}{
					"timeout": 50, // 50ms timeout
				},
			},
		},
		Edges: []*models.Edge{},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err == nil {
		t.Error("expected timeout error")
	}

	status, _ := execState.GetNodeStatus("node-1")
	if status != models.NodeExecutionStatusFailed {
		t.Errorf("expected Failed status, got %v", status)
	}
}

// TestDAGExecutor_RetrySuccess tests successful retry after failures
func TestDAGExecutor_RetrySuccess(t *testing.T) {
	attempts := 0
	var mu sync.Mutex

	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			mu.Lock()
			attempts++
			currentAttempt := attempts
			mu.Unlock()

			if currentAttempt < 3 {
				return nil, errors.New("temporary error")
			}
			return map[string]interface{}{"result": "success"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:    "wf-1",
		Name:  "Retry Test",
		Nodes: []*models.Node{{ID: "node-1", Name: "Retry Node", Type: "test", Config: map[string]interface{}{}}},
		Edges: []*models.Edge{},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()
	opts.RetryPolicy = &RetryPolicy{
		MaxAttempts:     3,
		InitialDelay:    10 * time.Millisecond,
		BackoffStrategy: BackoffConstant,
	}

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Errorf("expected success after retry, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	status, _ := execState.GetNodeStatus("node-1")
	if status != models.NodeExecutionStatusCompleted {
		t.Errorf("expected Completed status, got %v", status)
	}
}

// TestDAGExecutor_ContinueOnError tests continue-on-error mode
func TestDAGExecutor_ContinueOnError(t *testing.T) {
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID := config["nodeID"].(string)
			if nodeID == "node-2" {
				return nil, errors.New("node-2 failed")
			}
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Continue On Error Test",
		Nodes: []*models.Node{
			{ID: "node-1", Name: "Node 1", Type: "test", Config: map[string]interface{}{"nodeID": "node-1"}},
			{ID: "node-2", Name: "Node 2", Type: "test", Config: map[string]interface{}{"nodeID": "node-2"}},
			{ID: "node-3", Name: "Node 3", Type: "test", Config: map[string]interface{}{"nodeID": "node-3"}},
		},
		Edges: []*models.Edge{},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()
	opts.ContinueOnError = true

	err := dagExec.Execute(context.Background(), execState, opts)
	if err == nil {
		t.Error("expected aggregated error")
	}

	// Check that node-1 and node-3 succeeded despite node-2 failing
	status1, _ := execState.GetNodeStatus("node-1")
	if status1 != models.NodeExecutionStatusCompleted {
		t.Errorf("node-1 should be completed, got %v", status1)
	}

	status2, _ := execState.GetNodeStatus("node-2")
	if status2 != models.NodeExecutionStatusFailed {
		t.Errorf("node-2 should be failed, got %v", status2)
	}

	status3, _ := execState.GetNodeStatus("node-3")
	if status3 != models.NodeExecutionStatusCompleted {
		t.Errorf("node-3 should be completed, got %v", status3)
	}
}

// TestDAGExecutor_NodePriority tests priority-based execution order
func TestDAGExecutor_NodePriority(t *testing.T) {
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
		Name: "Priority Test",
		Nodes: []*models.Node{
			{ID: "low", Name: "Low Priority", Type: "test", Config: map[string]interface{}{}, Metadata: map[string]interface{}{"priority": 1}},
			{ID: "high", Name: "High Priority", Type: "test", Config: map[string]interface{}{}, Metadata: map[string]interface{}{"priority": 10}},
			{ID: "medium", Name: "Medium Priority", Type: "test", Config: map[string]interface{}{}, Metadata: map[string]interface{}{"priority": 5}},
		},
		Edges: []*models.Edge{},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Verify all nodes completed (priority doesn't affect completion, just scheduling)
	for _, node := range workflow.Nodes {
		status, _ := execState.GetNodeStatus(node.ID)
		if status != models.NodeExecutionStatusCompleted {
			t.Errorf("node %s should be completed, got %v", node.ID, status)
		}
	}
}

// TestDAGExecutor_ContextCancellation tests graceful cancellation
func TestDAGExecutor_ContextCancellation(t *testing.T) {
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			select {
			case <-time.After(500 * time.Millisecond):
				return map[string]interface{}{"result": "ok"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:    "wf-1",
		Name:  "Cancellation Test",
		Nodes: []*models.Node{{ID: "node-1", Name: "Slow Node", Type: "test", Config: map[string]interface{}{}}},
		Edges: []*models.Edge{},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := dagExec.Execute(ctx, execState, opts)
	if err == nil {
		t.Error("expected cancellation error")
	}
}

// TestDAGExecutor_MemoryLimit tests memory limit enforcement
func TestDAGExecutor_MemoryLimit(t *testing.T) {
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			// Generate large output
			largeData := make([]byte, 1000)
			return map[string]interface{}{"data": largeData}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	workflow := &models.Workflow{
		ID:    "wf-1",
		Name:  "Memory Limit Test",
		Nodes: []*models.Node{{ID: "node-1", Name: "Large Output", Type: "test", Config: map[string]interface{}{}}},
		Edges: []*models.Edge{},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()
	opts.MaxOutputSize = 100 // 100 bytes limit

	err := dagExec.Execute(context.Background(), execState, opts)
	if err == nil {
		t.Error("expected memory limit error")
	}

	status, _ := execState.GetNodeStatus("node-1")
	if status != models.NodeExecutionStatusFailed {
		t.Errorf("expected Failed status due to memory limit, got %v", status)
	}
}

// TestSortNodesByPriority tests node priority sorting
func TestSortNodesByPriority(t *testing.T) {
	nodes := []*models.Node{
		{ID: "low", Metadata: map[string]interface{}{"priority": 1}},
		{ID: "high", Metadata: map[string]interface{}{"priority": 10}},
		{ID: "medium", Metadata: map[string]interface{}{"priority": 5}},
		{ID: "default", Metadata: map[string]interface{}{}},
	}

	sorted := sortNodesByPriority(nodes)

	// Expected order: high (10) -> medium (5) -> low (1) -> default (0)
	if sorted[0].ID != "high" {
		t.Errorf("expected 'high' first, got %s", sorted[0].ID)
	}
	if sorted[1].ID != "medium" {
		t.Errorf("expected 'medium' second, got %s", sorted[1].ID)
	}
	if sorted[2].ID != "low" {
		t.Errorf("expected 'low' third, got %s", sorted[2].ID)
	}
	if sorted[3].ID != "default" {
		t.Errorf("expected 'default' fourth, got %s", sorted[3].ID)
	}
}

// TestAggregatedError tests aggregated error functionality
func TestAggregatedError(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	aggErr := &AggregatedError{
		Message: "multiple errors",
		Errors:  []error{err1, err2},
	}

	errMsg := aggErr.Error()
	if errMsg == "" {
		t.Error("expected non-empty error message")
	}

	// Test empty errors
	emptyAggErr := &AggregatedError{
		Message: "no errors",
		Errors:  []error{},
	}

	if emptyAggErr.Error() != "no errors" {
		t.Errorf("expected 'no errors', got %s", emptyAggErr.Error())
	}
}

// TestConditionCacheIntegration tests condition cache in DAG executor
func TestConditionCacheIntegration(t *testing.T) {
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return map[string]interface{}{"score": 100}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, nil)

	// Test that cache is used across multiple edge evaluations with same condition
	workflow := &models.Workflow{
		ID:   "wf-1",
		Name: "Cache Test",
		Nodes: []*models.Node{
			{ID: "node-1", Name: "Source", Type: "test", Config: map[string]interface{}{}},
			{ID: "node-2", Name: "Target 1", Type: "test", Config: map[string]interface{}{}},
			{ID: "node-3", Name: "Target 2", Type: "test", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "node-1", To: "node-2", Condition: "output.score >= 50"},
			{ID: "e2", From: "node-1", To: "node-3", Condition: "output.score >= 50"}, // Same condition
		},
	}

	execState := NewExecutionState("exec-1", "wf-1", workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// All nodes should complete
	for _, node := range workflow.Nodes {
		status, _ := execState.GetNodeStatus(node.ID)
		if status != models.NodeExecutionStatusCompleted {
			t.Errorf("node %s should be completed, got %v", node.ID, status)
		}
	}

	// Verify cache has the condition
	if dagExec.conditionCache.Len() == 0 {
		t.Error("expected condition to be cached")
	}
}
