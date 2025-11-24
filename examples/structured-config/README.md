# Structured Config Example

Этот пример демонстрирует использование структурных конфигураций для нод вместо `map[string]any`.

## Преимущества структурных конфигов

1. **Типобезопасность** - компилятор проверяет типы на этапе компиляции
2. **Автодополнение** - лучшая поддержка в IDE с подсказками полей
3. **Документация** - встроенная документация через комментарии к полям
4. **Простота рефакторинга** - легче находить и изменять использование конфигов
5. **Валидация** - использование struct tags для валидации

## Использование

### Старый способ (map[string]any)

```go
builder.AddNode(
    mbflow.NodeTypeHTTPRequest,
    "fetch_data",
    map[string]any{
        "url":        "https://api.example.com",
        "method":     "GET",
        "output_key": "response",
        "headers": map[string]string{
            "Accept": "application/json",
        },
    },
)
```

### Новый способ (структурный конфиг)

```go
builder.AddNodeWithConfig(
    mbflow.NodeTypeHTTPRequest,
    "fetch_data",
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

## Доступные конфигурации

- `OpenAICompletionConfig` - конфигурация для OpenAI completion нод
- `HTTPRequestConfig` - конфигурация для HTTP запросов
- `TelegramMessageConfig` - конфигурация для отправки сообщений в Telegram
- `ConditionalRouterConfig` - конфигурация для условной маршрутизации
- `DataMergerConfig` - конфигурация для объединения данных
- `DataAggregatorConfig` - конфигурация для агрегации данных
- `ScriptExecutorConfig` - конфигурация для выполнения скриптов
- `JSONParserConfig` - конфигурация для парсинга JSON
- `OpenAIResponsesConfig` - конфигурация для OpenAI Responses API
- `ConditionalEdgeConfig` - конфигурация для условных связей

## Запуск примера

```bash
cd examples/structured-config
go run main.go
```

## Создание собственных конфигов

Вы можете создать свои собственные структурные конфиги, реализовав интерфейс `mbflow.NodeConfig`:

```go
type MyCustomConfig struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}

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
builder.AddNodeWithConfig(
    "custom_type",
    "my_node",
    &MyCustomConfig{
        Field1: "value",
        Field2: 42,
    },
)
```
