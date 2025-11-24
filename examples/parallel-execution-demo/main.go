package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// ParallelExecutionDemo demonstrates parallel workflow execution with fork-join pattern.
// This example shows:
// 1. Multiple OpenAI nodes executing in parallel
// 2. Aggregation of results from parallel branches
// 3. Graph-based execution with the new architecture
func main() {
	fmt.Printf("=== Parallel Workflow Execution Demo ===\n\n")

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  WARNING: OPENAI_API_KEY environment variable is not set.")
		fmt.Println("This demo will run but OpenAI nodes will be simulated.")
		fmt.Println("Set OPENAI_API_KEY to enable real OpenAI API calls.")
	}

	// Create a workflow with parallel execution pattern
	workflow, err := mbflow.NewWorkflowBuilder("Parallel AI Processing", "1.0").
		WithDescription("Demonstrates parallel execution of multiple AI tasks").
		// Start node
		AddNode(string(mbflow.NodeTypeStart), "start", map[string]any{}).
		// Three parallel nodes that process different topics
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "task1", &mbflow.OpenAICompletionConfig{
			Model:     "gpt-4o",
			Prompt:    "Write a brief summary (2-3 sentences) about {{topic1}}",
			MaxTokens: 150,
			OutputKey: "result_1",
			APIKey:    apiKey,
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "task2", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"result_2": `"Summary of " + topic2 + ": Processed in parallel"`,
			},
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "task3", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"result_3": `"Summary of " + topic3 + ": Processed in parallel"`,
			},
		}).
		// Aggregation node - combines all results
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "aggregate", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"final_result": `"Combined: " + result_1['content'] + " | " + result_2 + " | " + result_3`,
			},
		}).
		// End node
		AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
			"output_keys": []string{"result_1", "result_2", "result_3", "final_result"},
		}).
		// Create fork-join pattern with edges
		// Fork: start -> task1, task2, task3 (parallel execution)
		AddEdge("start", "task1", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("start", "task2", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("start", "task3", string(mbflow.EdgeTypeDirect), nil).
		// Join: task1, task2, task3 -> aggregate (wait for all)
		AddEdge("task1", "aggregate", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("task2", "aggregate", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("task3", "aggregate", string(mbflow.EdgeTypeDirect), nil).
		// Complete the flow
		AddEdge("aggregate", "end", string(mbflow.EdgeTypeDirect), nil).
		// Add manual trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name":        "Start Parallel Processing",
			"description": "Execute AI tasks in parallel",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("✓ Workflow created: %s (version %s)\n", workflow.Name(), workflow.Version())
	fmt.Printf("  Nodes: %d\n", len(workflow.GetAllNodes()))
	fmt.Printf("  Edges: %d\n", len(workflow.GetAllEdges()))
	fmt.Printf("  Triggers: %d\n\n", len(workflow.GetAllTriggers()))

	fmt.Println("=== Workflow Graph Structure ===")
	fmt.Println("  start")
	fmt.Println("    ├──> task-1 (Transform)")
	fmt.Println("    ├──> task-2 (Transform)")
	fmt.Println("    └──> task-3 (Transform)")
	fmt.Println("         └──> aggregate")
	fmt.Println("              └──> end")
	fmt.Println()

	// Create executor with parallel execution enabled
	executor := mbflow.NewExecutorBuilder().
		EnableParallelExecution(10).
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("parallel-demo"))).
		EnableRetry(2).
		EnableMetrics().
		Build()

	fmt.Println("✓ Executor created with configuration:")
	fmt.Println("  - Parallel execution: enabled (max 10 nodes)")
	fmt.Println("  - Retry: enabled (max 2 attempts)")
	fmt.Println("  - Metrics: enabled")
	fmt.Println("  - Monitoring: enabled")

	// Get the trigger
	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal("No triggers found in workflow")
	}
	trigger := triggers[0]

	// Execute the workflow with initial variables
	ctx := context.Background()
	initialVars := map[string]any{
		"topic1":         "microservices architecture and containerization",
		"topic2":         "DevOps practices and CI/CD pipelines",
		"topic3":         "test-driven development and quality assurance",
		"openai_api_key": apiKey,
	}

	fmt.Println("▶ Executing workflow with parallel tasks...")
	fmt.Printf("  Topic 1: %s\n", initialVars["topic1"])
	fmt.Printf("  Topic 2: %s\n", initialVars["topic2"])
	fmt.Printf("  Topic 3: %s\n\n", initialVars["topic3"])

	execution, err := executor.ExecuteWorkflow(ctx, workflow, trigger, initialVars)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Println("✓ Parallel workflow execution completed successfully!")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n", execution.Phase())
	fmt.Printf("  Duration: %v\n", execution.Duration())

	// Get the final variables
	vars := execution.Variables().All()
	fmt.Println("\n📊 Execution Results:")
	if result1, ok := vars["result_1"]; ok {
		fmt.Printf("  Result 1: %v\n", result1)
	}
	if result2, ok := vars["result_2"]; ok {
		fmt.Printf("  Result 2: %s\n", result2)
	}
	if result3, ok := vars["result_3"]; ok {
		fmt.Printf("  Result 3: %s\n", result3)
	}

	fmt.Println("\n📈 Aggregated Result:")
	if finalResult, ok := vars["final_result"]; ok {
		fmt.Printf("  %s\n", finalResult)
	}

	// Get events from the event store
	events, err := executor.EventStore().GetEvents(ctx, execution.ID())
	if err != nil {
		log.Printf("Warning: Failed to get events: %v", err)
	} else {
		fmt.Printf("\n📝 Events recorded: %d\n", len(events))

		// Count node execution events
		nodeStartCount := 0
		nodeCompleteCount := 0
		for _, evt := range events {
			switch evt.EventType() {
			case mbflow.EventTypeNodeStarted:
				nodeStartCount++
			case mbflow.EventTypeNodeCompleted:
				nodeCompleteCount++
			}
		}
		fmt.Printf("  Nodes started: %d\n", nodeStartCount)
		fmt.Printf("  Nodes completed: %d\n", nodeCompleteCount)
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nNote: The three tasks can execute in parallel,")
	fmt.Println("which is faster than sequential execution.")
	fmt.Println("\nTip: Enable OPENAI_API_KEY to use real AI completions.")
}
