package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"mbflow"

	"github.com/google/uuid"
)

// AIContentPipelineDemo demonstrates a complex workflow with branching logic
// where OpenAI API is used to generate content, analyze it, and make decisions
// based on the analysis results. This example shows:
// 1. Complex workflow with conditional branching
// 2. Parallel execution of translation tasks
// 3. Join node aggregating results from parallel branches
// 4. Graph-based execution with edges
//
// Workflow structure:
// 1. Generate initial content using OpenAI
// 2. Analyze content quality (branching point)
//   - If quality is high -> Publish directly
//   - If quality is medium -> Enhance content -> Publish
//   - If quality is low -> Regenerate with different prompt -> Analyze again
//
// 3. Translate to multiple languages (parallel processing)
// 4. Generate SEO metadata for each language version
func main() {
	// Parse command line arguments
	topicFlag := flag.String("topic", "искусственный интеллект и машинное обучение", "Topic for the blog post")
	flag.Parse()

	topic := *topicFlag
	if topic == "" {
		topic = "искусственный интеллект и машинное обучение"
	}

	fmt.Println("=== AI Content Pipeline Demo ===\n")
	fmt.Printf("Topic: %s\n\n", topic)

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: OPENAI_API_KEY environment variable is required for this demo.")
		fmt.Println("Please set OPENAI_API_KEY to run this example.\n")
		os.Exit(1)
	}
	httpObserver, err := mbflow.NewHTTPCallbackObserver(mbflow.HTTPCallbackConfig{
		CallbackURL: "https://heabot.nl.tuna.am",
	})
	// Create executor with monitoring enabled
	executor := mbflow.NewExecutor(&mbflow.ExecutorConfig{
		OpenAIAPIKey:     apiKey,
		MaxRetryAttempts: 3,
		EnableMonitoring: true,
		VerboseLogging:   true,
	})
	executor.AddObserver(httpObserver)
	// Create workflow and execution IDs
	workflowID := uuid.NewString()
	executionID := uuid.NewString()

	fmt.Printf("Workflow ID: %s\n", workflowID)
	fmt.Printf("Execution ID: %s\n\n", executionID)

	// Create domain nodes (for demonstration of workflow structure)
	// Node 1: Generate initial content using OpenAI
	nodeGenerateContent, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Generate Initial Content",
		Config: map[string]any{
			"model":       "gpt-4o",
			"prompt":      "Write a comprehensive blog post about {{topic}} with using source language",
			"max_tokens":  2000,
			"temperature": 0.7,
			"output_key":  "generated_content",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateContent: %v", err)
	}

	// Node 2: Analyze content quality using OpenAI
	nodeAnalyzeQuality, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Analyze Content Quality",
		Config: map[string]any{
			"model":       "gpt-4o",
			"prompt":      "Analyze the following content and rate its quality as 'high', 'medium', or 'low'. Consider clarity, engagement, accuracy, and structure.\n\nContent: {{generated_content}}\n\nRespond with ONLY one word in lowercase: high, medium, or low.",
			"max_tokens":  10,
			"temperature": 0.1,
			"output_key":  "quality_rating",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeAnalyzeQuality: %v", err)
	}

	// Node 3: Quality-based router (decision node)
	nodeQualityRouter, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "conditional-router",
		Name:       "Route Based on Quality",
		Config: map[string]any{
			"input_key": "quality_rating",
			"routes": map[string]string{
				"high":   "publish",
				"medium": "enhance",
				"low":    "regenerate",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeQualityRouter: %v", err)
	}

	// Node 4: Enhance content (for medium quality)
	nodeEnhanceContent, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Enhance Content",
		Config: map[string]any{
			"model": "gpt-4o",
			"prompt": `Improve the following content by:
1. Adding more specific examples
2. Improving transitions between paragraphs
3. Enhancing the conclusion
4. Adding relevant statistics or data points

Original content: {{generated_content}}

Provide the enhanced version:`,
			"max_tokens":  2500,
			"temperature": 0.6,
			"output_key":  "enhanced_content",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeEnhanceContent: %v", err)
	}

	// Node 5: Regenerate content (for low quality)
	nodeRegenerateContent, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Regenerate Content",
		Config: map[string]any{
			"model": "gpt-4o",
			"prompt": `Write a high-quality, engaging blog post about {{topic}}.
Requirements:
- Clear structure with introduction, body, and conclusion
- Use specific examples and data
- Engaging writing style
- Professional tone
- 800-1000 words`,
			"max_tokens":  2500,
			"temperature": 0.8,
			"output_key":  "regenerated_content",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeRegenerateContent: %v", err)
	}

	// Node 6: Merge content (combines different paths)
	nodeMergeContent, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "data-merger",
		Name:       "Merge Content Versions",
		Config: map[string]any{
			"strategy":   "select_first_available",
			"sources":    []string{"generated_content", "enhanced_content", "regenerated_content"},
			"output_key": "final_content",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeMergeContent: %v", err)
	}

	// Node 7: Translate to Spanish (parallel branch 1)
	nodeTranslateSpanish, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Translate to Spanish",
		Config: map[string]any{
			"model":       "gpt-4o",
			"prompt":      "Translate the following content to Spanish, maintaining the tone and style:\n\n{{final_content}}",
			"max_tokens":  2500,
			"temperature": 0.3,
			"output_key":  "content_es",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeTranslateSpanish: %v", err)
	}

	// Node 8: Translate to Russian (parallel branch 2)
	nodeTranslateRussian, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Translate to Russian",
		Config: map[string]any{
			"model":       "gpt-4o",
			"prompt":      "Translate the following content to Russian, maintaining the tone and style:\n\n{{final_content}}",
			"max_tokens":  2500,
			"temperature": 0.3,
			"output_key":  "content_ru",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeTranslateRussian: %v", err)
	}

	// Node 9: Translate to German (parallel branch 3)
	nodeTranslateGerman, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Translate to German",
		Config: map[string]any{
			"model":       "gpt-4o",
			"prompt":      "Translate the following content to German, maintaining the tone and style:\n\n{{final_content}}",
			"max_tokens":  2500,
			"temperature": 0.3,
			"output_key":  "content_de",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeTranslateGerman: %v", err)
	}

	// Node 10: Generate SEO metadata for English
	nodeGenerateSEOEnglish, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Generate SEO Metadata (EN)",
		Config: map[string]any{
			"model": "gpt-4o",
			"prompt": `Generate SEO metadata for the following content in JSON format:
{
  "title": "SEO-optimized title (max 60 chars)",
  "description": "Meta description (max 160 chars)",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "slug": "url-friendly-slug"
}

Content: {{final_content}}`,
			"max_tokens":  300,
			"temperature": 0.4,
			"output_key":  "seo_en",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateSEOEnglish: %v", err)
	}

	// Node 11: Generate SEO metadata for Spanish
	nodeGenerateSEOSpanish, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Generate SEO Metadata (ES)",
		Config: map[string]any{
			"model": "gpt-4o",
			"prompt": `Generate SEO metadata for the following Spanish content in JSON format:
{
  "title": "SEO-optimized title (max 60 chars)",
  "description": "Meta description (max 160 chars)",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "slug": "url-friendly-slug"
}

Content: {{content_es}}`,
			"max_tokens":  300,
			"temperature": 0.4,
			"output_key":  "seo_es",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateSEOSpanish: %v", err)
	}

	// Node 12: Generate SEO metadata for Russian
	nodeGenerateSEORussian, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Generate SEO Metadata (RU)",
		Config: map[string]any{
			"model": "gpt-4o",
			"prompt": `Generate SEO metadata for the following Russian content in JSON format:
{
  "title": "SEO-optimized title (max 60 chars)",
  "description": "Meta description (max 160 chars)",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "slug": "url-friendly-slug"
}

Content: {{content_ru}}`,
			"max_tokens":  300,
			"temperature": 0.4,
			"output_key":  "seo_ru",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateSEORussian: %v", err)
	}

	// Node 13: Generate SEO metadata for German
	nodeGenerateSEOGerman, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Generate SEO Metadata (DE)",
		Config: map[string]any{
			"model": "gpt-4o",
			"prompt": `Generate SEO metadata for the following German content in JSON format:
{
  "title": "SEO-optimized title (max 60 chars)",
  "description": "Meta description (max 160 chars)",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "slug": "url-friendly-slug"
}

Content: {{content_de}}`,
			"max_tokens":  300,
			"temperature": 0.4,
			"output_key":  "seo_de",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateSEOGerman: %v", err)
	}

	// Node 14: Aggregate all results
	nodeAggregateResults, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "data-aggregator",
		Name:       "Aggregate All Results",
		Config: map[string]any{
			"output_format": "json",
			"fields": map[string]string{
				"content":    "final_content",
				"content_es": "content_es",
				"content_ru": "content_ru",
				"content_de": "content_de",
				"seo_en":     "seo_en",
				"seo_es":     "seo_es",
				"seo_ru":     "seo_ru",
				"seo_de":     "seo_de",
			},
			"output_key": "final_output",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeAggregateResults: %v", err)
	}

	// Collect all nodes
	nodes := []mbflow.Node{
		nodeGenerateContent, nodeAnalyzeQuality, nodeQualityRouter,
		nodeEnhanceContent, nodeRegenerateContent, nodeMergeContent,
		nodeTranslateSpanish, nodeTranslateRussian, nodeTranslateGerman,
		nodeGenerateSEOEnglish, nodeGenerateSEOSpanish, nodeGenerateSEORussian, nodeGenerateSEOGerman,
		nodeAggregateResults,
	}

	// Create edges (workflow connections)
	edges := []mbflow.Edge{
		// Main flow
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeGenerateContent.ID(),
			nodeAnalyzeQuality.ID(),
			"direct",
			nil,
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeAnalyzeQuality.ID(),
			nodeQualityRouter.ID(),
			"direct",
			nil,
		),

		// Branching based on quality - conditional edges
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeQualityRouter.ID(),
			nodeMergeContent.ID(),
			"conditional",
			map[string]any{"condition": "quality_rating == 'high'"},
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeQualityRouter.ID(),
			nodeEnhanceContent.ID(),
			"conditional",
			map[string]any{"condition": "quality_rating == 'medium'"},
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeQualityRouter.ID(),
			nodeRegenerateContent.ID(),
			"conditional",
			map[string]any{"condition": "quality_rating == 'low'"},
		),

		// Merge paths
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeEnhanceContent.ID(),
			nodeMergeContent.ID(),
			"direct",
			nil,
		),

		// Parallel translation branches
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeMergeContent.ID(),
			nodeTranslateSpanish.ID(),
			"direct",
			nil,
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeMergeContent.ID(),
			nodeTranslateRussian.ID(),
			"direct",
			nil,
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeMergeContent.ID(),
			nodeTranslateGerman.ID(),
			"direct",
			nil,
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeMergeContent.ID(),
			nodeGenerateSEOEnglish.ID(),
			"direct",
			nil,
		),

		// SEO generation (depends on translations)
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeTranslateSpanish.ID(),
			nodeGenerateSEOSpanish.ID(),
			"direct",
			nil,
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeTranslateRussian.ID(),
			nodeGenerateSEORussian.ID(),
			"direct",
			nil,
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeTranslateGerman.ID(),
			nodeGenerateSEOGerman.ID(),
			"direct",
			nil,
		),

		// Aggregate results (wait for all parallel branches)
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeGenerateSEOEnglish.ID(),
			nodeAggregateResults.ID(),
			"direct",
			nil,
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeGenerateSEOSpanish.ID(),
			nodeAggregateResults.ID(),
			"direct",
			nil,
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeGenerateSEORussian.ID(),
			nodeAggregateResults.ID(),
			"direct",
			nil,
		),
		mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			nodeGenerateSEOGerman.ID(),
			nodeAggregateResults.ID(),
			"direct",
			nil,
		),
	}

	// Convert domain nodes and edges to execution configs using helper functions
	nodeConfigs := mbflow.NodesToConfigs(nodes)
	edgeConfigs := mbflow.EdgesToConfigs(edges)

	// Set initial variables
	initialVariables := map[string]interface{}{
		"topic": topic,
	}

	fmt.Println("=== Workflow Graph Structure ===")
	fmt.Println("1. Generate Initial Content (OpenAI)")
	fmt.Println("2. Analyze Content Quality (OpenAI)")
	fmt.Println("3. Route Based on Quality:")
	fmt.Println("   - High Quality → Merge → Continue")
	fmt.Println("   - Medium Quality → Enhance Content → Merge → Continue")
	fmt.Println("   - Low Quality → Regenerate")
	fmt.Println("4. Parallel Processing:")
	fmt.Println("   - Translate to Spanish, Russian, German")
	fmt.Println("   - Generate SEO metadata for English")
	fmt.Println("5. Generate SEO for each translation")
	fmt.Println("6. Aggregate all results")
	fmt.Println()

	fmt.Printf("Nodes: %d\n", len(nodeConfigs))
	fmt.Printf("Edges: %d\n\n", len(edgeConfigs))

	fmt.Println("=== Executing Workflow ===\n")
	startTime := time.Now()

	// Execute workflow with edges for parallel execution
	ctx := context.Background()
	state, err := executor.ExecuteWorkflow(ctx, workflowID, executionID, nodeConfigs, edgeConfigs, initialVariables)

	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	executionDuration := time.Since(startTime)

	fmt.Println("\n=== Execution Results ===\n")
	fmt.Printf("Status: %s\n", state.Status())
	fmt.Printf("Execution Duration: %s\n", executionDuration)
	fmt.Printf("State Duration: %s\n\n", state.GetExecutionDuration())

	// Get all variables
	variables := state.GetAllVariables()

	// Display detailed content and SEO results
	fmt.Println("\n=== DETAILED CONTENT AND SEO RESULTS ===\n")

	// Extract final output
	var finalOutput map[string]interface{}
	if fo, ok := variables["final_output"]; ok {
		if resultMap, ok := fo.(map[string]interface{}); ok {
			finalOutput = resultMap
		}
	}

	// Helper function to safely get string value
	getStringValue := func(key string) string {
		if finalOutput != nil {
			if val, ok := finalOutput[key]; ok {
				return fmt.Sprintf("%v", val)
			}
		}
		if val, ok := variables[key]; ok {
			return fmt.Sprintf("%v", val)
		}
		return ""
	}

	// Display English Content
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📝 ORIGINAL CONTENT")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	contentEn := getStringValue("content")
	if contentEn == "" {
		contentEn = getStringValue("final_content")
	}
	if contentEn != "" {
		fmt.Printf("\n%s\n", contentEn)
		fmt.Printf("\n[Length: %d characters]\n", len(contentEn))
	} else {
		fmt.Println("\n[Content not available]")
	}
	fmt.Println()

	// Display English SEO
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔍 ENGLISH SEO METADATA")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	seoEn := getStringValue("seo_en")
	if seoEn != "" {
		fmt.Printf("\n%s\n", seoEn)
	} else {
		fmt.Println("\n[SEO metadata not available]")
	}
	fmt.Println()

	// Display Spanish Content
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📝 SPANISH CONTENT (ESPAÑOL)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	contentEs := getStringValue("content_es")
	if contentEs != "" {
		fmt.Printf("\n%s\n", contentEs)
		fmt.Printf("\n[Length: %d characters]\n", len(contentEs))
	} else {
		fmt.Println("\n[Content not available]")
	}
	fmt.Println()

	// Display Spanish SEO
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔍 SPANISH SEO METADATA (ESPAÑOL)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	seoEs := getStringValue("seo_es")
	if seoEs != "" {
		fmt.Printf("\n%s\n", seoEs)
	} else {
		fmt.Println("\n[SEO metadata not available]")
	}
	fmt.Println()

	// Display Russian Content
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📝 RUSSIAN CONTENT (РУССКИЙ)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	contentRu := getStringValue("content_ru")
	if contentRu != "" {
		fmt.Printf("\n%s\n", contentRu)
		fmt.Printf("\n[Length: %d characters]\n", len(contentRu))
	} else {
		fmt.Println("\n[Content not available]")
	}
	fmt.Println()

	// Display Russian SEO
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔍 RUSSIAN SEO METADATA (РУССКИЙ)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	seoRu := getStringValue("seo_ru")
	if seoRu != "" {
		fmt.Printf("\n%s\n", seoRu)
	} else {
		fmt.Println("\n[SEO metadata not available]")
	}
	fmt.Println()

	// Display German Content
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📝 GERMAN CONTENT (DEUTSCH)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	contentDe := getStringValue("content_de")
	if contentDe != "" {
		fmt.Printf("\n%s\n", contentDe)
		fmt.Printf("\n[Length: %d characters]\n", len(contentDe))
	} else {
		fmt.Println("\n[Content not available]")
	}
	fmt.Println()

	// Display German SEO
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔍 GERMAN SEO METADATA (DEUTSCH)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	seoDe := getStringValue("seo_de")
	if seoDe != "" {
		fmt.Printf("\n%s\n", seoDe)
	} else {
		fmt.Println("\n[SEO metadata not available]")
	}
	fmt.Println()

	// Display summary of all available variables
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 ALL EXECUTION VARIABLES SUMMARY")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for key, value := range variables {
		if key != "final_output" {
			valueStr := fmt.Sprintf("%v", value)
			if len(valueStr) > 200 {
				fmt.Printf("  %s: [%d characters]\n", key, len(valueStr))
			} else {
				fmt.Printf("  %s: %s\n", key, valueStr)
			}
		}
	}
	fmt.Println()

	// Display metrics
	fmt.Println("\n=== Execution Metrics ===\n")
	metrics := executor.GetMetrics()

	summary := metrics.GetSummary()
	fmt.Println("Summary:")
	for key, value := range summary {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Display workflow metrics
	fmt.Println("\nWorkflow Metrics:")
	workflowMetrics := metrics.GetWorkflowMetrics(workflowID)
	if workflowMetrics != nil {
		for key, value := range workflowMetrics {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Display node metrics
	fmt.Println("\nNode Type Metrics:")
	nodeTypes := []string{"openai-completion", "data-merger", "data-aggregator", "conditional-router"}

	for _, nodeType := range nodeTypes {
		nodeMetrics := metrics.GetNodeMetrics(nodeType)
		if nodeMetrics != nil {
			fmt.Printf("\n  %s:\n", nodeType)
			for key, value := range nodeMetrics {
				fmt.Printf("    %s: %v\n", key, value)
			}
		}
	}

	// Display AI metrics
	fmt.Println("\nAI API Metrics:")
	aiMetrics := metrics.GetAIMetrics()
	if aiMetrics != nil {
		for key, value := range aiMetrics {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nNote: This workflow demonstrates:")
	fmt.Println("- Conditional branching based on content quality")
	fmt.Println("- Parallel execution of translation tasks")
	fmt.Println("- Join node aggregating results from parallel branches")
}
