package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smilemakc/mbflow/pkg/sdk"
)

func main() {
	endpoint := os.Getenv("MBFLOW_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8585"
	}

	systemKey := os.Getenv("MBFLOW_SYSTEM_KEY")
	if systemKey == "" {
		log.Fatal("MBFLOW_SYSTEM_KEY environment variable is required")
	}

	config := sdk.ServiceClientConfig{
		Endpoint:  endpoint,
		SystemKey: systemKey,
	}

	client, err := sdk.NewServiceClient(config)
	if err != nil {
		log.Fatalf("Failed to create service client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("=== Service API Client Example ===\n")

	fmt.Println("1. List all workflows")
	workflows, total, err := client.Workflows.List(ctx, &sdk.ListOptions{
		Limit:  10,
		Offset: 0,
		Status: "active",
	})
	if err != nil {
		log.Printf("Failed to list workflows: %v", err)
	} else {
		fmt.Printf("Found %d workflows (total: %d)\n", len(workflows), total)
		for _, wf := range workflows {
			fmt.Printf("  - %s: %s (status: %s)\n", wf.ID, wf.Name, wf.Status)
		}
	}

	fmt.Println("\n2. Create a new workflow")
	createReq := &sdk.ServiceCreateWorkflowRequest{
		Name:        "Service API Test Workflow",
		Description: "Created via Service API",
		Variables: map[string]any{
			"env": "production",
		},
		Metadata: map[string]any{
			"created_by": "service_client_example",
		},
	}

	workflow, err := client.Workflows.Create(ctx, createReq)
	if err != nil {
		log.Printf("Failed to create workflow: %v", err)
	} else {
		fmt.Printf("Created workflow: %s (ID: %s)\n", workflow.Name, workflow.ID)

		fmt.Println("\n3. Update the workflow with nodes and edges")
		updateReq := &sdk.ServiceUpdateWorkflowRequest{
			Description: "Updated via Service API",
			Nodes: []sdk.ServiceNodeRequest{
				{
					ID:   "node-1",
					Name: "Start Node",
					Type: "http",
					Config: map[string]any{
						"url":    "https://api.example.com/data",
						"method": "GET",
					},
				},
				{
					ID:   "node-2",
					Name: "Process Node",
					Type: "transform",
					Config: map[string]any{
						"script": "return { processed: input.data }",
					},
				},
			},
			Edges: []sdk.ServiceEdgeRequest{
				{
					ID:   "edge-1",
					From: "node-1",
					To:   "node-2",
				},
			},
		}

		updated, err := client.Workflows.Update(ctx, workflow.ID, updateReq)
		if err != nil {
			log.Printf("Failed to update workflow: %v", err)
		} else {
			fmt.Printf("Updated workflow: %s (nodes: %d, edges: %d)\n",
				updated.Name, len(updated.Nodes), len(updated.Edges))
		}

		fmt.Println("\n4. Execute the workflow")
		execution, err := client.Executions.Start(ctx, workflow.ID, map[string]any{
			"user_id": "test-user-123",
		})
		if err != nil {
			log.Printf("Failed to start execution: %v", err)
		} else {
			fmt.Printf("Started execution: %s (status: %s)\n", execution.ID, execution.Status)
		}

		fmt.Println("\n5. List executions for the workflow")
		executions, total, err := client.Executions.List(ctx, &sdk.ServiceExecutionListOptions{
			Limit:      10,
			WorkflowID: workflow.ID,
		})
		if err != nil {
			log.Printf("Failed to list executions: %v", err)
		} else {
			fmt.Printf("Found %d executions for workflow (total: %d)\n", len(executions), total)
			for _, exec := range executions {
				fmt.Printf("  - %s: %s\n", exec.ID, exec.Status)
			}
		}

		fmt.Println("\n6. Create a trigger for the workflow")
		triggerReq := &sdk.ServiceCreateTriggerRequest{
			WorkflowID:  workflow.ID,
			Name:        "Daily Trigger",
			Description: "Runs every day at 9 AM",
			Type:        "cron",
			Config: map[string]any{
				"schedule": "0 9 * * *",
				"timezone": "UTC",
			},
			Enabled: true,
		}

		trigger, err := client.Triggers.Create(ctx, triggerReq)
		if err != nil {
			log.Printf("Failed to create trigger: %v", err)
		} else {
			fmt.Printf("Created trigger: %s (type: %s, enabled: %v)\n",
				trigger.Name, trigger.Type, trigger.Enabled)
		}

		fmt.Println("\n7. Using impersonation (As method)")
		userClient := client.As("user-123")
		userWorkflows, total, err := userClient.Workflows.List(ctx, &sdk.ListOptions{
			Limit: 5,
		})
		if err != nil {
			log.Printf("Failed to list workflows as user: %v", err)
		} else {
			fmt.Printf("Found %d workflows for impersonated user (total: %d)\n", len(userWorkflows), total)
		}

		fmt.Println("\n8. Using per-request impersonation (OnBehalfOf)")
		workflow2, err := client.Workflows.Get(ctx, workflow.ID, sdk.OnBehalfOf("user-456"))
		if err != nil {
			log.Printf("Failed to get workflow on behalf of user: %v", err)
		} else {
			fmt.Printf("Retrieved workflow on behalf of user-456: %s\n", workflow2.Name)
		}

		fmt.Println("\n9. Cleanup: Delete the workflow")
		err = client.Workflows.Delete(ctx, workflow.ID)
		if err != nil {
			log.Printf("Failed to delete workflow: %v", err)
		} else {
			fmt.Printf("Deleted workflow: %s\n", workflow.ID)
		}
	}

	fmt.Println("\n=== Example completed successfully ===")
}
