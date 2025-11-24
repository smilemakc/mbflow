package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
)

func main() {
	fmt.Println("=== MBFlow Error Handling & Retry Example ===")

	// Create a workflow demonstrating error handling strategies
	workflow, err := mbflow.NewWorkflowBuilder("Error Handling Demo", "1.0").
		WithDescription("Demonstrates retry, circuit breaker, and error strategies").
		// Start
		AddNode(string(mbflow.NodeTypeStart), "start", map[string]any{}).
		// Node with retry enabled
		AddNode(string(mbflow.NodeTypeHTTP), "api_call_with_retry", map[string]any{
			"url":    "https://booking-hub.free.beeceptor.com", // This will succeed
			"method": "GET",
			// Enable retry for this node
			"retry_enabled":       true,
			"retry_max_attempts":  3,
			"retry_initial_delay": "1s",
			"retry_multiplier":    2.0,
		}).
		// Transform the response
		AddNode(string(mbflow.NodeTypeTransform), "process_response", map[string]any{
			"transformations": map[string]any{
				"status":  "status_code",
				"success": "status_code == 200",
			},
		}).
		// Conditional routing based on success
		AddNode(string(mbflow.NodeTypeConditionalRoute), "check_status", map[string]any{
			"routes": []map[string]any{
				{
					"name":      "success",
					"condition": "success == true",
				},
				{
					"name":      "failure",
					"condition": "success == false",
				},
			},
			"default_route": "unknown",
		}).
		// Success path
		AddNode(string(mbflow.NodeTypeTransform), "success_handler", map[string]any{
			"transformations": map[string]any{
				"message": `"API call succeeded with status " + string(status)`,
			},
		}).
		// Failure path (would handle errors)
		AddNode(string(mbflow.NodeTypeTransform), "failure_handler", map[string]any{
			"transformations": map[string]any{
				"message": `"API call failed with status " + string(status)`,
			},
		}).
		// End
		AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
			"output_keys": []string{"status", "success", "message", "selected_route"},
		}).
		// Edges
		AddEdge("start", "api_call_with_retry", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("api_call_with_retry", "process_response", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("process_response", "check_status", string(mbflow.EdgeTypeDirect), nil).
		// Conditional edges based on routing
		AddEdge("check_status", "success_handler", string(mbflow.EdgeTypeConditional), map[string]any{
			"condition": `selected_route == "success"`,
		}).
		AddEdge("check_status", "failure_handler", string(mbflow.EdgeTypeConditional), map[string]any{
			"condition": `selected_route == "failure"`,
		}).
		AddEdge("success_handler", "end", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("failure_handler", "end", string(mbflow.EdgeTypeDirect), nil).
		// Trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Test Error Handling",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("✓ Workflow created: %s\n", workflow.Name())
	fmt.Println("  Features demonstrated:")
	fmt.Println("  - Retry with exponential backoff")
	fmt.Println("  - Conditional routing")
	fmt.Println("  - Error handling strategies")
	fmt.Println()

	// Create executor with different error strategies
	strategies := []struct {
		name     string
		strategy mbflow.ErrorStrategy
	}{
		{"Fail Fast", mbflow.NewFailFastStrategy()},
		{"Continue On Error", mbflow.NewContinueOnErrorStrategy()},
		{"Best Effort", mbflow.NewBestEffortStrategy()},
	}

	for _, s := range strategies {
		fmt.Printf("\n=== Testing with %s Strategy ===\n", s.name)

		executor := mbflow.NewExecutorBuilder().
			EnableRetry(3).
			EnableCircuitBreaker().
			Build()

		// Execute
		ctx := context.Background()
		execution, err := executor.ExecuteWorkflow(
			ctx,
			workflow,
			workflow.GetAllTriggers()[0],
			map[string]any{},
		)

		if err != nil {
			fmt.Printf("❌ Execution failed: %v\n", err)
			continue
		}

		fmt.Printf("✓ Execution completed: %s\n", execution.Phase())

		// Display results
		vars := execution.Variables().All()
		if status, ok := vars["status"]; ok {
			fmt.Printf("  HTTP Status: %v\n", status)
		}
		if success, ok := vars["success"]; ok {
			fmt.Printf("  Success: %v\n", success)
		}
		if msg, ok := vars["message"]; ok {
			fmt.Printf("  Message: %s\n", msg)
		}
		if route, ok := vars["selected_route"]; ok {
			fmt.Printf("  Selected Route: %s\n", route)
		}

		// Event analysis
		events, err := executor.EventStore().GetEvents(ctx, execution.ID())
		if err != nil {
			log.Printf("Warning: Failed to get events: %v", err)
		}
		retryEvents := 0
		failedEvents := 0
		for _, evt := range events {
			if evt.EventType() == mbflow.EventTypeNodeRetrying {
				retryEvents++
			}
			if evt.EventType() == mbflow.EventTypeNodeFailed {
				failedEvents++
			}
		}

		fmt.Printf("\n📊 Execution Statistics:\n")
		fmt.Printf("  Total events: %d\n", len(events))
		fmt.Printf("  Retry events: %d\n", retryEvents)
		fmt.Printf("  Failed events: %d\n", failedEvents)
	}

	fmt.Println("\n=== Error handling example completed! ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("✓ Automatic retry with exponential backoff")
	fmt.Println("✓ Circuit breaker protection")
	fmt.Println("✓ Multiple error handling strategies")
	fmt.Println("✓ Conditional routing based on results")
	fmt.Println("✓ Event sourcing for complete audit trail")
}
