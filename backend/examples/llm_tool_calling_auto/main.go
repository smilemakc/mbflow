// Example demonstrating LLM Tool Calling with Auto Mode
// This example shows how LLM can automatically call functions in a loop using built-in functions
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/pkg/builder"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
	"github.com/smilemakc/mbflow/pkg/visualization"
)

func main() {
	// Check for OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: OPENAI_API_KEY not set - workflow will fail during execution")
		log.Println("Set it with: export OPENAI_API_KEY='your-key-here'")
		log.Println()
	}

	// Create a standalone client (no database required)
	client, err := sdk.NewStandaloneClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create workflow with LLM that has tool calling capabilities
	workflow := builder.NewWorkflow("LLM Tool Calling - Auto Mode",
		builder.WithDescription("Demonstrates automatic tool calling with built-in functions"),
		builder.WithVariable("openai_api_key", apiKey),
		builder.WithVariable("model", "gpt-4"),
		builder.WithAutoLayout(),
	).AddNode(
		// LLM node with auto mode tool calling enabled
		builder.NewNode(
			"assistant",
			"AI Assistant with Tools",
			"llm",
			builder.WithConfig(map[string]interface{}{
				"provider": "openai",
				"model":    "{{env.model}}",
				"api_key":  "{{env.openai_api_key}}",
				"prompt":   "{{input.user_message}}",
				"instruction": `You are a helpful assistant with access to tools.
When the user asks about time or calculations, use the available tools to provide accurate information.
Always explain what you're doing when you use a tool.`,
				"temperature": 0.7,
				"max_tokens":  1000,

				// Tool calling configuration (auto mode)
				"tool_call_config": map[string]interface{}{
					"mode":                 "auto",
					"max_iterations":       5,
					"timeout_per_tool":     30,
					"total_timeout":        300,
					"stop_on_tool_failure": false,
				},

				// Function definitions
				"functions": []map[string]interface{}{
					{
						"type":         "builtin",
						"name":         "get_current_time",
						"description":  "Get the current date and time",
						"builtin_name": "get_current_time",
						"parameters": map[string]interface{}{
							"type":       "object",
							"properties": map[string]interface{}{},
						},
					},
					{
						"type":         "builtin",
						"name":         "calculate",
						"description":  "Perform mathematical calculations. Supports basic arithmetic operations.",
						"builtin_name": "calculate",
						"parameters": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"expression": map[string]interface{}{
									"type":        "string",
									"description": "Mathematical expression to evaluate (e.g., '2 + 2', '10 * 5 + 3')",
								},
							},
							"required": []string{"expression"},
						},
					},
					{
						"type":         "builtin",
						"name":         "get_weather",
						"description":  "Get weather information for a location",
						"builtin_name": "get_weather",
						"parameters": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"location": map[string]interface{}{
									"type":        "string",
									"description": "City name or location",
								},
							},
							"required": []string{"location"},
						},
					},
				},
			}),
		),
	).MustBuild()

	fmt.Printf("✓ Workflow defined: %s\n\n", workflow.Name)

	// Display workflow structure
	lrOpts := &visualization.RenderOptions{
		ShowConfig:     false,
		ShowConditions: true,
		Direction:      "TB",
	}
	err = visualization.PrintWorkflow(workflow, "mermaid", lrOpts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	showToolCallingInfo()

	// If no API key, show what would happen and exit
	if apiKey == "" {
		fmt.Println("⚠️  Skipping execution - no OPENAI_API_KEY set")
		fmt.Println()
		showKeyFeatures()
		return
	}

	// Test with different queries that require tool calls
	testQueries := []string{
		"What time is it right now?",
		"What's the weather like in London?",
		"Calculate 157 * 23 + 456",
		"What time is it and what's the weather in Paris?",
	}

	for i, query := range testQueries {
		fmt.Printf("\n╔════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║ Query %d: %-46s║\n", i+1, truncate(query, 46))
		fmt.Printf("╚════════════════════════════════════════════════════════╝\n\n")

		// Execute workflow in standalone mode
		input := map[string]interface{}{
			"user_message": query,
		}

		opts := &engine.ExecutionOptions{
			StrictMode:     false,
			MaxParallelism: 1,
			Variables:      make(map[string]interface{}),
		}

		execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, opts)
		if err != nil {
			log.Printf("❌ Execution failed: %v\n", err)
			continue
		}

		showExecutionResult(execution, query)

		// Add delay between queries
		if i < len(testQueries)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	showKeyFeatures()
}

func showToolCallingInfo() {
	fmt.Println("🔧 TOOL CALLING CONFIGURATION:")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("Mode:              auto (automatic tool execution loop)")
	fmt.Println("Max Iterations:    5 (prevents infinite loops)")
	fmt.Println("Timeout Per Tool:  30s")
	fmt.Println("Total Timeout:     300s (5 minutes)")
	fmt.Println("Stop on Failure:   false (continue even if tool fails)")
	fmt.Println()
	fmt.Println("Available Functions:")
	fmt.Println("  1. get_current_time  - Get current date and time")
	fmt.Println("  2. calculate         - Perform math calculations")
	fmt.Println("  3. get_weather       - Get weather for a location")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()
}

func showExecutionResult(execution *models.Execution, query string) {
	if execution == nil || len(execution.NodeExecutions) == 0 {
		return
	}

	nodeExec := execution.NodeExecutions[0]

	if nodeExec.Status == models.NodeExecutionStatusCompleted {
		fmt.Printf("✅ Status: %s\n", nodeExec.Status)
	} else {
		fmt.Printf("❌ Status: %s\n", nodeExec.Status)
		if nodeExec.Error != "" {
			fmt.Printf("   Error: %s\n", nodeExec.Error)
		}
		return
	}

	// Show final response
	if content, ok := nodeExec.Output["content"].(string); ok {
		fmt.Printf("\n💬 Assistant Response:\n")
		fmt.Printf("─────────────────────────────────────────────────────────\n")
		fmt.Println(content)
		fmt.Printf("─────────────────────────────────────────────────────────\n")
	}

	// Show tool calling statistics
	if totalIterations, ok := nodeExec.Output["total_iterations"].(int); ok {
		fmt.Printf("\n📊 Tool Calling Statistics:\n")
		fmt.Printf("   Total Iterations: %d\n", totalIterations)

		if stoppedReason, ok := nodeExec.Output["stopped_reason"].(string); ok {
			fmt.Printf("   Stopped Reason: %s\n", stoppedReason)
		}

		// Show tool executions
		if toolExecsRaw, ok := nodeExec.Output["tool_executions"].([]interface{}); ok && len(toolExecsRaw) > 0 {
			fmt.Printf("\n🔧 Tool Executions (%d):\n", len(toolExecsRaw))
			for i, execRaw := range toolExecsRaw {
				if execMap, ok := execRaw.(map[string]interface{}); ok {
					functionName := execMap["function_name"].(string)
					executionTime := execMap["execution_time"].(int64)

					fmt.Printf("   %d. %s (%dms)\n", i+1, functionName, executionTime)

					if result := execMap["result"]; result != nil {
						resultJSON, _ := json.Marshal(result)
						resultStr := string(resultJSON)
						if len(resultStr) > 80 {
							fmt.Printf("      Result: %s...\n", resultStr[:80])
						} else {
							fmt.Printf("      Result: %s\n", resultStr)
						}
					}

					if errorMsg, ok := execMap["error"].(string); ok && errorMsg != "" {
						fmt.Printf("      Error: %s\n", errorMsg)
					}
				}
			}
		}

		// Show message history summary
		if messagesRaw, ok := nodeExec.Output["messages"].([]interface{}); ok {
			fmt.Printf("\n💬 Message History (%d messages):\n", len(messagesRaw))
			for i, msgRaw := range messagesRaw {
				if msgMap, ok := msgRaw.(map[string]interface{}); ok {
					role := msgMap["role"].(string)
					content := ""
					if c, ok := msgMap["content"].(string); ok {
						content = c
					}

					switch role {
					case "system":
						fmt.Printf("   %d. [SYSTEM] %s\n", i+1, truncate(content, 60))
					case "user":
						fmt.Printf("   %d. [USER] %s\n", i+1, truncate(content, 60))
					case "assistant":
						if toolCalls, ok := msgMap["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
							fmt.Printf("   %d. [ASSISTANT] Called %d tool(s)\n", i+1, len(toolCalls))
						} else {
							fmt.Printf("   %d. [ASSISTANT] %s\n", i+1, truncate(content, 60))
						}
					case "tool":
						toolName := msgMap["name"].(string)
						fmt.Printf("   %d. [TOOL: %s] %s\n", i+1, toolName, truncate(content, 40))
					}
				}
			}
		}
	}

	fmt.Printf("\n⏱️  Execution Duration: %dms\n", execution.Duration)
	fmt.Println()
}

func showKeyFeatures() {
	fmt.Println("🎯 KEY FEATURES DEMONSTRATED:")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("✓ Auto Mode Tool Calling")
	fmt.Println("  - LLM automatically decides when to call tools")
	fmt.Println("  - Automatic loop: LLM → Tool Call → Tool Result → LLM")
	fmt.Println("  - Stops when LLM provides final answer (finish_reason: stop)")
	fmt.Println()
	fmt.Println("✓ Built-in Function Support")
	fmt.Println("  - get_current_time: Returns current date and time")
	fmt.Println("  - calculate: Performs mathematical calculations")
	fmt.Println("  - get_weather: Returns weather information (mock)")
	fmt.Println()
	fmt.Println("✓ Automatic History Management")
	fmt.Println("  - All messages automatically accumulated")
	fmt.Println("  - Tool calls and results preserved")
	fmt.Println("  - Complete conversation context maintained")
	fmt.Println()
	fmt.Println("✓ Configurable Limits")
	fmt.Println("  - Max iterations prevents infinite loops")
	fmt.Println("  - Timeout per tool and total timeout")
	fmt.Println("  - Optional stop on tool failure")
	fmt.Println()
	fmt.Println("✓ Execution Transparency")
	fmt.Println("  - Tool execution results tracked")
	fmt.Println("  - Message history available")
	fmt.Println("  - Iteration count and stopped reason")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()

	fmt.Println("📚 NEXT STEPS:")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("Phase 2: Sub-Workflow Functions")
	fmt.Println("  - Call other workflows as functions")
	fmt.Println("  - Input mapping and output extraction")
	fmt.Println()
	fmt.Println("Phase 3: Custom Code Functions")
	fmt.Println("  - Execute JavaScript/Python code")
	fmt.Println("  - Sandboxed execution environment")
	fmt.Println()
	fmt.Println("Phase 4: OpenAPI Functions")
	fmt.Println("  - Dynamic function generation from OpenAPI specs")
	fmt.Println("  - Authentication and rate limiting")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
