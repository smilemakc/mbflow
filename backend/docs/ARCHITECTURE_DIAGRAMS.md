# Диаграммы и архитектура

## ER диаграмма сущностей

```mermaid
erDiagram
    WORKFLOW ||--o{ TRIGGER : has
    WORKFLOW ||--o{ NODE : contains
    WORKFLOW ||--o{ EDGE : defines
    WORKFLOW ||--o{ EXECUTION : runs

    TRIGGER ||--o{ EXECUTION : initiates

    NODE ||--o{ NODE : "source of"
    NODE ||--o{ NODE : "target of"

    EDGE ||--|| NODE : "from"
    EDGE ||--|| NODE : "to"

    EXECUTION ||--o{ NODE_EXECUTION : contains
    EXECUTION ||--o{ EVENT : "logs"
    NODE ||--o{ NODE_EXECUTION : "executes"

    WORKFLOW {
        uuid id PK
        string name
        string description
        int version
        enum status "draft, active, archived"
        json variables "workflow variables"
        json metadata "additional metadata"
        uuid created_by FK "optional"
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at "soft delete"
    }

    TRIGGER {
        uuid id PK
        enum type "manual, cron, webhook, event, interval"
        uuid workflow_id FK
        json config
        boolean enabled
        timestamp last_triggered_at
        timestamp created_at
        timestamp updated_at
    }

    NODE {
        uuid id PK "internal UUID for FK"
        string node_id "logical ID for API/templates"
        uuid workflow_id FK
        string name
        enum type "http, transform, llm, conditional, merge, split, delay, webhook"
        json config "executor config, timeout, retry_policy, schemas"
        json position "UI coordinates"
        timestamp created_at
        timestamp updated_at
    }

    EDGE {
        uuid id PK "internal UUID"
        string edge_id "logical ID for API"
        uuid workflow_id FK
        string from_node_id "references node.node_id"
        string to_node_id "references node.node_id"
        json condition "conditional traversal"
        timestamp created_at
        timestamp updated_at
    }

    EXECUTION {
        uuid id PK
        uuid workflow_id FK
        uuid trigger_id FK "optional"
        enum status "pending, running, completed, failed, cancelled, paused"
        timestamp started_at
        timestamp completed_at
        json input_data
        json output_data
        json variables "runtime variables for templates"
        boolean strict_mode "fail on first error"
        string error "error message"
        json metadata
        timestamp created_at
        timestamp updated_at
    }

    NODE_EXECUTION {
        uuid id PK
        uuid execution_id FK
        uuid node_id FK "references node.id"
        enum status "pending, running, completed, failed, skipped, retrying"
        timestamp started_at
        timestamp completed_at
        json input_data
        json output_data
        string error
        int retry_count
        int wave "parallel execution wave"
        timestamp created_at
        timestamp updated_at
    }

    EVENT {
        uuid id PK
        uuid execution_id FK
        string event_type "workflow_started, node_completed, etc"
        bigint sequence "monotonic sequence number"
        json payload "event data"
        timestamp created_at
    }
```

## Архитектура системы

```mermaid
graph TB
    subgraph "UI Layer"
        Dashboard["Dashboard<br/>Мониторинг"]
        Editor["DAG Editor<br/>Конструктор"]
        TriggerUI["Trigger Manager<br/>Управление триггерами"]
        History["Execution History<br/>История запусков"]
    end
    
    subgraph "API Layer"
        WorkflowAPI["Workflow API<br/>CRUD операции"]
        NodeAPI["Node API<br/>Управление узлами"]
        ExecutionAPI["Execution API<br/>Запуск и мониторинг"]
        TriggerAPI["Trigger API<br/>Управление триггерами"]
    end
    
    subgraph "Orchestration Layer"
        Scheduler["Scheduler<br/>Планировщик"]
        ExecutionEngine["Execution Engine<br/>Движок выполнения"]
        DAGValidator["DAG Validator<br/>Валидация графа"]
        DepManager["Dependency Manager<br/>Управление зависимостями"]
    end
    
    subgraph "Execution Layer"
        NodeRunner["Node Runner<br/>Выполнение узлов"]
        ContextMgr["Context Manager<br/>Контекст выполнения"]
        CacheLayer["Cache Layer<br/>Кеширование"]
        WorkerPool["Worker Pool<br/>Параллельное исполнение"]
    end
    
    subgraph "Integration Layer"
        LLMAdapter["LLM Adapter<br/>OpenAI, Anthropic"]
        HTTPClient["HTTP Client<br/>Запросы"]
        DataAdapter["Data Adapter<br/>Трансформация"]
        CustomHandlers["Custom Handlers<br/>Пользовательские"]
    end
    
    subgraph "Persistence Layer"
        DB["PostgreSQL<br/>БД"]
        Redis["Redis<br/>Кеш & Queue"]
        Logs["Logging<br/>Логирование"]
    end
    
    Dashboard --> WorkflowAPI
    Editor --> WorkflowAPI
    TriggerUI --> TriggerAPI
    History --> ExecutionAPI
    
    WorkflowAPI --> DAGValidator
    NodeAPI --> DAGValidator
    TriggerAPI --> Scheduler
    ExecutionAPI --> ExecutionEngine
    
    Scheduler --> ExecutionEngine
    ExecutionEngine --> DepManager
    DepManager --> NodeRunner
    
    NodeRunner --> ContextMgr
    ContextMgr --> WorkerPool
    WorkerPool --> CacheLayer
    
    WorkerPool --> LLMAdapter
    WorkerPool --> HTTPClient
    WorkerPool --> DataAdapter
    WorkerPool --> CustomHandlers
    
    DAGValidator --> DB
    ExecutionEngine --> DB
    NodeRunner --> DB
    CacheLayer --> Redis
    Scheduler --> Redis
    NodeRunner --> Logs
```

## Жизненный цикл выполнения workflow

```mermaid
sequenceDiagram
    participant User
    participant API
    participant Scheduler
    participant Engine as ExecutionEngine
    participant DAGVal as DAGValidator
    participant Runner as NodeRunner
    participant DB as Database
    
    User->>API: Запустить workflow
    API->>DAGVal: Валидировать DAG
    DAGVal->>DB: Получить nodes/edges
    DAGVal-->>API: DAG валиден
    
    API->>Engine: Создать Execution
    Engine->>DB: Сохранить execution
    DB-->>Engine: execution_id
    
    Engine->>Engine: Вычислить порядок узлов
    Engine->>Runner: Выполнить начальные узлы
    
    Runner->>Runner: Получить input данные
    Runner->>DB: Сохранить NodeExecution (RUNNING)
    Runner->>Runner: Выполнить узел (LLM/HTTP/etc)
    
    alt Успешно
        Runner->>DB: Обновить NodeExecution (COMPLETED)
        Runner->>Engine: Узел завершен
        Engine->>Engine: Вычислить следующие узлы
        Engine->>Runner: Выполнить зависимые узлы
    else Ошибка
        Runner->>DB: Обновить NodeExecution (FAILED)
        Runner->>Engine: Ошибка в узле
        Engine->>Engine: Проверить retry_policy
        alt Повторить
            Engine->>Runner: Повторить узел
        else Остановить
            Engine->>DB: Обновить execution (FAILED)
        end
    end
    
    Engine->>DB: Обновить execution (COMPLETED)
    Engine-->>API: Готово
    API-->>User: Результат
```

## Типы узлов и их взаимодействие

```mermaid
graph LR
    Start["Start Node<br/>(No type)"]
    
    Start --> LLM["LLM Node<br/>Генерация текста"]
    Start --> HTTP["HTTP Node<br/>Запрос данных"]
    
    LLM --> Conditional["Conditional Node<br/>Проверка условия"]
    HTTP --> Transform["Data Adapter<br/>Трансформация"]
    
    Conditional -->|success| Merge["Merge Node<br/>Объединение"]
    Conditional -->|error| ErrorHandler["Error Handler<br/>Обработка ошибок"]
    
    Transform --> Merge
    ErrorHandler --> Merge
    
    Merge --> Custom["Custom Node<br/>Пользовательская логика"]
    Custom --> End["End Node<br/>Результат"]
```

## Состояния выполнения

```mermaid
stateDiagram-v2
    [*] --> PENDING: Создание
    
    PENDING --> RUNNING: Старт
    PENDING --> CANCELLED: Отмена
    
    RUNNING --> COMPLETED: Успех
    RUNNING --> FAILED: Ошибка
    RUNNING --> CANCELLED: Отмена
    
    FAILED --> RUNNING: Retry
    
    COMPLETED --> [*]
    FAILED --> [*]
    CANCELLED --> [*]
    
    note right of PENDING
        Ожидание начала
        выполнения
    end note
    
    note right of RUNNING
        Активное выполнение
        узлов workflow
    end note
    
    note right of COMPLETED
        Успешное завершение
        всех узлов
    end note
    
    note right of FAILED
        Сбой выполнения
        или узла
    end note
```

## Обработка ошибок и повторы

```mermaid
graph TD
    NodeExec["Выполнение узла"]
    NodeExec -->|Успех| Complete["✓ COMPLETED"]
    
    NodeExec -->|Ошибка| CheckRetry["Проверить<br/>retry_policy"]
    
    CheckRetry -->|retry_count<br/>max_attempts| Delay["Ожидать<br/>initial_delay"]
    CheckRetry -->|retry_count>=<br/>max_attempts| Fail["✗ FAILED"]
    
    Delay -->|Прошло время| CalcDelay["Вычислить новую<br/>задержку"]
    CalcDelay -->|backoff| UpdateDelay["new_delay =<br/>delay * multiplier"]
    UpdateDelay -->|max_delay| ClampDelay["min(new_delay,<br/>max_delay)"]
    ClampDelay --> Retry["Повторить узел"]
    
    Retry -->|Успех| Complete
    Retry -->|Ошибка| CheckRetry
```