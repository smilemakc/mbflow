// Simple example: ExecuteWorkflowStandalone with Observer (no HTTP calls)
package main

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/go/internal/application/observer"
	"github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/sdk"
)

func runSimpleExample() {
	ctx := context.Background()

	// Create standalone client
	client, _ := sdk.NewStandaloneClient()
	defer client.Close()

	// Create observer manager with custom observer
	observerManager := observer.NewObserverManager()
	progressObserver := &SimpleProgressObserver{}
	observerManager.Register(progressObserver)

	// Simple workflow: Transform data in 2 steps
	workflow := &models.Workflow{
		Name: "Simple Transform Pipeline",
		Nodes: []*models.Node{
			{
				ID:   "greet",
				Name: "Create Greeting",
				Type: "transform",
				Config: map[string]any{
					"type": "template",
					"output": map[string]any{
						"message": "Hello, {{input.name}}!",
						"name":    "{{input.name}}",
					},
				},
			},
			{
				ID:   "format",
				Name: "Format Output",
				Type: "transform",
				Config: map[string]any{
					"type": "template",
					"output": map[string]any{
						"result": "{{input.message}} Welcome to MBFlow.",
					},
				},
			},
		},
		Edges: []*models.Edge{
			{ID: "edge-1", From: "greet", To: "format"},
		},
	}

	// Execute with default options
	// Note: observer integration requires the full engine mode (not standalone)
	_ = observerManager
	opts := &engine.ExecutionOptions{}

	input := map[string]any{
		"name": "World",
	}

	fmt.Println("=== Simple Observer Example ===")

	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\n✓ Result: %s\n", execution.Status)
	fmt.Printf("  Duration: %dms\n", execution.Duration)
	fmt.Printf("\n  Nodes executed: %d\n", progressObserver.NodesCompleted)
	fmt.Printf("  Output: %v\n", execution.Output)
}

// SimpleProgressObserver tracks execution progress
type SimpleProgressObserver struct {
	NodesCompleted int
}

func (p *SimpleProgressObserver) Name() string {
	return "simple-progress"
}

func (p *SimpleProgressObserver) Filter() observer.EventFilter {
	return nil
}

func (p *SimpleProgressObserver) OnEvent(ctx context.Context, event observer.Event) error {
	switch event.Type {
	case observer.EventTypeNodeStarted:
		fmt.Printf("  → %s\n", *event.NodeName)

	case observer.EventTypeNodeCompleted:
		p.NodesCompleted++
		fmt.Printf("  ✓ %s\n", *event.NodeName)
	}
	return nil
}
