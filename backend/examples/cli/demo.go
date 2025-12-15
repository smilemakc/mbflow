//go:build !colors

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/visualization"
)

func main() {
	fmt.Println("=== MBFlow CLI Visualization Demo ===\n")

	// Load workflow from JSON file
	data, err := os.ReadFile("test_workflow.json")
	if err != nil {
		log.Fatalf("Failed to read workflow file: %v", err)
	}

	var workflow models.Workflow
	if err := json.Unmarshal(data, &workflow); err != nil {
		log.Fatalf("Failed to parse workflow JSON: %v", err)
	}

	// 1. Mermaid diagram with top-bottom layout
	fmt.Println("1. Mermaid Diagram (Top-Bottom Layout)")
	fmt.Println("======================================")
	opts := &visualization.RenderOptions{
		ShowConfig:     true,
		ShowConditions: true,
		Direction:      "TB",
	}
	diagram, err := visualization.RenderWorkflow(&workflow, "mermaid", opts)
	if err != nil {
		log.Fatalf("Failed to render Mermaid diagram: %v", err)
	}
	fmt.Println(diagram)

	// 2. Mermaid diagram with ELK adaptive layout
	fmt.Println("\n2. Mermaid Diagram (ELK Adaptive Layout)")
	fmt.Println("=========================================")
	optsElk := &visualization.RenderOptions{
		ShowConfig:     true,
		ShowConditions: true,
		Direction:      "elk",
	}
	diagramElk, err := visualization.RenderWorkflow(&workflow, "mermaid", optsElk)
	if err != nil {
		log.Fatalf("Failed to render ELK diagram: %v", err)
	}
	fmt.Println(diagramElk)

	// 3. ASCII tree (compact mode)
	fmt.Println("\n3. ASCII Tree (Compact Mode)")
	fmt.Println("=============================")
	asciiOpts := &visualization.RenderOptions{
		ShowConfig:     false,
		ShowConditions: true,
		CompactMode:    true,
		UseColor:       true,
	}
	asciiCompact, err := visualization.RenderWorkflow(&workflow, "ascii", asciiOpts)
	if err != nil {
		log.Fatalf("Failed to render ASCII tree: %v", err)
	}
	fmt.Println(asciiCompact)

	// 4. ASCII tree (detailed mode)
	fmt.Println("\n4. ASCII Tree (Detailed Mode)")
	fmt.Println("==============================")
	asciiDetailedOpts := &visualization.RenderOptions{
		ShowConfig:     true,
		ShowConditions: true,
		CompactMode:    false,
		UseColor:       true,
	}
	asciiDetailed, err := visualization.RenderWorkflow(&workflow, "ascii", asciiDetailedOpts)
	if err != nil {
		log.Fatalf("Failed to render ASCII tree: %v", err)
	}
	fmt.Println(asciiDetailed)

	// 5. Save to file
	fmt.Println("\n5. Saving to Files")
	fmt.Println("==================")

	// Save Mermaid diagram
	if err := visualization.SaveWorkflowToFile(&workflow, "mermaid", "workflow_diagram.mmd", optsElk); err != nil {
		log.Fatalf("Failed to save Mermaid diagram: %v", err)
	}
	fmt.Println("✓ Mermaid diagram saved to: workflow_diagram.mmd")

	// Save ASCII tree
	if err := visualization.SaveWorkflowToFile(&workflow, "ascii", "workflow_tree.txt", asciiDetailedOpts); err != nil {
		log.Fatalf("Failed to save ASCII tree: %v", err)
	}
	fmt.Println("✓ ASCII tree saved to: workflow_tree.txt")

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nNext steps:")
	fmt.Println("1. View the generated files: workflow_diagram.mmd and workflow_tree.txt")
	fmt.Println("2. Copy the Mermaid code to https://mermaid.live to see the interactive diagram")
	fmt.Println("3. Try the CLI tool: ../../bin/mbflow-cli workflow show <id> -format mermaid")
}
