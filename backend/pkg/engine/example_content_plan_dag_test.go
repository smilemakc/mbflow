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

// TestContentPlanDAG_FullWorkflow demonstrates the complete content plan generation workflow.
//
// Architecture:
//   - 43 nodes across 5 functional blocks + data sources + context preparation
//   - 50 edges including 4 loop edges
//   - Conditional branching via SourceHandle (true/false)
//   - 3 validation loops: grid (max=3), balance (max=3), drafts (max=2)
//   - 1 global patch loop: Block 4 → Block 3 (max=1)
//   - Parallel execution in data source and context waves
//
// Scenario:
//   - Grid check (N2) fails twice, passes on 3rd call → 2 loop iterations
//   - Balance check (N6) fails once, passes on 2nd call → 1 loop iteration
//   - Draft check (N11) passes immediately → no loop
//   - Critical check (CRIT) returns false → no patch loop
func TestContentPlanDAG_FullWorkflow(t *testing.T) {
	t.Parallel()

	// ── Call counter (thread-safe) ──
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

	// ── Execution log (thread-safe) ──
	var execLog []string
	var logMu sync.Mutex

	logNode := func(id string) {
		logMu.Lock()
		execLog = append(execLog, id)
		logMu.Unlock()
	}

	// ── Single executor for all node types ──
	// Dispatches behavior based on nodeID from config.
	// Conditional nodes return map{"result": bool} for SourceHandle routing.
	mainExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			nodeID, _ := config["nodeID"].(string)
			if nodeID == "" {
				return map[string]interface{}{"status": "ok"}, nil
			}
			count := countCall(nodeID)
			logNode(nodeID)

			switch nodeID {

			// ═══════════════════════════════════════════
			// Data sources (Wave 0)
			// ═══════════════════════════════════════════
			case "S0_WIZARD":
				return map[string]interface{}{
					"topic": "AI в маркетинге", "channels": []string{"telegram", "instagram"},
					"period": "2_weeks", "posts_per_week": 3,
				}, nil
			case "S1_SETTINGS":
				return map[string]interface{}{
					"brand": "TechCorp", "timezone": "Europe/Moscow",
				}, nil
			case "S2_STYLE":
				return map[string]interface{}{
					"tone": "professional", "emoji": false, "max_length": 2000,
				}, nil
			case "S3_TOPICS":
				return map[string]interface{}{
					"topics": []string{"AI trends", "marketing automation", "content strategy"},
				}, nil
			case "S4_ANALYTICS":
				return map[string]interface{}{
					"avg_engagement": 4.2, "top_posts": []string{"ai-intro"},
				}, nil
			case "S5_KB":
				return map[string]interface{}{"index_id": "kb-001", "doc_count": 150}, nil
			case "S6_RULES":
				return map[string]interface{}{"max_per_day": 2, "min_gap_hours": 4}, nil
			case "S7_CHANNELS":
				return map[string]interface{}{
					"telegram": map[string]interface{}{"max_len": 4096},
					"instagram": map[string]interface{}{"max_len": 2200},
				}, nil
			case "S8_POLICIES":
				return map[string]interface{}{
					"banned_words": []string{"guaranteed"}, "compliance": "standard",
				}, nil

			// ═══════════════════════════════════════════
			// Context preparation
			// ═══════════════════════════════════════════
			case "D1_SCENARIO":
				return map[string]interface{}{
					"scenario": "2-week AI marketing plan", "validated": true,
				}, nil
			case "D3_TOPICS":
				return map[string]interface{}{
					"ordered": []string{"AI trends", "automation", "strategy"},
				}, nil
			case "D4_STYLE":
				return map[string]interface{}{
					"rules": []string{"no-emoji", "professional-tone"},
				}, nil
			case "D5_SETTINGS":
				return map[string]interface{}{
					"brand_summary": "TechCorp - B2B AI solutions",
				}, nil
			case "D6_RAG":
				return map[string]interface{}{
					"queries": []string{"AI marketing trends", "automation practices"},
				}, nil
			case "D6_1_SEARCH":
				return map[string]interface{}{
					"fragments": []string{"AI market growth 40%...", "Automation cuts costs..."},
				}, nil
			case "D7_KB_SUM":
				return map[string]interface{}{
					"summary": "AI adoption growing, automation effective for B2B",
				}, nil
			case "D8_CTX":
				return map[string]interface{}{
					"context": "merged project context", "ready": true,
				}, nil

			// ═══════════════════════════════════════════
			// Block 1: Grid — conditional N2 with loop
			// ═══════════════════════════════════════════
			case "N1_GRID":
				return map[string]interface{}{
					"grid": []string{"Mon-AM-tg", "Wed-AM-tg", "Fri-AM-tg"},
				}, nil
			case "N2_CHECK_GRID":
				// Fails twice, passes on 3rd call
				if count <= 2 {
					return map[string]interface{}{
						"result": false,
						"errors": []string{fmt.Sprintf("grid_issue_%d", count)},
					}, nil
				}
				return map[string]interface{}{"result": true, "grid": "validated"}, nil
			case "N3_FIX_GRID":
				return map[string]interface{}{
					"grid": []string{"Mon-AM-tg", "Tue-PM-ig", "Thu-AM-tg"},
					"fix_iteration": count,
				}, nil
			case "N4_GRID_OK":
				return map[string]interface{}{"final_grid": "approved", "slots": 6}, nil
			case "F1_GRID_FAIL":
				return map[string]interface{}{"status": "manual_review_needed"}, nil

			// ═══════════════════════════════════════════
			// Block 2: Balance — conditional N6 with loop
			// ═══════════════════════════════════════════
			case "N5_ROLES":
				return map[string]interface{}{
					"cells": []string{"cell-1-expert", "cell-2-story", "cell-3-tips"},
				}, nil
			case "N6_CHECK_BAL":
				// Fails once, passes on 2nd call
				if count <= 1 {
					return map[string]interface{}{
						"result": false, "issues": []string{"topic_imbalance"},
					}, nil
				}
				return map[string]interface{}{"result": true, "balanced": true}, nil
			case "N7_FIX_BAL":
				return map[string]interface{}{
					"cells": []string{"cell-1-expert", "cell-2-how-to", "cell-3-story"},
				}, nil
			case "N8_GOALS":
				return map[string]interface{}{
					"cells_with_goals": []string{"cell-1:awareness", "cell-2:engagement", "cell-3:traffic"},
				}, nil
			case "F2_BAL_FAIL":
				return map[string]interface{}{"status": "manual_balance_needed"}, nil

			// ═══════════════════════════════════════════
			// Block 3: Cell generation — conditional N11
			// ═══════════════════════════════════════════
			case "CELL_KB":
				return map[string]interface{}{
					"cell_fragments": []string{"KB fragment about AI trends"},
				}, nil
			case "N9_PROMPT":
				return map[string]interface{}{
					"prompt": "Generate professional post about AI trends for Telegram...",
				}, nil
			case "N10_GEN":
				return map[string]interface{}{
					"drafts": []string{"Draft A: AI transforms...", "Draft B: In 2024..."},
				}, nil
			case "N11_CHECK":
				// Passes immediately — no loop
				return map[string]interface{}{"result": true, "quality": "good"}, nil
			case "N12_REGEN":
				return map[string]interface{}{
					"drafts": []string{"Draft C: Improved version..."},
				}, nil
			case "N13_SELECT":
				return map[string]interface{}{
					"publication": map[string]interface{}{
						"text": "AI is transforming marketing...", "channel": "telegram",
					},
				}, nil
			case "F3_CELL_FAIL":
				return map[string]interface{}{"status": "cell_manual_edit"}, nil
			case "R1_PUBS":
				return map[string]interface{}{
					"publications": []string{"pub-1", "pub-2", "pub-3"}, "total": 3,
				}, nil

			// ═══════════════════════════════════════════
			// Block 4: Global check — conditional CRIT
			// ═══════════════════════════════════════════
			case "N15_INTEGRITY":
				return map[string]interface{}{"issues": []string{}, "score": 95}, nil
			case "N16_INSTRUCT":
				return map[string]interface{}{"instructions": []string{}, "critical": false}, nil
			case "CRIT_CHECK":
				// Not critical — no patch loop
				return map[string]interface{}{"result": false}, nil
			case "PATCH_APPLY":
				return map[string]interface{}{"patched": true}, nil
			case "N17_STATUS":
				return map[string]interface{}{
					"statuses": map[string]interface{}{"pub-1": "ready", "pub-2": "ready"},
				}, nil
			case "R2_PLAN":
				return map[string]interface{}{"plan_id": "plan-001", "status": "complete"}, nil

			// ═══════════════════════════════════════════
			// Block 5: Save
			// ═══════════════════════════════════════════
			case "SAVE_PACK":
				return map[string]interface{}{"package": "ready", "entities": 3}, nil
			case "N20_SAVE":
				return map[string]interface{}{"saved": true, "plan_id": "plan-001"}, nil

			default:
				return map[string]interface{}{"status": "ok"}, nil
			}
		},
	}

	// ── Register executor for all node types ──
	registry := executor.NewManager()
	for _, typ := range []string{"llm", "store", "infra", "code", "conditional"} {
		registry.Register(typ, mainExec)
	}

	// ── Recording notifier for event verification ──
	notifier := &recordingNotifier{}

	nodeExec := NewNodeExecutor(registry)
	dagExec := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), notifier)

	// ── Build workflow ──
	// Node helper
	n := func(id, name, typ string) *models.Node {
		return &models.Node{
			ID: id, Name: name, Type: typ,
			Config: map[string]interface{}{"nodeID": id},
		}
	}
	// Regular edge
	e := func(id, from, to string) *models.Edge {
		return &models.Edge{ID: id, From: from, To: to}
	}
	// Conditional edge (SourceHandle)
	ce := func(id, from, to, handle string) *models.Edge {
		return &models.Edge{ID: id, From: from, To: to, SourceHandle: handle}
	}
	// Loop edge
	le := func(id, from, to string, maxIter int) *models.Edge {
		return &models.Edge{ID: id, From: from, To: to, Loop: &models.LoopConfig{MaxIterations: maxIter}}
	}

	workflow := &models.Workflow{
		ID:   "content-plan-wf",
		Name: "Content Plan Generation",
		Nodes: []*models.Node{
			// ── Data sources (9 nodes, wave 0) ──
			n("S0_WIZARD", "Ввод пользователя", "store"),
			n("S1_SETTINGS", "Настройки проекта", "store"),
			n("S2_STYLE", "Правила стиля", "store"),
			n("S3_TOPICS", "Тематики проекта", "store"),
			n("S4_ANALYTICS", "Историческая аналитика", "store"),
			n("S5_KB", "База знаний", "store"),
			n("S6_RULES", "Правила нагрузки", "store"),
			n("S7_CHANNELS", "Правила каналов", "store"),
			n("S8_POLICIES", "Политики проекта", "store"),

			// ── Context preparation (8 nodes) ──
			n("D1_SCENARIO", "Нормализовать сценарий", "llm"),
			n("D3_TOPICS", "Упорядочить тематики", "llm"),
			n("D4_STYLE", "Разобрать правила стиля", "code"),
			n("D5_SETTINGS", "Свести настройки", "code"),
			n("D6_RAG", "Сформировать запросы к БЗ", "llm"),
			n("D6_1_SEARCH", "Найти материалы в БЗ", "infra"),
			n("D7_KB_SUM", "Сжать найденное в конспект", "llm"),
			n("D8_CTX", "Собрать единый контекст", "code"),

			// ── Block 1: Grid (5 nodes) ──
			n("N1_GRID", "Спроектировать сетку", "llm"),
			n("N2_CHECK_GRID", "Проверить сетку", "conditional"),
			n("N3_FIX_GRID", "Исправить сетку", "llm"),
			n("N4_GRID_OK", "Финальная сетка", "store"),
			n("F1_GRID_FAIL", "Нужна ручная правка сетки", "store"),

			// ── Block 2: Balance (5 nodes) ──
			n("N5_ROLES", "Назначить роли и темы", "llm"),
			n("N6_CHECK_BAL", "Проверить баланс", "conditional"),
			n("N7_FIX_BAL", "Исправить баланс", "llm"),
			n("N8_GOALS", "Назначить цели", "llm"),
			n("F2_BAL_FAIL", "Нужна ручная правка баланса", "store"),

			// ── Block 3: Cell generation (8 nodes) ──
			n("CELL_KB", "Подобрать фрагменты под ячейку", "infra"),
			n("N9_PROMPT", "Собрать промпт", "code"),
			n("N10_GEN", "Сгенерировать черновики", "llm"),
			n("N11_CHECK", "Проверить черновики", "conditional"),
			n("N12_REGEN", "Перегенерировать", "llm"),
			n("N13_SELECT", "Выбрать лучший и упаковать", "llm"),
			n("F3_CELL_FAIL", "Ячейка требует правки", "store"),
			n("R1_PUBS", "Публикации по ячейкам", "store"),

			// ── Block 4: Global check (6 nodes) ──
			n("N15_INTEGRITY", "Проверить целостность плана", "llm"),
			n("N16_INSTRUCT", "Сформировать инструкции правок", "llm"),
			n("CRIT_CHECK", "Проблемы критичные?", "conditional"),
			n("PATCH_APPLY", "Применить инструкции правок", "code"),
			n("N17_STATUS", "Проставить статусы", "llm"),
			n("R2_PLAN", "План + статусы", "store"),

			// ── Block 5: Save (2 nodes) ──
			n("SAVE_PACK", "Подготовить пакет сохранения", "code"),
			n("N20_SAVE", "Сохранить", "store"),
		},
		Edges: []*models.Edge{
			// ── Sources → Context preparation ──
			e("e01", "S0_WIZARD", "D1_SCENARIO"),
			e("e02", "S2_STYLE", "D4_STYLE"),
			e("e03", "S1_SETTINGS", "D5_SETTINGS"),
			e("e04", "S3_TOPICS", "D3_TOPICS"),
			e("e05", "D1_SCENARIO", "D3_TOPICS"),

			// ── KB search chain ──
			e("e06", "D3_TOPICS", "D6_RAG"),
			e("e07", "D4_STYLE", "D6_RAG"),
			e("e08", "D5_SETTINGS", "D6_RAG"),
			e("e09", "D6_RAG", "D6_1_SEARCH"),
			e("e10", "S5_KB", "D6_1_SEARCH"),
			e("e11", "D6_1_SEARCH", "D7_KB_SUM"),

			// ── Context merge (all parents always complete → OR semantics safe) ──
			e("e12", "D1_SCENARIO", "D8_CTX"),
			e("e13", "D3_TOPICS", "D8_CTX"),
			e("e14", "D4_STYLE", "D8_CTX"),
			e("e15", "D5_SETTINGS", "D8_CTX"),
			e("e16", "D7_KB_SUM", "D8_CTX"),
			e("e17", "S4_ANALYTICS", "D8_CTX"),

			// ── Block 1: Grid ──
			e("e18", "D8_CTX", "N1_GRID"),
			e("e19", "D1_SCENARIO", "N1_GRID"),
			e("e20", "N1_GRID", "N2_CHECK_GRID"),
			e("e21", "S6_RULES", "N2_CHECK_GRID"),
			ce("e22", "N2_CHECK_GRID", "N4_GRID_OK", "true"),
			ce("e23", "N2_CHECK_GRID", "N3_FIX_GRID", "false"),
			le("e24", "N3_FIX_GRID", "N2_CHECK_GRID", 3), // Loop: fix → recheck (max 3)
			e("e25", "N3_FIX_GRID", "F1_GRID_FAIL"),       // Executes only when loop exhausted

			// ── Block 2: Balance ──
			e("e26", "N4_GRID_OK", "N5_ROLES"),
			e("e27", "N5_ROLES", "N6_CHECK_BAL"),
			ce("e28", "N6_CHECK_BAL", "N8_GOALS", "true"),
			ce("e29", "N6_CHECK_BAL", "N7_FIX_BAL", "false"),
			le("e30", "N7_FIX_BAL", "N6_CHECK_BAL", 3), // Loop: fix → recheck (max 3)
			e("e31", "N7_FIX_BAL", "F2_BAL_FAIL"),

			// ── Block 3: Cell generation ──
			e("e32", "N8_GOALS", "CELL_KB"),
			e("e33", "CELL_KB", "N9_PROMPT"),
			e("e34", "N9_PROMPT", "N10_GEN"),
			e("e35", "N10_GEN", "N11_CHECK"),
			ce("e36", "N11_CHECK", "N13_SELECT", "true"),
			ce("e37", "N11_CHECK", "N12_REGEN", "false"),
			le("e38", "N12_REGEN", "N10_GEN", 2), // Loop: regen → recheck (max 2)
			e("e39", "N12_REGEN", "F3_CELL_FAIL"),
			e("e40", "N13_SELECT", "R1_PUBS"),
			e("e41", "F3_CELL_FAIL", "R1_PUBS"),

			// ── Block 4: Global check ──
			e("e42", "R1_PUBS", "N15_INTEGRITY"),
			e("e43", "N15_INTEGRITY", "N16_INSTRUCT"),
			e("e44", "N16_INSTRUCT", "CRIT_CHECK"),
			ce("e45", "CRIT_CHECK", "PATCH_APPLY", "true"),
			ce("e46", "CRIT_CHECK", "N17_STATUS", "false"),
			le("e47", "PATCH_APPLY", "CELL_KB", 1), // Loop: patch → redo cells (max 1)
			e("e48", "N17_STATUS", "R2_PLAN"),

			// ── Block 5: Save ──
			e("e49", "R2_PLAN", "SAVE_PACK"),
			e("e50", "SAVE_PACK", "N20_SAVE"),
		},
	}

	// ── Execute ──
	execState := NewExecutionState("exec-plan-1", workflow.ID, workflow, map[string]interface{}{}, map[string]interface{}{})
	opts := DefaultExecutionOptions()

	err := dagExec.Execute(context.Background(), execState, opts)
	if err != nil {
		t.Fatalf("workflow execution failed: %v", err)
	}

	// ═══════════════════════════════════════════
	// Verification
	// ═══════════════════════════════════════════

	// 1. Verify loop iteration counts
	assertCallCount := func(nodeID string, expected int32) {
		t.Helper()
		actual := getCount(nodeID)
		if actual != expected {
			t.Errorf("node %s: expected %d calls, got %d", nodeID, expected, actual)
		}
	}

	assertCallCount("N2_CHECK_GRID", 3) // 1 original + 2 loop iterations
	assertCallCount("N3_FIX_GRID", 2)   // runs on 1st and 2nd false from N2
	assertCallCount("N6_CHECK_BAL", 2)  // 1 original + 1 loop iteration
	assertCallCount("N7_FIX_BAL", 1)    // runs on 1st false from N6
	assertCallCount("N11_CHECK", 1)     // passes immediately
	assertCallCount("CRIT_CHECK", 1)    // not critical
	assertCallCount("N20_SAVE", 1)      // final save

	// 2. Verify completed nodes
	completedNodes := []string{
		// Sources
		"S0_WIZARD", "S1_SETTINGS", "S2_STYLE", "S3_TOPICS",
		"S4_ANALYTICS", "S5_KB", "S6_RULES", "S7_CHANNELS", "S8_POLICIES",
		// Context
		"D1_SCENARIO", "D3_TOPICS", "D4_STYLE", "D5_SETTINGS",
		"D6_RAG", "D6_1_SEARCH", "D7_KB_SUM", "D8_CTX",
		// Block 1 (grid passes on 3rd attempt)
		"N1_GRID", "N2_CHECK_GRID", "N4_GRID_OK",
		// Block 2 (balance passes on 2nd attempt)
		"N5_ROLES", "N6_CHECK_BAL", "N8_GOALS",
		// Block 3 (drafts pass immediately)
		"CELL_KB", "N9_PROMPT", "N10_GEN", "N11_CHECK", "N13_SELECT", "R1_PUBS",
		// Block 4 (not critical)
		"N15_INTEGRITY", "N16_INSTRUCT", "CRIT_CHECK", "N17_STATUS", "R2_PLAN",
		// Block 5
		"SAVE_PACK", "N20_SAVE",
	}
	for _, nodeID := range completedNodes {
		status, ok := execState.GetNodeStatus(nodeID)
		if !ok {
			t.Errorf("node %s: no status recorded", nodeID)
		} else if status != models.NodeExecutionStatusCompleted {
			t.Errorf("node %s: expected completed, got %v", nodeID, status)
		}
	}

	// 3. Verify skipped nodes
	skippedNodes := []string{
		"F1_GRID_FAIL",  // Grid loop succeeded → fail node skipped
		"F2_BAL_FAIL",   // Balance loop succeeded → fail node skipped
		"N12_REGEN",     // Draft check passed → regeneration skipped
		"F3_CELL_FAIL",  // No regeneration → cell fail skipped
		"PATCH_APPLY",   // Not critical → patch skipped
		"N3_FIX_GRID",   // Last N2 call returned true → fix skipped (final status after reset)
		"N7_FIX_BAL",    // Last N6 call returned true → fix skipped (final status after reset)
	}
	for _, nodeID := range skippedNodes {
		status, _ := execState.GetNodeStatus(nodeID)
		if status != models.NodeExecutionStatusSkipped {
			t.Errorf("node %s: expected skipped, got %v", nodeID, status)
		}
	}

	// 4. Verify loop events
	notifier.mu.Lock()
	events := make([]ExecutionEvent, len(notifier.events))
	copy(events, notifier.events)
	notifier.mu.Unlock()

	loopIterEvents := 0
	for _, ev := range events {
		if ev.Type == EventTypeLoopIteration {
			loopIterEvents++
		}
	}
	// 2 grid loop iterations + 1 balance loop iteration = 3 total
	if loopIterEvents != 3 {
		t.Errorf("expected 3 loop iteration events, got %d", loopIterEvents)
	}

	// 5. Verify final output
	output, ok := execState.GetNodeOutput("N20_SAVE")
	if !ok {
		t.Fatal("N20_SAVE has no output")
	}
	outputMap, ok := output.(map[string]interface{})
	if !ok {
		t.Fatal("N20_SAVE output is not a map")
	}
	if saved, _ := outputMap["saved"].(bool); !saved {
		t.Error("expected N20_SAVE output saved=true")
	}

	// 6. Verify DAG structure: loop edges excluded from topo sort
	dag := BuildDAG(workflow)
	if len(dag.LoopEdges) != 4 {
		t.Errorf("expected 4 loop edges in DAG, got %d", len(dag.LoopEdges))
	}
	waves, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("topological sort should not fail with loop edges: %v", err)
	}
	if len(waves) < 10 {
		t.Errorf("expected at least 10 waves for 43-node workflow, got %d", len(waves))
	}

	// 7. Log summary
	t.Logf("Workflow executed successfully:")
	t.Logf("  Nodes: %d", len(workflow.Nodes))
	t.Logf("  Edges: %d (including %d loop edges)", len(workflow.Edges), len(dag.LoopEdges))
	t.Logf("  Waves: %d", len(waves))
	t.Logf("  Total node executions: %d", len(execLog))
	t.Logf("  Loop iterations: grid=%d, balance=%d, drafts=%d",
		getCount("N2_CHECK_GRID")-1, getCount("N6_CHECK_BAL")-1, int32(0))
}
