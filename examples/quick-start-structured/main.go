package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smilemakc/mbflow"
)

// Этот пример показывает, как использовать структурные конфиги
// для создания простого workflow с HTTP запросом
func main() {
	fmt.Println("=== Quick Start: Structured Configs ===")

	// Создаем workflow с типобезопасными конфигами
	workflow, err := mbflow.NewWorkflowBuilder("Quick Start", "1.0").
		// Используем структурный конфиг вместо map[string]any
		AddNodeWithConfig(
			string(mbflow.NodeTypeHTTPRequest),
			"get_weather",
			&mbflow.HTTPRequestConfig{
				URL:    "https://api.open-meteo.com/v1/forecast?latitude=52.52&longitude=13.41&current_weather=true",
				Method: "GET",
			},
		).
		AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{}).
		Build()

	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Println("✓ Workflow created successfully")
	fmt.Printf("  Name: %s\n", workflow.Name())
	fmt.Printf("  Nodes: %d\n", len(workflow.GetAllNodes()))

	// Выполняем workflow
	executor := mbflow.NewExecutorBuilder().Build()
	triggers := workflow.GetAllTriggers()

	fmt.Println("\n▶ Executing workflow...")
	execution, err := executor.ExecuteWorkflow(
		context.Background(),
		workflow,
		triggers[0],
		map[string]any{},
	)

	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Println("✓ Execution completed")
	fmt.Printf("  Status: %s\n", execution.Phase())

	// Показываем результат
	if weather, ok := execution.Variables().All()["weather"]; ok {
		fmt.Printf("\n📊 Weather data received: %v\n", weather)
	}

	fmt.Println("\n💡 Преимущества структурных конфигов:")
	fmt.Println("  ✓ Типобезопасность на этапе компиляции")
	fmt.Println("  ✓ Автодополнение в IDE")
	fmt.Println("  ✓ Встроенная документация")
	fmt.Println("  ✓ Легкость рефакторинга")
}
