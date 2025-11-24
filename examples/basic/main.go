package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
)

func main() {
	ctx := context.Background()

	// Строим workflow через новый fluent WorkflowBuilder
	workflow, err := mbflow.NewWorkflowBuilder("My First Workflow", "1.0.0").
		WithDescription("HTTP → Transform → End demo").
		AddNode("start", "Start", nil).
		AddNode(string(mbflow.NodeTypeHTTPRequest), "Fetch Data", map[string]any{
			"url":        "https://booking-hub.free.beeceptor.com",
			"method":     "GET",
			"output_key": "processed",
		}).
		AddNode("transform", "Process Data", map[string]any{
			"transformations": map[string]any{
				"processed": "input * 2",
			},
		}).
		AddNode("end", "Finish", map[string]any{
			"output_keys": []any{"processed"},
		}).
		AddEdge("Start", "Fetch Data", "direct", nil).
		AddEdge("Fetch Data", "Process Data", "direct", nil).
		AddEdge("Process Data", "Finish", "direct", nil).
		AddTrigger("manual", nil).
		Build()
	if err != nil {
		log.Fatalf("failed to build workflow: %v", err)
	}

	fmt.Printf("Workflow ready: %s (ID: %s)\n", workflow.Name(), workflow.ID())

	// Создаем executor c in-memory EventStore и стримингом событий
	exec := mbflow.NewExecutor()

	triggers := workflow.GetAllTriggers()
	if len(triggers) == 0 {
		log.Fatal("workflow must have at least one trigger")
	}
	trigger := triggers[0]

	// Выполняем workflow
	execution, err := exec.ExecuteWorkflow(ctx, workflow, trigger, map[string]any{
		"input": 21,
	})
	if err != nil {
		log.Fatalf("execution failed: %v", err)
	}

	fmt.Printf("Execution %s finished with phase=%s duration=%v\n",
		execution.ID(), execution.Phase(), execution.Duration())

	fmt.Printf("Final variables: %+v\n", execution.Variables().All())

	events, _ := exec.EventStore().GetEvents(ctx, execution.ID())
	fmt.Printf("Event log entries: %d\n", len(events))
}
