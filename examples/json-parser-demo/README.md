# JSON Parser Demo

Демонстрация работы ноды `json-parser` для преобразования JSON-строк в структурированные объекты.

## Описание

Этот пример показывает:

1. **Парсинг JSON-строк** в объекты для доступа к вложенным полям
2. **Условную маршрутизацию** на основе вложенных значений
3. **Обработку ошибок** парсинга с опцией `fail_on_error`
4. **Работу с уже распарсенными объектами** (passthrough)

## Структура workflow

```
1. Set JSON String (data-aggregator)
   ↓
2. Parse JSON (json-parser)
   ↓
3. Check User Status (conditional-router)
   ├─ active → Handle Active User
   └─ inactive → Handle Inactive User
```

## Запуск

```bash
cd examples/json-parser-demo
go run main.go
```

## Примеры

### Test Case 1: Active User

**Входные данные:**

```json
{
  "user": {
    "id": 123,
    "name": "John Doe",
    "email": "john@example.com",
    "status": "active",
    "roles": ["admin", "user"]
  },
  "timestamp": "2025-11-23T21:00:00Z"
}
```

**Результат:**

- JSON-строка парсится в объект
- Доступ к `parsed_response.user.status` возвращает `"active"`
- Workflow маршрутизируется в ветку "active_path"

### Test Case 2: Inactive User

**Входные данные:**

```json
{
  "user": {
    "id": 456,
    "name": "Jane Smith",
    "status": "inactive"
  }
}
```

**Результат:**

- Workflow маршрутизируется в ветку "inactive_path"

### Test Case 3: Invalid JSON

**Входные данные:**

```
"this is not valid JSON"
```

**Конфигурация:**

```go
Config: map[string]any{
    "input_key":     "invalid_json",
    "fail_on_error": false,  // Не падать на ошибке
}
```

**Результат:**

- Парсинг не удался, но workflow продолжается
- Исходное значение сохраняется без изменений
- Статус ноды: `"parse_error"`

## Использование в реальных сценариях

### Сценарий 1: Обработка API ответов

```go
// HTTP запрос возвращает JSON-строку
nodeHTTPRequest := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
    Type: "http-request",
    Config: map[string]any{
        "url": "https://api.example.com/user/123",
        "output_key": "api_response",
    },
})

// Парсим ответ
nodeParseResponse := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
    Type: "json-parser",
    Config: map[string]any{
        "input_key": "api_response",
    },
})

// Используем вложенные поля
nodeCheckStatus := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
    Type: "conditional-router",
    Config: map[string]any{
        "input_key": "api_response.status",
        "routes": map[string]string{
            "200": "success_path",
            "404": "not_found_path",
        },
    },
})
```

### Сценарий 2: Парсинг OpenAI ответов

```go
// OpenAI возвращает JSON-строку
nodeQualityCheck := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
    Type: "openai-completion",
    Config: map[string]any{
        "prompt": "Rate this: {{text}}. Return JSON: {\"score\": <1-10>, \"pass\": <bool>}",
        "output_key": "quality_score",
    },
})

// Парсим JSON
nodeParseQuality := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
    Type: "json-parser",
    Config: map[string]any{
        "input_key": "quality_score",
    },
})

// Маршрутизируем по вложенному полю
nodeRoute := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
    Type: "conditional-router",
    Config: map[string]any{
        "input_key": "quality_score.pass",
        "routes": map[string]string{
            "true":  "accept",
            "false": "reject",
        },
    },
})
```

## Ожидаемый вывод

```
=== JSON Parser Node Demo ===

Example 1: Parsing JSON string for nested field access
-------------------------------------------------------

▶ Test Case 1: Active User

Status: completed

✅ Parsed Response (type: map[string]interface {}):
   User ID: 123
   User Name: John Doe
   User Email: john@example.com
   User Status: active
   User Roles: [admin user]


▶ Test Case 2: Inactive User

Status: completed

✅ Parsed Response (type: map[string]interface {}):
   User ID: 456
   User Name: Jane Smith
   User Email: jane@example.com
   User Status: inactive
   User Roles: [user]


▶ Test Case 3: Invalid JSON with fail_on_error=false

Status: completed
Original value preserved: this is not valid JSON

=== Demo Complete ===

Key Takeaways:
1. ✅ JSON strings are parsed into structured objects
2. ✅ Nested fields can be accessed using dot notation (e.g., 'user.status')
3. ✅ Conditional routing works with nested fields
4. ✅ Parse errors can be handled gracefully with fail_on_error=false
5. ✅ Already-parsed objects are passed through unchanged
```

## См. также

- [JSON Parser Documentation](../../docs/nodes/json-parser.md)
- [Customer Support AI Example](../customer-support-ai/) - использует json-parser для quality score
- [Conditional Router](../../docs/nodes/conditional-router.md)
