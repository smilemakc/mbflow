package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"mbflow"

	"github.com/google/uuid"
)

func main() {
	// Создаем хранилище в памяти
	storage := mbflow.NewMemoryStorage()

	ctx := context.Background()

	// Создаем новый рабочий процесс
	var spec map[string]any
	_ = json.Unmarshal([]byte(`{"nodes": [], "edges": []}`), &spec)
	workflow := mbflow.NewWorkflow(
		uuid.NewString(),
		"My First Workflow",
		"1.0.0",
		spec,
	)

	// Сохраняем workflow
	if err := storage.SaveWorkflow(ctx, workflow); err != nil {
		log.Fatalf("Failed to save workflow: %v", err)
	}

	fmt.Printf("Created workflow: %s (ID: %s)\n", workflow.Name(), workflow.ID())

	// Создаем узлы с использованием NodeConfig для валидации UUID
	node1, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflow.ID(),
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Fetch Data",
		Config:     map[string]any{"url": "https://api.example.com/data"},
	})
	if err != nil {
		log.Fatalf("Failed to create node1: %v", err)
	}

	node2, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflow.ID(),
		Type:       "transform",
		Name:       "Process Data",
		Config:     map[string]any{"script": "data.map(x => x * 2)"},
	})
	if err != nil {
		log.Fatalf("Failed to create node2: %v", err)
	}

	// Сохраняем узлы
	if err := storage.SaveNode(ctx, node1); err != nil {
		log.Fatalf("Failed to save node1: %v", err)
	}
	if err := storage.SaveNode(ctx, node2); err != nil {
		log.Fatalf("Failed to save node2: %v", err)
	}

	// Создаем связь между узлами
	edge := mbflow.NewEdge(
		uuid.NewString(),
		workflow.ID(),
		node1.ID(),
		node2.ID(),
		"direct",
		map[string]any{},
	)

	if err := storage.SaveEdge(ctx, edge); err != nil {
		log.Fatalf("Failed to save edge: %v", err)
	}

	// Создаем выполнение
	execution := mbflow.NewExecution(
		uuid.NewString(),
		workflow.ID(),
	)

	if err := storage.SaveExecution(ctx, execution); err != nil {
		log.Fatalf("Failed to save execution: %v", err)
	}

	fmt.Printf("Created execution: %s (Status: %s)\n", execution.ID(), execution.Status())

	// Получаем все узлы workflow
	nodes, err := storage.ListNodes(ctx, workflow.ID())
	if err != nil {
		log.Fatalf("Failed to list nodes: %v", err)
	}

	fmt.Printf("\nWorkflow has %d nodes:\n", len(nodes))
	for _, n := range nodes {
		fmt.Printf("  - %s (%s)\n", n.Name(), n.Type())
	}

	// Получаем все связи
	edges, err := storage.ListEdges(ctx, workflow.ID())
	if err != nil {
		log.Fatalf("Failed to list edges: %v", err)
	}

	fmt.Printf("\nWorkflow has %d edges:\n", len(edges))
	for _, e := range edges {
		fmt.Printf("  - %s -> %s (%s)\n", e.FromNodeID(), e.ToNodeID(), e.Type())
	}
}
