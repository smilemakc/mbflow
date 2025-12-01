// Multi-Language Content Generation Workflow
//
// This example demonstrates a sophisticated content generation workflow with:
// - Quality-based routing (high/medium/low quality paths)
// - Content enhancement or regeneration loops
// - Parallel translations to multiple languages (Spanish, Russian, German)
// - SEO metadata generation for all language versions
// - Multi-parent aggregation with 8 input sources
//
// The workflow showcases advanced MBFlow patterns:
// 1. Conditional routing with numeric thresholds
// 2. Loop handling with explicit attempt counter
// 3. Wave-based parallel execution
// 4. Structured LLM outputs with JSON schemas
// 5. Complex multi-parent node aggregation
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
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
	"github.com/smilemakc/mbflow/pkg/visualization"
)

func main() {
	// Check for OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("⚠️  OPENAI_API_KEY not set")
		log.Println("   Set it with: export OPENAI_API_KEY='your-key-here'")
		log.Println()
		log.Println("Continuing with workflow definition (execution will require API key)...")
		log.Println()
		apiKey = "sk-placeholder"
	}

	// Create standalone client (no database required)
	client, err := sdk.NewStandaloneClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Build the content generation workflow
	workflow := buildContentGenerationWorkflow(apiKey)

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  Multi-Language Content Generation Workflow")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("✓ Workflow: %s\n", workflow.Name)
	fmt.Printf("  Nodes: %d\n", len(workflow.Nodes))
	fmt.Printf("  Edges: %d\n", len(workflow.Edges))
	fmt.Println()

	lrOpts := &visualization.RenderOptions{
		UseColor:        true,
		ShowDescription: true,
		ShowConfig:      true,
		ShowConditions:  true,
		Direction:       "elk",
	}
	mermaidLR, err := visualization.RenderWorkflow(workflow, "mermaid", lrOpts)
	if err != nil {
		log.Fatalf("Failed to render workflow: %v", err)
	}
	fmt.Println(mermaidLR)
	fmt.Println()

	// Display workflow structure
	displayWorkflowStructure(workflow)

	// Check if API key is valid for execution
	if apiKey == "sk-placeholder" {
		fmt.Println()
		fmt.Println("⚠️  Skipping execution - valid OPENAI_API_KEY required")
		fmt.Println()
		showUsageExample()
		return
	}

	// Sample input
	input := map[string]interface{}{
		"topic":           "The Future of Artificial Intelligence in Healthcare",
		"content_type":    "blog_post",
		"target_length":   1500,
		"tone":            "professional",
		"target_audience": "healthcare professionals",
	}

	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("  INPUT")
	fmt.Println("─────────────────────────────────────────────────────────────")
	inputJSON, _ := json.MarshalIndent(input, "  ", "  ")
	fmt.Printf("  %s\n", string(inputJSON))
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	// Execute workflow
	fmt.Println("🚀 Executing workflow...")
	fmt.Println()

	startTime := time.Now()
	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, nil)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("❌ Execution failed: %v\n", err)
		if execution != nil {
			displayExecutionDetails(execution)
		}
		return
	}

	fmt.Printf("✅ Execution completed in %v\n", duration)
	fmt.Println()

	// Display results
	displayExecutionDetails(execution)
	displayFinalOutput(execution)

	// Display cost estimation
	displayCostEstimation(execution, duration)
}

// buildContentGenerationWorkflow creates the multi-language content generation workflow
func buildContentGenerationWorkflow(apiKey string) *models.Workflow {
	return builder.NewWorkflow("Multi-Language Content Generation",
		builder.WithDescription("Generate, analyze, translate, and optimize content with quality-based routing"),
		builder.WithVariable("openai_api_key", apiKey),
		builder.WithVariable("model_turbo", "gpt-4o"),
		builder.WithVariable("model_standard", "gpt-4o"),
		builder.WithVariable("max_regeneration_attempts", 3),
		builder.WithAutoLayout(),
		builder.WithTags("content-generation", "llm", "translation", "seo"),
	).
		// Step 1: Generate initial content
		AddNode(
			builder.NewOpenAINode(
				"generate",
				"Generate Initial Content",
				"{{env.model_turbo}}",
				contentGenerationPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.7),
				builder.LLMMaxTokens(2500),
			),
		).
		// Step 2: Analyze content quality
		AddNode(
			builder.NewOpenAINode(
				"analyze",
				"Analyze Content Quality",
				"{{env.model_turbo}}",
				qualityAnalysisPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.0),
				builder.LLMJSONMode(),
				builder.LLMMaxTokens(800),
			),
		).
		// Step 3: Enhance content (medium quality path)
		AddNode(
			builder.NewOpenAINode(
				"enhance",
				"Enhance Content",
				"{{env.model_turbo}}",
				enhancementPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.5),
				builder.LLMMaxTokens(2500),
			),
		).
		// Step 4: Regenerate content (low quality path)
		AddNode(
			builder.NewOpenAINode(
				"regenerate",
				"Regenerate Content",
				"{{env.model_turbo}}",
				regenerationPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.8),
				builder.LLMMaxTokens(2500),
			),
		).
		// Step 5: Merge node (consolidate quality paths)
		AddNode(
			builder.NewExpressionNode(
				"merge",
				"Merge Content Versions",
				`{content: (input.enhance.content ?? input.regenerate.content ?? input.generate.content)}`,
			),
		).
		// Step 6-8: Parallel translations
		AddNode(
			builder.NewOpenAINode(
				"trans_es",
				"Translate to Spanish",
				"{{env.model_standard}}",
				translationPromptES,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.3),
				builder.LLMMaxTokens(2500),
			),
		).
		AddNode(
			builder.NewOpenAINode(
				"trans_ru",
				"Translate to Russian",
				"{{env.model_standard}}",
				translationPromptRU,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.3),
				builder.LLMMaxTokens(2500),
			),
		).
		AddNode(
			builder.NewOpenAINode(
				"trans_de",
				"Translate to German",
				"{{env.model_standard}}",
				translationPromptDE,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.3),
				builder.LLMMaxTokens(2500),
			),
		).
		// Step 9-12: SEO generation for all languages
		AddNode(
			builder.NewOpenAINode(
				"seo_original",
				"Generate SEO Metadata (Original)",
				"{{env.model_turbo}}",
				seoGenerationPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.0),
				builder.LLMJSONMode(),
				builder.LLMMaxTokens(300),
				builder.WithConfigValue("language", "English"),
			),
		).
		AddNode(
			builder.NewOpenAINode(
				"seo_es",
				"Generate SEO Metadata (Spanish)",
				"{{env.model_turbo}}",
				seoGenerationPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.0),
				builder.LLMJSONMode(),
				builder.LLMMaxTokens(300),
				builder.WithConfigValue("language", "Spanish"),
			),
		).
		AddNode(
			builder.NewOpenAINode(
				"seo_ru",
				"Generate SEO Metadata (Russian)",
				"{{env.model_turbo}}",
				seoGenerationPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.0),
				builder.LLMJSONMode(),
				builder.LLMMaxTokens(300),
				builder.WithConfigValue("language", "Russian"),
			),
		).
		AddNode(
			builder.NewOpenAINode(
				"seo_de",
				"Generate SEO Metadata (German)",
				"{{env.model_turbo}}",
				seoGenerationPrompt,
				builder.LLMAPIKey("{{env.openai_api_key}}"),
				builder.LLMTemperature(0.0),
				builder.LLMJSONMode(),
				builder.LLMMaxTokens(300),
				builder.WithConfigValue("language", "German"),
			),
		).
		// Step 13: Aggregate all results
		AddNode(
			builder.NewJQNode(
				"aggregate",
				"Aggregate All Results",
				jqAggregationFilter,
			),
		).
		// Connect nodes - Linear flow
		Connect("generate", "analyze").
		// Quality-based routing (numeric thresholds)
		Connect("analyze", "merge", builder.WithCondition("output.score >= 80")).
		Connect("analyze", "enhance", builder.WithCondition("output.score >= 50 && output.score < 80")).
		Connect("analyze", "regenerate", builder.WithCondition("output.score < 50")).
		// Provide original content to all paths
		Connect("generate", "merge").
		Connect("generate", "enhance").
		Connect("generate", "regenerate").
		// Enhancement path
		Connect("enhance", "merge").
		// Regeneration path (no loop - single regeneration attempt)
		Connect("regenerate", "merge").
		// Parallel translations and SEO generation
		Connect("merge", "trans_es").
		Connect("merge", "trans_ru").
		Connect("merge", "trans_de").
		Connect("merge", "seo_original").
		// SEO for translations
		Connect("trans_es", "seo_es").
		Connect("trans_ru", "seo_ru").
		Connect("trans_de", "seo_de").
		// Final aggregation (8 parent inputs)
		Connect("merge", "aggregate").
		Connect("trans_es", "aggregate").
		Connect("trans_ru", "aggregate").
		Connect("trans_de", "aggregate").
		Connect("seo_original", "aggregate").
		Connect("seo_es", "aggregate").
		Connect("seo_ru", "aggregate").
		Connect("seo_de", "aggregate").
		MustBuild()
}

// displayWorkflowStructure shows the workflow nodes and edges
func displayWorkflowStructure(workflow *models.Workflow) {
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("  WORKFLOW STRUCTURE")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("Nodes:")
	for i, node := range workflow.Nodes {
		fmt.Printf("  %2d. %-30s (%s)\n", i+1, node.Name, node.Type)
	}
	fmt.Println()
	fmt.Println("Conditional Edges:")
	for _, edge := range workflow.Edges {
		if edge.Condition != "" {
			conditionPreview := edge.Condition
			if len(conditionPreview) > 40 {
				conditionPreview = conditionPreview[:40] + "..."
			}
			fmt.Printf("  %s → %s [%s]\n", edge.From, edge.To, conditionPreview)
		}
	}
	fmt.Println("─────────────────────────────────────────────────────────────")
}

// displayExecutionDetails shows node execution results
func displayExecutionDetails(execution *models.Execution) {
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("  EXECUTION DETAILS")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Printf("  Status: %s\n", execution.Status)
	fmt.Printf("  Duration: %dms\n", execution.Duration)
	fmt.Printf("  Nodes Executed: %d\n", len(execution.NodeExecutions))
	fmt.Println()

	// Show execution path
	fmt.Println("  Execution Path:")
	for i, nodeExec := range execution.NodeExecutions {
		status := "✓"
		if nodeExec.Status != models.NodeExecutionStatusCompleted {
			status = "✗"
		}
		fmt.Printf("  %s %2d. %-30s [%s, %dms]\n",
			status, i+1, nodeExec.NodeName, nodeExec.Status, nodeExec.Duration)

		if nodeExec.Error != "" {
			fmt.Printf("       Error: %s\n", nodeExec.Error)
		}
	}
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()
}

// displayFinalOutput shows the aggregated final output
func displayFinalOutput(execution *models.Execution) {
	if execution.Output == nil {
		fmt.Println("No output available")
		return
	}

	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("  FINAL OUTPUT")
	fmt.Println("─────────────────────────────────────────────────────────────")

	outputJSON, _ := json.MarshalIndent(execution.Output, "  ", "  ")
	fmt.Printf("  %s\n", string(outputJSON))

	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()
}

// displayCostEstimation estimates OpenAI API costs
func displayCostEstimation(execution *models.Execution, duration time.Duration) {
	// Rough cost estimation based on typical token usage
	// GPT-4-turbo: $0.01/1K input tokens, $0.03/1K output tokens
	// GPT-4: $0.03/1K input tokens, $0.06/1K output tokens

	llmNodeCount := 0
	for _, nodeExec := range execution.NodeExecutions {
		if nodeExec.Status == models.NodeExecutionStatusCompleted {
			// Count LLM nodes (all except check_attempts, merge, aggregate)
			if !strings.Contains(nodeExec.NodeID, "check_attempts") &&
				!strings.Contains(nodeExec.NodeID, "merge") &&
				!strings.Contains(nodeExec.NodeID, "aggregate") {
				llmNodeCount++
			}
		}
	}

	// Rough estimation
	estimatedCost := float64(llmNodeCount) * 0.035 // Average $0.035 per LLM call

	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("  COST ESTIMATION")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Printf("  LLM API Calls: %d\n", llmNodeCount)
	fmt.Printf("  Estimated Cost: $%.2f - $%.2f\n", estimatedCost*0.8, estimatedCost*1.2)
	fmt.Printf("  Execution Time: %v\n", duration)
	fmt.Println()
	fmt.Println("  Note: Actual costs vary based on token usage and model pricing.")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()
}

// showUsageExample displays usage instructions
func showUsageExample() {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  USAGE")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("To execute this workflow:")
	fmt.Println()
	fmt.Println("1. Set your OpenAI API key:")
	fmt.Println("   export OPENAI_API_KEY='sk-your-key-here'")
	fmt.Println()
	fmt.Println("2. Run the example:")
	fmt.Println("   go run backend/examples/content_generation/*.go")
	fmt.Println()
	fmt.Println("Expected execution time:")
	fmt.Println("  - High quality path: ~50-60 seconds")
	fmt.Println("  - Medium quality (with enhancement): ~70-80 seconds")
	fmt.Println("  - Low quality (with regeneration): ~90-120 seconds")
	fmt.Println()
	fmt.Println("Expected cost per execution:")
	fmt.Println("  - High quality path: $0.30-$0.40")
	fmt.Println("  - With enhancement/regeneration: $0.40-$0.50")
	fmt.Println()
	fmt.Println("See README.md for more details and configuration options.")
	fmt.Println("═══════════════════════════════════════════════════════════════")
}
