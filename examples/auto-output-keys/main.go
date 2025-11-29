package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// AutoOutputKeysDemo demonstrates automatic output key generation based on node ID
// This prevents conflicts in parallel execution and makes node chaining explicit
func main() {
	fmt.Println("=== MBFlow Auto Output Keys Demo ===")

	// Create workflow with parallel nodes that don't specify output_key
	// The engine will automatically generate unique keys: {node_id}_output
	workflow, err := mbflow.NewWorkflowBuilder("Auto Output Keys Demo", "1.0").
		WithDescription("Demonstrates automatic output key generation").
		// Transform node 1 - no output_key specified
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "transform1", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"result": "\"Hello from transform1\"",
				"value":  "42",
			},
		}).
		// Transform node 2 - no output_key specified
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "transform2", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"result": "\"Hello from transform2\"",
				"value":  "100",
			},
		}).
		// Aggregator that reads from both transforms using their auto-generated keys
		AddNodeWithConfig(string(mbflow.NodeTypeDataAggregator), "aggregate", &mbflow.DataAggregatorConfig{
			Fields: map[string]string{
				"transform1_result": "result", // From auto-merged transform1 output
				"transform2_result": "result", // From auto-merged transform2 output
			},
		}).
		// Connect nodes
		AddEdge("transform1", "aggregate", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("transform2", "aggregate", string(mbflow.EdgeTypeDirect), nil).
		// Add trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Start Demo",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("✓ Workflow created: %s\n", workflow.Name())
	fmt.Printf("  Nodes: %d\n", len(workflow.GetAllNodes()))
	fmt.Println()

	// Create executor
	executor := mbflow.NewExecutorBuilder().
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("auto-keys"))).
		EnableMetrics().
		EnableParallelExecution(10).
		Build()

	fmt.Println("✓ Executor created")

	// Execute workflow
	ctx := context.Background()
	triggers := workflow.GetAllTriggers()

	fmt.Println("▶ Executing workflow...")
	fmt.Println("  Note: transform1 and transform2 run in parallel")
	fmt.Println("  Each gets a unique output key: {node_id}_output")

	execution, err := executor.ExecuteWorkflow(ctx, workflow, triggers[0], map[string]any{})
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Printf("\n✓ Workflow execution completed!\n")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n\n", execution.Phase())

	// Display results
	vars := execution.Variables().All()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 RESULTS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Show auto-generated keys
	fmt.Println("\n🔑 Auto-generated output keys:")
	for key, value := range vars {
		if len(key) > 0 {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nKey Benefits:")
	fmt.Println("1. No manual output_key configuration needed")
	fmt.Println("2. Guaranteed unique keys prevent conflicts")
	fmt.Println("3. Node ID-based keys make data flow explicit")
	fmt.Println("4. Parallel nodes never overwrite each other's data")
}
