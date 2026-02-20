// Sub-Workflow Fan-Out Example
//
// This example demonstrates the sub_workflow node type in MBFlow:
// - A parent workflow with a sub_workflow node that fans out over an array
// - A child workflow that processes each item independently
// - Parallel execution with configurable max_parallelism
// - collect_partial error handling mode
//
// The pattern:
//
//	parent: [source] --> [fanout (sub_workflow)] --> [aggregate]
//	child:  [process] (runs N times in parallel, once per array item)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow/go/pkg/builder"
	"github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func main() {
	// 1. Create child workflow: processes a single content cell
	childWF := builder.NewWorkflow("Cell Processor",
		builder.WithDescription("Process a single content cell"),
	).
		AddNode(builder.NewNode("process", "transform", "Process Cell")).
		MustBuild()
	childWF.ID = "cell-processor-wf"

	fmt.Printf("Child workflow: %s (ID: %s, %d nodes)\n", childWF.Name, childWF.ID, len(childWF.Nodes))

	// 2. Create parent workflow with sub_workflow fan-out
	parentWF := builder.NewWorkflow("Content Plan Generator",
		builder.WithDescription("Generate content for all cells in parallel"),
	).
		AddNode(builder.NewSubWorkflowNode("fanout", "Generate All Cells", childWF.ID,
			builder.WithForEach("input.cells"),
			builder.WithItemVar("cell"),
			builder.WithMaxParallelism(3),
			builder.WithOnError("collect_partial"),
		)).
		MustBuild()

	fmt.Printf("Parent workflow: %s (%d nodes)\n", parentWF.Name, len(parentWF.Nodes))
	fmt.Println()

	// 3. Set up engine with mock executor
	mockExec := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			inputMap, _ := input.(map[string]any)
			cell, _ := inputMap["cell"].(map[string]any)
			topic, _ := cell["topic"].(string)
			channel, _ := cell["channel"].(string)
			return map[string]any{
				"text":    fmt.Sprintf("Generated post about '%s' for %s", topic, channel),
				"channel": channel,
				"topic":   topic,
			}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("transform", mockExec)

	loader := engine.NewMockWorkflowLoader(map[string]*models.Workflow{
		childWF.ID: childWF,
	})

	nodeExec := engine.NewNodeExecutor(registry)
	dagExec := engine.NewDAGExecutor(
		nodeExec,
		engine.NewExprConditionEvaluator(),
		engine.NewNoOpNotifier(),
		loader,
	)

	// 4. Execute with sample content cells
	input := map[string]any{
		"cells": []any{
			map[string]any{"topic": "AI trends 2026", "channel": "telegram"},
			map[string]any{"topic": "Go 1.24 release", "channel": "blog"},
			map[string]any{"topic": "Remote work tips", "channel": "instagram"},
			map[string]any{"topic": "Cloud cost optimization", "channel": "telegram"},
			map[string]any{"topic": "Rust vs Go", "channel": "blog"},
		},
	}

	fmt.Printf("Input: %d cells to process (max_parallelism=3)\n", len(input["cells"].([]any)))
	fmt.Println()

	execState := engine.NewExecutionState("demo-exec", parentWF.ID, parentWF, input, nil)

	err := dagExec.Execute(context.Background(), execState, engine.DefaultExecutionOptions())
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	// 5. Print results
	output, _ := execState.GetNodeOutput("fanout")
	outputMap, _ := output.(map[string]any)

	// Print summary
	summary, _ := outputMap["summary"].(map[string]any)
	fmt.Println("=== Fan-Out Results ===")
	fmt.Printf("Total:     %v\n", summary["total"])
	fmt.Printf("Completed: %v\n", summary["completed"])
	fmt.Printf("Failed:    %v\n", summary["failed"])
	fmt.Println()

	// Print each item
	items, _ := outputMap["items"].([]any)
	for _, item := range items {
		itemMap, _ := item.(map[string]any)
		fmt.Printf("  [%v] %s", itemMap["index"], itemMap["status"])
		if itemOutput, ok := itemMap["output"].(map[string]any); ok {
			fmt.Printf(" → %s", itemOutput["text"])
		}
		fmt.Println()
	}

	// Full JSON output
	fmt.Println()
	jsonOutput, _ := json.MarshalIndent(output, "", "  ")
	fmt.Printf("Full output:\n%s\n", jsonOutput)
}
