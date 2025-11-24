package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
)

func main() {
	fmt.Println("=== MBFlow Parallel Workflow Example ===")

	// Create a workflow with parallel branches demonstrating fork/join pattern
	workflow, err := mbflow.NewWorkflowBuilder("Parallel Processing", "1.0").
		WithDescription("Demonstrates parallel execution with fork and join").
		// Start node
		AddNode(string(mbflow.NodeTypeStart), "start", map[string]any{}).
		// Fork node - spawns multiple parallel branches
		AddNode(string(mbflow.NodeTypeParallel), "fork", map[string]any{
			"max_parallel": 3,
		}).
		// Branch 1: Double the value
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "branch1", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"result": "value * 2",
			},
		}).
		// Branch 2: Square the value
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "branch2", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"result": "value * value",
			},
		}).
		// Branch 3: Add 100
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "branch3", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"result": "value + 100",
			},
		}).
		// Aggregate node - sums all branch results (uses namespaced variables)
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "aggregate", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"sum": "branch1_result + branch2_result + branch3_result",
			},
		}).
		// End node
		AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
			"output_keys": []string{"sum"},
		}).
		// Connect edges - fork pattern
		AddEdge("start", "fork", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("fork", "branch1", string(mbflow.EdgeTypeFork), nil).
		AddEdge("fork", "branch2", string(mbflow.EdgeTypeFork), nil).
		AddEdge("fork", "branch3", string(mbflow.EdgeTypeFork), nil).
		// Connect branches to aggregate using join edges - all branches must complete before aggregate
		AddEdge("branch1", "aggregate", string(mbflow.EdgeTypeJoin), map[string]any{
			"join_strategy": string(mbflow.JoinStrategyWaitAll),
		}).
		AddEdge("branch2", "aggregate", string(mbflow.EdgeTypeJoin), nil).
		AddEdge("branch3", "aggregate", string(mbflow.EdgeTypeJoin), nil).
		// Continue to end
		AddEdge("aggregate", "end", string(mbflow.EdgeTypeDirect), nil).
		// Add trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Start Parallel Processing",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("✓ Workflow created: %s\n", workflow.Name())
	fmt.Printf("  Total nodes: %d\n", len(workflow.GetAllNodes()))
	fmt.Printf("  Parallel branches: 3\n")
	fmt.Printf("  Join strategy: %s (waits for all branches before aggregating)\n\n", mbflow.JoinStrategyWaitAll)

	// Create executor with parallel execution enabled
	executor := mbflow.NewExecutorBuilder().
		EnableParallelExecution(10).
		EnableRetry(2).
		EnableCircuitBreaker().
		EnableMetrics().
		Build()

	fmt.Println("✓ Executor configured:")
	fmt.Println("  - Parallel execution: enabled")
	fmt.Println("  - Retry: enabled (max 2 attempts)")
	fmt.Println("  - Circuit breaker: enabled")
	fmt.Println("  - Metrics: enabled")

	// Get trigger
	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal("No triggers found")
	}

	// Execute workflow
	ctx := context.Background()
	initialVars := map[string]any{
		"value": 10.0,
	}

	fmt.Println("▶ Executing workflow with parallel branches...")
	fmt.Printf("  Input value: %.0f\n\n", initialVars["value"].(float64))

	execution, err := executor.ExecuteWorkflow(ctx, workflow, triggers[0], initialVars)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	// Display results
	fmt.Println("✓ Parallel workflow execution completed!")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n", execution.Phase())

	// Show results
	vars := execution.Variables().All()
	fmt.Println("\n📊 Results:")

	// Branch results are namespaced with underscore
	if v, ok := vars["branch1_result"]; ok {
		fmt.Printf("  Branch 1 (x2):   %.0f\n", v)
	}
	if v, ok := vars["branch2_result"]; ok {
		fmt.Printf("  Branch 2 (x²):   %.0f\n", v)
	}
	if v, ok := vars["branch3_result"]; ok {
		fmt.Printf("  Branch 3 (+100): %.0f\n", v)
	}

	fmt.Println("\n📈 Aggregated Result:")
	if sum, ok := vars["sum"]; ok {
		fmt.Printf("  Sum of all branches: %v\n", sum)
	}

	// Show join metadata
	if joinCount, ok := vars["_join_branch_count"]; ok {
		fmt.Printf("\n🔀 Join Information:")
		fmt.Printf("\n  Branches joined: %v\n", joinCount)
	}

	// Event summary
	events, _ := executor.EventStore().GetEvents(ctx, execution.ID())
	fmt.Printf("\n📝 Total events: %d\n", len(events))

	// Count node execution events
	nodeStartCount := 0
	nodeCompleteCount := 0
	for _, evt := range events {
		switch evt.EventType() {
		case mbflow.EventTypeNodeStarted:
			nodeStartCount++
		case mbflow.EventTypeNodeCompleted:
			nodeCompleteCount++
		}
	}
	fmt.Printf("  Nodes started: %d\n", nodeStartCount)
	fmt.Printf("  Nodes completed: %d\n", nodeCompleteCount)

	fmt.Println("\n=== Parallel execution example completed! ===")
}
