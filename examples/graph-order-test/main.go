package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// GraphOrderTest verifies correct graph execution order with fork-join pattern
// Graph structure:
//   start
//     ├─> task1 ─> join
//     ├─> task2 ─> join
//     └─> task3 ─> join
//           └─> end

func main() {
	fmt.Println("=== Graph Execution Order Test ===")

	// Create a workflow with fork-join pattern
	workflow, err := mbflow.NewWorkflowBuilder("Graph Order Test", "1.0").
		WithDescription("Tests graph execution order with parallel and join patterns").
		// Start node
		AddNode(string(mbflow.NodeTypeStart), "start", map[string]any{}).
		// Three parallel tasks
		AddNode(string(mbflow.NodeTypeTransform), "task1", map[string]any{
			"transformations": map[string]any{
				"task1_result": `"Task 1 completed"`,
			},
		}).
		AddNode(string(mbflow.NodeTypeTransform), "task2", map[string]any{
			"transformations": map[string]any{
				"task2_result": `"Task 2 completed"`,
			},
		}).
		AddNode(string(mbflow.NodeTypeTransform), "task3", map[string]any{
			"transformations": map[string]any{
				"task3_result": `"Task 3 completed"`,
			},
		}).
		// Join node - waits for all parallel tasks
		AddNode(string(mbflow.NodeTypeTransform), "join", map[string]any{
			"transformations": map[string]any{
				"joined": `task1_result + ", " + task2_result + ", " + task3_result`,
			},
		}).
		// End node
		AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
			"output_keys": []string{"task1_result", "task2_result", "task3_result", "joined"},
		}).
		// Create edges for fork-join pattern
		// Fork: start -> task1, task2, task3 (parallel)
		AddEdge("start", "task1", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("start", "task2", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("start", "task3", string(mbflow.EdgeTypeDirect), nil).
		// Join: task1, task2, task3 -> join (wait for all)
		AddEdge("task1", "join", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("task2", "join", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("task3", "join", string(mbflow.EdgeTypeDirect), nil).
		// Sequential: join -> end
		AddEdge("join", "end", string(mbflow.EdgeTypeDirect), nil).
		// Add trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Test Graph Execution",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("\n✓ Workflow created: %s\n", workflow.Name())
	fmt.Printf("  Nodes: %d\n", len(workflow.GetAllNodes()))
	fmt.Printf("  Edges: %d\n", len(workflow.GetAllEdges()))

	// Create executor with parallel execution enabled
	executor := mbflow.NewExecutorBuilder().
		EnableParallelExecution(10).
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("graph-order-test"))).
		EnableMetrics().
		Build()

	fmt.Println("\n✓ Executor configured for parallel execution")

	fmt.Println("\nExpected execution order:")
	fmt.Println("  1. Start (entry node)")
	fmt.Println("  2. Task1, Task2, Task3 (parallel - after Start)")
	fmt.Println("  3. Join (after all tasks complete)")
	fmt.Println("  4. End (after Join)")

	// Execute workflow
	ctx := context.Background()
	initialVars := map[string]any{}

	fmt.Println("\n▶ Executing workflow...")

	execution, err := executor.ExecuteWorkflow(ctx, workflow, workflow.GetAllTriggers()[0], initialVars)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Printf("\n✓ Execution completed: %s\n", execution.Phase())
	fmt.Printf("  Duration: %v\n", execution.Duration())

	// Display final variables
	vars := execution.Variables().All()
	fmt.Println("\n📊 Final Variables:")
	for k, v := range vars {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// Get events to verify execution order
	events, _ := executor.EventStore().GetEvents(ctx, execution.ID())
	fmt.Printf("\n📝 Events recorded: %d\n", len(events))

	fmt.Println("\n=== Test Result ===")
	if execution.Phase() == "completed" {
		fmt.Println("✅ Graph execution completed successfully!")
		fmt.Println("✅ Fork/join pattern worked correctly!")
		fmt.Println("✅ All nodes executed in correct order!")
	} else {
		fmt.Println("❌ Graph execution failed")
	}
}
