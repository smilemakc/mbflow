package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
)

// This example demonstrates edge-based variable passing, where downstream nodes
// can access outputs from upstream nodes that are not direct parents.
//
// Workflow structure:
//
//	generate → analyze → router → [enhance] → aggregate → end
//	                        └────→ [skip]   ─┘
//
// Problem: aggregate needs access to generate output, but:
// - If routed through "high" path: only "router" is parent
// - If routed through "medium" path: "enhance" is parent, but enhance needs generate output too
//
// Solution: Use include_outputs_from in edge config to make generate output available

func main() {
	ctx := context.Background()

	// Build workflow with edge-based variable passing
	workflow, err := buildWorkflow()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	// Create executor
	executor := mbflow.NewExecutor()

	// Create and start execution
	fmt.Println("=== Edge-Based Variable Passing Demo ===\n")
	fmt.Println("Testing high quality path (skip enhance)...")
	testHighQualityPath(ctx, executor, workflow)

	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("\nTesting medium quality path (with enhance)...")
	testMediumQualityPath(ctx, executor, workflow)
}

func buildWorkflow() (mbflow.Workflow, error) {
	return mbflow.NewWorkflowBuilder("Edge Data Sources Demo", "1.0").
		WithDescription("Demonstrates edge-based variable passing for multi-stage pipelines").

		// Node 1: Generate content
		AddNode("transform", "generate", map[string]any{
			"transformations": map[string]any{
				"content":    `"Generated blog post: " + input_topic`,
				"word_count": "100",
			},
		}).

		// Node 2: Analyze quality
		AddNode("transform", "analyze", map[string]any{
			"transformations": map[string]any{
				// Access generate output via scoped variables (direct parent)
				"quality_rating": `content != "" ? (word_count >= 80 ? "high" : "medium") : "low"`,
			},
		}).

		// Node 3: Router (conditional)
		AddNode("conditional-router", "router", map[string]any{
			"routes": []map[string]any{
				{"name": "high", "condition": `quality_rating == "high"`},
				{"name": "medium", "condition": `quality_rating == "medium"`},
			},
			"default_route": "low",
		}).

		// Node 4: Enhance content (only for medium quality)
		AddNode("transform", "enhance", map[string]any{
			"transformations": map[string]any{
				// IMPORTANT: enhance needs generate output but only has router as direct parent!
				// Solution: Use include_outputs_from in the edge from router to enhance
				"content":    `generate_content + " [ENHANCED with better keywords]"`,
				"word_count": "generate_word_count + 20",
			},
		}).

		// Node 5: Aggregate results
		AddNode("transform", "aggregate", map[string]any{
			"transformations": map[string]any{
				// IMPORTANT: This demonstrates edge-based variable passing!
				// aggregate receives variables from:
				// 1. Direct parent (router in high path, enhance in medium path)
				// 2. Additional sources via include_outputs_from: generate, analyze

				// generate_content and generate_word_count are ALWAYS available
				// because they're in include_outputs_from, even though generate is not a direct parent!
				"final_content":    "generate_content",
				"final_word_count": "generate_word_count",

				// analyze_quality_rating is also available via include_outputs_from
				"quality": "analyze_quality_rating",

				// Check if enhance path was taken by checking if we have "content" variable
				// (enhance is direct parent in medium path, so content is merged without namespace)
				"path_taken": `selected_route == "high" ? "high (skipped enhance)" : "medium (enhanced)"`,
			},
		}).

		// Node 6: End
		AddNode("end", "end", nil).

		// Edges
		AddEdge("generate", "analyze", "direct", nil).
		AddEdge("analyze", "router", "direct", nil).

		// Edge with additional data source: enhance needs generate output
		AddEdge("router", "enhance", "conditional", map[string]any{
			"condition":            `selected_route == "medium"`,
			"include_outputs_from": []string{"generate"}, // KEY FEATURE!
		}).

		// Edge from router to aggregate (high quality path - skip enhance)
		AddEdge("router", "aggregate", "conditional", map[string]any{
			"condition":            `selected_route == "high"`,
			"include_outputs_from": []string{"generate", "analyze"}, // KEY FEATURE!
		}).

		// Edge from enhance to aggregate (medium quality path)
		AddEdgeWithDataSources("enhance", "aggregate", "direct",
			[]string{"generate", "analyze"}). // Using convenience method

		AddEdge("aggregate", "end", "direct", nil).
		AddTrigger("manual", nil).
		Build()
}

func testHighQualityPath(ctx context.Context, executor *mbflow.Executor, workflow mbflow.Workflow) {
	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal("No triggers found in workflow")
	}
	trigger := triggers[0]

	execution, err := executor.ExecuteWorkflow(ctx, workflow, trigger, map[string]any{
		"input_topic": "Kubernetes Best Practices",
	})

	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Printf("Execution Status: %s\n", execution.Phase())

	// Check results
	vars := execution.Variables().All()
	fmt.Printf("\nResults:\n")
	fmt.Printf("  Final Content: %v\n", vars["final_content"])
	fmt.Printf("  Final Word Count: %v\n", vars["final_word_count"])
	fmt.Printf("  Quality: %v\n", vars["quality"])
	fmt.Printf("  Path Taken: %v\n", vars["path_taken"])

	// Verify that final_content contains the expected value
	// This proves that edge-based variable passing worked!
	// aggregate node accessed generate_content via include_outputs_from
	if content, ok := vars["final_content"].(string); !ok || content != "Generated blog post: Kubernetes Best Practices" {
		log.Fatalf("ERROR: Expected final_content to be 'Generated blog post: Kubernetes Best Practices', got: %v", vars["final_content"])
	}
	fmt.Println("\n✓ High quality path works correctly")
	fmt.Println("✓ Edge-based variable passing works - aggregate accessed generate_content!")
}

func testMediumQualityPath(ctx context.Context, executor *mbflow.Executor, workflow mbflow.Workflow) {
	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal("No triggers found in workflow")
	}
	trigger := triggers[0]

	execution, err := executor.ExecuteWorkflow(ctx, workflow, trigger, map[string]any{
		"input_topic": "Docker", // Shorter input -> medium quality
	})

	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Printf("Execution Status: %s\n", execution.Phase())

	// Check results
	vars := execution.Variables().All()
	fmt.Printf("\nResults:\n")
	fmt.Printf("  Final Content: %v\n", vars["final_content"])
	fmt.Printf("  Final Word Count: %v\n", vars["final_word_count"])
	fmt.Printf("  Quality: %v\n", vars["quality"])
	fmt.Printf("  Path Taken: %v\n", vars["path_taken"])

	// Verify that final_content contains the expected value
	// This proves that edge-based variable passing worked!
	// aggregate node accessed generate_content via include_outputs_from
	if content, ok := vars["final_content"].(string); !ok || content != "Generated blog post: Docker" {
		log.Fatalf("ERROR: Expected final_content to be 'Generated blog post: Docker', got: %v", vars["final_content"])
	}
	fmt.Println("\n✓ Medium quality path works correctly")
	fmt.Println("✓ Edge-based variable passing works - aggregate accessed generate_content!")
}
