# MBFlow REST API - Quick Start Guide

Полный REST API для управления workflow и executions в MBFlow.

## Запуск сервера

### Локальный запуск

```bash
# Сборка сервера
go build -o mbflow-server ./cmd/server

# Запуск с настройками по умолчанию (порт 8181)
./mbflow-server

# Запуск с пользовательскими параметрами
./mbflow-server -port 9000 -cors=true -metrics=true

# Запуск с API key аутентификацией
./mbflow-server -api-keys="key1,key2,key3"
```

### Docker Compose

```bash
# Запустить все сервисы (API, PostgreSQL, Redis, Swagger UI)
docker-compose up -d

# Просмотр логов
docker-compose logs -f mbflow-api

# Остановка
docker-compose down
```

## Доступные endpoints

### Health Checks

- `GET /health` - Проверка здоровья сервера
- `GET /ready` - Проверка готовности сервера

### Управление Workflow

- `GET /api/v1/workflows` - Список всех workflows
- `GET /api/v1/workflows/{id}` - Получить workflow по ID
- `POST /api/v1/workflows` - Создать новый workflow
- `PUT /api/v1/workflows/{id}` - Обновить workflow
- `DELETE /api/v1/workflows/{id}` - Удалить workflow

### Управление Нодами

- `GET /api/v1/node-types` - Список доступных типов нод
- `GET /api/v1/workflows/{workflow_id}/nodes` - Список всех нод в workflow
- `GET /api/v1/workflows/{workflow_id}/nodes/{node_id}` - Получить ноду по ID
- `POST /api/v1/workflows/{workflow_id}/nodes` - Создать новую ноду
- `PUT /api/v1/workflows/{workflow_id}/nodes/{node_id}` - Обновить ноду
- `DELETE /api/v1/workflows/{workflow_id}/nodes/{node_id}` - Удалить ноду

### Управление Edges (Связями)

- `GET /api/v1/edge-types` - Список доступных типов связей
- `GET /api/v1/workflows/{workflow_id}/edges` - Список всех связей в workflow
- `GET /api/v1/workflows/{workflow_id}/edges/{edge_id}` - Получить связь по ID
- `POST /api/v1/workflows/{workflow_id}/edges` - Создать новую связь
- `PUT /api/v1/workflows/{workflow_id}/edges/{edge_id}` - Обновить связь
- `DELETE /api/v1/workflows/{workflow_id}/edges/{edge_id}` - Удалить связь
- `GET /api/v1/workflows/{workflow_id}/graph` - Визуализация графа workflow

### Управление Executions

- `GET /api/v1/executions` - Список всех executions
- `GET /api/v1/executions/{id}` - Получить execution по ID
- `POST /api/v1/executions` - Выполнить workflow
- `GET /api/v1/executions/{id}/events` - Получить события execution
- `POST /api/v1/executions/{id}/cancel` - Отменить execution (в разработке)
- `POST /api/v1/executions/{id}/pause` - Приостановить execution (в разработке)
- `POST /api/v1/executions/{id}/resume` - Возобновить execution (в разработке)

## Примеры использования

### 1. Создание простого workflow

```bash
curl -X POST http://localhost:8181/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "simple-workflow",
    "version": "1.0.0",
    "description": "Простой workflow с трансформацией",
    "nodes": [
      {
        "type": "start",
        "name": "start"
      },
      {
        "type": "transform",
        "name": "double",
        "config": {
          "transformations": {
            "result": "input * 2"
          }
        }
      },
      {
        "type": "end",
        "name": "end"
      }
    ],
    "edges": [
      {
        "from": "start",
        "to": "double",
        "type": "direct"
      },
      {
        "from": "double",
        "to": "end",
        "type": "direct"
      }
    ],
    "triggers": [
      {
        "type": "manual"
      }
    ]
  }'
```

### 2. Получение списка workflows

```bash
curl http://localhost:8181/api/v1/workflows
```

### 3. Выполнение workflow

```bash
curl -X POST http://localhost:8181/api/v1/executions \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "{workflow-id}",
    "variables": {
      "input": 42
    }
  }'
```

### 4. Получение информации об execution

```bash
curl http://localhost:8181/api/v1/executions/{execution-id}
```

### 5. Получение событий execution

```bash
curl http://localhost:8181/api/v1/executions/{execution-id}/events
```

### 6. Фильтрация executions

```bash
# По workflow ID
curl "http://localhost:8181/api/v1/executions?workflow_id={workflow-id}"

# По статусу
curl "http://localhost:8181/api/v1/executions?status=completed"
```

### 7. Работа с нодами

#### Получить список доступных типов нод

```bash
curl http://localhost:8181/api/v1/node-types
```

#### Получить все ноды в workflow

```bash
curl http://localhost:8181/api/v1/workflows/{workflow-id}/nodes
```

#### Создать новую ноду

```bash
curl -X POST http://localhost:8181/api/v1/workflows/{workflow-id}/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "type": "transform",
    "name": "calculate",
    "config": {
      "transformations": {
        "result": "input * 2"
      }
    }
  }'
```

#### Обновить существующую ноду

```bash
curl -X PUT http://localhost:8181/api/v1/workflows/{workflow-id}/nodes/{node-id} \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "transformations": {
        "result": "input * 3"
      }
    }
  }'
```

#### Удалить ноду

```bash
curl -X DELETE http://localhost:8181/api/v1/workflows/{workflow-id}/nodes/{node-id}
```

### 8. Работа со связями (edges)

#### Получить список типов связей

```bash
curl http://localhost:8181/api/v1/edge-types
```

Доступные типы:

- `direct` - простой последовательный переход
- `conditional` - переход по условию
- `fork` - начало параллельных веток
- `join` - синхронизация параллельных веток

#### Получить все связи в workflow

```bash
curl http://localhost:8181/api/v1/workflows/{workflow-id}/edges
```

#### Создать новую связь

```bash
# Простая связь
curl -X POST http://localhost:8181/api/v1/workflows/{workflow-id}/edges \
  -H "Content-Type: application/json" \
  -d '{
    "from": "start",
    "to": "process",
    "type": "direct"
  }'

# Условная связь
curl -X POST http://localhost:8181/api/v1/workflows/{workflow-id}/edges \
  -H "Content-Type: application/json" \
  -d '{
    "from": "router",
    "to": "success_handler",
    "type": "conditional",
    "config": {
      "expression": "status_code >= 200 && status_code < 300"
    }
  }'

# Fork связь (начало параллельных веток)
curl -X POST http://localhost:8181/api/v1/workflows/{workflow-id}/edges \
  -H "Content-Type: application/json" \
  -d '{
    "from": "parallel_start",
    "to": "branch_a",
    "type": "fork"
  }'
```

#### Обновить существующую связь

```bash
curl -X PUT http://localhost:8181/api/v1/workflows/{workflow-id}/edges/{edge-id} \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "expression": "status_code == 200"
    }
  }'
```

#### Удалить связь

```bash
curl -X DELETE http://localhost:8181/api/v1/workflows/{workflow-id}/edges/{edge-id}
```

#### Визуализация графа workflow

```bash
# Получить полный граф с нодами и связями
curl http://localhost:8181/api/v1/workflows/{workflow-id}/graph | jq

# Упрощенный вывод - только имена и связи
curl -s http://localhost:8181/api/v1/workflows/{workflow-id}/graph | \
  jq '{
    nodes: [.nodes[].name],
    connections: [.edges[] | {from: .from, to: .to, type: .type}]
  }'
```

## Примеры сложных workflows

### Workflow с параллельным выполнением

```json
{
  "name": "parallel-workflow",
  "version": "1.0.0",
  "description": "Workflow с параллельными ветками",
  "nodes": [
    {
      "type": "start",
      "name": "start"
    },
    {
      "type": "parallel",
      "name": "fork"
    },
    {
      "type": "transform",
      "name": "branch1",
      "config": {
        "transformations": {
          "result": "input * 2"
        }
      }
    },
    {
      "type": "transform",
      "name": "branch2",
      "config": {
        "transformations": {
          "result": "input * 3"
        }
      }
    },
    {
      "type": "parallel",
      "name": "join",
      "config": {
        "join_strategy": "wait_all"
      }
    },
    {
      "type": "end",
      "name": "end"
    }
  ],
  "edges": [
    {"from": "start", "to": "fork", "type": "direct"},
    {"from": "fork", "to": "branch1", "type": "fork"},
    {"from": "fork", "to": "branch2", "type": "fork"},
    {"from": "branch1", "to": "join", "type": "join"},
    {"from": "branch2", "to": "join", "type": "join"},
    {"from": "join", "to": "end", "type": "direct"}
  ],
  "triggers": [
    {"type": "manual"}
  ]
}
```

### Workflow с условной маршрутизацией

```json
{
  "name": "conditional-workflow",
  "version": "1.0.0",
  "nodes": [
    {"type": "start", "name": "start"},
    {"type": "conditional-route", "name": "router"},
    {
      "type": "transform",
      "name": "positive",
      "config": {
        "transformations": {
          "message": "\"Number is positive\""
        }
      }
    },
    {
      "type": "transform",
      "name": "negative",
      "config": {
        "transformations": {
          "message": "\"Number is negative or zero\""
        }
      }
    },
    {"type": "end", "name": "end"}
  ],
  "edges": [
    {"from": "start", "to": "router", "type": "direct"},
    {
      "from": "router",
      "to": "positive",
      "type": "conditional",
      "condition": {"expression": "input > 0"}
    },
    {
      "from": "router",
      "to": "negative",
      "type": "conditional",
      "condition": {"expression": "input <= 0"}
    },
    {"from": "positive", "to": "end", "type": "direct"},
    {"from": "negative", "to": "end", "type": "direct"}
  ],
  "triggers": [{"type": "manual"}]
}
```

## Аутентификация

Если настроены API ключи, включайте их в запросы:

```bash
# Использование X-API-Key header
curl -H "X-API-Key: your-api-key" http://localhost:8181/api/v1/workflows

# Использование Authorization Bearer token
curl -H "Authorization: Bearer your-api-key" http://localhost:8181/api/v1/workflows
```

## Интерактивная документация (Swagger UI)

После запуска через Docker Compose, Swagger UI доступен по адресу:

```
http://localhost:8081/docs
```

Swagger UI предоставляет:

- Интерактивный explorer API
- Примеры запросов/ответов
- Документацию схем
- Функцию "Try it out"

## Обработка ошибок

Все ошибки возвращаются в стандартном формате:

```json
{
  "error": "Описание ошибки"
}
```

Стандартные HTTP коды:

- `200 OK` - Успешный запрос
- `201 Created` - Ресурс создан успешно
- `204 No Content` - Ресурс удален успешно
- `400 Bad Request` - Невалидное тело запроса или параметры
- `404 Not Found` - Ресурс не найден
- `500 Internal Server Error` - Ошибка сервера
- `501 Not Implemented` - Функция еще не реализована

## Конфигурация

### Переменные окружения

- `PORT` - Порт сервера (по умолчанию: 8181)
- `LOG_LEVEL` - Уровень логирования: debug, info, warn, error (по умолчанию: info)
- `DATABASE_DSN` - DSN для подключения к PostgreSQL (по умолчанию: `postgres://postgres:postgres@localhost:5432/mbflow?sslmode=disable`)
- `CORS_ENABLED` - Включить CORS (по умолчанию: true)
- `METRICS_ENABLED` - Включить сбор метрик (по умолчанию: true)

### Флаги командной строки

- `-port` - Переопределить порт сервера
- `-cors` - Включить/отключить CORS
- `-metrics` - Включить/отключить сбор метрик
- `-api-keys` - API ключи через запятую для аутентификации

## Типы узлов (Node Types)

Доступные типы узлов для workflows:

- `start` - Начальный узел
- `end` - Конечный узел
- `transform` - Трансформация данных (expr-lang)
- `http` - HTTP запросы
- `conditional-route` - Условная маршрутизация
- `parallel` - Параллельное выполнение
- `openai-completion` - Интеграция с OpenAI
- `json-parser` - Парсинг JSON

## Типы рёбер (Edge Types)

- `direct` - Простой последовательный переход
- `conditional` - Переход по условию
- `fork` - Начало параллельных веток
- `join` - Синхронизация параллельных веток

## Типы триггеров (Trigger Types)

- `manual` - Ручной запуск
- `http` - HTTP webhook
- `schedule` - По расписанию
- `event` - По событию

## Разработка и тестирование

### Запуск тестов

```bash
# Все тесты
go test ./...

# Тесты с покрытием
go test -cover ./...

# Тесты конкретного пакета
go test ./internal/infrastructure/api/rest/...
```

### Сборка

```bash
# Сборка бинарника сервера
go build -o mbflow-server ./cmd/server

# Сборка Docker образа
docker build -t mbflow-api:latest .
```

## Структура проекта

```
mbflow/
├── cmd/
│   └── server/          # REST API server
├── internal/
│   ├── domain/          # Domain layer (DDD)
│   ├── application/     # Application layer
│   └── infrastructure/
│       ├── api/rest/    # REST API handlers
│       ├── storage/     # Storage implementations
│       └── monitoring/  # Monitoring & observability
├── docs/
│   └── api/
│       ├── openapi.yaml # OpenAPI specification
│       └── README.md    # API documentation
├── examples/            # Example workflows
├── docker-compose.yml   # Docker Compose configuration
└── Dockerfile          # Docker image definition
```

## Поддержка

Для вопросов и отчетов об ошибках:
<https://github.com/smilemakc/mbflow/issues>

## Лицензия

MIT License
