package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

func main() {
	fmt.Println("=== MBFlow Simple Workflow Example ===")

	// Create a simple workflow using the builder pattern
	workflow, err := mbflow.NewWorkflowBuilder("Simple Workflow", "1.0").
		WithDescription("A simple workflow demonstrating the new architecture").
		// Add Start node
		AddNode(string(mbflow.NodeTypeStart), "start", map[string]any{}).
		// Add Transform node to process data using structured config
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "process", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"doubled": "input * 2", // Double the input
				"message": `"Processed: " + string(doubled)`,
			},
		}).
		// Add End node
		AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
			"output_keys": []string{"doubled", "message"},
		}).
		// Connect nodes with edges
		AddEdge("start", "process", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("process", "end", string(mbflow.EdgeTypeDirect), nil).
		// Add a manual trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name":        "Manual Start",
			"description": "Manually triggered execution",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("✓ Workflow created: %s (version %s)\n", workflow.Name(), workflow.Version())
	fmt.Printf("  Nodes: %d\n", len(workflow.GetAllNodes()))
	fmt.Printf("  Edges: %d\n", len(workflow.GetAllEdges()))
	fmt.Printf("  Triggers: %d\n\n", len(workflow.GetAllTriggers()))

	// Create an executor using the builder pattern
	executor := mbflow.NewExecutorBuilder().
		EnableParallelExecution(10).
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("simple-workflow"))).
		EnableRetry(3).
		EnableMetrics().
		Build()

	fmt.Println("✓ Executor created with configuration:")
	fmt.Println("  - Parallel execution: enabled (max 10 nodes)")
	fmt.Println("  - Retry: enabled (max 3 attempts)")
	fmt.Println("  - Metrics: enabled")

	// Get the trigger
	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal("No triggers found in workflow")
	}
	trigger := triggers[0]

	// Execute the workflow with initial variables
	ctx := context.Background()
	initialVars := map[string]any{
		"input": 21.0, // Input value to be processed
	}

	fmt.Println("▶ Executing workflow with initial variables:")
	fmt.Printf("  input = %.0f\n\n", initialVars["input"].(float64))

	execution, err := executor.ExecuteWorkflow(ctx, workflow, trigger, initialVars)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Println("✓ Workflow execution completed successfully!")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n", execution.Phase())

	// Get the final variables
	vars := execution.Variables().All()
	fmt.Println("\n📊 Final Output:")
	if doubled, ok := vars["doubled"]; ok {
		fmt.Printf("  doubled = %.0f\n", doubled)
	}
	if message, ok := vars["message"]; ok {
		fmt.Printf("  message = %s\n", message)
	}

	// Get events from the event store
	events, err := executor.EventStore().GetEvents(ctx, execution.ID())
	if err != nil {
		log.Printf("Warning: Failed to get events: %v", err)
	} else {
		fmt.Printf("\n📝 Events recorded: %d\n", len(events))
		for i, event := range events {
			fmt.Printf("  %d. %s (seq: %d)\n", i+1, event.EventType(), event.SequenceNumber())
		}
	}
	fmt.Println("\n=== Example completed successfully! ===")
}
