//go:build colors

package main

import (
	"fmt"
	"log"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/visualization"
)

func main() {
	fmt.Println("=== Mermaid Color Styling Demo ===\n")

	// Create workflow with all node types to demonstrate color styling
	workflow := &models.Workflow{
		Name:        "Multi-Type Workflow Demo",
		Description: "Demonstrates Mermaid color styling for different node types",
		Nodes: []*models.Node{
			{
				ID:   "fetch_data",
				Name: "Fetch User Data",
				Type: "http",
				Config: map[string]interface{}{
					"method": "GET",
					"url":    "https://api.example.com/users",
				},
			},
			{
				ID:   "analyze_sentiment",
				Name: "Analyze Sentiment",
				Type: "llm",
				Config: map[string]interface{}{
					"provider": "openai",
					"model":    "gpt-4",
				},
			},
			{
				ID:   "transform_data",
				Name: "Transform to JSON",
				Type: "transform",
				Config: map[string]interface{}{
					"type": "expression",
				},
			},
			{
				ID:   "quality_check",
				Name: "Quality Gate",
				Type: "conditional",
			},
			{
				ID:   "merge_results",
				Name: "Combine Results",
				Type: "merge",
			},
			{
				ID:   "send_notification",
				Name: "Send Email",
				Type: "http",
				Config: map[string]interface{}{
					"method": "POST",
					"url":    "https://api.sendgrid.com/v3/mail/send",
				},
			},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "fetch_data", To: "analyze_sentiment"},
			{ID: "e2", From: "analyze_sentiment", To: "transform_data"},
			{ID: "e3", From: "transform_data", To: "quality_check"},
			{ID: "e4", From: "quality_check", To: "merge_results", Condition: "score > 0.8"},
			{ID: "e5", From: "quality_check", To: "send_notification", Condition: "score <= 0.8"},
			{ID: "e6", From: "merge_results", To: "send_notification"},
		},
	}

	fmt.Println("1. Mermaid Diagram with ELK Layout and Color Styling")
	fmt.Println("=====================================================\n")

	opts := &visualization.RenderOptions{
		ShowConfig:     true,
		ShowConditions: true,
		Direction:      "elk",
	}

	diagram, err := visualization.RenderWorkflow(workflow, "mermaid", opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(diagram)

	fmt.Println("\n2. Color Legend")
	fmt.Println("===============")
	fmt.Println("🔵 HTTP nodes (fetch_data, send_notification) - Light Blue (#e1f5ff)")
	fmt.Println("🟣 LLM nodes (analyze_sentiment) - Light Purple (#f3e5f5)")
	fmt.Println("🟠 Transform nodes (transform_data) - Light Orange (#fff3e0)")
	fmt.Println("🟢 Conditional nodes (quality_check) - Light Green (#e8f5e9)")
	fmt.Println("🔴 Merge nodes (merge_results) - Light Pink (#fce4ec)")

	fmt.Println("\n3. Node Type Shapes")
	fmt.Println("===================")
	fmt.Println("HTTP:        Rectangle [...]")
	fmt.Println("LLM:         Stadium ([...])")
	fmt.Println("Transform:   Trapezoid [/...\\]")
	fmt.Println("Conditional: Diamond {...}")
	fmt.Println("Merge:       Hexagon {{...}}")

	fmt.Println("\n4. Saving Diagram")
	fmt.Println("=================")

	if err := visualization.SaveWorkflowToFile(workflow, "mermaid", "colored_workflow.mmd", opts); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Saved to: colored_workflow.mmd")
	fmt.Println("\nOpen https://mermaid.live and paste the diagram to see it rendered with colors!")
}
