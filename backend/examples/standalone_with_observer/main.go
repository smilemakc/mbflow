// Example: ExecuteWorkflowStandalone with Observer
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
)

func main() {
	ctx := context.Background()

	// Create a standalone client (no database required)
	client, err := sdk.NewStandaloneClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create observer manager
	observerManager := observer.NewObserverManager()

	// Register logger observer to see events in console
	loggerInstance := logger.New(config.LoggingConfig{
		Level:  "info",
		Format: "text",
	})
	loggerObserver := observer.NewLoggerObserver(
		observer.WithLoggerInstance(loggerInstance),
		// Optional: filter only specific events
		// observer.WithLoggerFilter(observer.NewEventTypeFilter(
		// 	observer.EventTypeExecutionStarted,
		// 	observer.EventTypeNodeCompleted,
		// 	observer.EventTypeExecutionCompleted,
		// )),
	)
	if err := observerManager.Register(loggerObserver); err != nil {
		log.Fatalf("Failed to register logger observer: %v", err)
	}

	// Create custom observer to track execution progress
	progressObserver := &ProgressObserver{}
	if err := observerManager.Register(progressObserver); err != nil {
		log.Fatalf("Failed to register progress observer: %v", err)
	}

	// Create a simple workflow
	workflow := &models.Workflow{
		Name:        "Workflow with Observers",
		Description: "Demonstrates observer functionality in standalone mode",
		Variables: map[string]interface{}{
			"api_base": "https://jsonplaceholder.typicode.com",
		},
		Nodes: []*models.Node{
			{
				ID:   "fetch-user",
				Name: "Fetch User Data",
				Type: "http",
				Config: map[string]interface{}{
					"method": "GET",
					"url":    "{{env.api_base}}/users/{{input.user_id}}",
				},
			},
			{
				ID:   "fetch-posts",
				Name: "Fetch User Posts",
				Type: "http",
				Config: map[string]interface{}{
					"method": "GET",
					"url":    "{{env.api_base}}/posts?userId={{input.user_id}}",
				},
			},
			{
				ID:   "combine-data",
				Name: "Combine Results",
				Type: "transform",
				Config: map[string]interface{}{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "edge-1",
				From: "fetch-user",
				To:   "combine-data",
			},
			{
				ID:   "edge-2",
				From: "fetch-posts",
				To:   "combine-data",
			},
		},
	}

	fmt.Printf("Starting workflow: %s\n\n", workflow.Name)

	// Prepare execution options with observer manager
	opts := &engine.ExecutionOptions{
		MaxParallelism:  10,
		ObserverManager: observerManager,
	}

	// Prepare input
	input := map[string]interface{}{
		"user_id": 1,
	}

	// Execute workflow with observers
	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, opts)
	if err != nil {
		log.Printf("Execution failed: %v\n", err)
		return
	}

	fmt.Printf("\n✓ Execution completed successfully!\n")
	fmt.Printf("  Execution ID: %s\n", execution.ID)
	fmt.Printf("  Status: %s\n", execution.Status)
	fmt.Printf("  Duration: %dms\n", execution.Duration)

	// Show progress observer stats
	fmt.Printf("\nProgress Observer Stats:\n")
	fmt.Printf("  Nodes Started: %d\n", progressObserver.NodesStarted)
	fmt.Printf("  Nodes Completed: %d\n", progressObserver.NodesCompleted)
	fmt.Printf("  Nodes Failed: %d\n", progressObserver.NodesFailed)
	fmt.Printf("  Waves: %d\n", progressObserver.Waves)
}

// ProgressObserver tracks execution progress
type ProgressObserver struct {
	NodesStarted   int
	NodesCompleted int
	NodesFailed    int
	Waves          int
}

func (p *ProgressObserver) Name() string {
	return "progress"
}

func (p *ProgressObserver) Filter() observer.EventFilter {
	return nil // Accept all events
}

func (p *ProgressObserver) OnEvent(ctx context.Context, event observer.Event) error {
	switch event.Type {
	case observer.EventTypeWaveStarted:
		p.Waves++
		fmt.Printf("→ Wave %d started (%d nodes)\n", *event.WaveIndex, *event.NodeCount)

	case observer.EventTypeNodeStarted:
		p.NodesStarted++
		fmt.Printf("  ⋯ Node started: %s\n", *event.NodeName)

	case observer.EventTypeNodeCompleted:
		p.NodesCompleted++
		fmt.Printf("  ✓ Node completed: %s (took %dms)\n", *event.NodeName, *event.DurationMs)

	case observer.EventTypeNodeFailed:
		p.NodesFailed++
		fmt.Printf("  ✗ Node failed: %s - %v\n", *event.NodeName, event.Error)

	case observer.EventTypeExecutionStarted:
		fmt.Printf("\n→ Execution started: %s\n", event.ExecutionID)

	case observer.EventTypeExecutionCompleted:
		fmt.Printf("\n✓ Execution completed: %s\n", event.ExecutionID)
	}

	return nil
}
