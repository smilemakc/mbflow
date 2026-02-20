package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// TestContentPlanSubWorkflow_FanOutWithLoopInChild demonstrates the content plan pipeline
// using sub_workflow fan-out where each child workflow contains a conditional loop.
//
// Architecture:
//   - Main workflow: CTX_NODE → GRID_NODE → SUB_WF_NODE → COLLECT_NODE
//   - Child "cell-gen-wf": C1_PROMPT → C2_GEN → C3_CHECK → C4_SELECT (true branch)
//                                                          → C5_REGEN (false branch)
//     Loop edge: C5_REGEN → C2_GEN (MaxIterations=2)
//     Forward edge: C5_REGEN → C6_FAIL (executes only when loop exhausted)
//
// Scenario:
//   - 3 slots fan out to 3 parallel child executions (max_parallelism=2)
//   - Each child: C3_CHECK fails on 1st call (result=false), passes on 2nd call (result=true)
//   - Loop fires once per child (C5_REGEN → C2_GEN), then C3_CHECK passes
//   - C6_FAIL is skipped in all children (loop succeeds before exhaustion)
//   - Total: C2_GEN=6, C3_CHECK=6, C5_REGEN=3, C4_SELECT=3, C6_FAIL=0
func TestContentPlanSubWorkflow_FanOutWithLoopInChild(t *testing.T) {
	t.Parallel()

	// ── Thread-safe call counter ──
	callCounts := &sync.Map{}

	countCall := func(nodeID string) int32 {
		val, _ := callCounts.LoadOrStore(nodeID, new(int32))
		return atomic.AddInt32(val.(*int32), 1)
	}

	getCount := func(nodeID string) int32 {
		val, ok := callCounts.Load(nodeID)
		if !ok {
			return 0
		}
		return atomic.LoadInt32(val.(*int32))
	}

	// ── Single executor dispatching by nodeID ──
	// All node types (code, llm, conditional, store) share one executor.
	// Conditional nodes return map{"result": bool} for SourceHandle routing.
	mainExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			nodeID, _ := config["nodeID"].(string)
			if nodeID == "" {
				return map[string]any{"status": "ok"}, nil
			}

			count := countCall(nodeID)
			inputMap, _ := input.(map[string]any)

			switch nodeID {

			// ═══════════════════════════════════════════
			// Main workflow nodes
			// ═══════════════════════════════════════════

			case "CTX_NODE":
				return map[string]any{
					"context": "generation context data",
				}, nil

			case "GRID_NODE":
				return map[string]any{
					"slots": []any{
						map[string]any{"topic": "AI", "channel": "tg"},
						map[string]any{"topic": "ML", "channel": "ig"},
						map[string]any{"topic": "Data", "channel": "tg"},
					},
				}, nil

			case "COLLECT_NODE":
				// Pass-through: return input as output
				return inputMap, nil

			// ═══════════════════════════════════════════
			// Child workflow nodes
			// ═══════════════════════════════════════════

			case "C1_PROMPT":
				cell, _ := inputMap["cell"].(map[string]any)
				topic, _ := cell["topic"].(string)
				return map[string]any{
					"prompt": fmt.Sprintf("Generate post about %s", topic),
				}, nil

			case "C2_GEN":
				return map[string]any{
					"drafts":    []any{"Draft A", "Draft B"},
					"gen_count": count,
				}, nil

			case "C3_CHECK":
				// Each child independently fails once, succeeds on 2nd call.
				// count is the TOTAL across all children; per-child determination
				// relies on the loop: loop fires when count%2==1 (1st call odd → false).
				// Since children run in parallel, we use per-child counters stored in
				// a sync.Map keyed by goroutine-unique child execution ID passed via context.
				// However, the test spec says: count <=1 → false, else → true.
				// With 3 parallel children each calling once initially → counts 1,2,3.
				// After loop: each calls again → counts 4,5,6.
				// So we cannot use a simple global counter reliably for per-child logic.
				// Instead, use per-child item_var to track state via a child-scoped map.
				//
				// Practical approach: use sync.Map with child-execution-scoped key.
				// The child input has "index" field set by sub_workflow executor.
				index, _ := inputMap["index"].(int)
				key := fmt.Sprintf("C3_CHECK_child_%d", index)
				childVal, _ := callCounts.LoadOrStore(key, new(int32))
				childCount := atomic.AddInt32(childVal.(*int32), 1)

				// Count the global call to maintain total tracking (already done above)
				if childCount <= 1 {
					return map[string]any{
						"result": false,
						"issues": []any{fmt.Sprintf("quality_check_failed_attempt_%d", childCount)},
					}, nil
				}
				return map[string]any{
					"result": true,
				}, nil

			case "C4_SELECT":
				cell, _ := inputMap["cell"].(map[string]any)
				channel, _ := cell["channel"].(string)
				return map[string]any{
					"publication": map[string]any{
						"text":    "Final text",
						"channel": channel,
					},
				}, nil

			case "C5_REGEN":
				return map[string]any{
					"drafts":      []any{"Improved draft"},
					"regen_count": count,
				}, nil

			case "C6_FAIL":
				return map[string]any{
					"status": "manual_edit_needed",
				}, nil

			default:
				return map[string]any{"status": "ok"}, nil
			}
		},
	}

	// ── Register executor for all node types used in both workflows ──
	registry := executor.NewManager()
	for _, typ := range []string{"code", "llm", "conditional", "store"} {
		registry.Register(typ, mainExec)
	}

	// ── Build child workflow "cell-gen-wf" ──
	//
	//   C1_PROMPT → C2_GEN → C3_CHECK --[true]→  C4_SELECT  (terminal: publication)
	//                               └──[false]→  C5_REGEN
	//               ↑ loop (max=2)────────────────┘
	//                                  C5_REGEN → C6_FAIL    (forward: when loop exhausted)
	//
	childWorkflow := &models.Workflow{
		ID:   "cell-gen-wf",
		Name: "Cell Generation Workflow",
		Nodes: []*models.Node{
			{ID: "C1_PROMPT", Name: "Build Prompt", Type: "code", Config: map[string]any{"nodeID": "C1_PROMPT"}},
			{ID: "C2_GEN", Name: "Generate Drafts", Type: "llm", Config: map[string]any{"nodeID": "C2_GEN"}},
			{ID: "C3_CHECK", Name: "Quality Check", Type: "conditional", Config: map[string]any{"nodeID": "C3_CHECK"}},
			{ID: "C4_SELECT", Name: "Select Best", Type: "code", Config: map[string]any{"nodeID": "C4_SELECT"}},
			{ID: "C5_REGEN", Name: "Regenerate", Type: "llm", Config: map[string]any{"nodeID": "C5_REGEN"}},
			{ID: "C6_FAIL", Name: "Manual Edit Required", Type: "store", Config: map[string]any{"nodeID": "C6_FAIL"}},
		},
		Edges: []*models.Edge{
			{ID: "ce1", From: "C1_PROMPT", To: "C2_GEN"},
			{ID: "ce2", From: "C2_GEN", To: "C3_CHECK"},
			{ID: "ce3", From: "C3_CHECK", To: "C4_SELECT", SourceHandle: "true"},
			{ID: "ce4", From: "C3_CHECK", To: "C5_REGEN", SourceHandle: "false"},
			{ID: "ce5", From: "C5_REGEN", To: "C2_GEN", Loop: &models.LoopConfig{MaxIterations: 2}},
			{ID: "ce6", From: "C5_REGEN", To: "C6_FAIL"}, // forward edge: fires only when loop exhausted
		},
	}

	// ── Build main workflow ──
	//
	//   CTX_NODE → GRID_NODE → SUB_WF_NODE → COLLECT_NODE
	//
	mainWorkflow := &models.Workflow{
		ID:   "content-plan-main-wf",
		Name: "Content Plan Main Workflow",
		Nodes: []*models.Node{
			{
				ID:     "CTX_NODE",
				Name:   "Context Node",
				Type:   "code",
				Config: map[string]any{"nodeID": "CTX_NODE"},
			},
			{
				ID:     "GRID_NODE",
				Name:   "Grid Node",
				Type:   "code",
				Config: map[string]any{"nodeID": "GRID_NODE"},
			},
			{
				ID:   "SUB_WF_NODE",
				Name: "Sub Workflow Fan-Out",
				Type: "sub_workflow",
				Config: map[string]any{
					"workflow_id":     "cell-gen-wf",
					"for_each":        "slots",
					"item_var":        "cell",
					"max_parallelism": 2,
					"on_error":        "collect_partial",
				},
			},
			{
				ID:     "COLLECT_NODE",
				Name:   "Collect Results",
				Type:   "code",
				Config: map[string]any{"nodeID": "COLLECT_NODE"},
			},
		},
		Edges: []*models.Edge{
			{ID: "me1", From: "CTX_NODE", To: "GRID_NODE"},
			{ID: "me2", From: "GRID_NODE", To: "SUB_WF_NODE"},
			{ID: "me3", From: "SUB_WF_NODE", To: "COLLECT_NODE"},
		},
	}

	// ── Setup executor pipeline ──
	loader := NewMockWorkflowLoader(map[string]*models.Workflow{
		"cell-gen-wf": childWorkflow,
	})

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), NewNoOpNotifier(), loader)

	// ── Execute ──
	execState := NewExecutionState("exec-content-subwf-1", mainWorkflow.ID, mainWorkflow, map[string]any{}, map[string]any{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("workflow execution failed: %v", err)
	}

	// ═══════════════════════════════════════════
	// Assertions
	// ═══════════════════════════════════════════

	assertCallCount := func(nodeID string, expected int32) {
		t.Helper()
		actual := getCount(nodeID)
		if actual != expected {
			t.Errorf("node %s: expected %d calls, got %d", nodeID, expected, actual)
		}
	}

	// 1. Main workflow node call counts
	assertCallCount("CTX_NODE", 1)
	assertCallCount("GRID_NODE", 1)
	assertCallCount("COLLECT_NODE", 1)

	// 2. Child workflow node call counts (total across all 3 children)
	// Each child: C2_GEN called twice (initial + after loop), C3_CHECK called twice
	// (fails on 1st, passes on 2nd), C5_REGEN called once (loop fires once per child)
	// C4_SELECT called once per child (success path), C6_FAIL never called (loop succeeds)
	assertCallCount("C1_PROMPT", 3) // 1 per child × 3 children
	assertCallCount("C2_GEN", 6)    // 2 per child × 3 children (initial + 1 loop)
	assertCallCount("C3_CHECK", 6)  // 2 per child × 3 children (fails once, passes once)
	assertCallCount("C5_REGEN", 3)  // 1 per child × 3 children (loop fires once)
	assertCallCount("C4_SELECT", 3) // 1 per child × 3 children (success path)
	assertCallCount("C6_FAIL", 0)   // never called — loop succeeds before exhaustion

	// 3. SUB_WF_NODE output structure
	subWFOutput, ok := execState.GetNodeOutput("SUB_WF_NODE")
	if !ok {
		t.Fatal("SUB_WF_NODE has no output")
	}

	subWFOutputMap, ok := subWFOutput.(map[string]any)
	if !ok {
		t.Fatalf("SUB_WF_NODE output is not a map, got: %T", subWFOutput)
	}

	summary, ok := subWFOutputMap["summary"].(map[string]any)
	if !ok {
		t.Fatal("SUB_WF_NODE output missing 'summary' field")
	}

	if summary["total"] != 3 {
		t.Errorf("expected summary.total=3, got: %v", summary["total"])
	}
	if summary["completed"] != 3 {
		t.Errorf("expected summary.completed=3, got: %v", summary["completed"])
	}
	if summary["failed"] != 0 {
		t.Errorf("expected summary.failed=0, got: %v", summary["failed"])
	}

	items, ok := subWFOutputMap["items"].([]any)
	if !ok {
		t.Fatal("SUB_WF_NODE output missing 'items' field")
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items in sub_workflow output, got: %d", len(items))
	}

	for i, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			t.Errorf("item[%d] is not a map: %T", i, rawItem)
			continue
		}
		if item["status"] != "completed" {
			t.Errorf("item[%d]: expected status='completed', got: %v", i, item["status"])
		}
	}

	// 4. COLLECT_NODE was executed and has output
	collectOutput, ok := execState.GetNodeOutput("COLLECT_NODE")
	if !ok {
		t.Fatal("COLLECT_NODE has no output")
	}
	if collectOutput == nil {
		t.Fatal("COLLECT_NODE output is nil")
	}

	// 5. All main workflow nodes completed successfully
	for _, nodeID := range []string{"CTX_NODE", "GRID_NODE", "SUB_WF_NODE", "COLLECT_NODE"} {
		status, ok := execState.GetNodeStatus(nodeID)
		if !ok {
			t.Errorf("node %s: no status recorded", nodeID)
			continue
		}
		if status != models.NodeExecutionStatusCompleted {
			t.Errorf("node %s: expected completed, got %v", nodeID, status)
		}
	}

	// 6. Log summary
	t.Logf("Content plan sub-workflow fan-out executed successfully:")
	t.Logf("  Children: 3 (AI/tg, ML/ig, Data/tg)")
	t.Logf("  C2_GEN total calls: %d (2 per child × 3)", getCount("C2_GEN"))
	t.Logf("  C3_CHECK total calls: %d (fails once, passes once per child)", getCount("C3_CHECK"))
	t.Logf("  C5_REGEN total calls: %d (1 loop iteration per child)", getCount("C5_REGEN"))
	t.Logf("  C4_SELECT total calls: %d (success path per child)", getCount("C4_SELECT"))
	t.Logf("  C6_FAIL total calls: %d (loop exhausted path — skipped)", getCount("C6_FAIL"))
}
