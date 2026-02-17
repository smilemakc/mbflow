package engine

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

func TestSubWorkflow_SingleItem(t *testing.T) {
	t.Parallel()

	childWF := &models.Workflow{
		ID:   "child-wf",
		Name: "Child",
		Nodes: []*models.Node{
			{ID: "double", Name: "Double", Type: "transform", Config: map[string]any{}},
		},
	}

	mockExec := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			inputMap, _ := input.(map[string]any)
			if item, ok := inputMap["item"]; ok {
				if num, ok := item.(float64); ok {
					return map[string]any{"result": num * 2}, nil
				}
			}
			return map[string]any{"result": 0}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("transform", mockExec)

	loader := NewMockWorkflowLoader(map[string]*models.Workflow{
		"child-wf": childWF,
	})

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), loader)

	parentWF := &models.Workflow{
		ID:   "parent-wf",
		Name: "Parent",
		Nodes: []*models.Node{
			{
				ID:   "fanout",
				Name: "Fan Out",
				Type: "sub_workflow",
				Config: map[string]any{
					"workflow_id": "child-wf",
					"for_each":    "input.items",
				},
			},
		},
	}

	input := map[string]any{
		"items": []any{float64(5)},
	}
	execState := NewExecutionState("exec-1", "parent-wf", parentWF, input, nil)

	err := dagExec.Execute(context.Background(), execState, DefaultExecutionOptions())
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	output, ok := execState.GetNodeOutput("fanout")
	if !ok {
		t.Fatal("expected fanout output")
	}

	outputMap, ok := output.(map[string]any)
	if !ok {
		t.Fatalf("expected map output, got: %T", output)
	}

	summary, ok := outputMap["summary"].(map[string]any)
	if !ok {
		t.Fatal("expected summary in output")
	}

	if summary["total"] != 1 {
		t.Fatalf("expected total=1, got: %v", summary["total"])
	}
	if summary["completed"] != 1 {
		t.Fatalf("expected completed=1, got: %v", summary["completed"])
	}
}

func TestSubWorkflow_MultipleItems(t *testing.T) {
	t.Parallel()

	childWF := &models.Workflow{
		ID:   "child-wf",
		Name: "Child",
		Nodes: []*models.Node{
			{ID: "process", Name: "Process", Type: "transform", Config: map[string]any{}},
		},
	}

	mockExec := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			inputMap, _ := input.(map[string]any)
			item := inputMap["item"]
			return map[string]any{"processed": item}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("transform", mockExec)

	loader := NewMockWorkflowLoader(map[string]*models.Workflow{"child-wf": childWF})
	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), loader)

	parentWF := &models.Workflow{
		ID:   "parent-wf",
		Name: "Parent",
		Nodes: []*models.Node{
			{
				ID:   "fanout",
				Name: "Fan Out",
				Type: "sub_workflow",
				Config: map[string]any{
					"workflow_id":     "child-wf",
					"for_each":        "input.items",
					"max_parallelism": 2,
				},
			},
		},
	}

	input := map[string]any{
		"items": []any{"a", "b", "c", "d", "e"},
	}
	execState := NewExecutionState("exec-1", "parent-wf", parentWF, input, nil)

	err := dagExec.Execute(context.Background(), execState, DefaultExecutionOptions())
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	output, ok := execState.GetNodeOutput("fanout")
	if !ok {
		t.Fatal("expected fanout output")
	}

	outputMap := output.(map[string]any)
	summary := outputMap["summary"].(map[string]any)

	if summary["total"] != 5 {
		t.Fatalf("expected total=5, got: %v", summary["total"])
	}
	if summary["completed"] != 5 {
		t.Fatalf("expected completed=5, got: %v", summary["completed"])
	}
}

func TestSubWorkflow_CollectPartial(t *testing.T) {
	t.Parallel()

	childWF := &models.Workflow{
		ID:   "child-wf",
		Name: "Child",
		Nodes: []*models.Node{
			{ID: "process", Name: "Process", Type: "transform", Config: map[string]any{}},
		},
	}

	var callCount int64
	mockExec := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			n := atomic.AddInt64(&callCount, 1)
			if n == 3 {
				return nil, fmt.Errorf("simulated failure")
			}
			inputMap, _ := input.(map[string]any)
			return map[string]any{"processed": inputMap["item"]}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("transform", mockExec)

	loader := NewMockWorkflowLoader(map[string]*models.Workflow{"child-wf": childWF})
	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), loader)

	parentWF := &models.Workflow{
		ID:   "parent-wf",
		Name: "Parent",
		Nodes: []*models.Node{
			{
				ID:   "fanout",
				Name: "Fan Out",
				Type: "sub_workflow",
				Config: map[string]any{
					"workflow_id":     "child-wf",
					"for_each":        "input.items",
					"on_error":        "collect_partial",
					"max_parallelism": 1, // sequential to make failure deterministic
				},
			},
		},
	}

	input := map[string]any{
		"items": []any{"a", "b", "c", "d", "e"},
	}
	execState := NewExecutionState("exec-1", "parent-wf", parentWF, input, nil)

	err := dagExec.Execute(context.Background(), execState, DefaultExecutionOptions())
	if err != nil {
		t.Fatalf("collect_partial should not return error, got: %v", err)
	}

	output, _ := execState.GetNodeOutput("fanout")
	outputMap := output.(map[string]any)
	summary := outputMap["summary"].(map[string]any)

	if summary["failed"] != 1 {
		t.Fatalf("expected failed=1, got: %v", summary["failed"])
	}
	if summary["completed"] != 4 {
		t.Fatalf("expected completed=4, got: %v", summary["completed"])
	}
}

func TestSubWorkflow_FailFast(t *testing.T) {
	t.Parallel()

	childWF := &models.Workflow{
		ID:   "child-wf",
		Name: "Child",
		Nodes: []*models.Node{
			{ID: "process", Name: "Process", Type: "transform", Config: map[string]any{}},
		},
	}

	mockExec := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return nil, fmt.Errorf("always fails")
		},
	}

	registry := executor.NewManager()
	registry.Register("transform", mockExec)

	loader := NewMockWorkflowLoader(map[string]*models.Workflow{"child-wf": childWF})
	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), loader)

	parentWF := &models.Workflow{
		ID:   "parent-wf",
		Name: "Parent",
		Nodes: []*models.Node{
			{
				ID:   "fanout",
				Name: "Fan Out",
				Type: "sub_workflow",
				Config: map[string]any{
					"workflow_id": "child-wf",
					"for_each":    "input.items",
					"on_error":    "fail_fast",
				},
			},
		},
	}

	input := map[string]any{
		"items": []any{"a", "b", "c"},
	}
	execState := NewExecutionState("exec-1", "parent-wf", parentWF, input, nil)

	err := dagExec.Execute(context.Background(), execState, DefaultExecutionOptions())
	if err == nil {
		t.Fatal("expected error with fail_fast")
	}

	status, _ := execState.GetNodeStatus("fanout")
	if status != models.NodeExecutionStatusFailed {
		t.Fatalf("expected failed status, got: %s", status)
	}
}

func TestSubWorkflow_EmptyArray(t *testing.T) {
	t.Parallel()

	loader := NewMockWorkflowLoader(map[string]*models.Workflow{
		"child-wf": {ID: "child-wf", Name: "Child", Nodes: []*models.Node{{ID: "n1", Name: "N1", Type: "transform", Config: map[string]any{}}}},
	})

	registry := executor.NewManager()
	registry.Register("transform", &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{}, nil
		},
	})

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), loader)

	parentWF := &models.Workflow{
		ID:   "parent-wf",
		Name: "Parent",
		Nodes: []*models.Node{
			{
				ID: "fanout", Name: "Fan Out", Type: "sub_workflow",
				Config: map[string]any{
					"workflow_id": "child-wf",
					"for_each":    "input.items",
				},
			},
		},
	}

	execState := NewExecutionState("exec-1", "parent-wf", parentWF, map[string]any{
		"items": []any{},
	}, nil)

	err := dagExec.Execute(context.Background(), execState, DefaultExecutionOptions())
	if err != nil {
		t.Fatalf("empty array should not error: %v", err)
	}

	output, _ := execState.GetNodeOutput("fanout")
	outputMap := output.(map[string]any)
	summary := outputMap["summary"].(map[string]any)
	if summary["total"] != 0 {
		t.Fatalf("expected total=0, got: %v", summary["total"])
	}
}

func TestSubWorkflow_WorkflowNotFound(t *testing.T) {
	t.Parallel()

	// Empty loader - no workflows registered
	loader := NewMockWorkflowLoader(map[string]*models.Workflow{})

	registry := executor.NewManager()
	registry.Register("transform", &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{}, nil
		},
	})

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), loader)

	parentWF := &models.Workflow{
		ID:   "parent-wf",
		Name: "Parent",
		Nodes: []*models.Node{
			{
				ID: "fanout", Name: "Fan Out", Type: "sub_workflow",
				Config: map[string]any{
					"workflow_id": "nonexistent-wf",
					"for_each":    "input.items",
				},
			},
		},
	}

	execState := NewExecutionState("exec-1", "parent-wf", parentWF, map[string]any{
		"items": []any{"a"},
	}, nil)

	err := dagExec.Execute(context.Background(), execState, DefaultExecutionOptions())
	if err == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestSubWorkflow_InvalidConfig(t *testing.T) {
	t.Parallel()

	loader := NewMockWorkflowLoader(map[string]*models.Workflow{})
	registry := executor.NewManager()
	registry.Register("transform", &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return nil, nil
		},
	})

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), loader)

	// Missing workflow_id
	parentWF := &models.Workflow{
		ID:   "parent-wf",
		Name: "Parent",
		Nodes: []*models.Node{
			{
				ID: "fanout", Name: "Fan Out", Type: "sub_workflow",
				Config: map[string]any{
					"for_each": "input.items",
				},
			},
		},
	}

	execState := NewExecutionState("exec-1", "parent-wf", parentWF, map[string]any{
		"items": []any{"a"},
	}, nil)

	err := dagExec.Execute(context.Background(), execState, DefaultExecutionOptions())
	if err == nil {
		t.Fatal("expected error for missing workflow_id")
	}
}
