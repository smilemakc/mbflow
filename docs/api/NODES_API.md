# Node Management API

Полное CRUD API для управления нодами в workflows.

## Обзор

Node Management API позволяет динамически управлять узлами в существующих workflows без необходимости пересоздавать весь workflow.

## Endpoints

### 1. Получить список доступных типов нод

```bash
GET /api/v1/node-types
```

**Пример запроса:**

```bash
curl http://localhost:8080/api/v1/node-types
```

**Пример ответа:**

```json
[
  {
    "type": "start",
    "name": "Start Node",
    "description": "Entry point of the workflow",
    "category": "control"
  },
  {
    "type": "transform",
    "name": "Transform",
    "description": "Transform data using expressions",
    "category": "data",
    "config_schema": {
      "transformations": "map[string]string - expressions to evaluate"
    }
  },
  {
    "type": "openai-completion",
    "name": "OpenAI Completion",
    "description": "Call OpenAI API for text completion",
    "category": "ai",
    "config_schema": {
      "api_key": "string - OpenAI API key",
      "model": "string - model name (gpt-4, gpt-3.5-turbo, etc.)",
      "prompt": "string - prompt template"
    }
  }
]
```

### 2. Получить список всех нод в workflow

```bash
GET /api/v1/workflows/{workflow_id}/nodes
```

**Пример запроса:**

```bash
curl http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes
```

**Пример ответа:**

```json
[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "start",
    "name": "start",
    "config": {}
  },
  {
    "id": "223e4567-e89b-12d3-a456-426614174001",
    "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "transform",
    "name": "double",
    "config": {
      "transformations": {
        "result": "input * 2"
      }
    }
  },
  {
    "id": "323e4567-e89b-12d3-a456-426614174002",
    "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "end",
    "name": "end",
    "config": {}
  }
]
```

### 3. Получить конкретную ноду

```bash
GET /api/v1/workflows/{workflow_id}/nodes/{node_id}
```

**Пример запроса:**

```bash
curl http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes/223e4567-e89b-12d3-a456-426614174001
```

**Пример ответа:**

```json
{
  "id": "223e4567-e89b-12d3-a456-426614174001",
  "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "transform",
  "name": "double",
  "config": {
    "transformations": {
      "result": "input * 2"
    }
  }
}
```

### 4. Создать новую ноду

```bash
POST /api/v1/workflows/{workflow_id}/nodes
```

**Пример запроса - Transform Node:**

```bash
curl -X POST http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "type": "transform",
    "name": "triple",
    "config": {
      "transformations": {
        "result": "input * 3"
      }
    }
  }'
```

**Пример запроса - HTTP Node:**

```bash
curl -X POST http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "type": "http",
    "name": "fetch-data",
    "config": {
      "url": "https://api.example.com/data",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer {{api_token}}"
      }
    }
  }'
```

**Пример запроса - OpenAI Node:**

```bash
curl -X POST http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "type": "openai-completion",
    "name": "ai-summary",
    "config": {
      "api_key": "sk-...",
      "model": "gpt-4",
      "prompt": "Summarize the following text: {{text}}"
    }
  }'
```

**Пример запроса - Conditional Route Node:**

```bash
curl -X POST http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "type": "conditional-route",
    "name": "check-value",
    "config": {
      "routes": [
        {
          "condition": "value > 100",
          "target": "high-path"
        },
        {
          "condition": "value <= 100",
          "target": "low-path"
        }
      ]
    }
  }'
```

**Пример ответа:**

```json
{
  "id": "423e4567-e89b-12d3-a456-426614174003",
  "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "transform",
  "name": "triple",
  "config": {
    "transformations": {
      "result": "input * 3"
    }
  }
}
```

### 5. Обновить существующую ноду

```bash
PUT /api/v1/workflows/{workflow_id}/nodes/{node_id}
```

**Пример запроса - обновление конфигурации:**

```bash
curl -X PUT http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes/223e4567-e89b-12d3-a456-426614174001 \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "transformations": {
        "result": "input * 4",
        "squared": "input * input"
      }
    }
  }'
```

**Пример запроса - изменение имени:**

```bash
curl -X PUT http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes/223e4567-e89b-12d3-a456-426614174001 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quadruple"
  }'
```

**Пример запроса - полное обновление:**

```bash
curl -X PUT http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes/223e4567-e89b-12d3-a456-426614174001 \
  -H "Content-Type: application/json" \
  -d '{
    "type": "transform",
    "name": "multiply",
    "config": {
      "transformations": {
        "result": "input * multiplier"
      }
    }
  }'
```

**Пример ответа:**

```json
{
  "id": "523e4567-e89b-12d3-a456-426614174004",
  "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "transform",
  "name": "multiply",
  "config": {
    "transformations": {
      "result": "input * multiplier"
    }
  }
}
```

**Примечание:** При обновлении создается новая нода с новым ID, так как ноды в домене неизменяемы.

### 6. Удалить ноду

```bash
DELETE /api/v1/workflows/{workflow_id}/nodes/{node_id}
```

**Пример запроса:**

```bash
curl -X DELETE http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/nodes/223e4567-e89b-12d3-a456-426614174001
```

**Ответ:** HTTP 204 No Content

## Примеры конфигураций для различных типов нод

### Transform Node

```json
{
  "type": "transform",
  "name": "calculate",
  "config": {
    "transformations": {
      "total": "price * quantity",
      "tax": "total * 0.2",
      "final_price": "total + tax"
    }
  }
}
```

### HTTP Node

```json
{
  "type": "http",
  "name": "api-call",
  "config": {
    "url": "https://api.example.com/users/{{user_id}}",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {{token}}",
      "Content-Type": "application/json"
    },
    "timeout": "30s"
  }
}
```

### OpenAI Completion Node

```json
{
  "type": "openai-completion",
  "name": "generate-content",
  "config": {
    "api_key": "sk-...",
    "model": "gpt-4",
    "prompt": "Write a blog post about {{topic}}",
    "max_tokens": 1000,
    "temperature": 0.7
  }
}
```

### Conditional Route Node

```json
{
  "type": "conditional-route",
  "name": "router",
  "config": {
    "default_route": "fallback"
  }
}
```

### Parallel Node (Fork)

```json
{
  "type": "parallel",
  "name": "fork",
  "config": {
    "mode": "fork"
  }
}
```

### Parallel Node (Join)

```json
{
  "type": "parallel",
  "name": "join",
  "config": {
    "mode": "join",
    "join_strategy": "wait_all",
    "timeout": "60s"
  }
}
```

### JSON Parser Node

```json
{
  "type": "json-parser",
  "name": "parse-response",
  "config": {
    "source_field": "response_body",
    "schema": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "name": {"type": "string"}
      }
    }
  }
}
```

### Telegram Message Node

```json
{
  "type": "telegram-message",
  "name": "notify",
  "config": {
    "bot_token": "123456:ABC-DEF...",
    "chat_id": "{{chat_id}}",
    "message": "Task completed: {{task_name}}",
    "parse_mode": "HTML"
  }
}
```

## Сценарии использования

### Сценарий 1: Добавление обработки ошибок

```bash
# 1. Создать workflow
WORKFLOW_ID=$(curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-workflow",
    "version": "1.0.0",
    "nodes": [
      {"type": "start", "name": "start"},
      {"type": "end", "name": "end"}
    ],
    "edges": [
      {"from": "start", "to": "end", "type": "direct"}
    ],
    "triggers": [{"type": "manual"}]
  }' | jq -r '.id')

# 2. Добавить HTTP ноду
HTTP_NODE_ID=$(curl -X POST http://localhost:8080/api/v1/workflows/$WORKFLOW_ID/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "type": "http",
    "name": "api-call",
    "config": {
      "url": "https://api.example.com/data",
      "method": "GET"
    }
  }' | jq -r '.id')

# 3. Добавить обработчик ошибок
ERROR_NODE_ID=$(curl -X POST http://localhost:8080/api/v1/workflows/$WORKFLOW_ID/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "type": "telegram-message",
    "name": "error-notification",
    "config": {
      "bot_token": "YOUR_TOKEN",
      "chat_id": "YOUR_CHAT_ID",
      "message": "API call failed: {{error}}"
    }
  }' | jq -r '.id')
```

### Сценарий 2: Динамическое изменение логики

```bash
# Обновить transform ноду с новой логикой
curl -X PUT http://localhost:8080/api/v1/workflows/$WORKFLOW_ID/nodes/$NODE_ID \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "transformations": {
        "discount": "price * 0.1",
        "final_price": "price - discount"
      }
    }
  }'
```

### Сценарий 3: A/B тестирование

```bash
# Создать две версии обработки
curl -X POST http://localhost:8080/api/v1/workflows/$WORKFLOW_ID/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "type": "transform",
    "name": "variant-a",
    "config": {
      "transformations": {
        "result": "input * 1.5"
      }
    }
  }'

curl -X POST http://localhost:8080/api/v1/workflows/$WORKFLOW_ID/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "type": "transform",
    "name": "variant-b",
    "config": {
      "transformations": {
        "result": "input * 2.0"
      }
    }
  }'
```

## Обработка ошибок

### 400 Bad Request

```json
{
  "error": "Node type is required"
}
```

```json
{
  "error": "Failed to add node: node with name 'duplicate' already exists"
}
```

### 404 Not Found

```json
{
  "error": "Workflow not found"
}
```

```json
{
  "error": "Node not found"
}
```

### 500 Internal Server Error

```json
{
  "error": "Failed to save workflow"
}
```

## Best Practices

1. **Уникальные имена**: Всегда используйте уникальные имена для нод в пределах одного workflow
2. **Валидация конфигурации**: Проверяйте конфигурацию перед созданием ноды
3. **Использование типов**: Используйте GET /api/v1/node-types для получения актуальной информации о доступных типах
4. **Версионирование**: При значительных изменениях создавайте новую версию workflow
5. **Тестирование**: Тестируйте изменения на копии workflow перед применением в production

## Интеграция с Workflow API

Node API тесно интегрирован с Workflow API. После создания/обновления/удаления нод не забывайте также обновить edges (рёбра) для правильной маршрутизации.

Пример:

```bash
# 1. Добавить новую ноду
NODE_ID=$(curl -X POST .../nodes -d '{"type":"transform","name":"new-step"}' | jq -r '.id')

# 2. Добавить edge к этой ноде
curl -X POST .../workflows/$WORKFLOW_ID \
  -d '{
    "edges": [
      {"from": "previous-step", "to": "new-step", "type": "direct"}
    ]
  }'
```

## Ограничения

1. **Immutability**: При обновлении ноды создается новая нода с новым ID
2. **Валидация**: Нельзя создать ноду с дублирующимся именем в одном workflow
3. **Зависимости**: Удаление ноды не удаляет автоматически связанные edges (требуется ручная очистка)

## Дополнительные ресурсы

- [OpenAPI Specification](openapi.yaml) - полная спецификация API
- [Workflow API Documentation](README.md) - документация по Workflow API
- [Node Types Reference](../NODE_TYPES.md) - подробное описание всех типов нод
