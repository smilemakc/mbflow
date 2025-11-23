# ExecutionLogger Interface Demo

Этот пример демонстрирует использование интерфейса `ExecutionLogger` и его реализаций в MBFlow.

## Обзор

`ExecutionLogger` - это интерфейс для логирования событий выполнения workflow. Он позволяет записывать события в различные назначения:
- Консоль (stdout/stderr)
- Файлы
- Буферы памяти
- Базы данных (ClickHouse)
- Пользовательские реализации

## Доступные реализации

### 1. ConsoleLogger

Консольный логер с настраиваемым `io.Writer`. Поддерживает:
- Запись в stdout/stderr
- Запись в файлы
- Запись в буферы памяти
- Любой другой `io.Writer`
- Verbose режим для отладки

**Пример использования:**

```go
import "mbflow/internal/infrastructure/monitoring"

// Логирование в stdout
logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
    Prefix:  "MyApp",
    Verbose: true,
    Writer:  os.Stdout,
})

// Логирование в файл
file, _ := os.Create("workflow.log")
defer file.Close()

logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
    Prefix:  "MyApp",
    Verbose: false,
    Writer:  file,
})

// Логирование в буфер
var buffer bytes.Buffer
logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
    Prefix:  "MyApp",
    Verbose: false,
    Writer:  &buffer,
})

// Использование логера
logger.LogExecutionStarted("workflow-1", "exec-1")
logger.LogInfo("exec-1", "Workflow started")
logger.LogExecutionCompleted("workflow-1", "exec-1", 100*time.Millisecond)
```

**Дополнительные методы:**

```go
// Изменить writer во время выполнения
logger.SetWriter(newWriter)

// Включить/выключить verbose режим
logger.SetVerbose(true)

// Сбросить буфер (если writer поддерживает Flush)
logger.Flush()
```

### 2. ClickHouseLogger

Логер для записи событий в ClickHouse. Поддерживает:
- Батчинг событий для эффективной записи
- Асинхронную запись
- Автоматическое создание таблицы
- Настраиваемый интервал сброса
- Structured logging с метаданными

**Пример использования:**

```go
import (
    "database/sql"
    "mbflow/internal/infrastructure/monitoring"
    _ "github.com/ClickHouse/clickhouse-go/v2"
)

// Подключение к ClickHouse
db, err := sql.Open("clickhouse", "tcp://localhost:9000?database=default")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Создание логера
logger, err := monitoring.NewClickHouseLogger(monitoring.ClickHouseLoggerConfig{
    DB:            db,
    TableName:     "workflow_execution_logs",
    BatchSize:     100,                   // Размер батча
    FlushInterval: 5 * time.Second,       // Интервал сброса
    Verbose:       true,
    CreateTable:   true,                  // Создать таблицу если не существует
})
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// Использование логера
logger.LogExecutionStarted("workflow-1", "exec-1")
logger.LogNodeStartedFromConfig("exec-1", "node-1", "workflow-1", "http", "API Call",
    map[string]any{"url": "https://api.example.com"}, 1)
logger.LogExecutionCompleted("workflow-1", "exec-1", 200*time.Millisecond)
```

**Структура таблицы:**

```sql
CREATE TABLE workflow_execution_logs (
    timestamp DateTime64(3),
    execution_id String,
    workflow_id String,
    node_id String,
    node_type String,
    node_name String,
    event_type String,
    level String,
    message String,
    duration_ms Int64,
    attempt_number Int32,
    will_retry UInt8,
    error_message String,
    metadata String
) ENGINE = MergeTree()
ORDER BY (workflow_id, execution_id, timestamp)
PARTITION BY toYYYYMM(timestamp);
```

**Примеры запросов:**

```sql
-- Все события для конкретного workflow
SELECT * FROM workflow_execution_logs
WHERE workflow_id = 'workflow-1'
ORDER BY timestamp;

-- Статистика по узлам
SELECT
    node_type,
    count() as total_executions,
    avg(duration_ms) as avg_duration,
    countIf(level = 'error') as errors
FROM workflow_execution_logs
WHERE event_type = 'node_completed'
GROUP BY node_type;

-- События за последний час
SELECT * FROM workflow_execution_logs
WHERE timestamp >= now() - INTERVAL 1 HOUR
ORDER BY timestamp DESC;
```

## Использование в WorkflowEngine

```go
import (
    "mbflow/internal/application/executor"
    "mbflow/internal/infrastructure/monitoring"
)

// Создание логера
logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
    Prefix:  "ENGINE",
    Verbose: true,
})

// Создание WorkflowEngine
engine := executor.NewWorkflowEngine(&executor.EngineConfig{
    EnableMonitoring: false, // Отключаем встроенный мониторинг
})

// Добавление логера через CompositeObserver
observer := monitoring.NewCompositeObserver(logger, nil, nil)
engine.AddObserver(observer)

// Теперь все события выполнения будут логироваться
```

## Создание собственной реализации

Вы можете создать свою реализацию интерфейса `ExecutionLogger`:

```go
type MyCustomLogger struct {
    // ваши поля
}

func (l *MyCustomLogger) LogExecutionStarted(workflowID, executionID string) {
    // ваша реализация
}

func (l *MyCustomLogger) LogExecutionCompleted(workflowID, executionID string, duration time.Duration) {
    // ваша реализация
}

// ... реализация остальных методов интерфейса
```

Примеры возможных реализаций:
- **SyslogLogger** - отправка в syslog
- **ElasticsearchLogger** - запись в Elasticsearch
- **PrometheusLogger** - экспорт метрик в Prometheus
- **WebhookLogger** - отправка событий через webhook
- **MultiLogger** - запись в несколько логеров одновременно

## Запуск примера

```bash
cd examples/logger-demo
go run main.go
```

## Требования

- Go 1.21+
- Для ClickHouseLogger: ClickHouse сервер и драйвер `github.com/ClickHouse/clickhouse-go/v2`

## Дополнительная информация

- Все логеры потокобезопасны
- ClickHouseLogger автоматически сбрасывает буфер при закрытии
- ConsoleLogger поддерживает любой `io.Writer`
- Verbose режим логирует дополнительные события (переменные, переходы состояний)
