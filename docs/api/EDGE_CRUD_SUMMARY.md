# Edge CRUD API - Summary

## Что было реализовано

Полный CRUD API для управления связями (edges) между нодами в workflows. Edges определяют поток управления и данных через workflow, поддерживая последовательное выполнение, условную маршрутизацию и параллельную обработку.

## Созданные файлы

### 1. Handlers
- **handlers_edges.go** - полная реализация CRUD операций для связей
  - `handleListEdges` - список всех связей в workflow
  - `handleGetEdge` - получение конкретной связи
  - `handleCreateEdge` - создание новой связи
  - `handleUpdateEdge` - обновление существующей связи
  - `handleDeleteEdge` - удаление связи
  - `handleGetEdgeTypes` - справочник доступных типов связей
  - `handleGetWorkflowGraph` - визуализация графа workflow

### 2. Routes
Обновлен **server.go** с новыми эндпоинтами:
- `GET /api/v1/edge-types`
- `GET /api/v1/workflows/{workflow_id}/edges`
- `GET /api/v1/workflows/{workflow_id}/edges/{edge_id}`
- `POST /api/v1/workflows/{workflow_id}/edges`
- `PUT /api/v1/workflows/{workflow_id}/edges/{edge_id}`
- `DELETE /api/v1/workflows/{workflow_id}/edges/{edge_id}`
- `GET /api/v1/workflows/{workflow_id}/graph`

### 3. Documentation
- **openapi.yaml** - обновлена OpenAPI спецификация с schemas и paths для edges
- **EDGES_API.md** - полная документация с примерами использования (21KB)
- **EDGE_CRUD_SUMMARY.md** - техническое резюме реализации
- **API_README.md** - обновлена секция с примерами работы с edges

## API ENDPOINTS

```
GET    /api/v1/edge-types                              → Catalog
GET    /api/v1/workflows/{id}/edges                    → List
GET    /api/v1/workflows/{id}/edges/{edge_id}          → Read
POST   /api/v1/workflows/{id}/edges                    → Create
PUT    /api/v1/workflows/{id}/edges/{edge_id}          → Update
DELETE /api/v1/workflows/{id}/edges/{edge_id}          → Delete
GET    /api/v1/workflows/{id}/graph                    → Visualize
```

## Ключевые возможности

### 1. Динамическое управление связями
Можно добавлять, изменять и удалять связи в существующих workflows без пересоздания всего workflow.

```bash
# Добавить новую связь
curl -X POST /api/v1/workflows/{id}/edges \
  -d '{"from":"start","to":"process","type":"direct"}'

# Обновить связь
curl -X PUT /api/v1/workflows/{id}/edges/{edge_id} \
  -d '{"config":{"expression":"status == 200"}}'

# Удалить связь
curl -X DELETE /api/v1/workflows/{id}/edges/{edge_id}
```

### 2. Справочник типов связей
GET `/api/v1/edge-types` возвращает список всех доступных типов:
- **direct** - простой последовательный переход
- **conditional** - переход по условию (expr-lang)
- **fork** - начало параллельных веток
- **join** - синхронизация параллельных веток

### 3. Двойное представление
Edges используют удобное для пользователя представление:
- API принимает имена нод (`"from": "start"`)
- Внутри хранятся UUID нод
- Response содержит оба варианта для гибкости:
  ```json
  {
    "from": "start",
    "from_id": "uuid-1",
    "to": "process",
    "to_id": "uuid-2"
  }
  ```

### 4. Визуализация графа
GET `/api/v1/workflows/{id}/graph` возвращает полное представление:
```json
{
  "workflow_id": "uuid",
  "nodes": [
    {"id": "uuid-1", "name": "start", "type": "start"},
    {"id": "uuid-2", "name": "process", "type": "transform"}
  ],
  "edges": [
    {
      "id": "edge-uuid",
      "from": "start",
      "from_id": "uuid-1",
      "to": "process",
      "to_id": "uuid-2",
      "type": "direct"
    }
  ]
}
```

### 5. Валидация
- Автоматическая валидация типов связей
- Проверка существования исходной и целевой нод
- Валидация конфигурации согласно типу связи
- Предотвращение циклов в графе (DAG validation)

### 6. Безопасность
- Все операции проходят через middleware chain
- Поддержка API key аутентификации
- CORS защита
- Rate limiting
- Recovery middleware

## Поддерживаемые типы связей

### Control Flow (Управление потоком)
- **direct** - простой последовательный переход
- **conditional** - условный переход на основе выражений expr-lang

### Parallel Processing (Параллельная обработка)
- **fork** - разветвление на параллельные ветки
- **join** - синхронизация параллельных веток

## Примеры использования

### Пример 1: Построение линейного workflow

```bash
# 1. Создать базовый workflow
WORKFLOW_ID=$(curl -X POST /api/v1/workflows \
  -d '{
    "name":"linear-flow",
    "version":"1.0",
    "nodes":[
      {"type":"start","name":"start"},
      {"type":"transform","name":"process"},
      {"type":"end","name":"end"}
    ],
    "edges":[],
    "triggers":[{"type":"manual"}]
  }' | jq -r '.id')

# 2. Создать связи
curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"start","to":"process","type":"direct"}'

curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"process","to":"end","type":"direct"}'

# 3. Визуализировать результат
curl /api/v1/workflows/$WORKFLOW_ID/graph | jq
```

### Пример 2: Условная маршрутизация

```bash
# Создать workflow с роутером
WORKFLOW_ID=$(curl -X POST /api/v1/workflows \
  -d '{
    "name":"conditional-router",
    "version":"1.0",
    "nodes":[
      {"type":"start","name":"start"},
      {"type":"http","name":"api_call"},
      {"type":"conditional-route","name":"router"},
      {"type":"transform","name":"success"},
      {"type":"transform","name":"error"},
      {"type":"end","name":"end"}
    ],
    "edges":[],
    "triggers":[{"type":"manual"}]
  }' | jq -r '.id')

# Добавить прямые связи
curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"start","to":"api_call","type":"direct"}'

curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"api_call","to":"router","type":"direct"}'

# Добавить условные связи
curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{
    "from":"router",
    "to":"success",
    "type":"conditional",
    "config":{"expression":"status_code >= 200 && status_code < 300"}
  }'

curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{
    "from":"router",
    "to":"error",
    "type":"conditional",
    "config":{"expression":"status_code >= 400"}
  }'

# Соединить с концом
curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"success","to":"end","type":"direct"}'

curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"error","to":"end","type":"direct"}'
```

### Пример 3: Параллельное выполнение

```bash
# Создать workflow с параллельными ветками
WORKFLOW_ID=$(curl -X POST /api/v1/workflows \
  -d '{
    "name":"parallel-processing",
    "version":"1.0",
    "nodes":[
      {"type":"start","name":"start"},
      {"type":"parallel","name":"fork","config":{"mode":"fork"}},
      {"type":"transform","name":"branch_a"},
      {"type":"transform","name":"branch_b"},
      {"type":"transform","name":"branch_c"},
      {"type":"parallel","name":"join","config":{"mode":"join","join_strategy":"wait_all"}},
      {"type":"end","name":"end"}
    ],
    "edges":[],
    "triggers":[{"type":"manual"}]
  }' | jq -r '.id')

# Start → Fork
curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"start","to":"fork","type":"direct"}'

# Fork → Branches (параллельно)
curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"fork","to":"branch_a","type":"fork"}'

curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"fork","to":"branch_b","type":"fork"}'

curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"fork","to":"branch_c","type":"fork"}'

# Branches → Join (синхронизация)
curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"branch_a","to":"join","type":"join"}'

curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"branch_b","to":"join","type":"join"}'

curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"branch_c","to":"join","type":"join"}'

# Join → End
curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"join","to":"end","type":"direct"}'
```

### Пример 4: Динамическое изменение маршрутизации

```bash
# Найти связь для обновления
EDGE_ID=$(curl -s /api/v1/workflows/$WORKFLOW_ID/edges | \
  jq -r '.[] | select(.from == "router" and .to == "success") | .id')

# Изменить условие
curl -X PUT /api/v1/workflows/$WORKFLOW_ID/edges/$EDGE_ID \
  -d '{
    "config":{
      "expression":"status_code == 200 || status_code == 201"
    }
  }'

# Переключить ноду на другую цель
curl -X DELETE /api/v1/workflows/$WORKFLOW_ID/edges/$EDGE_ID

curl -X POST /api/v1/workflows/$WORKFLOW_ID/edges \
  -d '{"from":"router","to":"new_target","type":"conditional","config":{...}}'
```

## Технические детали

### Архитектура
- RESTful дизайн с использованием стандартных HTTP методов
- Следование принципам DDD (Domain-Driven Design)
- Работа через Workflow aggregate для поддержания консистентности
- Валидация DAG структуры при изменениях

### Обработка обновлений
При обновлении связи (PUT) создается новая связь с новым ID, так как edges в domain layer являются immutable. Старая связь удаляется, новая добавляется.

```go
// В handlers_edges.go
workflow.RemoveEdge(edgeID)
newEdgeID, err := workflow.AddEdge(fromID, toID, edgeType, config)
```

### Response Types
```typescript
interface EdgeDetailResponse {
  id: string;           // UUID связи
  workflow_id: string;  // UUID workflow
  from: string;         // Имя исходной ноды
  from_id: string;      // UUID исходной ноды
  to: string;           // Имя целевой ноды
  to_id: string;        // UUID целевой ноды
  type: string;         // Тип связи
  config?: object;      // Конфигурация связи
}
```

### Node Name Resolution
Handlers автоматически разрешают имена нод в ID:

```go
// Создание mapping name → ID
nodeNameToID := make(map[string]uuid.UUID)
nodeIDToName := make(map[uuid.UUID]string)

for _, node := range nodes {
    nodeNameToID[node.Name()] = node.ID()
    nodeIDToName[node.ID()] = node.Name()
}

// Использование при создании
fromID := nodeNameToID[req.From]
toID := nodeNameToID[req.To]
```

## Интеграция с другими API

Edge API тесно интегрирован с:
- **Workflow API** - для получения workflow перед операциями
- **Node API** - для разрешения имен нод в ID
- **Execution API** - измененные связи используются в новых executions
- **Graph Visualization** - `/graph` endpoint для визуализации

## Паттерны использования

### Pattern 1: Условная маршрутизация с fallback
```bash
# Всегда добавляйте default путь
curl -X POST /api/v1/workflows/$WF/edges \
  -d '{
    "from":"router",
    "to":"default_handler",
    "type":"conditional",
    "config":{"expression":"true"}
  }'
```

### Pattern 2: Fork/Join для параллельной обработки
```bash
# Fork node → N branches → Join node
# Join strategy: wait_all (ждать все ветки)
```

### Pattern 3: Динамическое A/B тестирование
```bash
# Создать две версии обработки
# Использовать conditional edges с условиями на % трафика
```

### Pattern 4: Error handling paths
```bash
# Всегда добавлять error edges
curl -X POST /api/v1/workflows/$WF/edges \
  -d '{
    "from":"api_call",
    "to":"error_handler",
    "type":"conditional",
    "config":{"expression":"error != null"}
  }'
```

## Следующие шаги

Возможные улучшения:
1. **Batch operations** - массовое создание/обновление связей
2. **Edge templates** - готовые шаблоны связей
3. **Validation API** - предварительная валидация конфигурации
4. **Visual editor integration** - интеграция с визуальным редактором
5. **Edge versioning** - отслеживание версий связей
6. **Complex routing** - поддержка более сложных условий маршрутизации

## Тестирование

```bash
# Запуск сервера
go run cmd/server/main.go -port 8181

# Тестирование API
curl http://localhost:8181/api/v1/edge-types
curl http://localhost:8181/api/v1/workflows/{id}/edges
curl http://localhost:8181/api/v1/workflows/{id}/graph

# Swagger UI (при использовании Docker Compose)
open http://localhost:8081/docs
```

## Код статистика

```
handlers_edges.go:    ~400 lines
Documentation:        ~1500 lines
OpenAPI updates:      ~200 lines
Total additions:      ~2100 lines
```

## ✅ TESTING

- ✓ handlers_edges.go создан
- ✓ Endpoints зарегистрированы в server.go
- ✓ OpenAPI spec обновлен
- ✓ Документация создана
- ⏳ Build test (pending)
- ⏳ Integration tests (pending)

## Поддержка

Для вопросов и отчетов об ошибках:
https://github.com/smilemakc/mbflow/issues
