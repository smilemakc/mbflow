package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// OpenAIResponsesDemo demonstrates using OpenAI Responses API for structured JSON output.
// This example shows:
// 1. Using response_format to get structured JSON responses
// 2. Defining JSON schema for output validation
// 3. Configuring various OpenAI parameters (temperature, top_p, etc.)
// 4. Parsing and using structured JSON data in workflow
//
// Workflow structure:
// 1. Extract structured product information from text
// 2. Generate product recommendation based on structured data
func main() {
	// Parse command line arguments
	productDescFlag := flag.String("description", "High-performance laptop with 16GB RAM, Intel i7 processor, 512GB SSD, 15.6 inch display", "Product description")
	flag.Parse()

	productDesc := *productDescFlag
	if productDesc == "" {
		productDesc = "High-performance laptop with 16GB RAM, Intel i7 processor, 512GB SSD, 15.6 inch display"
	}

	fmt.Printf("=== OpenAI Responses API Demo ===\n\n")
	fmt.Printf("Product Description: %s\n\n", productDesc)

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: OPENAI_API_KEY environment variable is required for this demo.")
		fmt.Printf("Please set OPENAI_API_KEY to run this example.\n\n")
		os.Exit(1)
	}

	// Create executor with monitoring enabled
	workflow, err := mbflow.NewWorkflowBuilder("OpenAIResponsesDemo", "1.0.0").
		AddNode(string(mbflow.NodeTypeOpenAICompletion), "extract_info", map[string]any{
			"model":  "gpt-4o",
			"prompt": "Extract structured product information from the following description: {{product_description}}",
			"response_format": map[string]interface{}{
				"type": "json_schema",
				"json_schema": map[string]interface{}{
					"name": "product_info",
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Product name",
							},
							"category": map[string]interface{}{
								"type":        "string",
								"description": "Product category",
							},
							"specifications": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"processor": map[string]interface{}{
										"type": "string",
									},
									"ram": map[string]interface{}{
										"type": "string",
									},
									"storage": map[string]interface{}{
										"type": "string",
									},
									"display": map[string]interface{}{
										"type": "string",
									},
								},
								"additionalProperties": false,
								"required":             []string{"processor", "ram", "storage", "display"},
							},
							"price_range": map[string]interface{}{
								"type":        "string",
								"description": "Estimated price range",
							},
							"target_audience": map[string]interface{}{
								"type":        "string",
								"description": "Target audience for this product",
							},
						},
						"required":             []string{"name", "category", "specifications", "price_range", "target_audience"},
						"additionalProperties": false,
					},
					"strict": true,
				},
			},
			"temperature": 0.3,
			"output_key":  "product_info",
		}).
		AddNode(string(mbflow.NodeTypeOpenAICompletion), "generate_recommendation", map[string]any{
			"model":  "gpt-4o",
			"prompt": "Based on the following product information, generate a personalized recommendation: {{content}}",
			"response_format": map[string]interface{}{
				"type": "json_schema",
				"json_schema": map[string]interface{}{
					"name": "recommendation",
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"recommendation_text": map[string]interface{}{
								"type":        "string",
								"description": "Personalized recommendation text",
							},
							"pros": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "string",
								},
								"description": "List of product advantages",
							},
							"cons": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "string",
								},
								"description": "List of product disadvantages",
							},
							"rating": map[string]interface{}{
								"type":        "number",
								"description": "Overall rating from 1 to 10",
								"minimum":     1,
								"maximum":     10,
							},
							"best_for": map[string]interface{}{
								"type":        "string",
								"description": "What this product is best suited for",
							},
						},
						"required":             []string{"recommendation_text", "pros", "cons", "rating", "best_for"},
						"additionalProperties": false,
					},
					"strict": true,
				},
			},
			"temperature":       0.7,
			"top_p":             0.9,
			"frequency_penalty": 0.3,
			"presence_penalty":  0.2,
			"output_key":        "recommendation",
		}).
		AddEdge("extract_info", "generate_recommendation", string(mbflow.EdgeTypeDirect), nil).
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Start Content Pipeline",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}
	// Set initial variables
	initialVars := map[string]interface{}{
		"product_description": productDesc,
		"openai_api_key":      apiKey,
	}

	executor := mbflow.NewExecutorBuilder().
		EnableParallelExecution(10).
		EnableRetry(2).
		EnableMetrics().
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("ai-pipeline"))).
		Build()

	// Execute workflow
	ctx := context.Background()
	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal("No triggers found in workflow")
	}

	fmt.Println("▶ Executing workflow...")
	execution, err := executor.ExecuteWorkflow(ctx, workflow, triggers[0], initialVars)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}
	fmt.Printf("\n✓ Workflow execution completed!\n")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n", execution.Phase())
	fmt.Printf("  Duration: %v\n\n", execution.Duration())

	// Display results
	vars := execution.Variables().All()

	log.Printf("vars: %+v\n", vars["content"])

	fmt.Println("\n=== Demo Completed Successfully ===")
}
