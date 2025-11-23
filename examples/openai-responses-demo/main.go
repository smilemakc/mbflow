package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"mbflow"

	"github.com/google/uuid"
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

	fmt.Println("=== OpenAI Responses API Demo ===\n")
	fmt.Printf("Product Description: %s\n\n", productDesc)

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: OPENAI_API_KEY environment variable is required for this demo.")
		fmt.Println("Please set OPENAI_API_KEY to run this example.\n")
		os.Exit(1)
	}

	// Create executor with monitoring enabled
	executor := mbflow.NewWorkflowEngine(&mbflow.EngineConfig{
		OpenAIAPIKey:     apiKey,
		EnableMonitoring: true,
		VerboseLogging:   true,
	})

	// Create workflow and execution IDs
	workflowID := uuid.NewString()
	executionID := uuid.NewString()

	fmt.Printf("Workflow ID: %s\n", workflowID)
	fmt.Printf("Execution ID: %s\n\n", executionID)

	// Node 1: Extract structured product information using OpenAI Responses API
	nodeExtractProduct, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAIResponses,
		Name:       "Extract Product Information",
		Config: map[string]any{
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
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeExtractProduct: %v", err)
	}

	// Node 2: Generate recommendation using structured data
	nodeGenerateRecommendation, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAIResponses,
		Name:       "Generate Product Recommendation",
		Config: map[string]any{
			"model":  "gpt-4o",
			"prompt": "Based on the following product information, generate a personalized recommendation: {{product_info}}",
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
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateRecommendation: %v", err)
	}

	// Convert domain nodes to executor node configs using helper function
	nodes := []mbflow.NodeConfig{
		mbflow.NodeToConfig(nodeExtractProduct),
		mbflow.NodeToConfig(nodeGenerateRecommendation),
	}

	// Define edges: extract -> generate
	edges := []mbflow.ExecutorEdgeConfig{
		{
			FromNodeID: nodeExtractProduct.ID(),
			ToNodeID:   nodeGenerateRecommendation.ID(),
			EdgeType:   "direct",
		},
	}

	// Set initial variables
	initialVars := map[string]interface{}{
		"product_description": productDesc,
	}

	// Execute workflow
	fmt.Println("Starting workflow execution...\n")
	startTime := time.Now()

	state, err := executor.ExecuteWorkflow(
		context.Background(),
		workflowID,
		executionID,
		nodes,
		edges,
		initialVars,
	)

	duration := time.Since(startTime)

	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Print results
	fmt.Println("\n=== Execution Results ===\n")
	fmt.Printf("Status: %s\n", state.GetStatusString())
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("State Duration: %s\n\n", state.GetExecutionDuration())

	// Get all variables
	variables := state.GetAllVariables()

	// Print extracted product information
	if productInfo, ok := variables["product_info"]; ok {
		fmt.Println("📦 Extracted Product Information:")
		prettyJSON, _ := json.MarshalIndent(productInfo, "", "  ")
		fmt.Println(string(prettyJSON))
		fmt.Println()
	}

	// Print recommendation
	if recommendation, ok := variables["recommendation"]; ok {
		fmt.Println("💡 Product Recommendation:")
		prettyJSON, _ := json.MarshalIndent(recommendation, "", "  ")
		fmt.Println(string(prettyJSON))
		fmt.Println()
	}
	var nodeIDs []string
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}
	mbflow.DisplayMetrics(executor.GetMetrics(), workflowID, nodeIDs, true)

	fmt.Println("\n=== Demo Completed Successfully ===")
}
