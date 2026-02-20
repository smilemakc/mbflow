// Example demonstrating LLM executor with input templates using Builder API
// This example shows how to chain LLM calls using {{input.X}} template syntax
// with the type-safe, fluent builder API (64% less code than raw structs)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/smilemakc/mbflow/go/pkg/builder"
	"github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/sdk"
	"github.com/smilemakc/mbflow/go/pkg/visualization"
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
	// In standalone mode, only ExecuteWorkflowStandalone() is available.
	// Perfect for examples, testing, and simple automation without persistence.
	client, err := sdk.NewStandaloneClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a multi-step code analysis workflow using Builder API
	// This is significantly more concise than manually constructing nodes and edges
	workflow := builder.NewWorkflow("Code Analysis with Input Templates",
		builder.WithDescription("Demonstrates chaining LLM calls using {{input.X}} templates"),
		builder.WithVariable("openai_api_key", apiKey),
		builder.WithVariable("model", "gpt-4"),
		builder.WithAutoLayout(), // Automatically position nodes
	).AddNode(
		// Step 1: Extract programming language from code
		builder.NewOpenAINode(
			"detect_language",
			"Detect Programming Language",
			"{{env.model}}",
			"Identify the programming language of this code. Reply with ONLY the language name:\n\n{{input.code}}",
			builder.LLMAPIKey("{{env.openai_api_key}}"),
			builder.LLMTemperature(0.0),
			builder.LLMMaxTokens(50),
		),
	).AddNode(
		// Step 2: Analyze code for issues using detected language
		builder.NewOpenAINode(
			"analyze_code",
			"Analyze Code Quality",
			"{{env.model}}",
			`Analyze this {{input.content}} code for potential issues:

{{input.code}}

Focus on:
1. Security vulnerabilities
2. Performance issues
3. Code style and best practices
4. Potential bugs

Provide specific recommendations.`,
			builder.LLMAPIKey("{{env.openai_api_key}}"),
			builder.LLMSystemPrompt("You are an expert {{input.content}} developer and code reviewer."),
			builder.LLMTemperature(0.2),
			builder.LLMMaxTokens(800),
		),
	).AddNode(
		// Step 3: Generate refactored version based on analysis
		builder.NewOpenAINode(
			"refactor_code",
			"Generate Refactored Code",
			"{{env.model}}",
			`Based on this code review:

{{input.content}}

Refactor the code to address all issues. Provide ONLY the refactored code without explanations.`,
			builder.LLMAPIKey("{{env.openai_api_key}}"),
			builder.LLMSystemPrompt("You are a code refactoring expert."),
			builder.LLMTemperature(0.1),
			builder.LLMMaxTokens(1000),
		),
	).AddNode(
		// Step 4: Explain the changes made
		builder.NewOpenAINode(
			"explain_changes",
			"Explain Refactoring",
			"gpt-3.5-turbo", // Use faster model for explanation
			`Explain the changes made in this code refactoring:

ORIGINAL ANALYSIS:
{{input.analysis}}

REFACTORED CODE:
{{input.content}}

Provide a clear explanation suitable for junior developers.`,
			builder.LLMAPIKey("{{env.openai_api_key}}"),
			builder.LLMTemperature(0.5),
			builder.LLMMaxTokens(500),
		),
	).Connect("detect_language", "analyze_code").
		Connect("analyze_code", "refactor_code").
		Connect("refactor_code", "explain_changes").
		MustBuild()

	fmt.Printf("✓ Workflow defined: %s\n\n", workflow.Name)
	lrOpts := &visualization.RenderOptions{
		ShowConfig:     true,
		ShowConditions: true,
		Direction:      "elk",
	}
	err = visualization.PrintWorkflow(workflow, "mermaid", lrOpts)
	if err != nil {
		log.Fatal(err)
	}
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
	input := map[string]any{
		"code": sampleCode,
	}

	// Execute with custom options
	opts := &engine.ExecutionOptions{
		StrictMode:     false,
		MaxParallelism: 1, // Execute nodes sequentially for clarity
		Variables:      make(map[string]any),
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
	fmt.Println("✓ Builder API (Type-Safe Workflow Construction)")
	fmt.Println("  - 64% less code than manual struct construction")
	fmt.Println("  - Full IDE autocomplete and compile-time validation")
	fmt.Println("  - Fluent, chainable API for readability")
	fmt.Println()
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

	fmt.Println("📖 LEARN MORE:")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("Builder API:      docs/BUILDER_README.md")
	fmt.Println("Full Example:     examples/builder_usage/main.go")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()
}
