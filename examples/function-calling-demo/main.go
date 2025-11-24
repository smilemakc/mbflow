package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// FunctionCallingDemo demonstrates OpenAI function calling with MBFlow
// This example shows:
// 1. Defining functions that OpenAI can call
// 2. Processing function call requests from OpenAI
// 3. Executing functions with script handlers
func main() {
	fmt.Println("=== MBFlow Function Calling Demo ===")

	// Get OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: OPENAI_API_KEY environment variable is required.")
		os.Exit(1)
	}

	// Define a function that OpenAI can call
	getTemperatureTool := mbflow.OpenAITool{
		Type: "function",
		Function: mbflow.OpenAIFunction{
			Name:        "get_temperature",
			Description: "Get the current temperature for a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city and country, e.g. London, UK",
					},
					"unit": map[string]interface{}{
						"type": "string",
						"enum": []string{"celsius", "fahrenheit"},
					},
				},
				"required": []string{"location"},
			},
		},
	}

	// Create workflow with function calling and conversation continuation
	workflow, err := mbflow.NewWorkflowBuilder("Function Calling Demo", "1.0").
		WithDescription("Demonstrates OpenAI function calling with conversation continuation").
		// OpenAI completion with function calling
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAICompletion), "ask_ai", &mbflow.OpenAICompletionConfig{
			Model:      "gpt-4o",
			Prompt:     "What's the weather like in {{city}}?",
			MaxTokens:  150,
			Tools:      []mbflow.OpenAITool{getTemperatureTool},
			ToolChoice: "auto",
		}).
		// Execute the function call
		AddNodeWithConfig(string(mbflow.NodeTypeFunctionCall), "execute_function", &mbflow.FunctionCallConfig{
			InputKey:      "ask_ai_output",
			AiResponseKey: "completion_response",
			Handler:       "script",
			HandlerConfig: map[string]interface{}{
				"script": `{
					"temperature": 22,
					"unit": "celsius",
					"location": location,
					"forecast": "Sunny with light clouds",
					"humidity": 65,
					"wind_speed": 12
				}`,
			},
		}).
		// Continue conversation with function result
		AddNodeWithConfig(string(mbflow.NodeTypeOpenAIFunctionResult), "continue_conversation", &mbflow.OpenAIFunctionResponseConfig{
			AIResponseKey:     "ask_ai_output",
			FunctionResultKey: "execute_function_output",
			Model:             "gpt-4o",
			MaxTokens:         300,
		}).
		// End node
		AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
			"output_keys": []string{"ai_response", "function_result", "final_response"},
		}).
		// Connect nodes
		AddEdge("ask_ai", "execute_function", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("execute_function", "continue_conversation", string(mbflow.EdgeTypeDirect), nil).
		AddEdge("continue_conversation", "end", string(mbflow.EdgeTypeDirect), nil).
		// Add trigger
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
			"name": "Start Function Call Demo",
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("✓ Workflow created: %s\n", workflow.Name())
	fmt.Printf("  Nodes: %d\n", len(workflow.GetAllNodes()))
	fmt.Println()

	// Create executor
	executor := mbflow.NewExecutorBuilder().
		WithObserver(monitoring.NewLogObserver(monitoring.NewDefaultConsoleLogger("function-calling-example"))).
		EnableMetrics().
		Build()

	fmt.Println("✓ Executor created")

	// Execute workflow
	ctx := context.Background()
	triggers := workflow.GetAllTriggers()

	initialVars := map[string]any{
		"city":           "San Francisco, CA",
		"openai_api_key": apiKey,
	}

	fmt.Println("▶ Executing workflow...")
	fmt.Printf("  Question: What's the weather like in %s?\n\n", initialVars["city"])

	execution, err := executor.ExecuteWorkflow(ctx, workflow, triggers[0], initialVars)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Printf("\n✓ Workflow execution completed!\n")
	fmt.Printf("  Execution ID: %s\n", execution.ID())
	fmt.Printf("  Phase: %s\n\n", execution.Phase())

	// Display results
	vars := execution.Variables().All()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 RESULTS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if aiResponse, ok := vars["ask_ai_output"]; ok {
		fmt.Println("\n🤖 Initial AI Response (with tool_calls):")
		if respMap, ok := aiResponse.(map[string]any); ok {
			if toolCalls, ok := respMap["tool_calls"]; ok {
				fmt.Printf("  Tool Calls: %+v\n", toolCalls)
			}
		}
	}

	if toolCall, ok := vars["execute_function_output"]; ok {
		fmt.Println("\n Tool call output:")
		if respMap, ok := toolCall.(map[string]any); ok {
			if functionName, ok := respMap["function_name"]; ok {
				fmt.Printf("  Function Name: %s\n", functionName)
			}
			if content, ok := respMap["arguments"]; ok {
				fmt.Printf("  Arguments: %+v\n", content)
			}
			if result, ok := respMap["result"]; ok {
				fmt.Printf("  Result: %+v\n", result)
			}
		}
	}

	if funcResult, ok := vars["continue_conversation_output"]; ok {
		fmt.Println("\n⚙️  Function Execution Result:")
		if resultMap, ok := funcResult.(map[string]any); ok {
			if result, ok := resultMap["content"]; ok {
				fmt.Printf("  %+v\n", result)
			}
		}
	}

	if finalResponse, ok := vars["final_response"]; ok {
		fmt.Println("\n💬 Final AI Response (after function execution):")
		fmt.Printf("  %v\n", finalResponse)
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nThis demo demonstrates the complete OpenAI function calling flow:")
	fmt.Println("1. AI receives a question and decides to call a function")
	fmt.Println("2. Function is executed with extracted parameters")
	fmt.Println("3. Result is sent back to AI")
	fmt.Println("4. AI formulates a natural language response using the function result")
}
