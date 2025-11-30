// Example demonstrating LLM executor with input templates
// This example shows how to chain LLM calls using {{input.X}} template syntax
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
)

func main() {
	// Check for OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: OPENAI_API_KEY not set - workflow will fail during execution")
		log.Println("Set it with: export OPENAI_API_KEY='your-key-here'")
		log.Println()
	}

	// Create an embedded client (in-memory mode)
	client, err := sdk.NewClient(
		sdk.WithEmbeddedMode("memory://", "memory://"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a multi-step code analysis workflow with input templates
	workflow := &models.Workflow{
		Name:        "Code Analysis with Input Templates",
		Description: "Demonstrates chaining LLM calls using {{input.X}} templates",
		Variables: map[string]interface{}{
			"openai_api_key": apiKey,
			"model":          "gpt-4",
		},
		Nodes: []*models.Node{
			// Step 1: Extract programming language from code
			{
				ID:   "detect_language",
				Name: "Detect Programming Language",
				Type: "llm",
				Config: map[string]interface{}{
					"provider":    "openai",
					"model":       "{{env.model}}",
					"api_key":     "{{env.openai_api_key}}",
					"prompt":      "Identify the programming language of this code. Reply with ONLY the language name:\n\n{{input.code}}",
					"temperature": 0.0,
					"max_tokens":  50,
				},
			},
			// Step 2: Analyze code for issues using detected language
			{
				ID:   "analyze_code",
				Name: "Analyze Code Quality",
				Type: "llm",
				Config: map[string]interface{}{
					"provider":    "openai",
					"model":       "{{env.model}}",
					"api_key":     "{{env.openai_api_key}}",
					"instruction": "You are an expert {{input.content}} developer and code reviewer.",
					"prompt": `Analyze this {{input.content}} code for potential issues:

{{input.code}}

Focus on:
1. Security vulnerabilities
2. Performance issues
3. Code style and best practices
4. Potential bugs

Provide specific recommendations.`,
					"temperature": 0.2,
					"max_tokens":  800,
				},
			},
			// Step 3: Generate refactored version based on analysis
			{
				ID:   "refactor_code",
				Name: "Generate Refactored Code",
				Type: "llm",
				Config: map[string]interface{}{
					"provider":    "openai",
					"model":       "{{env.model}}",
					"api_key":     "{{env.openai_api_key}}",
					"instruction": "You are a code refactoring expert.",
					"prompt": `Based on this code review:

{{input.content}}

Refactor the code to address all issues. Provide ONLY the refactored code without explanations.`,
					"temperature": 0.1,
					"max_tokens":  1000,
				},
			},
			// Step 4: Explain the changes made
			{
				ID:   "explain_changes",
				Name: "Explain Refactoring",
				Type: "llm",
				Config: map[string]interface{}{
					"provider": "openai",
					"model":    "gpt-3.5-turbo", // Use faster model for explanation
					"api_key":  "{{env.openai_api_key}}",
					"prompt": `Explain the changes made in this code refactoring:

ORIGINAL ANALYSIS:
{{input.analysis}}

REFACTORED CODE:
{{input.content}}

Provide a clear explanation suitable for junior developers.`,
					"temperature": 0.5,
					"max_tokens":  500,
				},
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "edge-1",
				From: "detect_language",
				To:   "analyze_code",
			},
			{
				ID:   "edge-2",
				From: "analyze_code",
				To:   "refactor_code",
			},
			{
				ID:   "edge-3",
				From: "refactor_code",
				To:   "explain_changes",
			},
		},
	}

	fmt.Printf("✓ Workflow defined: %s\n\n", workflow.Name)

	// Display workflow structure
	fmt.Println("📋 WORKFLOW STRUCTURE:")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Printf("Nodes: %d\n", len(workflow.Nodes))
	for i, node := range workflow.Nodes {
		fmt.Printf("  %d. %s (%s)\n", i+1, node.Name, node.ID)
		if prompt, ok := node.Config["prompt"].(string); ok {
			// Show first 60 chars of prompt
			promptPreview := strings.ReplaceAll(prompt, "\n", " ")
			if len(promptPreview) > 60 {
				fmt.Printf("     Prompt: %s...\n", promptPreview[:60])
			} else {
				fmt.Printf("     Prompt: %s\n", promptPreview)
			}
		}
	}
	fmt.Println()

	fmt.Printf("Edges: %d\n", len(workflow.Edges))
	for i, edge := range workflow.Edges {
		fmt.Printf("  %d. %s → %s\n", i+1, edge.From, edge.To)
	}
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()

	// Sample code that will be analyzed
	sampleCode := `func calculateTotal(items []Item) float64 {
    total := 0.0
    for i := 0; i < len(items); i++ {
        total = total + items[i].Price * float64(items[i].Quantity)
    }
    return total
}`

	fmt.Println("📝 INPUT CODE:")
	fmt.Println(sampleCode)
	fmt.Println()

	// If no API key, show what would happen and exit
	if apiKey == "" {
		fmt.Println("⚠️  Skipping execution - no OPENAI_API_KEY set")
		fmt.Println()
		showTemplateFlow()
		showKeyFeatures()
		return
	}

	// Execute workflow in standalone mode (no database required)
	fmt.Println("🚀 EXECUTING WORKFLOW IN STANDALONE MODE...")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()

	// Prepare execution input
	input := map[string]interface{}{
		"code": sampleCode,
	}

	// Execute with custom options
	opts := &engine.ExecutionOptions{
		StrictMode:     false,
		MaxParallelism: 1, // Execute nodes sequentially for clarity
		Variables:      make(map[string]interface{}),
	}

	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, opts)
	if err != nil {
		log.Printf("❌ Execution failed: %v\n", err)
		fmt.Println()
		showNodeResults(execution)
		return
	}

	fmt.Printf("✅ Execution completed successfully!\n\n")

	// Display results
	showNodeResults(execution)
	showFinalOutput(execution)
	showTemplateFlow()
	showKeyFeatures()
}

func showNodeResults(execution *models.Execution) {
	if execution == nil {
		return
	}

	fmt.Println("📊 NODE EXECUTION RESULTS:")
	fmt.Println("─────────────────────────────────────────────────────────")
	for i, nodeExec := range execution.NodeExecutions {
		fmt.Printf("%d. %s (%s)\n", i+1, nodeExec.NodeName, nodeExec.NodeID)
		fmt.Printf("   Status: %s\n", nodeExec.Status)

		if nodeExec.Error != "" {
			fmt.Printf("   Error: %s\n", nodeExec.Error)
		}

		if nodeExec.Output != nil && len(nodeExec.Output) > 0 {
			if content, ok := nodeExec.Output["content"].(string); ok {
				// Truncate long outputs
				if len(content) > 100 {
					fmt.Printf("   Output: %s...\n", content[:100])
				} else {
					fmt.Printf("   Output: %s\n", content)
				}
			} else {
				// Pretty print JSON
				outputJSON, _ := json.MarshalIndent(nodeExec.Output, "   ", "  ")
				fmt.Printf("   Output: %s\n", string(outputJSON))
			}
		}
		fmt.Println()
	}
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()
}

func showFinalOutput(execution *models.Execution) {
	if execution == nil || execution.Output == nil {
		return
	}

	fmt.Println("🎯 FINAL OUTPUT:")
	fmt.Println("─────────────────────────────────────────────────────────")

	if content, ok := execution.Output["content"].(string); ok {
		fmt.Println(content)
	} else {
		outputJSON, _ := json.MarshalIndent(execution.Output, "", "  ")
		fmt.Println(string(outputJSON))
	}

	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()

	fmt.Printf("📈 Execution Stats:\n")
	fmt.Printf("   Duration: %dms\n", execution.Duration)
	fmt.Printf("   Status: %s\n", execution.Status)
	fmt.Println()
}

func showTemplateFlow() {
	fmt.Println("🔄 TEMPLATE RESOLUTION FLOW:")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("Step 1: detect_language node receives execution input")
	fmt.Println("  Input: {code: \"func calculateTotal...\"}")
	fmt.Println("  Template: {{input.code}}")
	fmt.Println("  Resolved: \"func calculateTotal...\"")
	fmt.Println("  Output: {content: \"Go\"}")
	fmt.Println()

	fmt.Println("Step 2: analyze_code node receives merged input")
	fmt.Println("  Merged Input: {code: \"func calculateTotal...\", content: \"Go\"}")
	fmt.Println("  (Parent output merged with execution input)")
	fmt.Println("  Template: {{input.content}} → \"Go\"")
	fmt.Println("  Template: {{input.code}} → \"func calculateTotal...\"")
	fmt.Println("  Instruction: \"You are an expert Go developer\"")
	fmt.Println("  Output: {content: \"Security analysis...\"}")
	fmt.Println()

	fmt.Println("Step 3: refactor_code node receives merged input")
	fmt.Println("  Merged Input: {code: \"func calculateTotal...\", content: \"Security analysis...\"}")
	fmt.Println("  Template: {{input.content}} → analysis text")
	fmt.Println("  Output: {content: \"refactored code\"}")
	fmt.Println()

	fmt.Println("Step 4: explain_changes node receives merged input")
	fmt.Println("  Merged Input: {code: \"func calculateTotal...\", content: \"refactored code\"}")
	fmt.Println("  Note: {{input.analysis}} is not available (would need multi-parent)")
	fmt.Println("  Template: {{input.content}} → refactored code")
	fmt.Println("  Output: {content: \"Explanation of changes\"}")
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()
}

func showKeyFeatures() {
	fmt.Println("🎯 KEY FEATURES DEMONSTRATED:")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("✓ Standalone Execution (No Database Required)")
	fmt.Println("  - Execute workflows in-memory without persistence")
	fmt.Println("  - Perfect for examples, testing, and simple automation")
	fmt.Println()
	fmt.Println("✓ Automatic Template Resolution")
	fmt.Println("  - Templates {{input.X}} and {{env.X}} are resolved BEFORE execution")
	fmt.Println("  - No manual template handling in LLM executor needed")
	fmt.Println()
	fmt.Println("✓ Parent Output Namespace Management")
	fmt.Println("  - Single parent: output merged directly into input")
	fmt.Println("  - Multiple parents: outputs namespaced by parent node ID")
	fmt.Println()
	fmt.Println("✓ Variable Precedence")
	fmt.Println("  - Execution variables override workflow variables")
	fmt.Println("  - {{env.model}} resolves from workflow.Variables")
	fmt.Println("  - {{env.openai_api_key}} resolves from workflow.Variables")
	fmt.Println()
	fmt.Println("✓ Chain of LLM Calls")
	fmt.Println("  - Each node output becomes next node input")
	fmt.Println("  - Language detection → Code analysis → Refactoring → Explanation")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()

	fmt.Println("📚 TEMPLATE SYNTAX REFERENCE:")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("{{input.field}}        - Access field from parent node output")
	fmt.Println("{{env.variable}}       - Access workflow/execution variable")
	fmt.Println("{{input.parent.field}} - Access specific parent (multi-parent)")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()
}
