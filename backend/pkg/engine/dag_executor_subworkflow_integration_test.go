package engine

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// TestSubWorkflow_Integration_ChildWithLoop tests the full parent-child flow
// where each child workflow contains a loop (generate → check → fix → loop back).
// Pattern matches existing loop tests: loop edge originates from the "fix" node
// which only executes when check fails (via SourceHandle "false").
func TestSubWorkflow_Integration_ChildWithLoop(t *testing.T) {
	t.Parallel()

	// Child workflow:
	//   generate → check → finalize (SourceHandle "true")
	//                    → fix      (SourceHandle "false")
	//   fix → generate   (Loop, MaxIterations: 2)
	childWF := &models.Workflow{
		ID:   "child-with-loop",
		Name: "Child With Loop",
		Nodes: []*models.Node{
			{ID: "generate", Name: "Generate", Type: "llm", Config: map[string]interface{}{}},
			{ID: "check", Name: "Check", Type: "conditional", Config: map[string]interface{}{}},
			{ID: "fix", Name: "Fix", Type: "transform", Config: map[string]interface{}{}},
			{ID: "finalize", Name: "Finalize", Type: "transform", Config: map[string]interface{}{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "generate", To: "check"},
			{ID: "e2", From: "check", To: "finalize", SourceHandle: "true"},
			{ID: "e3", From: "check", To: "fix", SourceHandle: "false"},
			{ID: "loop1", From: "fix", To: "generate", Loop: &models.LoopConfig{MaxIterations: 2}},
		},
	}

	var genCount int64
	mockLLM := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			n := atomic.AddInt64(&genCount, 1)
			return map[string]interface{}{"text": "generated", "attempt": n}, nil
		},
	}

	// Check passes on every 2nd call (odd=false, even=true)
	var checkCount int64
	mockCheck := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			n := atomic.AddInt64(&checkCount, 1)
			return n%2 == 0, nil
		},
	}

	mockTransform := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return map[string]interface{}{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("llm", mockLLM)
	registry.Register("conditional", mockCheck)
	registry.Register("transform", mockTransform)

	loader := NewMockWorkflowLoader(map[string]*models.Workflow{
		"child-with-loop": childWF,
	})

	events := &recordingNotifier{}
	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), events, loader)

	parentWF := &models.Workflow{
		ID:   "parent",
		Name: "Parent",
		Nodes: []*models.Node{
			{
				ID:   "fanout",
				Name: "Fan Out",
				Type: "sub_workflow",
				Config: map[string]interface{}{
					"workflow_id":     "child-with-loop",
					"for_each":        "input.cells",
					"item_var":        "cell",
					"max_parallelism": 1, // Sequential for deterministic shared counter
				},
			},
		},
	}

	input := map[string]interface{}{
		"cells": []interface{}{
			map[string]interface{}{"topic": "AI"},
			map[string]interface{}{"topic": "Go"},
		},
	}

	execState := NewExecutionState("exec-1", "parent", parentWF, input, nil)
	err := dagExec.Execute(context.Background(), execState, DefaultExecutionOptions())
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	output, ok := execState.GetNodeOutput("fanout")
	if !ok {
		t.Fatal("expected fanout output")
	}

	outputMap, ok := output.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map output, got: %T", output)
	}

	summary, ok := outputMap["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("expected summary in output")
	}

	if summary["completed"] != 2 {
		t.Fatalf("expected 2 completed, got: %v", summary["completed"])
	}

	// Each child: generate(1), check(1=false), fix, [loop], generate(2), check(2=true), finalize
	// 2 children × 2 generate calls = 4 total
	finalGenCount := atomic.LoadInt64(&genCount)
	if finalGenCount != 4 {
		t.Fatalf("expected 4 generate calls (2 per child), got: %d", finalGenCount)
	}

	// 2 children × 2 check calls = 4 total
	finalCheckCount := atomic.LoadInt64(&checkCount)
	if finalCheckCount != 4 {
		t.Fatalf("expected 4 check calls, got: %d", finalCheckCount)
	}

	// Verify sub_workflow events were emitted
	events.mu.Lock()
	defer events.mu.Unlock()
	progressEvents := 0
	for _, e := range events.events {
		if e.Type == EventTypeSubWorkflowProgress {
			progressEvents++
		}
	}
	if progressEvents == 0 {
		t.Fatal("expected sub_workflow.progress events")
	}
}
