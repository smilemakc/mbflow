package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// AIContentPipelineDemo demonstrates a complex workflow with branching logic
// where OpenAI API is used to generate content, analyze it, and make decisions
// based on the analysis results. This example shows:
// 1. Complex workflow with conditional branching
// 2. Parallel execution of translation tasks
// 3. Aggregation of results from parallel branches
//
// Workflow structure:
// 1. Generate initial content using OpenAI
// 2. Analyze content quality (branching point)
//   - If quality is high -> Merge directly
//   - If quality is medium -> Enhance content -> Merge
//
// 3. Translate to multiple languages (parallel processing)
// 4. Generate SEO metadata for each language version
// 5. Aggregate all results
func main() {
	// Parse command line arguments
	topicFlag := flag.String("topic", "искусственный интеллект и машинное обучение", "Topic for the blog post")
	flag.Parse()

	topic := *topicFlag
	if topic == "" {
		topic = "искусственный интеллект и машинное обучение"
	}

	fmt.Printf("=== AI Content Pipeline Demo ===\n\n")
	fmt.Printf("Topic: %s\n\n", topic)

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: OPENAI_API_KEY environment variable is required for this demo.")
		fmt.Printf("Please set OPENAI_API_KEY to run this example.\n\n")
		os.Exit(1)
	}

	seoResponseSchema := map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"type": "object",
			"name": "SEO_Metadata",
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"maxLength":   60,
						"description": "SEO-optimized title (max 60 chars)",
					},
					"description": map[string]any{
						"type":        "string",
						"maxLength":   160,
						"description": "Meta description (max 160 chars)",
					},
					"keywords": map[string]any{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "List of keywords",
					},
					"slug": map[string]any{
						"type":        "string",
						"description": "url-friendly-slug",
					},
				},
				"required": []string{"title", "description", "keywords", "slug"},
			},
		},
	}
	// Build workflow using the new WorkflowBuilder API
	maxTokens := 200
	workflow, err := mbflow.NewWorkflowBuilder("AI Content Pipeline", "1.0").
		WithDescription("Demonstrates complex AI content generation with parallel processing").
		// 1. Generate initial content
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "generate_content", &mbflow.OpenAICompletionConfig{
			Model:       "gpt-4o",
			Prompt:      `Write a comprehensive blog post about "{{topic}}" using source language. Do not add any additional content, just write fully completed to posting text without editing befor.`,
			MaxTokens:   maxTokens,
			Temperature: 0.7,
		}).
		// 2. Analyze content quality
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "analyze_quality", &mbflow.OpenAICompletionConfig{
			Model: "gpt-4o",
			Prompt: `Analyze the following content and rate its quality as 'high', 'medium', or 'low'.
Consider clarity, engagement, accuracy, and structure.
Content: {{generate_content.content}}
Respond with ONLY one word in lowercase: high, medium, or low.`,
			MaxTokens:   10,
			Temperature: 0.1,
		}).
		// 3. Conditional router
		AddNode(string(mbflow.NodeTypeConditionalRoute), "quality_router", map[string]any{
			"routes": []map[string]any{
				{"name": "high", "condition": "analyze_quality.content == 'high'"},
				{"name": "medium", "condition": "analyze_quality.content == 'medium'"},
			},
			"default_route": "low",
		}).
		// 4. Enhance content (for medium quality)
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "enhance_content", &mbflow.OpenAICompletionConfig{
			Model: "gpt-4o",
			Prompt: `Improve the following content by:
1. Adding more specific examples
2. Improving transitions between paragraphs
3. Enhancing the conclusion
4. Adding relevant statistics or data points

Original content: {{generate_content_content}}

Provide the enhanced version:`,
			MaxTokens:   maxTokens,
			Temperature: 0.6,
		}).
		// 5. Select final content (prefer enhanced if available, otherwise use generated)
		AddNodeWithConfig(string(mbflow.NodeTypeDataAggregator), "select_final_content", &mbflow.DataAggregatorConfig{
			Fields: map[string]string{
				"final_content": "enhance_content.content",
				"fallback":      "generate_content.content",
			},
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeTransform), "extract_final_content", &mbflow.TransformConfig{
			Transformations: map[string]string{
				"final_content": "select_final_content.final_content != nil ? select_final_content.final_content : select_final_content.fallback",
			},
		}).
		// 6. Parallel translations
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "translate_spanish", &mbflow.OpenAICompletionConfig{
			Model:       "gpt-4o",
			Prompt:      "Translate the following content to Spanish, maintaining the tone and style:\n\n{{extract_final_content.final_content}}",
			MaxTokens:   maxTokens,
			Temperature: 0.3,
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "translate_russian", &mbflow.OpenAICompletionConfig{
			Model:       "gpt-4o",
			Prompt:      "Translate the following content to Russian, maintaining the tone and style:\n\n{{extract_final_content.final_content}}",
			MaxTokens:   maxTokens,
			Temperature: 0.3,
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "translate_german", &mbflow.OpenAICompletionConfig{
			Model:       "gpt-4o",
			Prompt:      "Translate the following content to German, maintaining the tone and style:\n\n{{extract_final_content.final_content}}",
			MaxTokens:   maxTokens,
			Temperature: 0.3,
		}).
		// 7. Generate SEO metadata for each language
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "seo", &mbflow.OpenAICompletionConfig{
			Model:          "gpt-4o",
			Prompt:         `Generate SEO metadata for the following content in JSON format with using source language: Content: {{extract_final_content.final_content}}`,
			ResponseFormat: seoResponseSchema,
			MaxTokens:      300,
			Temperature:    0.4,
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "seo_spanish", &mbflow.OpenAICompletionConfig{
			Model:          "gpt-4o",
			Prompt:         `Generate SEO metadata for the following Spanish content in JSON format:Content: {{translate_spanish.content}}`,
			ResponseFormat: seoResponseSchema,
			MaxTokens:      300,
			Temperature:    0.4,
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "seo_russian", &mbflow.OpenAICompletionConfig{
			Model:          "gpt-4o",
			Prompt:         `Generate SEO metadata for the following Russian content in JSON format: Content: {{translate_russian.content}}`,
			ResponseFormat: seoResponseSchema,
			MaxTokens:      300,
			Temperature:    0.4,
		}).
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "seo_german", &mbflow.OpenAICompletionConfig{
			Model:          "gpt-4o",
			Prompt:         `Generate SEO metadata for the following German content in JSON format: Content: {{translate_german.content}}`,
			ResponseFormat: seoResponseSchema,
			MaxTokens:      300,
			Temperature:    0.4,
		}).
		// 8. Aggregate all results
		AddNodeWithConfig(string(mbflow.NodeTypeDataAggregator), "aggregate_results", &mbflow.DataAggregatorConfig{
			Fields: map[string]string{
				"content":    "extract_final_content.final_content",
				"seo":        "seo.content",
				"content_es": "translate_spanish.content",
				"content_ru": "translate_russian.content",
				"content_de": "translate_german.content",
				"seo_es":     "seo_spanish.content",
				"seo_ru":     "seo_russian.content",
				"seo_de":     "seo_german.content",
			},
		}).
		AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
			"output_keys": []string{"aggregate_results_output"},
		}).
		// Define edges (workflow flow)
		AddEdge("generate_content", "analyze_quality", string(mbflow.EdgeTypeDirect), nil).
		AddEdgeWithDataSources("analyze_quality", "quality_router", string(mbflow.EdgeTypeDirect), []string{"generate_content"}).
		// Conditional branches
		AddEdge("quality_router", "enhance_content", string(mbflow.EdgeTypeConditional), map[string]any{
			"condition":            "quality_router_selected_route == 'medium'",
			"include_outputs_from": []string{"generate_content"},
		}).
		AddEdge("quality_router", "enhance_content", string(mbflow.EdgeTypeConditional), map[string]any{
			"condition":            "quality_router_selected_route == 'low'",
			"include_outputs_from": []string{"generate_content"},
		}).
		AddEdge("quality_router", "select_final_content", string(mbflow.EdgeTypeConditional), map[string]any{
			"condition":            "quality_router_selected_route == 'high'",
			"include_outputs_from": []string{"generate_content"},
		}).
		// Для веток medium/low узел select_final_content должен запускаться после enhance_content
		AddEdgeWithDataSources("enhance_content", "select_final_content", string(mbflow.EdgeTypeDirect), []string{
			"generate_content", // нужен fallback
			"enhance_content",  // нужен enhanced
		}).
		AddEdge("select_final_content", "extract_final_content", string(mbflow.EdgeTypeDirect), nil).
		// Parallel translations fork
		AddEdge("extract_final_content", "translate_spanish", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("extract_final_content", "translate_russian", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("extract_final_content", "translate_german", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("extract_final_content", "seo", string(mbflow.EdgeTypeDirect), nil).
		// SEO generation
		AddEdge("translate_spanish", "seo_spanish", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("translate_russian", "seo_russian", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("translate_german", "seo_german", string(mbflow.EdgeTypeDirect), nil).
		// Aggregate all results (join point)
		// Use include_outputs_from to access translation and final content nodes
		AddEdge("extract_final_content", "aggregate_results", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("translate_spanish", "aggregate_results", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("translate_russian", "aggregate_results", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("translate_german", "aggregate_results", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("seo", "aggregate_results", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("seo_spanish", "aggregate_results", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("seo_russian", "aggregate_results", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("seo_german", "aggregate_results", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("aggregate_results", "end", string(mbflow.EdgeTypeDirect), nil).
		// Add trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Start Content Pipeline",
		}).
		Build()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create workflow")
	}

	fmt.Printf("✓ Workflow created: %s (version %s)\n", workflow.Name(), workflow.Version())
	fmt.Printf("  Nodes: %d\n", len(workflow.GetAllNodes()))
	fmt.Printf("  Edges: %d\n\n", len(workflow.GetAllEdges()))
	// Create executor with parallel execution and monitoring
	executor := mbflow.NewExecutorBuilder().
		EnableParallelExecution(10).
		EnableRetry(2).
		EnableMetrics().
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("ai-pipeline"))).
		Build()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("	Executor Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✓ Parallel execution: enabled (max 10 nodes)")
	fmt.Println("✓ Retry: enabled (max 2 attempts)")
	fmt.Println("✓ Metrics: enabled")
	fmt.Println("✓ Monitoring: enabled")

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("	Workflow Graph Structure")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("1. Generate initial content on topic language (OpenAI)")
	fmt.Println("2. Analyze content quality (OpenAI)")

	fmt.Println("3. Route based on quality:")
	fmt.Println("     • High   → Select Final Content")
	fmt.Println("     • Medium → Enhance Content → Select Final Content")

	fmt.Println("4. Select and extract final content")

	fmt.Println("5. Parallel processing:")
	fmt.Println("     • Translate to: Spanish, Russian, German")
	fmt.Println("     • Generate SEO metadata for the source content")

	fmt.Println("6. Generate SEO for each translation")
	fmt.Println("7. Aggregate all results")

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Execute workflow
	ctx := context.Background()
	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal().Msg("No triggers found in workflow")
	}

	initialVars := map[string]any{
		"topic":          topic,
		"openai_api_key": apiKey,
	}

	fmt.Println("▶ Executing workflow...")
	execution, err := executor.ExecuteWorkflow(ctx, workflow, triggers[0], initialVars)
	if err != nil {
		log.Fatal().Err(err).Msg("Workflow execution failed")
	}

	fmt.Printf("\n✓ Workflow execution completed!\n")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n", execution.Phase())
	fmt.Printf("  Duration: %v\n\n", execution.Duration())

	// Display results
	vars := execution.Variables().All()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("	EXECUTION RESULTS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n	All Variables:")
	for key, value := range vars {
		if key == "openai_api_key" {
			continue
		}
		valueStr := fmt.Sprintf("%v", value)
		fmt.Printf("  %s: [%d...]\n", key, len(valueStr))
	}

	PrintAggregated(vars)

	fmt.Println("\n=== Demo Complete ===")
}

type AggregatedResult struct {
	Content   string         `json:"content"`
	Seo       map[string]any `json:"seo"`
	ContentEs string         `json:"content_es"`
	ContentRu string         `json:"content_ru"`
	ContentDe string         `json:"content_de"`
	SeoEs     map[string]any `json:"seo_es"`
	SeoRu     map[string]any `json:"seo_ru"`
	SeoDe     map[string]any `json:"seo_de"`
}

// prettyJSON formats maps into readable indented JSON.
// Comments in English only.
func prettyJSON(m map[string]any) string {
	if m == nil {
		return "{}"
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// printSection prints both text and JSON blocks nicely.
func printSection(title, content string, seo map[string]any) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("		%s CONTENT\n", title)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%v\n\n", content)
	fmt.Printf("		%s SEO\n", title)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(prettyJSON(seo))
}

func PrintAggregated(vars map[string]any) {
	finalOutput, ok := vars["aggregate_results"]
	if !ok {
		return
	}

	var result AggregatedResult
	data, _ := json.Marshal(finalOutput)
	_ = json.Unmarshal(data, &result)

	printSection("ORIGINAL", result.Content, result.Seo)

	printSection("SPANISH", result.ContentEs, result.SeoEs)

	printSection("GERMAN", result.ContentDe, result.SeoDe)
}
