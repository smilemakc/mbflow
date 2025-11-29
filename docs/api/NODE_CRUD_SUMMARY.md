# Node CRUD API - Summary

## Что было реализовано

Полный CRUD API для управления нодами в workflows с поддержкой динамического изменения структуры workflow без необходимости пересоздания.

## Созданные файлы

### 1. Handlers
- **handlers_nodes.go** - полная реализация CRUD операций для нод
  - `handleListNodes` - список всех нод в workflow
  - `handleGetNode` - получение конкретной ноды
  - `handleCreateNode` - создание новой ноды
  - `handleUpdateNode` - обновление существующей ноды
  - `handleDeleteNode` - удаление ноды
  - `handleGetNodeTypes` - справочник доступных типов нод

### 2. Routes
Обновлен **server.go** с новыми эндпоинтами:
- `GET /api/v1/node-types`
- `GET /api/v1/workflows/{workflow_id}/nodes`
- `GET /api/v1/workflows/{workflow_id}/nodes/{node_id}`
- `POST /api/v1/workflows/{workflow_id}/nodes`
- `PUT /api/v1/workflows/{workflow_id}/nodes/{node_id}`
- `DELETE /api/v1/workflows/{workflow_id}/nodes/{node_id}`

### 3. Documentation
- **openapi.yaml** - обновлена OpenAPI спецификация с schemas и paths для нод
- **NODES_API.md** - полная документация с примерами использования
- **API_README.md** - обновлен главный README с секцией о Node API

## Ключевые возможности

### 1. Динамическое управление нодами
Можно добавлять, изменять и удалять ноды в существующих workflows без пересоздания всего workflow.

```bash
# Добавить новую ноду
curl -X POST /api/v1/workflows/{id}/nodes \
  -d '{"type":"transform","name":"new-step","config":{...}}'

# Обновить ноду
curl -X PUT /api/v1/workflows/{id}/nodes/{node_id} \
  -d '{"config":{...}}'

# Удалить ноду
curl -X DELETE /api/v1/workflows/{id}/nodes/{node_id}
```

### 2. Справочник типов нод
GET `/api/v1/node-types` возвращает список всех доступных типов с описаниями и схемами конфигурации:
- start, end - контрольные узлы
- transform - трансформация данных
- http - HTTP запросы
- conditional-route - условная маршрутизация
- parallel - параллельное выполнение
- openai-completion - интеграция с OpenAI
- json-parser - парсинг JSON
- telegram-message - отправка сообщений в Telegram

### 3. Валидация
- Автоматическая валидация типов нод
- Проверка уникальности имен нод
- Валидация конфигурации согласно типу ноды

### 4. Безопасность
- Все операции проходят через middleware chain
- Поддержка API key аутентификации
- CORS защита
- Rate limiting

## Примеры использования

### Пример 1: Построение pipeline пошагово

```bash
# 1. Создать базовый workflow
WORKFLOW_ID=$(curl -X POST /api/v1/workflows \
  -d '{"name":"my-pipeline","version":"1.0","nodes":[
    {"type":"start","name":"start"},
    {"type":"end","name":"end"}
  ],"edges":[{"from":"start","to":"end","type":"direct"}],"triggers":[{"type":"manual"}]}' \
  | jq -r '.id')

# 2. Добавить HTTP ноду для получения данных
curl -X POST /api/v1/workflows/$WORKFLOW_ID/nodes \
  -d '{
    "type":"http",
    "name":"fetch-data",
    "config":{
      "url":"https://api.example.com/data",
      "method":"GET"
    }
  }'

# 3. Добавить Transform ноду для обработки
curl -X POST /api/v1/workflows/$WORKFLOW_ID/nodes \
  -d '{
    "type":"transform",
    "name":"process",
    "config":{
      "transformations":{
        "result":"data.value * 2"
      }
    }
  }'

# 4. Добавить OpenAI ноду для анализа
curl -X POST /api/v1/workflows/$WORKFLOW_ID/nodes \
  -d '{
    "type":"openai-completion",
    "name":"analyze",
    "config":{
      "api_key":"sk-...",
      "model":"gpt-4",
      "prompt":"Analyze this data: {{result}}"
    }
  }'
```

### Пример 2: Обновление логики на лету

```bash
# Изменить конфигурацию transform ноды без остановки workflow
curl -X PUT /api/v1/workflows/$WORKFLOW_ID/nodes/$NODE_ID \
  -d '{
    "config":{
      "transformations":{
        "result":"data.value * 3",
        "timestamp":"now()"
      }
    }
  }'
```

### Пример 3: A/B тестирование

```bash
# Создать две версии обработки
curl -X POST /api/v1/workflows/$WORKFLOW_ID/nodes \
  -d '{"type":"transform","name":"variant-a","config":{...}}'

curl -X POST /api/v1/workflows/$WORKFLOW_ID/nodes \
  -d '{"type":"transform","name":"variant-b","config":{...}}'

# Добавить роутер для распределения
curl -X POST /api/v1/workflows/$WORKFLOW_ID/nodes \
  -d '{
    "type":"conditional-route",
    "name":"ab-router",
    "config":{...}
  }'
```

## Технические детали

### Архитектура
- RESTful дизайн с использованием стандартных HTTP методов
- Следование принципам DDD (Domain-Driven Design)
- Работа через Workflow aggregate для поддержания консистентности
- Event Sourcing для отслеживания изменений

### Обработка обновлений
При обновлении ноды (PUT) создается новая нода с новым ID, так как ноды в domain layer являются immutable. Старая нода удаляется, новая добавляется.

### Response Types
```typescript
interface NodeDetailResponse {
  id: string;           // UUID ноды
  workflow_id: string;  // UUID workflow
  type: string;         // Тип ноды
  name: string;         // Имя ноды
  config: object;       // Конфигурация
  description?: string; // Опциональное описание
}
```

## Интеграция с другими API

Node API тесно интегрирован с:
- **Workflow API** - для получения workflow перед операциями с нодами
- **Execution API** - измененные ноды используются в новых executions
- **Edge API** (будущая разработка) - для управления связями между нодами

## Следующие шаги

Возможные улучшения:
1. **Edge CRUD API** - управление ребрами между нодами
2. **Batch operations** - массовое создание/обновление нод
3. **Node templates** - готовые шаблоны нод
4. **Validation API** - предварительная валидация конфигурации
5. **Version control** - отслеживание версий нод

## Тестирование

```bash
# Запуск сервера
go run cmd/server/main.go -port 8181

# Тестирование API
curl http://localhost:8181/api/v1/node-types
curl http://localhost:8181/api/v1/workflows/{id}/nodes

# Swagger UI (при использовании Docker Compose)
open http://localhost:8081/docs
```

## Поддержка

Для вопросов и отчетов об ошибках:
https://github.com/smilemakc/mbflow/issues
