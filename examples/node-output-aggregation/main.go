package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// NodeOutputAggregationDemo demonstrates collecting outputs from multiple parallel nodes
// This is critical for workflows where you need to combine results from parallel branches
func main() {
	fmt.Println("=== MBFlow Node Output Aggregation Demo ===")

	// Create workflow with parallel nodes
	workflow, err := mbflow.NewWorkflowBuilder("Node Output Aggregation Demo", "1.0").
		WithDescription("Demonstrates aggregating outputs from multiple parallel nodes").
		// Parallel transform nodes - each processes different data
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "process_user", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"user_id":   "123",
				"user_name": `"John Doe"`,
				"role":      `"admin"`,
			},
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "process_order", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"order_id":    "456",
				"order_total": "99.99",
				"status":      `"completed"`,
			},
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "process_shipping", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"tracking_number": `"TRACK123456"`,
				"carrier":         `"UPS"`,
				"eta":             `"2024-12-01"`,
			},
		}).
		// Aggregator collects all outputs using NodeOutputs mode
		// Strategy "separate" - each node output under its own key
		AddNodeWithConfig(string(mbflow.NodeTypeDataAggregator), "combine_separate", &mbflow.DataAggregatorConfig{
			NodeOutputs: map[string]string{
				"user_data":     "process_user",     // Will read from process_user_output
				"order_data":    "process_order",    // Will read from process_order_output
				"shipping_data": "process_shipping", // Will read from process_shipping_output
			},
			MergeStrategy: "separate", // Each under its alias
		}).
		// Second aggregator demonstrates "flatten" strategy
		// All fields merged into one level
		AddNodeWithConfig(string(mbflow.NodeTypeDataAggregator), "combine_flatten", &mbflow.DataAggregatorConfig{
			NodeOutputs: map[string]string{
				"user":     "process_user",
				"order":    "process_order",
				"shipping": "process_shipping",
			},
			MergeStrategy: "flatten", // All fields in one map
		}).
		// All parallel nodes feed into both aggregators
		AddEdge("process_user", "combine_separate", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("process_order", "combine_separate", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("process_shipping", "combine_separate", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("process_user", "combine_flatten", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("process_order", "combine_flatten", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("process_shipping", "combine_flatten", string(mbflow.EdgeTypeDirect), nil).
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
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("node-agg"))).
		EnableMetrics().
		EnableParallelExecution(10).
		Build()

	fmt.Println("✓ Executor created")

	// Execute workflow
	ctx := context.Background()
	triggers := workflow.GetAllTriggers()

	fmt.Println("▶ Executing workflow...")
	fmt.Println("  Note: process_user, process_order, process_shipping run in parallel")
	fmt.Println("  Then both aggregators collect their outputs")

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

	// Debug: Show all variables
	fmt.Println("\n🔍 All Variables:")
	for key, value := range vars {
		fmt.Printf("  %s: %+v\n", key, value)
	}

	// Show separate aggregation result
	if separateResult, ok := vars["combine_separate_output"]; ok {
		fmt.Println("\n🔹 Strategy: SEPARATE (each node under its alias)")
		fmt.Printf("  combine_separate_output: %+v\n", separateResult)
	} else {
		fmt.Println("\n⚠️ combine_separate_output not found in variables")
	}

	// Show flatten aggregation result
	if flattenResult, ok := vars["combine_flatten_output"]; ok {
		fmt.Println("\n🔹 Strategy: FLATTEN (all fields merged)")
		fmt.Printf("  combine_flatten_output: %+v\n", flattenResult)
	} else {
		fmt.Println("\n⚠️ combine_flatten_output not found in variables")
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nKey Benefits:")
	fmt.Println("1. Explicitly specify which node outputs to collect")
	fmt.Println("2. No need to remember auto-generated _output keys")
	fmt.Println("3. Choose merge strategy: separate or flatten")
	fmt.Println("4. Automatic waiting for all dependent nodes")
	fmt.Println("5. Type-safe configuration with DataAggregatorConfig")
}
