package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"mbflow"

	"github.com/google/uuid"
)

func main() {
	// Start a simple HTTP server to receive callbacks
	go startCallbackServer()
	time.Sleep(100 * time.Millisecond) // Give server time to start

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create executor with monitoring enabled
	executor := mbflow.NewExecutor(&mbflow.ExecutorConfig{
		OpenAIAPIKey:     apiKey,
		MaxRetryAttempts: 3,
		EnableMonitoring: true,
		VerboseLogging:   true,
	})

	ctx := context.Background()
	workflowID := "callback-demo-workflow"
	executionID := fmt.Sprintf("exec-%d", time.Now().Unix())

	// Define nodes with callback configuration
	nodes := []mbflow.ExecutorNodeConfig{
		{
			NodeID:   uuid.NewString(),
			Name:     "Generate Text",
			NodeType: "openai-completion",
			Config: map[string]any{
				"model":       "gpt-4o-mini",
				"prompt":      "Write a short poem about {{topic}}",
				"max_tokens":  100,
				"temperature": 0.7,
				"output_key":  "poem",
				// Configure callback to be called after successful execution
				"on_success_callback": map[string]any{
					"url":               "http://localhost:8080/callback",
					"method":            "POST",
					"timeout_seconds":   10,
					"include_variables": true, // Include all execution variables in callback
					"headers": map[string]string{
						"X-Custom-Header": "callback-demo",
					},
				},
			},
		},
		{
			NodeID:   uuid.NewString(),
			Name:     "Analyze Poem",
			NodeType: "openai-completion",
			Config: map[string]any{
				"model":       "gpt-4o-mini",
				"prompt":      "Analyze this poem and provide a brief critique:\n\n{{poem}}",
				"max_tokens":  150,
				"temperature": 0.3,
				"output_key":  "critique",
				// This node also has a callback
				"on_success_callback": map[string]any{
					"url":               "http://localhost:8080/callback",
					"method":            "POST",
					"timeout_seconds":   10,
					"include_variables": false, // Don't include variables in this callback
				},
			},
		},
	}

	// Define workflow edges
	edges := []mbflow.ExecutorEdgeConfig{
		{
			FromNodeID: nodes[0].NodeID,
			ToNodeID:   nodes[1].NodeID,
			EdgeType:   "sequence",
		},
	}

	// Execute workflow
	fmt.Println("Starting workflow execution...")
	state, err := executor.ExecuteWorkflow(
		ctx,
		workflowID,
		executionID,
		nodes,
		edges,
		map[string]interface{}{
			"topic": "artificial intelligence",
		},
	)

	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Printf("\nWorkflow completed with status: %s\n", state.Status())
	fmt.Printf("Execution duration: %s\n", state.GetExecutionDuration())

	// Display results
	if poem, ok := state.GetVariable("poem"); ok {
		fmt.Printf("\n=== Generated Poem ===\n%v\n", poem)
	}

	if critique, ok := state.GetVariable("critique"); ok {
		fmt.Printf("\n=== Critique ===\n%v\n", critique)
	}

	// Get metrics
	metrics := executor.GetMetrics()
	fmt.Printf("\n=== Metrics ===\n")
	summary := metrics.GetSummary()
	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(summaryJSON))

	// Wait a bit to ensure callbacks complete
	fmt.Println("\nWaiting for callbacks to complete...")
	time.Sleep(2 * time.Second)
}

// startCallbackServer starts a simple HTTP server to receive callbacks
func startCallbackServer() {
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Parse callback data
		var callbackData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&callbackData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Log callback data
		fmt.Printf("\n=== Callback Received ===\n")
		fmt.Printf("Node ID: %v\n", callbackData["node_id"])
		fmt.Printf("Node Type: %v\n", callbackData["node_type"])
		fmt.Printf("Execution ID: %v\n", callbackData["execution_id"])
		fmt.Printf("Duration: %v ms\n", callbackData["duration_ms"])

		// Check if variables are included
		if variables, ok := callbackData["variables"].(map[string]interface{}); ok {
			fmt.Printf("Variables included: %d\n", len(variables))
			for k := range variables {
				fmt.Printf("  - %s\n", k)
			}
		} else {
			fmt.Println("Variables: not included")
		}

		// Pretty print the full callback data
		prettyJSON, _ := json.MarshalIndent(callbackData, "", "  ")
		fmt.Printf("\nFull callback data:\n%s\n", string(prettyJSON))
		fmt.Println("======================")

		// Respond with success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	fmt.Println("Callback server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Printf("Callback server error: %v", err)
	}
}
