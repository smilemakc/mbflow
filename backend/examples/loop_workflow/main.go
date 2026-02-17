// Iterative Article Review Workflow with Loop Edges
//
// This example demonstrates the loop edge feature in MBFlow:
// - Generate an article draft (LLM)
// - Review and score the content (LLM → JSON with score + article)
// - Parse the review JSON into structured data (JQ transform)
// - Edge-condition routing: score >= 80 → format, otherwise → improve
// - Loop edge: improve → review (max 3 iterations)
// - Final formatting and SEO generation
//
// The workflow showcases:
// 1. Loop edges with max_iterations for controlled retry cycles
// 2. Edge conditions for quality-based routing
// 3. Builder API with WithLoop() and WithCondition()
// 4. Mermaid visualization with dotted loop edges
// 5. JQ transform for JSON parsing
//
// Data flow through the loop:
//
//	generate → review (LLM outputs JSON: {score, issues, article})
//	  → parse_review (JQ: extracts structured data from JSON string)
//	    → [score >= 80] → format → seo
//	    → [score < 80]  → improve (gets article + issues from parse_review)
//	                        ↺ loop(max=3) → review
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
	input := map[string]any{
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
//	generate → review → parse_review
//	                       ├─ [score >= 80] → format → seo
//	                       └─ [score < 80]  → improve
//	                                           ↺ loop(max=3) → review
//
// Data flow note:
// The LLM executor wraps all output in {"content": "..."}, so the review node's
// JSON response (score, issues, article) is a string inside content.
// The parse_review JQ node extracts it into a proper object with top-level fields.
// The review prompt echoes back the article text so it survives through the pipeline.
func buildWorkflow(apiKey string) *models.Workflow {
	// JQ filter: parse the JSON string from LLM's "content" field
	// into a structured object with score, issues, and article
	parseReviewFilter := `.content | fromjson`

	return builder.NewWorkflow("Iterative Article Review",
		builder.WithDescription("Generate, review, and iteratively improve articles with loop-based retry"),
		builder.WithVariable("openai_api_key", apiKey),
		builder.WithVariable("model", "gpt-4o"),
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
		// Step 2: Review content quality (returns JSON with score + echoed article)
		AddNode(
			builder.NewOpenAINode(
				"review",
				"Review Quality",
				"{{env.model}}",
				reviewPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.0),
				builder.LLMJSONMode(),
				builder.LLMMaxTokens(4000),
			),
		).
		// Step 3: Parse the review JSON string into structured data
		// Input:  {"content": "{\"score\":75,\"issues\":[...],\"article\":\"...\"}"}
		// Output: {"score": 75, "issues": [...], "article": "..."}
		AddNode(
			builder.NewJQNode(
				"parse_review",
				"Parse Review",
				parseReviewFilter,
			),
		).
		// Step 4: Improve content (runs when score < 80)
		// Receives: {score, issues, article} from parse_review
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
		// Step 5: Format final output (runs when score >= 80)
		// Receives: {score, issues, article} from parse_review
		AddNode(
			builder.NewExpressionNode(
				"format",
				"Format Output",
				`{"article": input.article, "quality_score": input.score, "status": "approved"}`,
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
		Connect("review", "parse_review").
		// Quality-based routing via edge conditions
		Connect("parse_review", "format", builder.WithCondition("output.score >= 80")).
		Connect("parse_review", "improve", builder.WithCondition("output.score < 80")).
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
	outputStr := string(outputJSON)
	if len(outputStr) > 2000 {
		outputStr = outputStr[:2000] + "\n  ... (truncated)"
	}
	fmt.Printf("  %s\n", outputStr)

	fmt.Println("-----------------------------------------------------------------")
	fmt.Println()
}

func displayLoopStats(execution *models.Execution) {
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
	fmt.Println("  2. 'review' scores the content and echoes article in JSON")
	fmt.Println("  3. 'parse_review' extracts score, issues, article from JSON")
	fmt.Println("  4. Edge conditions route based on score:")
	fmt.Println("     - score >= 80: 'format' -> 'seo' (publish path)")
	fmt.Println("     - score < 80:  'improve' fixes issues")
	fmt.Println("  5. Loop edge sends improved content back to 'review'")
	fmt.Println("  6. Repeat up to 3 times until quality passes")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("  - First try: ~30s (generate + review + parse)")
	fmt.Println("  - Each loop iteration: ~20s (improve + review + parse)")
	fmt.Println("  - Total with 2 iterations: ~70s")
	fmt.Println("=================================================================")
}
