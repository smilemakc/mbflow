package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// ExecutionDemo demonstrates the workflow execution engine with monitoring and error handling.
// This example shows:
// 1. Creating an executor with monitoring enabled
// 2. Executing a workflow with multiple node types
// 3. EventStore integration and metrics
func main() {
	fmt.Printf("=== Workflow Execution Engine Demo ===\n\n")

	// Create a workflow with multiple node types
	workflow, err := mbflow.NewWorkflowBuilder("Demo Workflow", "1.0").
		WithDescription("Demonstrates various node types and execution features").
		// Start node
		AddNode(string(mbflow.NodeTypeStart), "start", map[string]any{}).
		// Transform node - process input data
		AddNode(string(mbflow.NodeTypeTransform), "process_data", map[string]any{
			"transformations": map[string]any{
				"processed": `input_data + " [PROCESSED]"`,
				"timestamp": `execution_time`,
				"status":    `"success"`,
			},
		}).
		// Another transform - aggregate results
		AddNode(string(mbflow.NodeTypeTransform), "aggregate", map[string]any{
			"transformations": map[string]any{
				"final_result": `"Data: " + processed + ", Time: " + timestamp + ", Status: " + status`,
			},
		}).
		// End node
		AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
			"output_keys": []string{"processed", "timestamp", "status", "final_result"},
		}).
		// Connect nodes
		AddEdge("start", "process_data", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("process_data", "aggregate", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("aggregate", "end", string(mbflow.EdgeTypeDirect), nil).
		// Add manual trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Execute Demo",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("✓ Workflow created: %s\n", workflow.Name())
	fmt.Printf("  Nodes: %d\n", len(workflow.GetAllNodes()))
	fmt.Printf("  Edges: %d\n\n", len(workflow.GetAllEdges()))

	// Create executor with monitoring and metrics enabled
	executor := mbflow.NewExecutorBuilder().
		EnableParallelExecution(5).
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("execution-demo"))).
		EnableRetry(2).
		EnableCircuitBreaker().
		EnableMetrics().
		Build()

	fmt.Println("✓ Executor configured:")
	fmt.Println("  - Monitoring: enabled")
	fmt.Println("  - Metrics: enabled")
	fmt.Println("  - Retry: enabled (max 2 attempts)")
	fmt.Println("  - Circuit breaker: enabled")

	// Set initial variables
	initialVars := map[string]any{
		"input_data": `**Execution Engine Core**

- Workflow orchestration and state management
- Node-by-node execution with dependency handling
- Thread-safe execution state tracking
- Variable storage and substitution`,
		"execution_time": time.Now().Format(time.RFC3339),
	}

	fmt.Println("▶ Executing workflow...")
	fmt.Printf("  Input data: %s\n", initialVars["input_data"].(string))
	fmt.Printf("  Execution time: %s\n\n", initialVars["execution_time"].(string))

	// Execute workflow
	ctx := context.Background()
	execution, err := executor.ExecuteWorkflow(
		ctx,
		workflow,
		workflow.GetAllTriggers()[0],
		initialVars,
	)

	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Println("\n✓ Workflow execution completed!")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n", execution.Phase())
	fmt.Printf("  Duration: %v\n", execution.Duration())

	// Display variables
	fmt.Println("\n📊 Final Variables:")
	vars := execution.Variables().All()
	for key, value := range vars {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Get and display events from event store
	events, err := executor.EventStore().GetEvents(ctx, execution.ID())
	if err != nil {
		log.Printf("Warning: Failed to get events: %v", err)
	} else {
		fmt.Printf("\n📝 Event Store - Total events: %d\n", len(events))
		for i, evt := range events {
			fmt.Printf("  %d. [Seq:%d] %s\n", i+1, evt.SequenceNumber(), evt.EventType())
		}
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("✓ Workflow Builder pattern")
	fmt.Println("✓ Multiple node types (Start, Transform, End)")
	fmt.Println("✓ Event sourcing with EventStore")
	fmt.Println("✓ Execution monitoring")
	fmt.Println("✓ Variable substitution and transformations")
}
