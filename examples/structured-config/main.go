package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

func main() {
	fmt.Println("=== MBFlow Structured Config Example ===")

	// Create a workflow using structured configs instead of map[string]any
	// This provides type safety and better IDE support
	workflow, err := mbflow.NewWorkflowBuilder("Structured Config Demo", "1.0").
		WithDescription("Demonstrates using type-safe structured configs").
		// Add Start node
		AddNode(string(mbflow.NodeTypeStart), "start", map[string]any{}).
		// Add HTTP Request node with structured config
		AddNodeWithConfig(
			string(mbflow.NodeTypeHTTPRequest),
			"fetch_data",
			&mbflow.HTTPRequestConfig{
				URL:    "https://api.github.com/repos/golang/go",
				Method: "GET",
				Headers: map[string]string{
					"Accept": "application/json",
				},
			},
		).
		// Add JSON Parser with structured config
		AddNodeWithConfig(
			string(mbflow.NodeTypeJSONParser),
			"parse_json",
			&mbflow.JSONParserConfig{
				InputKey:    "repo_data",
				FailOnError: true,
			},
		).
		// Add Data Aggregator with structured config
		AddNodeWithConfig(
			string(mbflow.NodeTypeDataAggregator),
			"extract_info",
			&mbflow.DataAggregatorConfig{
				Fields: map[string]string{
					"name":        "parsed_data.name",
					"stars":       "parsed_data.stargazers_count",
					"description": "parsed_data.description",
				},
			},
		).
		// Add End node
		AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
			"output_keys": []string{"repo_info"},
		}).
		// Connect nodes
		AddEdge("start", "fetch_data", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("fetch_data", "parse_json", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("parse_json", "extract_info", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("extract_info", "end", string(mbflow.EdgeTypeDirect), nil).
		// Add manual trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Start Processing",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("✓ Workflow created: %s (version %s)\n", workflow.Name(), workflow.Version())
	fmt.Println("\n📋 Workflow Structure:")
	for _, node := range workflow.GetAllNodes() {
		fmt.Printf("  • %s (%s)\n", node.Name(), node.Type())
	}

	// Create executor
	executor := mbflow.NewExecutorBuilder().
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("structured-config-demo"))).
		EnableMetrics().
		Build()

	fmt.Println("\n✓ Executor created")

	// Get trigger and execute
	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal("No triggers found")
	}

	fmt.Println("▶ Executing workflow...")
	ctx := context.Background()
	execution, err := executor.ExecuteWorkflow(ctx, workflow, triggers[0], map[string]any{})
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Println("\n✓ Workflow execution completed!")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n\n", execution.Phase())

	// Show results
	vars := execution.Variables().All()
	fmt.Println("📊 Results:")
	if repoInfo, ok := vars["repo_info"]; ok {
		fmt.Printf("  Repository Info: %+v\n", repoInfo)
	}

	fmt.Println("\n=== Example completed successfully! ===")
	fmt.Println("\n💡 Key Benefits of Structured Configs:")
	fmt.Println("  ✓ Type safety at compile time")
	fmt.Println("  ✓ Better IDE autocomplete and documentation")
	fmt.Println("  ✓ Easier to refactor and maintain")
	fmt.Println("  ✓ Validation through struct tags")
}
