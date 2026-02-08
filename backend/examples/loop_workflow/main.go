// Iterative Article Review Workflow with Loop Edges
//
// This example demonstrates the loop edge feature in MBFlow:
// - Generate an article draft (LLM)
// - Review and score the content (LLM → JSON)
// - Conditional routing: score >= 80 → publish, otherwise → improve
// - Loop edge: improve → review (max 3 iterations)
// - Final formatting and SEO generation
//
// The workflow showcases:
// 1. Loop edges with max_iterations for controlled retry cycles
// 2. Conditional branching via SourceHandle (true/false)
// 3. Builder API with WithLoop(), FromTrueBranch(), FromFalseBranch()
// 4. Mermaid visualization with dotted loop edges
// 5. Wave-based parallel execution
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/pkg/builder"
	"github.com/smilemakc/mbflow/pkg/engine"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
	"github.com/smilemakc/mbflow/pkg/visualization"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("OPENAI_API_KEY not set")
		log.Println("Set it with: export OPENAI_API_KEY='your-key-here'")
		log.Println()
		log.Println("Continuing with workflow definition (execution will require API key)...")
		log.Println()
		apiKey = "sk-placeholder"
	}

	client, err := sdk.NewStandaloneClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	workflow := buildWorkflow(apiKey)

	fmt.Println("=================================================================")
	fmt.Println("  Iterative Article Review Workflow (Loop Edges Demo)")
	fmt.Println("=================================================================")
	fmt.Println()
	fmt.Printf("Workflow: %s\n", workflow.Name)
	fmt.Printf("Nodes:    %d\n", len(workflow.Nodes))
	fmt.Printf("Edges:    %d (including loop edges)\n", len(workflow.Edges))
	fmt.Println()

	// Count loop edges
	loopCount := 0
	for _, edge := range workflow.Edges {
		if edge.IsLoop() {
			loopCount++
		}
	}
	fmt.Printf("Loop edges: %d\n", loopCount)
	fmt.Println()

	// Render Mermaid diagram
	renderOpts := &visualization.RenderOptions{
		UseColor:        true,
		ShowDescription: true,
		ShowConfig:      true,
		ShowConditions:  true,
		Direction:       "elk",
	}
	mermaid, err := visualization.RenderWorkflow(workflow, "mermaid", renderOpts)
	if err != nil {
		log.Fatalf("Failed to render workflow: %v", err)
	}
	fmt.Println("Mermaid Diagram:")
	fmt.Println(mermaid)
	fmt.Println()

	// Display structure
	displayStructure(workflow)

	if apiKey == "sk-placeholder" {
		fmt.Println()
		fmt.Println("Skipping execution - valid OPENAI_API_KEY required")
		fmt.Println()
		showUsage()
		return
	}

	// Execute
	input := map[string]interface{}{
		"topic":         "How Microservices Architecture Improves Scalability",
		"style":         "technical blog post",
		"target_length": 800,
		"audience":      "software engineers",
	}

	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("  INPUT")
	fmt.Println("-----------------------------------------------------------------")
	inputJSON, _ := json.MarshalIndent(input, "  ", "  ")
	fmt.Printf("  %s\n", string(inputJSON))
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println()

	fmt.Println("Executing workflow...")
	fmt.Println()

	startTime := time.Now()
	opts := &engine.ExecutionOptions{
		MaxParallelism: 5,
	}

	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, opts)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Execution failed: %v\n", err)
		if execution != nil {
			displayExecution(execution)
		}
		return
	}

	fmt.Printf("Execution completed in %v\n", duration)
	fmt.Println()

	displayExecution(execution)
	displayOutput(execution)
	displayLoopStats(execution)
}

// buildWorkflow creates the iterative article review workflow using the builder API.
//
// Flow:
//
//	generate → review → check_score
//	                       ├─ [true]  → format → seo
//	                       └─ [false] → improve
//	                                     └─ loop(max=3) → review
func buildWorkflow(apiKey string) *models.Workflow {
	return builder.NewWorkflow("Iterative Article Review",
		builder.WithDescription("Generate, review, and iteratively improve articles with loop-based retry"),
		builder.WithVariable("openai_api_key", apiKey),
		builder.WithVariable("model", "gpt-4o"),
		builder.WithVariable("quality_threshold", 80),
		builder.WithAutoLayout(),
		builder.WithTags("loop-edges", "content-review", "iterative"),
	).
		// Step 1: Generate initial draft
		AddNode(
			builder.NewOpenAINode(
				"generate",
				"Generate Draft",
				"{{env.model}}",
				draftPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.7),
				builder.LLMMaxTokens(2000),
			),
		).
		// Step 2: Review content quality (returns JSON with score)
		AddNode(
			builder.NewOpenAINode(
				"review",
				"Review Quality",
				"{{env.model}}",
				reviewPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.0),
				builder.LLMJSONMode(),
				builder.LLMMaxTokens(500),
			),
		).
		// Step 3: Check if quality score meets threshold
		AddNode(
			builder.NewNode("check_score", "conditional", "Quality Check",
				builder.WithConfigValue("condition", "input.score >= 80"),
			),
		).
		// Step 4: Improve content (runs on false branch)
		AddNode(
			builder.NewOpenAINode(
				"improve",
				"Improve Content",
				"{{env.model}}",
				improvePrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.5),
				builder.LLMMaxTokens(2000),
			),
		).
		// Step 5: Format final output (runs on true branch)
		AddNode(
			builder.NewExpressionNode(
				"format",
				"Format Output",
				`{
					"article": input.content,
					"quality_score": input.score,
					"status": "approved"
				}`,
			),
		).
		// Step 6: Generate SEO metadata
		AddNode(
			builder.NewOpenAINode(
				"seo",
				"Generate SEO",
				"{{env.model}}",
				seoPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.0),
				builder.LLMJSONMode(),
				builder.LLMMaxTokens(300),
			),
		).
		// Edges: linear flow
		Connect("generate", "review").
		// Conditional branching from check_score
		Connect("review", "check_score").
		Connect("check_score", "format", builder.FromTrueBranch()).
		Connect("check_score", "improve", builder.FromFalseBranch()).
		// Continue after format
		Connect("format", "seo").
		// Loop edge: improve → review (retry up to 3 times)
		Connect("improve", "review", builder.WithLoop(3)).
		MustBuild()
}

func displayStructure(workflow *models.Workflow) {
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("  WORKFLOW STRUCTURE")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println()
	fmt.Println("Nodes:")
	for i, node := range workflow.Nodes {
		fmt.Printf("  %d. %-25s (%s)\n", i+1, node.Name, node.Type)
	}
	fmt.Println()

	fmt.Println("Edges:")
	for _, edge := range workflow.Edges {
		arrow := " --> "
		label := ""
		if edge.IsLoop() {
			arrow = " -. loop .-> "
			label = fmt.Sprintf(" [max %d iterations]", edge.Loop.MaxIterations)
		} else if edge.SourceHandle != "" {
			label = fmt.Sprintf(" [%s branch]", edge.SourceHandle)
		} else if edge.Condition != "" {
			label = fmt.Sprintf(" [%s]", edge.Condition)
		}
		fmt.Printf("  %s%s%s%s\n", edge.From, arrow, edge.To, label)
	}
	fmt.Println("-----------------------------------------------------------------")
}

func displayExecution(execution *models.Execution) {
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("  EXECUTION DETAILS")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("  Status: %s\n", execution.Status)
	fmt.Printf("  Duration: %dms\n", execution.Duration)
	fmt.Printf("  Nodes Executed: %d\n", len(execution.NodeExecutions))
	fmt.Println()

	fmt.Println("  Execution Path:")
	for i, nodeExec := range execution.NodeExecutions {
		status := "+"
		if nodeExec.Status == models.NodeExecutionStatusSkipped {
			status = "-"
		} else if nodeExec.Status != models.NodeExecutionStatusCompleted {
			status = "x"
		}
		fmt.Printf("  %s %2d. %-25s [%s, %dms]\n",
			status, i+1, nodeExec.NodeName, nodeExec.Status, nodeExec.Duration)

		if nodeExec.Error != "" {
			fmt.Printf("       Error: %s\n", nodeExec.Error)
		}
	}
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println()
}

func displayOutput(execution *models.Execution) {
	if execution.Output == nil {
		fmt.Println("No output available")
		return
	}

	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("  FINAL OUTPUT")
	fmt.Println("-----------------------------------------------------------------")

	outputJSON, _ := json.MarshalIndent(execution.Output, "  ", "  ")
	// Truncate long output for display
	outputStr := string(outputJSON)
	if len(outputStr) > 2000 {
		outputStr = outputStr[:2000] + "\n  ... (truncated)"
	}
	fmt.Printf("  %s\n", outputStr)

	fmt.Println("-----------------------------------------------------------------")
	fmt.Println()
}

func displayLoopStats(execution *models.Execution) {
	// Count how many times review and improve nodes executed
	reviewCount := 0
	improveCount := 0
	for _, nodeExec := range execution.NodeExecutions {
		if nodeExec.Status != models.NodeExecutionStatusCompleted {
			continue
		}
		if strings.Contains(nodeExec.NodeID, "review") {
			reviewCount++
		}
		if strings.Contains(nodeExec.NodeID, "improve") {
			improveCount++
		}
	}

	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("  LOOP STATISTICS")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("  Review executions:  %d\n", reviewCount)
	fmt.Printf("  Improve executions: %d\n", improveCount)
	if improveCount > 0 {
		fmt.Printf("  Loop iterations:    %d (content was improved %d time(s))\n",
			improveCount, improveCount)
	} else {
		fmt.Println("  Loop iterations:    0 (content passed quality check on first try)")
	}
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println()
}

func showUsage() {
	fmt.Println("=================================================================")
	fmt.Println("  USAGE")
	fmt.Println("=================================================================")
	fmt.Println()
	fmt.Println("To execute this workflow:")
	fmt.Println()
	fmt.Println("1. Set your OpenAI API key:")
	fmt.Println("   export OPENAI_API_KEY='sk-your-key-here'")
	fmt.Println()
	fmt.Println("2. Run the example:")
	fmt.Println("   go run backend/examples/loop_workflow/*.go")
	fmt.Println()
	fmt.Println("How the loop works:")
	fmt.Println("  1. 'generate' creates an initial article draft")
	fmt.Println("  2. 'review' scores the content (0-100)")
	fmt.Println("  3. 'check_score' routes based on score >= 80")
	fmt.Println("     - true:  'format' -> 'seo' (publish path)")
	fmt.Println("     - false: 'improve' fixes issues")
	fmt.Println("  4. Loop edge sends improved content back to 'review'")
	fmt.Println("  5. Repeat up to 3 times until quality passes")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("  - First try: ~30s (generate + review)")
	fmt.Println("  - Each loop iteration: ~20s (improve + review)")
	fmt.Println("  - Total with 2 iterations: ~70s")
	fmt.Println("=================================================================")
}
