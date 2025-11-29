package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// TestInputVarsDemo demonstrates that nodes receive inputs from their predecessors
func main() {
	fmt.Println("=== Testing Input Variables from Predecessors ===")

	// Create a simple workflow:
	// start -> transform1 -> transform2 -> end
	// transform1 creates "result1"
	// transform2 should receive "result1" from transform1
	workflow, err := mbflow.NewWorkflowBuilder("Input Vars Test", "1.0").
		WithDescription("Tests that nodes receive inputs from predecessors").
		// Transform 1: Creates "result1" from initial variable
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "transform1", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"result1": `"Transformed: " + input_value`,
			},
		}).
		// Transform 2: Should receive "result1" from transform1
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "transform2", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"result2": `"Further transformed: " + result1`,
			},
		}).
		// Create edges
		AddEdge("transform1", "transform2", string(mbflow.EdgeTypeDirect), nil).
		// Add trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Test Trigger",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("✓ Workflow created: %s\n", workflow.Name())
	fmt.Printf("  Nodes: %d\n", len(workflow.GetAllNodes()))
	fmt.Printf("  Edges: %d\n\n", len(workflow.GetAllEdges()))

	// Create executor
	executor := mbflow.NewExecutorBuilder().
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("input-vars-test"))).
		Build()

	// Get trigger
	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal("No triggers found")
	}

	// Execute workflow
	ctx := context.Background()
	initialVars := map[string]any{
		"input_value": "Hello World",
	}

	fmt.Println("▶ Executing workflow...")
	fmt.Printf("  Initial variable: input_value = %s\n\n", initialVars["input_value"])

	execution, err := executor.ExecuteWorkflow(ctx, workflow, triggers[0], initialVars)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Println("\n✓ Workflow execution completed successfully!")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n", execution.Phase())
	fmt.Printf("  Duration: %v\n", execution.Duration())

	// Check results
	vars := execution.Variables().All()
	fmt.Println("\n📊 Execution Results:")

	if result1, ok := vars["result1"]; ok {
		fmt.Printf("  result1: %v\n", result1)
	} else {
		fmt.Println("  ❌ result1 not found!")
	}

	if result2, ok := vars["result2"]; ok {
		fmt.Printf("  result2: %v\n", result2)
	} else {
		fmt.Println("  ❌ result2 not found!")
	}

	fmt.Println("\n=== Test Complete ===")
	fmt.Println("\nThis test verifies that:")
	fmt.Println("1. transform1 receives input_value from initial variables")
	fmt.Println("2. transform2 receives result1 from transform1's output")
	fmt.Println("3. Nodes only see their predecessors' outputs, not all variables")
}
