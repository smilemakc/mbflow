package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"mbflow"

	"github.com/google/uuid"
)

// ExecutionDemo demonstrates the workflow execution engine with monitoring and error handling.
// This example shows:
// 1. Creating an executor with monitoring enabled
// 2. Executing a simple workflow with multiple node types
// 3. Viewing execution metrics and results
func main() {
	fmt.Println("=== Workflow Execution Engine Demo ===\n")
	// Get OpenAI API key from environment (optional for this demo)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Note: OPENAI_API_KEY not set. OpenAI nodes will be skipped.")
		fmt.Println("Set OPENAI_API_KEY environment variable to test AI features.\n")
	}

	// Create executor with monitoring enabled
	executor := mbflow.NewWorkflowEngine(&mbflow.EngineConfig{
		OpenAIAPIKey:     apiKey,
		EnableMonitoring: true,
		VerboseLogging:   true,
	})
	httpObserver, err := mbflow.NewHTTPCallbackObserver(mbflow.HTTPCallbackObserverConfig{
		CallbackURL: "https://heabot.nl.tuna.am",
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP callback observer: %v", err)
	}
	executor.AddObserver(httpObserver)
	// Create a simple workflow
	workflowID := uuid.NewString()
	executionID := uuid.NewString()

	fmt.Printf("Workflow ID: %s\n", workflowID)
	fmt.Printf("Execution ID: %s\n\n", executionID)

	// Define nodes to execute
	nodes := []mbflow.NodeConfig{
		// Node 1: Data merger (simulates selecting from multiple sources)
		{
			ID:   uuid.NewString(),
			Name: "Data Merger",
			Type: "data-merger",
			Config: map[string]any{
				"strategy":   "select_first_available",
				"sources":    []string{"input_data", "fallback_data"},
				"output_key": "merged_data",
			},
		},

		// Node 2: Data aggregator (combines multiple fields)
		{
			ID:   uuid.NewString(),
			Name: "Data Aggregator",
			Type: "data-aggregator",
			Config: map[string]any{
				"fields": map[string]string{
					"data":      "merged_data",
					"timestamp": "execution_time",
					"status":    "execution_status",
				},
				"output_key": "aggregated_result",
			},
		},

		// Node 3: Conditional router (routes based on status)
		{
			ID:   uuid.NewString(),
			Name: "Conditional Router",
			Type: "conditional-router",
			Config: map[string]any{
				"input_key": "execution_status",
				"routes": map[string]string{
					"success": "success_path",
					"failure": "failure_path",
					"default": "default_path",
				},
			},
		},
	}

	// If OpenAI API key is available, add an OpenAI node
	if apiKey != "" {
		openaiNode := mbflow.NodeConfig{
			ID:   uuid.NewString(),
			Name: "OpenAI Summarizer",
			Type: "openai-completion",
			Config: map[string]any{
				"model":      "gpt-4o",
				"prompt":     "Summarize this data in one sentence: {{merged_data}}",
				"max_tokens": 100,
				"output_key": "ai_summary",
			},
		}
		nodes = append(nodes, openaiNode)
	}

	// Set initial variables
	initialVariables := map[string]interface{}{
		"input_data": `**Execution Engine Core**

- Workflow orchestration and state management
- Node-by-node execution with dependency handling
- Thread-safe execution state tracking
- Variable storage and substitution`,
		"execution_time":   time.Now().Format(time.RFC3339),
		"execution_status": "success",
	}

	fmt.Println("=== Executing Workflow ===\n")

	// Execute workflow (with empty edges for sequential execution)
	ctx := context.Background()
	state, err := executor.ExecuteWorkflow(ctx, workflowID, executionID, nodes, nil, initialVariables)

	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Println("\n=== Execution Results ===\n")
	fmt.Printf("Execution Status: %s\n", state.GetStatusString())
	fmt.Printf("Execution Duration: %s\n", state.GetExecutionDuration())
	fmt.Printf("Execution ID: %s\n\n", state.GetExecutionID())

	// Display variables
	fmt.Println("=== Execution Variables ===\n")
	variables := state.GetAllVariables()
	for key, value := range variables {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Display metrics
	var nodeIDs []string
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}
	mbflow.DisplayMetrics(executor.GetMetrics(), workflowID, nodeIDs, apiKey != "")

	fmt.Println("\n=== Demo Complete ===")
}
