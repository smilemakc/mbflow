# MBFlow - Workflow Engine Library

MBFlow - это библиотека для создания и выполнения рабочих процессов (workflows) на Go, следующая принципам Domain-Driven Design (DDD).

## Установка

```bash
go get github.com/yourusername/mbflow
```

## Быстрый старт

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    
    "github.com/google/uuid"
    "mbflow"
)

func main() {
    // Создаем хранилище в памяти
    storage := mbflow.NewMemoryStorage()
    
    ctx := context.Background()
    
    // Создаем новый рабочий процесс
    var spec map[string]any
    json.Unmarshal([]byte(`{"description": "My first workflow"}`), &spec)
    workflow := mbflow.NewWorkflow(
        uuid.NewString(),
        "My Workflow",
        "1.0.0",
        spec,
    )
    
    // Сохраняем workflow
    if err := storage.SaveWorkflow(ctx, workflow); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Created workflow: %s\n", workflow.Name())
}
```

## Основные концепции

### Workflow (Рабочий процесс)

Workflow представляет собой граф из узлов (nodes) и связей (edges), определяющий последовательность операций.

```go
workflow := mbflow.NewWorkflow(id, name, version, spec)
```

### Node (Узел)

Node - это отдельная операция в рабочем процессе.

```go
node := mbflow.NewNode(
    id,
    workflowID,
    "http-request",  // тип узла
    "Fetch Data",    // имя
    map[string]any{"url": "https://api.example.com"}, // конфигурация
)
```

Для отправки уведомлений доступен тип узла `telegram-message`, использующий Telegram Bot API:

```go
node := mbflow.NewNode(
    id,
    workflowID,
    "telegram-message",
    "Send update",
    map[string]any{
        "chat_id": "@my_channel",
        "text":    "Build finished with status {{status}}",
    },
)
```

### Edge (Связь)

Edge определяет переход между узлами.

```go
edge := mbflow.NewEdge(
    id,
    workflowID,
    fromNodeID,
    toNodeID,
    "direct",  // тип связи
    map[string]any{}, // конфигурация
)
```

### Execution (Выполнение)

Execution представляет собой конкретный запуск workflow.

```go
execution := mbflow.NewExecution(id, workflowID)
```

## Хранилища

### In-Memory Storage

Для разработки и тестирования:

```go
storage := mbflow.NewMemoryStorage()
```

### PostgreSQL Storage

Для production использования:

```go
storage := mbflow.NewPostgresStorage("postgres://user:pass@localhost:5432/dbname?sslmode=disable")
```

## Примеры

Полные примеры использования находятся в директории [examples/](./examples/).

### Базовый пример

```bash
cd examples/basic
go run main.go
```

## API Reference

### Storage Interface

```go
type Storage interface {
    // Workflows
    SaveWorkflow(ctx context.Context, w Workflow) error
    GetWorkflow(ctx context.Context, id string) (Workflow, error)
    ListWorkflows(ctx context.Context) ([]Workflow, error)
    
    // Executions
    SaveExecution(ctx context.Context, e Execution) error
    GetExecution(ctx context.Context, id string) (Execution, error)
    ListExecutions(ctx context.Context) ([]Execution, error)
    
    // Nodes
    SaveNode(ctx context.Context, n Node) error
    GetNode(ctx context.Context, id string) (Node, error)
    ListNodes(ctx context.Context, workflowID string) ([]Node, error)
    
    // Edges
    SaveEdge(ctx context.Context, e Edge) error
    GetEdge(ctx context.Context, id string) (Edge, error)
    ListEdges(ctx context.Context, workflowID string) ([]Edge, error)
    
    // Triggers
    SaveTrigger(ctx context.Context, t Trigger) error
    GetTrigger(ctx context.Context, id string) (Trigger, error)
    ListTriggers(ctx context.Context, workflowID string) ([]Trigger, error)
    
    // Events
    AppendEvent(ctx context.Context, e Event) error
    ListEventsByExecution(ctx context.Context, executionID string) ([]Event, error)
}
```

## Архитектура

Проект следует принципам Domain-Driven Design (DDD):

```
mbflow/
├── mbflow.go           # Публичные интерфейсы
├── factory.go          # Фабричные функции
├── adapter.go          # Адаптеры для внутренних реализаций
├── internal/           # Внутренняя реализация (не экспортируется)
│   ├── domain/         # Доменная логика
│   ├── infrastructure/ # Инфраструктурный слой
│   └── application/    # Слой приложения
└── examples/           # Примеры использования
```

## Разработка

### Запуск тестов

```bash
go test ./...
```

### Запуск сервера (для разработки)

```bash
cd cmd/server
go run main.go
```

## Лицензия

MIT

## Вклад в проект

Мы приветствуем вклад в проект! Пожалуйста, создавайте issues и pull requests.
