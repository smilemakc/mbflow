package main

import (
	"context"
	"fmt"
	"log"

	"mbflow"

	"github.com/google/uuid"
)

// AIContentPipelineWorkflow demonstrates a complex workflow with branching logic
// where OpenAI API is used to generate content, analyze it, and make decisions
// based on the analysis results.
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
	storage := mbflow.NewMemoryStorage()
	ctx := context.Background()

	// Create the workflow
	workflowID := uuid.NewString()
	workflow := mbflow.NewWorkflow(
		workflowID,
		"AI Content Pipeline with Branching",
		"1.0.0",
		[]byte(`{
			"description": "Complex AI content generation pipeline with quality checks and branching logic",
			"features": ["branching", "parallel_processing", "iterative_refinement"]
		}`),
	)

	if err := storage.SaveWorkflow(ctx, workflow); err != nil {
		log.Fatalf("Failed to save workflow: %v", err)
	}

	fmt.Printf("Created workflow: %s (ID: %s)\n\n", workflow.Name(), workflow.ID())

	// Node 1: Generate initial content using OpenAI
	nodeGenerateContent := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Initial Content",
		map[string]any{
			"model":       "gpt-4",
			"prompt":      "Write a comprehensive blog post about {{topic}}",
			"max_tokens":  2000,
			"temperature": 0.7,
			"output_key":  "generated_content",
		},
	)

	// Node 2: Analyze content quality using OpenAI
	nodeAnalyzeQuality := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Analyze Content Quality",
		map[string]any{
			"model":       "gpt-4",
			"prompt":      "Analyze the following content and rate its quality as 'high', 'medium', or 'low'. Consider clarity, engagement, accuracy, and structure.\n\nContent: {{generated_content}}\n\nRespond with ONLY one word: high, medium, or low.",
			"max_tokens":  10,
			"temperature": 0.1,
			"output_key":  "quality_rating",
		},
	)

	// Node 3: Quality-based router (decision node)
	nodeQualityRouter := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"conditional-router",
		"Route Based on Quality",
		map[string]any{
			"input_key": "quality_rating",
			"routes": map[string]string{
				"high":   "publish",
				"medium": "enhance",
				"low":    "regenerate",
			},
		},
	)

	// Node 4: Enhance content (for medium quality)
	nodeEnhanceContent := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Enhance Content",
		map[string]any{
			"model": "gpt-4",
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
	)

	// Node 5: Regenerate content (for low quality)
	nodeRegenerateContent := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Regenerate Content",
		map[string]any{
			"model": "gpt-4",
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
	)

	// Node 6: Merge content (combines different paths)
	nodeMergeContent := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"data-merger",
		"Merge Content Versions",
		map[string]any{
			"strategy":   "select_first_available",
			"sources":    []string{"generated_content", "enhanced_content", "regenerated_content"},
			"output_key": "final_content",
		},
	)

	// Node 7: Translate to Spanish (parallel branch 1)
	nodeTranslateSpanish := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Translate to Spanish",
		map[string]any{
			"model":       "gpt-4",
			"prompt":      "Translate the following content to Spanish, maintaining the tone and style:\n\n{{final_content}}",
			"max_tokens":  2500,
			"temperature": 0.3,
			"output_key":  "content_es",
		},
	)

	// Node 8: Translate to French (parallel branch 2)
	nodeTranslateFrench := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Translate to French",
		map[string]any{
			"model":       "gpt-4",
			"prompt":      "Translate the following content to French, maintaining the tone and style:\n\n{{final_content}}",
			"max_tokens":  2500,
			"temperature": 0.3,
			"output_key":  "content_fr",
		},
	)

	// Node 9: Translate to German (parallel branch 3)
	nodeTranslateGerman := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Translate to German",
		map[string]any{
			"model":       "gpt-4",
			"prompt":      "Translate the following content to German, maintaining the tone and style:\n\n{{final_content}}",
			"max_tokens":  2500,
			"temperature": 0.3,
			"output_key":  "content_de",
		},
	)

	// Node 10: Generate SEO metadata for English
	nodeGenerateSEOEnglish := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate SEO Metadata (EN)",
		map[string]any{
			"model": "gpt-4",
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
	)

	// Node 11: Generate SEO metadata for Spanish
	nodeGenerateSEOSpanish := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate SEO Metadata (ES)",
		map[string]any{
			"model": "gpt-4",
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
	)

	// Node 12: Generate SEO metadata for French
	nodeGenerateSEOFrench := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate SEO Metadata (FR)",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Generate SEO metadata for the following French content in JSON format:
{
  "title": "SEO-optimized title (max 60 chars)",
  "description": "Meta description (max 160 chars)",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "slug": "url-friendly-slug"
}

Content: {{content_fr}}`,
			"max_tokens":  300,
			"temperature": 0.4,
			"output_key":  "seo_fr",
		},
	)

	// Node 13: Generate SEO metadata for German
	nodeGenerateSEOGerman := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate SEO Metadata (DE)",
		map[string]any{
			"model": "gpt-4",
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
	)

	// Node 14: Aggregate all results
	nodeAggregateResults := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"data-aggregator",
		"Aggregate All Results",
		map[string]any{
			"output_format": "json",
			"fields": map[string]string{
				"content_en": "final_content",
				"content_es": "content_es",
				"content_fr": "content_fr",
				"content_de": "content_de",
				"seo_en":     "seo_en",
				"seo_es":     "seo_es",
				"seo_fr":     "seo_fr",
				"seo_de":     "seo_de",
			},
			"output_key": "final_output",
		},
	)

	// Node 15: Publish to CMS
	nodePublish := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Publish to CMS",
		map[string]any{
			"url":    "https://cms.example.com/api/publish",
			"method": "POST",
			"body":   "{{final_output}}",
			"headers": map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer {{api_token}}",
			},
		},
	)

	// Save all nodes
	nodes := []mbflow.Node{
		nodeGenerateContent, nodeAnalyzeQuality, nodeQualityRouter,
		nodeEnhanceContent, nodeRegenerateContent, nodeMergeContent,
		nodeTranslateSpanish, nodeTranslateFrench, nodeTranslateGerman,
		nodeGenerateSEOEnglish, nodeGenerateSEOSpanish, nodeGenerateSEOFrench, nodeGenerateSEOGerman,
		nodeAggregateResults, nodePublish,
	}

	for _, node := range nodes {
		if err := storage.SaveNode(ctx, node); err != nil {
			log.Fatalf("Failed to save node %s: %v", node.Name(), err)
		}
	}

	// Create edges (workflow connections)
	edges := []struct {
		from     mbflow.Node
		to       mbflow.Node
		edgeType string
		config   map[string]any
	}{
		// Main flow
		{nodeGenerateContent, nodeAnalyzeQuality, "direct", nil},
		{nodeAnalyzeQuality, nodeQualityRouter, "direct", nil},

		// Branching based on quality
		{nodeQualityRouter, nodeMergeContent, "conditional", map[string]any{"condition": "quality_rating == 'high'"}},
		{nodeQualityRouter, nodeEnhanceContent, "conditional", map[string]any{"condition": "quality_rating == 'medium'"}},
		{nodeQualityRouter, nodeRegenerateContent, "conditional", map[string]any{"condition": "quality_rating == 'low'"}},

		// Merge paths
		{nodeEnhanceContent, nodeMergeContent, "direct", nil},
		{nodeRegenerateContent, nodeAnalyzeQuality, "direct", map[string]any{"retry": true}}, // Loop back for re-analysis

		// Parallel translation branches
		{nodeMergeContent, nodeTranslateSpanish, "parallel", nil},
		{nodeMergeContent, nodeTranslateFrench, "parallel", nil},
		{nodeMergeContent, nodeTranslateGerman, "parallel", nil},
		{nodeMergeContent, nodeGenerateSEOEnglish, "parallel", nil},

		// SEO generation (depends on translations)
		{nodeTranslateSpanish, nodeGenerateSEOSpanish, "direct", nil},
		{nodeTranslateFrench, nodeGenerateSEOFrench, "direct", nil},
		{nodeTranslateGerman, nodeGenerateSEOGerman, "direct", nil},

		// Aggregate results (wait for all parallel branches)
		{nodeGenerateSEOEnglish, nodeAggregateResults, "join", nil},
		{nodeGenerateSEOSpanish, nodeAggregateResults, "join", nil},
		{nodeGenerateSEOFrench, nodeAggregateResults, "join", nil},
		{nodeGenerateSEOGerman, nodeAggregateResults, "join", nil},

		// Final publish
		{nodeAggregateResults, nodePublish, "direct", nil},
	}

	for i, e := range edges {
		config := e.config
		if config == nil {
			config = map[string]any{}
		}

		edge := mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			e.from.ID(),
			e.to.ID(),
			e.edgeType,
			config,
		)

		if err := storage.SaveEdge(ctx, edge); err != nil {
			log.Fatalf("Failed to save edge %d: %v", i, err)
		}
	}

	// Create trigger
	trigger := mbflow.NewTrigger(
		uuid.NewString(),
		workflowID,
		"http",
		map[string]any{
			"path":   "/api/content/generate",
			"method": "POST",
			"schema": map[string]any{
				"topic": "string",
			},
		},
	)

	if err := storage.SaveTrigger(ctx, trigger); err != nil {
		log.Fatalf("Failed to save trigger: %v", err)
	}

	// Print workflow summary
	fmt.Println("=== Workflow Summary ===")
	fmt.Printf("Workflow: %s\n", workflow.Name())
	fmt.Printf("Nodes: %d\n", len(nodes))
	fmt.Printf("Edges: %d\n\n", len(edges))

	fmt.Println("=== Workflow Structure ===")
	fmt.Println("1. Generate Initial Content (OpenAI)")
	fmt.Println("2. Analyze Content Quality (OpenAI)")
	fmt.Println("3. Route Based on Quality:")
	fmt.Println("   - High Quality → Merge → Continue")
	fmt.Println("   - Medium Quality → Enhance Content → Merge → Continue")
	fmt.Println("   - Low Quality → Regenerate → Re-analyze (loop)")
	fmt.Println("4. Parallel Processing:")
	fmt.Println("   - Translate to Spanish, French, German")
	fmt.Println("   - Generate SEO metadata for English")
	fmt.Println("5. Generate SEO for each translation")
	fmt.Println("6. Aggregate all results")
	fmt.Println("7. Publish to CMS")

	fmt.Println("\n=== Trigger Configuration ===")
	fmt.Println("Type: HTTP POST")
	fmt.Println("Path: /api/content/generate")
	fmt.Println("Input: { \"topic\": \"your topic here\" }")

	// List all nodes
	savedNodes, err := storage.ListNodes(ctx, workflowID)
	if err != nil {
		log.Fatalf("Failed to list nodes: %v", err)
	}

	fmt.Printf("\n=== All Nodes (%d) ===\n", len(savedNodes))
	for i, n := range savedNodes {
		fmt.Printf("%d. %s (%s)\n", i+1, n.Name(), n.Type())
	}

	// List all edges
	savedEdges, err := storage.ListEdges(ctx, workflowID)
	if err != nil {
		log.Fatalf("Failed to list edges: %v", err)
	}

	fmt.Printf("\n=== All Edges (%d) ===\n", len(savedEdges))
	for i, e := range savedEdges {
		fromNode, _ := storage.GetNode(ctx, e.FromNodeID())
		toNode, _ := storage.GetNode(ctx, e.ToNodeID())
		fmt.Printf("%d. %s → %s (%s)\n", i+1, fromNode.Name(), toNode.Name(), e.Type())
	}
}
