// OpenAI Vision Example
//
// This example demonstrates how to:
// 1. Fetch an image from URL using HTTP node
// 2. Pass the image (as base64) to OpenAI GPT-4o for vision analysis
//
// Usage:
//   export OPENAI_API_KEY=sk-...
//   go run main.go
//
// The pipeline:
//   HTTP (fetch image) → LLM (analyze with GPT-4o vision)

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/smilemakc/mbflow/go/pkg/builder"
	"github.com/smilemakc/mbflow/go/pkg/executor/builtin"
)

func main() {
	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create workflow using builder
	wf := builder.NewWorkflow("image-analysis", builder.WithDescription("Analyze image with GPT-4o Vision")).
		AddNode(
			builder.NewNode("fetch_image", "http", "Fetch Image",
				builder.WithConfigValue("method", "GET"),
				builder.WithConfigValue("url", "https://httpbin.org/image/jpeg"),
			),
		).
		AddNode(
			builder.NewNode("analyze_image", "llm", "Analyze Image",
				builder.WithConfigValue("provider", "openai"),
				builder.WithConfigValue("model", "gpt-4o"),
				builder.WithConfigValue("api_key", apiKey),
				builder.WithConfigValue("prompt", "Describe this image in detail. What do you see?"),
				builder.WithConfigValue("max_tokens", 500),
				builder.WithConfigValue("files", []map[string]any{
					{
						"data":      "{{input.body_base64}}",
						"mime_type": "{{input.content_type}}",
						"name":      "image.jpg",
					},
				}),
			),
		).
		Connect("fetch_image", "analyze_image").
		MustBuild()

	// Print workflow structure
	fmt.Println("=== Workflow Structure ===")
	fmt.Printf("Name: %s\n", wf.Name)
	fmt.Printf("Description: %s\n", wf.Description)
	fmt.Printf("Nodes: %d\n", len(wf.Nodes))
	fmt.Printf("Edges: %d\n", len(wf.Edges))
	fmt.Println()

	// Create executors
	httpExec := builtin.NewHTTPExecutor()
	llmExec := builtin.NewLLMExecutor()

	ctx := context.Background()

	// Step 1: Execute HTTP node to fetch image
	fmt.Println("=== Step 1: Fetching image from httpbin.org ===")
	httpResult, err := httpExec.Execute(ctx, wf.Nodes[0].Config, nil)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}

	httpOutput := httpResult.(map[string]any)
	fmt.Printf("Status: %v\n", httpOutput["status"])
	fmt.Printf("Content-Type: %v\n", httpOutput["content_type"])
	fmt.Printf("Image size: %v bytes\n", httpOutput["size"])
	fmt.Printf("Has base64: %v\n", httpOutput["body_base64"] != nil && httpOutput["body_base64"] != "")
	fmt.Println()

	// Step 2: Prepare LLM config with resolved templates
	fmt.Println("=== Step 2: Analyzing image with GPT-4o Vision ===")

	// Manually resolve templates (in real workflow, engine does this automatically)
	llmConfig := map[string]any{
		"provider":   "openai",
		"model":      "gpt-4o",
		"api_key":    apiKey,
		"prompt":     "Describe this image in detail. What do you see?",
		"max_tokens": 500,
		"files": []any{
			map[string]any{
				"data":      httpOutput["body_base64"],
				"mime_type": httpOutput["content_type"],
				"name":      "image.jpg",
			},
		},
	}

	llmResult, err := llmExec.Execute(ctx, llmConfig, httpOutput)
	if err != nil {
		log.Fatalf("LLM request failed: %v", err)
	}

	llmOutput := llmResult.(map[string]any)
	fmt.Println()
	fmt.Println("=== GPT-4o Vision Response ===")
	fmt.Println(llmOutput["content"])
	fmt.Println()

	// Print usage stats
	if usage, ok := llmOutput["usage"].(map[string]any); ok {
		fmt.Println("=== Token Usage ===")
		fmt.Printf("Prompt tokens: %v\n", usage["prompt_tokens"])
		fmt.Printf("Completion tokens: %v\n", usage["completion_tokens"])
		fmt.Printf("Total tokens: %v\n", usage["total_tokens"])
	}

	// Save workflow to JSON for reference
	wfJSON, _ := json.MarshalIndent(wf, "", "  ")
	os.WriteFile("workflow.json", wfJSON, 0644)
	fmt.Println("\nWorkflow saved to workflow.json")
}
