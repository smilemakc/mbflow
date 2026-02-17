// Basic usage example for MBFlow SDK - Embedded Mode
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
)

func main() {
	// Create a standalone client (no database required)
	// In standalone mode, only ExecuteWorkflowStandalone() is available (no persistence).
	// For production with persistence, use:
	// sdk.NewClient(sdk.WithEmbeddedMode("postgres://...", "redis://..."))
	client, err := sdk.NewStandaloneClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a simple workflow with template resolution
	workflow := &models.Workflow{
		Name:        "Hello World Workflow",
		Description: "A workflow demonstrating automatic template resolution",
		Variables: map[string]any{
			"api_base": "https://jsonplaceholder.typicode.com",
		},
		Nodes: []*models.Node{
			{
				ID:   "fetch-data",
				Name: "Fetch User Data",
				Type: "http",
				Config: map[string]any{
					"method": "GET",
					"url":    "{{env.api_base}}/users/{{input.user_id}}",
				},
			},
			{
				ID:   "transform-data",
				Name: "Transform Data",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "edge-1",
				From: "fetch-data",
				To:   "transform-data",
			},
		},
	}

	fmt.Printf("Workflow defined: %s\n\n", workflow.Name)

	// Execute workflow in standalone mode (no database required)
	fmt.Println("Executing workflow in standalone mode...")

	// Prepare execution input
	input := map[string]any{
		"user_id": 1,
	}

	// Execute workflow
	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, nil)
	if err != nil {
		log.Printf("Execution failed: %v\n", err)
		return
	}

	fmt.Printf("\n✓ Execution completed successfully!\n")
	fmt.Printf("  Execution ID: %s\n", execution.ID)
	fmt.Printf("  Status: %s\n", execution.Status)
	fmt.Printf("  Duration: %dms\n", execution.Duration)

	// Display node results
	fmt.Println("\nNode Execution Results:")
	for _, nodeExec := range execution.NodeExecutions {
		fmt.Printf("  - %s: %s\n", nodeExec.NodeName, nodeExec.Status)
		if nodeExec.Error != "" {
			fmt.Printf("    Error: %s\n", nodeExec.Error)
		}
	}

	fmt.Println("\nWorkflow Template Features:")
	fmt.Println("  - Template engine automatically resolves {{env.variable}} and {{input.field}}")
	fmt.Println("  - Node: fetch-data uses {{env.api_base}}/users/{{input.user_id}}")
	fmt.Println("  - During execution, templates are resolved to actual values")
	fmt.Println("  - Standalone mode: no database required, perfect for testing!")
}
