# MBFlow - Концепция проекта

## 🎯 Что такое MBFlow ?

**MBFlow ** - это **универсальная платформа для автоматизации бизнес-процессов** через систему направленных ациклических графов (DAG - Directed Acyclic Graph).

---

## 📌 Основная идея

Вместо того чтобы писать линейный код вида:
```
Шаг 1 → Шаг 2 → Шаг 3 → Результат
```

Система позволяет строить **гибкие, визуальные рабочие процессы** вида:
```
                    ┌─→ [Обработка A] ─┐
[Получить данные] ──┤                    ├─→ [Слить результаты] → [Результат]
                    └─→ [Обработка B] ─┘
```

---

## 💡 Для чего нужна система?

### 1. **Автоматизация сложных процессов**
- Обработка данных из API
- Интеграция с внешними сервисами (OpenAI, Anthropic, etc.)
- Трансформация и валидация данных
- Условное выполнение (if-then логика)
- Параллельное выполнение (DAG позволяет выполнять независимые узлы одновременно)

### 2. **Без кода (Low-Code)**
- Строить workflow через **визуальный редактор** (как Zapier, IFTTT)
- Соединять компоненты drag-and-drop
- Не нужно писать код - конфигурируешь JSON

### 3. **Интеграция с LLM**
- Вызывать OpenAI/Anthropic Claude из workflow
- Использовать результаты LLM как input для следующих шагов
- Строить AI-powered automation

### 4. **Встраивание в приложения**
- SDK позволяет использовать как **подключаемый модуль**
- Встроить в существующее Go приложение
- Управлять workflows через публичный API (SDK)
- Использовать собственное БД соединение

---

## 🏗️ Архитектура системы

```
┌─────────────────────────────────────────────────────────────┐
│                        REST API                              │
│         (создание, управление, выполнение workflow)          │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    SDK Client                                │
│  (для встраивания в сторонние Go приложения)                │
│                                                               │
│  • WorkflowAPI      • ExecutionAPI    • TriggerAPI           │
│  • Custom Executors • Real-time Watch                        │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│               Orchestration Layer                            │
│                                                               │
│  • DAG Validator (проверка циклов, structure)               │
│  • Execution Engine (запуск workflow)                        │
│  • Node Runner (выполнение узлов с retry)                   │
│  • Executor Registry (управление executors)                 │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│              Node Executors (Что может делать)              │
│                                                               │
│  • LLM Executor      (OpenAI, Anthropic Claude)             │
│  • HTTP Executor     (вызов внешних API)                    │
│  • Data Adapter      (трансформация данных)                 │
│  • Conditional       (if-then логика)                       │
│  • Merge             (объединение результатов)              │
│  • Custom Executors  (пишите свои)                          │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│           Trigger  (Когда выполнять)                  │
│                                                               │
│  • Time Trigger      (по расписанию - cron)                 │
│  • Webhook Trigger   (по HTTP запросу)                      │
│  • Manual Trigger    (вручную запустить)                    │
│  • Event Trigger     (по событию)                           │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   Persistence Layer                          │
│                                                               │
│  • PostgreSQL (хранение workflows, executions, history)     │
│  • Bun ORM (простая работа с БД)                            │
│  • Redis (кеш, очередь событий)                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔄 Как это работает? (Жизненный цикл)

### Этап 1: Создание Workflow

```
Пользователь/SDK
    ↓
Создает nodes (узлы)
    ↓
Соединяет edges (связи)
    ↓
Валидирует DAG (проверка циклов, структуры)
    ↓
Публикует workflow
    ↓
Workflow готов к запуску
```

**Пример:**
```json
{
  "workflow": {
    "id": "wf-123",
    "name": "Process User Data",
    "nodes": [
      {
        "id": "n1",
        "name": "Fetch from API",
        "type": "HTTP",
        "config": {
          "url": "https://api.example.com/users",
          "method": "GET"
        }
      },
      {
        "id": "n2",
        "name": "Transform Data",
        "type": "DATA_ADAPTER",
        "config": {
          "operation": "map",
          "mapping": {"username": "name"}
        }
      }
    ],
    "edges": [
      {
        "source": "n1",
        "target": "n2",
        "type": "DIRECT"
      }
    ]
  }
}
```

### Этап 2: Запуск Workflow

```
Trigger (Time, Webhook, Manual, Event)
    ↓
API получает запрос на запуск
    ↓
Создается Execution запись (история)
    ↓
DAG Validator проверяет структуру
    ↓
Execution Engine вычисляет порядок выполнения (topological sort)
    ↓
Для каждого узла в порядке:
    • Получить executor
    • Передать input данные
    • Выполнить с timeout
    • Обработать ошибки (retry с exponential backoff)
    • Сохранить output
    ↓
Объединить результаты всех узлов
    ↓
Сохранить финальный результат
```

### Этап 3: Выполнение Узла (с retry)

```
NodeRunner.run()
    ↓
Попытка 1:
    • Выполнить узел
    • Если успех → сохранить output, вернуть
    • Если ошибка → перейти к попытке 2
    ↓
Попытка 2 (через 1 сек):
    • Повторить выполнение
    • Если успех → сохранить, вернуть
    • Если ошибка → перейти к попытке 3
    ↓
Попытка 3 (через 2 сек):
    • Финальная попытка
    • Если успех → сохранить, вернуть
    • Если ошибка → отметить как FAILED
```

**Exponential Backoff:**
- 1-я попытка: 1 сек
- 2-я попытка: 2 сек (1 * 2)
- 3-я попытка: 4 сек (2 * 2)
- Max: 30 сек

---

## 📊 Примеры использования

### Пример 1: Обработка данных (Pipeline)

```
[Fetch Data] → [Transform] → [Validate] → [Save]
```

1. Fetch Data (HTTP) - получить JSON с API
2. Transform (Data Adapter) - преобразовать формат
3. Validate (Conditional) - проверить корректность
4. Save (HTTP) - сохранить результат

### Пример 2: AI-powered Automation

```
[Get Text] → [Analyze with LLM] → [Decision] ┬─→ [Action A]
                                              └─→ [Action B]
```

1. Get Text - получить текст
2. Analyze with LLM - отправить в ChatGPT
3. Decision - проверить что вернул LLM
4. Action A или B - выполнить разные действия

### Пример 3: Параллельная обработка

```
         ┌─→ [Process A] ─┐
[Input] ─┤                 ├─→ [Merge] → [Result]
         └─→ [Process B] ─┘
```

1. Input - получить данные
2. Process A и B выполняются **параллельно**
3. Merge - объединить результаты
4. Result - финальный output

---

## 🛠️ Инструменты и технологии

### Backend
| Компонент | Инструмент     | Назначение |
|-----------|----------------|-----------|
| **Language** | Go 1.23+       | Высокая производительность, простота |
| **Web Framework** | Gin            | Fast HTTP框架 (JSON API) |
| **ORM** | Bun            | Простая работа с PostgreSQL |
| **Database** | PostgreSQL 14+ | Хранение workflows, executions |
| **Cache/Queue** | Redis          | Кеширование, события |
| **Concurrency** | Goroutines     | Параллельное выполнение узлов |
| **Logging** | slog       | Структурированное логирование |
| **Scheduling** | robfig/cron    | Time-based triggers |

### Frontend
| Компонент | Инструмент | Назначение |
|-----------|-----------|-----------|
| **Framework** | Vue 3+ | UI компоненты |
| **DAG Visualization** | VueFlow | Рисование workflow |
| **Styling** | Tailwind CSS | Стили и UI |
| **HTTP Client** | Axios | Запросы к API |
| **Real-time** | WebSocket | WebSocket для updates |
| **State Management** | Zustand или Pinia | Управление состоянием |
| **Charts** | Chart.js или ECharts | Графики и метрики |

### DevOps
| Компонент | Инструмент | Назначение |
|-----------|-----------|-----------|
| **Containerization** | Docker | Контейнеризация приложения |
| **Orchestration** | Docker Compose (dev) / Kubernetes (prod) | Управление сервисами |
| **CI/CD** | GitHub Actions | Автоматизация тестирования и деплоя |
| **Monitoring** | Prometheus + Grafana | Метрики и мониторинг |
| **Logging** | ELK Stack | Централизованные логи |

---

## 📦 Компоненты системы

### 1. **REST API** (30+ endpoints)
```
POST   /workflows                      # Создать workflow
GET    /workflows/:id                  # Получить workflow
PUT    /workflows/:id                  # Обновить workflow
DELETE /workflows/:id                  # Удалить workflow
POST   /workflows/:id/nodes            # Добавить узел
POST   /workflows/:id/publish          # Публиковать
POST   /workflows/:id/execute          # Запустить
GET    /executions/:id                 # Получить результат
POST   /triggers/:id                   # Создать триггер
POST   /webhooks/:trigger_id           # Webhook endpoint
```

### 2. **SDK для встраивания**
```go
client, _ := sdk.NewClient(opts)

// Создать workflow
workflow, _ := client.Workflows.Create(ctx, req)

// Добавить узлы
node, _ := client.Workflows.AddNode(ctx, workflowID, nodeReq)

// Запустить
execution, _ := client.Executions.Execute(ctx, executeReq)

// Получить результат
result, _ := client.Executions.GetResult(ctx, executionID)
```

### 3. **Web UI**
- Dashboard (статистика, recent executions)
- Workflow Editor (drag-and-drop DAG создание)
- Execution Monitor (история, логи, результаты)
- Trigger Manager (создание триггеров)
- Logs & Monitoring (Prometheus метрики)

---

## 🎓 Примеры реальных сценариев

### Сценарий 1: Email Marketing Automation
```
[Schedule Time Trigger]
    ↓
[Get Customer List from CRM]
    ↓
[Filter Active Customers]
    ↓
[Generate Email with LLM] → For Each Customer
    ↓
[Send Email via SMTP]
    ↓
[Log Result to Analytics]
```

### Сценарий 2: Data Pipeline
```
[Fetch Data from DB]
    ↓
[Clean & Transform]
    ↓
┌─→ [Generate Report] ───┐
│                          ├─→ [Send Report]
└─→ [Create Dashboard] ──┘
    ↓
[Store Results]
```

### Сценарий 3: AI Content Generation
```
[Webhook Event: New Article]
    ↓
[Extract Key Points]
    ↓
[Generate Summary with LLM]
    ↓
[Create Social Media Posts]
    ↓
[Post to Twitter/LinkedIn]
    ↓
[Send Notification to User]
```

---

## 🚀 Возможности системы

### ✅ Что она делает
1. **Создает workflow** - визуально или через SDK
2. **Валидирует структуру** - проверяет циклы, типы узлов
3. **Выполняет workflow** - запускает узлы по порядку
4. **Управляет ошибками** - автоматический retry с backoff
5. **Интегрируется с AI** - вызывает OpenAI/Anthropic
6. **Вызывает API** - HTTP GET/POST/PUT/DELETE
7. **Трансформирует данные** - JSON mapping, filtering
8. **Принимает решения** - условное выполнение (if-then)
9. **Работает параллельно** - независимые узлы одновременно
10. **Мониторит выполнение** - WebSocket updates, история, логи

### ❌ Что она не делает
- Не требует написания кода (Low-Code)
- Не требует DevOps навыков (Docker + Git)
- Не требует специальной инфраструктуры (PostgreSQL + Redis хватает)
- Не требует зависимости от облачных сервисов (self-hosted)

---

## 💰 Бизнес ценность

1. **Скорость разработки** - в 5-10х раз быстрее чем писать код
2. **Без программистов** - бизнес-аналитики могут создавать workflow
3. **Гибкость** - легко менять процессы без перезагрузки
4. **Надежность** - retry, error handling, мониторинг
5. **Масштабируемость** - поддержка параллельного выполнения
6. **Интеграция** - с любыми API через HTTP executor
7. **AI-powered** - использование LLM для автоматизации

---

## 📈 Метрики проекта

### Размер проекта
- **18 недель разработки** (4.5 месяца)
- **7 фаз реализации**
- **30+ REST endpoints**
- **6 основных сущностей**
- **5+ встроенных executors**
- **85%+ test coverage**

### Стек
- **Backend:** Go + Gin + Bun ORM
- **Frontend:** React/Vue + React Flow
- **Database:** PostgreSQL + Redis
- **DevOps:** Docker + GitHub Actions
- **Monitoring:** Prometheus + Grafana + ELK

### Производительность
- **Параллельное выполнение** через goroutines
- **Connection pooling** для БД
- **Кеширование** через Redis
- **Exponential backoff** для retry
- **Topological sort** для оптимизации порядка

---

## 🎯 Сравнение с аналогами

| Функция | DAG System | Zapier | Make | n8n |
|---------|-----------|--------|------|-----|
| Self-hosted | ✅ | ❌ | ❌ | ✅ |
| SDK для встраивания | ✅ | ❌ | ❌ | ❌ |
| Custom executors | ✅ | ✅ | ✅ | ✅ |
| LLM интеграция | ✅ | ✅ | ✅ | ✅ |
| Real-time monitoring | ✅ | ✅ | ✅ | ✅ |
| Open Source | ✅ | ❌ | ❌ | ✅ |
| Go-based | ✅ | ❌ | ❌ | ❌ |
| REST API | ✅ | ✅ | ✅ | ✅ |

---

## 🔮 Будущие расширения

### Phase 1 (MVP) - 18 недель
- Core DAG system
- REST API
- Web UI
- 5 встроенных executors
- Time & Webhook triggers

### Phase 2 (Enterprise)
- Advanced scheduling
- Message queue (RabbitMQ)
- Multi-tenant support
- Role-based access control
- Advanced monitoring

### Phase 3 (AI/ML)
- Fine-tuning для LLM
- Model versioning
- A/B testing workflows
- Auto-optimization
- Cost analytics

---

## 📝 Итоговое резюме

**MBFlow ** - это **универсальная платформа автоматизации**, которая позволяет:

1. 🏗️ **Строить** сложные бизнес-процессы через DAG
2. 🚀 **Запускать** автоматически по расписанию или триггерам
3. 🔗 **Интегрировать** с любыми API и LLM
4. 📊 **Мониторить** выполнение в реальном времени
5. 🔌 **Встраивать** как SDK в существующие приложения
6. 🛠️ **Расширять** через custom executors
7. 💾 **Хранить** историю и результаты

**Используется для:**
- Email automation
- Data pipelines
- AI-powered content generation
- API integrations
- Business process automation
- ETL/ELT workflows
- Micro-service orchestration

**Технологический стек:**
- Backend: Go + Gin + Bun + PostgreSQL
- Frontend: React + React Flow + Tailwind
- DevOps: Docker + GitHub Actions + Prometheus

**Ценность:**
- 5-10x быстрее разработки
- No-code/Low-code подход
- Self-hosted и открытый
- Встраиваемый SDK
- Enterprise-ready