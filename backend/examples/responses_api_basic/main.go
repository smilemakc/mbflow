package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smilemakc/mbflow/pkg/builder"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/executor/builtin"
	"github.com/smilemakc/mbflow/pkg/models"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Build workflow using Responses API
	workflow, err := builder.NewWorkflow("responses-demo").
		AddNode(
			builder.NewOpenAIResponsesNode(
				"story",
				"llm",
				"Story Generator",
				"gpt-4.1",
				"Tell me a three sentence bedtime story about a unicorn.",
				builder.WithInstructions("You are a creative storyteller."),
				builder.WithConfigValue("api_key", apiKey),
				builder.WithConfigValue("max_tokens", 200),
			),
		).
		Build()

	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	fmt.Println("=== OpenAI Responses API Basic Example ===")
	fmt.Printf("Workflow: %s\n", workflow.Name)
	fmt.Printf("Node: %s (provider: %s, model: %s)\n\n",
		workflow.Nodes[0].Name,
		workflow.Nodes[0].Config["provider"],
		workflow.Nodes[0].Config["model"])

	// Execute workflow
	fmt.Println("Executing workflow...")
	result, err := executeWorkflow(workflow)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	// Display results
	fmt.Println("\n=== Results ===")
	if content, ok := result["content"].(string); ok {
		fmt.Printf("\nStory:\n%s\n", content)
	}

	if usage, ok := result["usage"].(map[string]any); ok {
		fmt.Printf("\nUsage:\n")
		fmt.Printf("  Prompt tokens: %v\n", usage["prompt_tokens"])
		fmt.Printf("  Completion tokens: %v\n", usage["completion_tokens"])
		fmt.Printf("  Total tokens: %v\n", usage["total_tokens"])
	}

	if status, ok := result["status"].(string); ok {
		fmt.Printf("\nStatus: %s\n", status)
	}

	// Show output items (Responses API specific)
	if outputItems, ok := result["output_items"].([]map[string]any); ok && len(outputItems) > 0 {
		fmt.Printf("\nOutput Items: %d\n", len(outputItems))
		for i, item := range outputItems {
			fmt.Printf("  [%d] Type: %s, Status: %s\n", i, item["type"], item["status"])
		}
	}

	if reasoning, ok := result["reasoning"].(map[string]any); ok {
		fmt.Printf("\nReasoning:\n")
		if effort, ok := reasoning["effort"].(string); ok && effort != "" {
			fmt.Printf("  Effort: %s\n", effort)
		}
	}
}

// executeWorkflow executes a workflow with a single node
func executeWorkflow(workflow *models.Workflow) (map[string]any, error) {
	// Create executor manager and register LLM executor
	executorManager := executor.NewManager()
	if err := executorManager.Register("llm", builtin.NewLLMExecutor()); err != nil {
		return nil, fmt.Errorf("failed to register executor: %w", err)
	}

	// Get the node
	if len(workflow.Nodes) == 0 {
		return nil, fmt.Errorf("workflow has no nodes")
	}
	node := workflow.Nodes[0]

	// Get executor for the node
	exec, err := executorManager.Get(node.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor: %w", err)
	}

	// Validate config
	if err := exec.Validate(node.Config); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Execute
	result, err := exec.Execute(context.Background(), node.Config, nil)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Convert to map
	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	return resultMap, nil
}
