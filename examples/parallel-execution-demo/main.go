package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/smilemakc/mbflow"

	"github.com/google/uuid"
)

// ParallelExecutionDemo demonstrates parallel workflow execution with fork-join pattern.
// This example shows:
// 1. Multiple OpenAI nodes executing in parallel
// 2. Join node that aggregates results from parallel branches
// 3. Graph-based execution with edges
func main() {
	fmt.Printf("=== Parallel Workflow Execution Demo ===\n\n")

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: OPENAI_API_KEY environment variable is required for this demo.")
		fmt.Printf("Please set OPENAI_API_KEY to run this example.\n\n")
		os.Exit(1)
	}

	// Create executor with monitoring enabled
	executor := mbflow.NewWorkflowEngine(&mbflow.EngineConfig{
		OpenAIAPIKey:     apiKey,
		EnableMonitoring: true,
		VerboseLogging:   true,
	})

	// Create workflow and execution IDs
	workflowID := uuid.NewString()
	executionID := uuid.NewString()

	fmt.Printf("Workflow ID: %s\n", workflowID)
	fmt.Printf("Execution ID: %s\n\n", executionID)

	// Define nodes for parallel execution
	// Structure: Start -> [Task1, Task2, Task3] -> Join
	startNode, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:   uuid.NewString(),
		Name: "Start Node",
		Type: "data-aggregator",
		Config: map[string]any{
			"fields": map[string]string{
				"topic1": "topic1",
				"topic2": "topic2",
				"topic3": "topic3",
			},
			"output_key": "start_output",
		},
	})
	// Task 1: OpenAI completion for first topic

	task1, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:   uuid.NewString(),
		Name: "Task 1: Summarize Topic 1",
		Type: "openai-completion",
		Config: map[string]any{
			"model":      "gpt-4o",
			"prompt":     "Write a brief summary (2-3 sentences) about {{topic1}}",
			"max_tokens": 150,
			"output_key": "result_1",
		},
	})
	// Task 2: OpenAI completion for second topic
	task2, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:   uuid.NewString(),
		Name: "Task 2: Summarize Topic 2",
		Type: "openai-completion",
		Config: map[string]any{
			"model":      "gpt-4o",
			"prompt":     "Write a brief summary (2-3 sentences) about {{topic2}}",
			"max_tokens": 150,
			"output_key": "result_2",
		},
	})
	// Task 3: OpenAI completion for third topic
	task3, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:   uuid.NewString(),
		Name: "Task 3: Summarize Topic 3",
		Type: "openai-completion",
		Config: map[string]any{
			"model":      "gpt-4o",
			"prompt":     "Write a brief summary (2-3 sentences) about {{topic3}}",
			"max_tokens": 150,
			"output_key": "result_3",
		},
	})
	// Join node: Aggregate all results from parallel tasks
	joinNode, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:   uuid.NewString(),
		Name: "Join Node",
		Type: "data-aggregator",
		Config: map[string]any{
			"fields": map[string]string{
				"summary_1": "result_1",
				"summary_2": "result_2",
				"summary_3": "result_3",
			},
			"output_key": "final_result",
		},
	})
	nodes := []mbflow.NodeConfig{
		mbflow.NodeToConfig(startNode),
		mbflow.NodeToConfig(task1),
		mbflow.NodeToConfig(task2),
		mbflow.NodeToConfig(task3),
		mbflow.NodeToConfig(joinNode),
	}

	// Define edges to create fork-join pattern:
	// start -> task-1, task-2, task-3 (fork)
	// task-1, task-2, task-3 -> join (join)
	edges := mbflow.NewRelationshipBuilder(workflowID).
		Direct(startNode, task1).
		Direct(startNode, task2).
		Direct(startNode, task3).
		Join(task1, joinNode).
		Join(task2, joinNode).
		Join(task3, joinNode).
		Build()
	// Set initial variables with topics for parallel processing
	initialVariables := map[string]interface{}{
		"topic1": "microservices architecture and containerization",
		"topic2": "DevOps practices and CI/CD pipelines",
		"topic3": "test-driven development and quality assurance",
	}

	fmt.Println("=== Workflow Graph Structure ===")
	fmt.Println("  start")
	fmt.Println("    |")
	fmt.Println("    +---> task-1 (OpenAI)")
	fmt.Println("    +---> task-2 (OpenAI)")
	fmt.Println("    +---> task-3 (OpenAI)")
	fmt.Println("    |")
	fmt.Println("  join (aggregator)")
	fmt.Println()

	fmt.Printf("=== Executing Workflow (Parallel Execution) ===\n\n")
	startTime := time.Now()

	// Execute workflow with edges for parallel execution
	ctx := context.Background()
	state, err := executor.ExecuteWorkflow(ctx, workflowID, executionID, nodes, mbflow.EdgesToConfigs(edges), initialVariables)

	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	executionDuration := time.Since(startTime)

	fmt.Printf("\n=== Execution Results ===\n\n")
	fmt.Printf("Status: %s\n", state.GetStatusString())
	fmt.Printf("Execution Duration: %s\n", executionDuration)
	fmt.Printf("State Duration: %s\n\n", state.GetExecutionDuration())

	// Display all variables
	fmt.Printf("=== Execution Variables ===\n\n")
	variables := state.GetAllVariables()
	for key, value := range variables {
		if key == "final_result" {
			fmt.Printf("  %s:\n", key)
			if resultMap, ok := value.(map[string]interface{}); ok {
				for k, v := range resultMap {
					fmt.Printf("    %s: %v\n", k, v)
				}
			} else {
				fmt.Printf("    %v\n", value)
			}
		} else {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Display final aggregated result
	fmt.Printf("\n=== Final Aggregated Result ===\n\n")
	if finalResult, ok := variables["final_result"]; ok {
		if resultMap, ok := finalResult.(map[string]interface{}); ok {
			fmt.Println("Combined summaries from all parallel tasks:")
			for i := 1; i <= 3; i++ {
				key := fmt.Sprintf("summary_%d", i)
				if summary, exists := resultMap[key]; exists {
					fmt.Printf("\n  Task %d:\n    %v\n", i, summary)
				}
			}
		} else {
			fmt.Printf("  %v\n", finalResult)
		}
	}

	// Display metrics
	nodeIDs := []string{"start", "task-1", "task-2", "task-3", "join"}
	mbflow.DisplayMetrics(executor.GetMetrics(), workflowID, nodeIDs, true)

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nNote: The three OpenAI tasks executed in parallel,")
	fmt.Println("which should be faster than sequential execution.")
}
