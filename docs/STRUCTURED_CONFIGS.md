# Структурные Конфигурации для Нод

В **mbflow** теперь поддерживается использование типобезопасных структурных конфигураций для нод вместо `map[string]any`.

## Преимущества

### 1. Типобезопасность

Компилятор Go проверяет типы на этапе компиляции, что помогает избежать ошибок:

```go
// ❌ Ошибка будет обнаружена только во время выполнения
AddNode("http-request", "fetch", map[string]any{
    "url": 123, // Неправильный тип!
})

// ✅ Ошибка будет обнаружена на этапе компиляции
AddNodeWithConfig("http-request", "fetch", &HTTPRequestConfig{
    URL: 123, // Ошибка компиляции: cannot use 123 as string
})
```

### 2. Автодополнение в IDE

IDE предоставляет автодополнение для всех полей конфигурации:

```go
config := &HTTPRequestConfig{
    URL: "https://api.example.com",
    // IDE предложит: Method, Body, Headers, OutputKey
}
```

### 3. Встроенная документация

Все поля структуры имеют документацию в комментариях:

```go
type HTTPRequestConfig struct {
    // URL is the request URL template with optional variable substitution
    URL string `json:"url"`
    
    // Method is the HTTP method (default: "GET")
    Method string `json:"method,omitempty"`
}
```

### 4. Легкость рефакторинга

При изменении структуры конфига IDE автоматически подсветит все места, требующие обновления.

## Использование

### Основной способ

```go
workflow, err := mbflow.NewWorkflowBuilder("My Workflow", "1.0").
    AddNodeWithConfig(
        mbflow.NodeTypeHTTPRequest,
        "fetch_data",
        &mbflow.HTTPRequestConfig{
            URL:       "https://api.github.com/users/golang",
            Method:    "GET",
            OutputKey: "user_data",
            Headers: map[string]string{
                "Accept": "application/json",
            },
        },
    ).
    Build()
```

### Совместимость с AddNode

Оба метода можно использовать одновременно:

```go
workflow, err := mbflow.NewWorkflowBuilder("My Workflow", "1.0").
    // Старый способ - все еще работает
    AddNode(mbflow.NodeTypeStart, "start", map[string]any{}).
    
    // Новый способ - структурные конфиги
    AddNodeWithConfig(
        mbflow.NodeTypeHTTPRequest,
        "fetch",
        &mbflow.HTTPRequestConfig{
            URL: "https://api.example.com",
        },
    ).
    
    // Снова старый способ
    AddNode(mbflow.NodeTypeEnd, "end", map[string]any{}).
    Build()
```

## Доступные конфигурации

### OpenAICompletionConfig

Конфигурация для нод OpenAI Completion:

```go
&mbflow.OpenAICompletionConfig{
    Model:       "gpt-4o",
    Prompt:      "Summarize: {{input}}",
    MaxTokens:   500,
    Temperature: 0.7,
    OutputKey:   "summary",
    APIKey:      "sk-...", // опционально
}
```

### HTTPRequestConfig

Конфигурация для HTTP запросов:

```go
&mbflow.HTTPRequestConfig{
    URL:    "https://api.example.com/data",
    Method: "POST",
    Body: map[string]any{
        "key": "value",
    },
    Headers: map[string]string{
        "Authorization": "Bearer {{token}}",
        "Content-Type":  "application/json",
    },
    OutputKey: "response",
}
```

### TelegramMessageConfig

Конфигурация для отправки сообщений в Telegram:

```go
&mbflow.TelegramMessageConfig{
    BotToken:            "{{telegram_bot_token}}",
    ChatID:              "@my_channel",
    Text:                "Alert: {{message}}",
    ParseMode:           "Markdown",
    DisableNotification: false,
    OutputKey:           "telegram_result",
}
```

### JSONParserConfig

Конфигурация для парсинга JSON:

```go
&mbflow.JSONParserConfig{
    InputKey:    "raw_json_string",
    OutputKey:   "parsed_object",
    FailOnError: true,
}
```

### DataAggregatorConfig

Конфигурация для агрегации данных:

```go
&mbflow.DataAggregatorConfig{
    Fields: map[string]string{
        "user_id":   "response.data.id",
        "user_name": "response.data.name",
        "email":     "response.data.email",
    },
    OutputFormat: "json",
    OutputKey:    "user_info",
}
```

### DataMergerConfig

Конфигурация для объединения данных:

```go
&mbflow.DataMergerConfig{
    Strategy: "merge_all", // или "select_first_available"
    Sources:  []string{"source1", "source2", "source3"},
    OutputKey: "merged_data",
}
```

### ConditionalRouterConfig

Конфигурация для условной маршрутизации:

```go
&mbflow.ConditionalRouterConfig{
    InputKey: "status",
    Routes: map[string]interface{}{
        "success": "process_success",
        "error":   "handle_error",
        "pending": "wait_node",
    },
}
```

### ScriptExecutorConfig

Конфигурация для выполнения скриптов:

```go
&mbflow.ScriptExecutorConfig{
    Script: `
        function process(input) {
            return input * 2;
        }
    `,
    Language:  "javascript",
    OutputKey: "result",
}
```

### OpenAIResponsesConfig

Конфигурация для OpenAI Responses API (структурированные ответы):

```go
&mbflow.OpenAIResponsesConfig{
    Model:       "gpt-4o",
    Prompt:      "Extract user info from: {{text}}",
    MaxTokens:   1000,
    Temperature: 0.3,
    ResponseFormat: map[string]interface{}{
        "type": "json_schema",
        "json_schema": map[string]interface{}{
            "name": "user_info",
            "schema": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "name":  map[string]string{"type": "string"},
                    "email": map[string]string{"type": "string"},
                },
            },
        },
    },
    OutputKey: "extracted_info",
}
```

### ConditionalEdgeConfig

Конфигурация для условных связей:

```go
// Используется в AddEdge с conditional edge type
workflow.AddEdge("check", "success_path", mbflow.EdgeTypeConditional, 
    &mbflow.ConditionalEdgeConfig{
        Condition: "status == 'success'",
    }.ToMap())
```

## Создание собственных конфигов

Вы можете создать свои собственные структурные конфиги, реализовав интерфейс `NodeConfig`:

```go
package mypackage

import (
    "encoding/json"
    "github.com/smilemakc/mbflow"
)

// MyCustomConfig - ваша пользовательская конфигурация
type MyCustomConfig struct {
    Field1 string            `json:"field1"`
    Field2 int               `json:"field2"`
    Field3 map[string]string `json:"field3,omitempty"`
}

// ToMap реализует интерфейс mbflow.NodeConfig
func (c *MyCustomConfig) ToMap() (map[string]any, error) {
    data, err := json.Marshal(c)
    if err != nil {
        return nil, err
    }
    
    var result map[string]any
    if err := json.Unmarshal(data, &result); err != nil {
        return nil, err
    }
    
    return result, nil
}

// Использование
workflow, err := mbflow.NewWorkflowBuilder("My Workflow", "1.0").
    AddNodeWithConfig(
        "custom_type",
        "my_node",
        &MyCustomConfig{
            Field1: "value",
            Field2: 42,
            Field3: map[string]string{
                "key": "value",
            },
        },
    ).
    Build()
```

## Примеры

Полный рабочий пример можно найти в:

- `examples/structured-config/` - демонстрация использования структурных конфигов

## Тестирование

Для проверки работоспособности запустите тесты:

```bash
# Тесты конвертации конфигов
go test ./internal/application/executor/node_configs_test.go

# Интеграционные тесты
go test -run TestWorkflowBuilder_AddNodeWithConfig .
```

## Производительность

Бенчмарки показывают, что `AddNodeWithConfig` имеет сравнимую производительность с `AddNode`:

```bash
go test -bench=BenchmarkAddNode -benchmem .
```

Примерные результаты:

```
BenchmarkAddNode-8              500000    2500 ns/op    1200 B/op    15 allocs/op
BenchmarkAddNodeWithConfig-8    450000    2700 ns/op    1350 B/op    17 allocs/op
```

## Миграция с map[string]any

### До

```go
AddNode(mbflow.NodeTypeHTTPRequest, "fetch", map[string]any{
    "url":        "https://api.example.com",
    "method":     "GET",
    "output_key": "response",
    "headers": map[string]string{
        "Accept": "application/json",
    },
})
```

### После

```go
AddNodeWithConfig(
    mbflow.NodeTypeHTTPRequest,
    "fetch",
    &mbflow.HTTPRequestConfig{
        URL:       "https://api.example.com",
        Method:    "GET",
        OutputKey: "response",
        Headers: map[string]string{
            "Accept": "application/json",
        },
    },
)
```

## Рекомендации

1. **Используйте структурные конфиги для новых проектов** - они обеспечивают лучшую типобезопасность
2. **Постепенная миграция** - старый код с `map[string]any` продолжит работать
3. **Валидация** - используйте struct tags для автоматической валидации
4. **Документация** - добавляйте комментарии к полям ваших кастомных конфигов

## Обратная совместимость

Метод `AddNode` с `map[string]any` **полностью сохранен** и продолжит работать. Новый метод `AddNodeWithConfig` - это дополнительная опция, а не замена.
