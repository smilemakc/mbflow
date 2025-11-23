# Демо: Сбор и сохранение метрик и трейсов выполнения

Этот пример демонстрирует полный цикл работы с метриками и трейсами выполнения в MBFlow:
- Сбор метрик выполнения workflows и nodes
- Запись трейсов событий
- Сохранение данных в файлы (JSON)
- Автоматическая периодическая персистентность
- Анализ собранных данных

## Возможности

### MetricsCollector

`MetricsCollector` собирает следующие метрики:

**Метрики Workflow:**
- Количество выполнений (успешных/неудачных)
- Длительность выполнения (средняя, мин, макс)
- Процент успешности
- Время последнего выполнения

**Метрики Node:**
- Количество выполнений по типам узлов
- Количество ретраев
- Длительность выполнения узлов
- Процент успешности по типам

**AI метрики:**
- Количество запросов к AI API
- Использованные токены (prompt/completion)
- Оценка стоимости в USD
- Средняя задержка

### ExecutionTrace

`ExecutionTrace` записывает детальный трейс событий:

- События выполнения (старт, завершение, ошибка)
- События узлов (старт, завершение, ошибка, ретрай)
- Установка переменных
- Переходы состояний
- Callback события

Трейс предоставляет методы для:
- Фильтрации событий по типу/узлу
- Поиска ошибок
- Расчета длительности
- Анализа событий

### Персистентность

**Сохранение метрик:**
```go
// Создать снимок метрик
snapshot := metricsCollector.Snapshot()

// Сохранить в файл
monitoring.SaveMetricsToFile(snapshot, "metrics.json")

// Сохранить с timestamp в имени
monitoring.SaveMetricsToFileWithTimestamp(snapshot, "./metrics", "prefix")
```

**Автоматическое сохранение:**
```go
// Создать менеджер персистентности
persistence := monitoring.NewMetricsPersistence(
    metricsCollector,
    "./metrics",
    5*time.Second, // интервал сохранения
)

// Настроить
persistence.SetFilePrefix("auto-metrics")
persistence.SetRetention(10) // хранить последние 10 файлов

// Запустить автоматическое сохранение
persistence.Start()
defer persistence.Stop()

// Или сохранить немедленно
filePath, err := persistence.SaveNow()
```

**Сохранение трейсов:**
```go
// Сохранить трейс в JSON
monitoring.SaveTraceToFile(trace, "trace.json")

// Сохранить с timestamp
monitoring.SaveTraceToFileWithTimestamp(trace, "./traces")

// Экспортировать несколько трейсов в текстовый файл
monitoring.ExportTracesAsText(traces, "all-traces.txt")
```

**Загрузка данных:**
```go
// Загрузить метрики
snapshot, err := monitoring.LoadMetricsFromFile("metrics.json")

// Загрузить трейс
trace, err := monitoring.LoadTraceFromFile("trace.json")
```

## Использование с Workflow Engine

```go
// Создать компоненты мониторинга
metrics := monitoring.NewMetricsCollector()
trace := monitoring.NewExecutionTrace(executionID, workflowID)
logger := monitoring.NewConsoleLogger(config)

// Объединить в CompositeObserver
observer := monitoring.NewCompositeObserver(logger, metrics, trace)

// Добавить в workflow engine
engine := executor.NewWorkflowEngine(config)
engine.AddObserver(observer)

// После выполнения workflow:
// - metrics содержит собранные метрики
// - trace содержит полный трейс событий
// - Можно сохранить оба в файлы для анализа
```

## Запуск примера

```bash
cd examples/metrics-trace-demo
go run main.go
```

Пример создаст директорию `./output` с двумя поддиректориями:
- `./output/metrics` - JSON файлы с метриками
- `./output/traces` - JSON и текстовые файлы с трейсами

## Структура сохраненных данных

### Формат метрик (JSON)

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "workflow_metrics": {
    "workflow-id": {
      "workflow_id": "workflow-id",
      "execution_count": 10,
      "success_count": 8,
      "failure_count": 2,
      "total_duration": 5000000000,
      "average_duration": 500000000,
      "min_duration": 250000000,
      "max_duration": 800000000,
      "last_execution_at": "2024-01-15T10:29:55Z"
    }
  },
  "node_metrics": {
    "http": {
      "node_type": "http",
      "execution_count": 15,
      "success_count": 13,
      "failure_count": 2,
      "retry_count": 3,
      "average_duration": 120000000
    }
  },
  "ai_metrics": {
    "total_requests": 5,
    "total_tokens": 2500,
    "prompt_tokens": 1500,
    "completion_tokens": 1000,
    "estimated_cost_usd": 0.105,
    "average_latency": 2000000000
  },
  "summary": {
    "total_workflows": 1,
    "total_executions": 10,
    "overall_success_rate": 0.8,
    "total_ai_cost_usd": 0.105
  }
}
```

### Формат трейса (JSON)

```json
{
  "execution_id": "exec-123",
  "workflow_id": "workflow-1",
  "timestamp": "2024-01-15T10:30:00Z",
  "event_count": 8,
  "events": [
    {
      "timestamp": "2024-01-15T10:30:00.100Z",
      "event_type": "execution_started",
      "message": "Workflow execution started"
    },
    {
      "timestamp": "2024-01-15T10:30:00.150Z",
      "event_type": "node_started",
      "node_id": "node-1",
      "node_type": "http",
      "message": "HTTP node started",
      "data": {
        "url": "https://api.example.com"
      }
    }
  ]
}
```

## Примеры анализа данных

### Найти самые медленные workflows

```go
snapshot, _ := monitoring.LoadMetricsFromFile("metrics.json")
for id, metrics := range snapshot.WorkflowMetrics {
    if metrics.AverageDuration > threshold {
        fmt.Printf("Slow workflow: %s (avg: %v)\n", id, metrics.AverageDuration)
    }
}
```

### Найти узлы с высоким процентом ошибок

```go
snapshot, _ := monitoring.LoadMetricsFromFile("metrics.json")
for nodeType, metrics := range snapshot.NodeMetrics {
    failureRate := float64(metrics.FailureCount) / float64(metrics.ExecutionCount)
    if failureRate > 0.1 { // > 10% failures
        fmt.Printf("Problematic node type: %s (%.1f%% failures)\n",
            nodeType, failureRate*100)
    }
}
```

### Анализировать ошибки в трейсе

```go
trace, _ := monitoring.LoadTraceFromFile("trace.json")
errorEvents := trace.GetErrorEvents()
for _, event := range errorEvents {
    fmt.Printf("Error at %s: %s - %v\n",
        event.Timestamp, event.NodeID, event.Error)
}
```

## Интеграция с базами данных

Для долгосрочного хранения метрик и трейсов можно:

1. Использовать ClickHouse для логов событий (см. `ClickHouseLogger`)
2. Сохранять метрики в PostgreSQL/TimescaleDB
3. Использовать Prometheus для метрик в реальном времени
4. Экспортировать в Grafana для визуализации

## Best Practices

1. **Периодическое сохранение**: Используйте `MetricsPersistence` для автоматического сохранения метрик
2. **Ротация файлов**: Настройте retention policy для управления дисковым пространством
3. **Сжатие**: Для долгосрочного хранения сжимайте JSON файлы (gzip)
4. **Индексация**: При сохранении в БД создавайте индексы по timestamp, execution_id, workflow_id
5. **Мониторинг**: Настройте алерты на основе метрик (высокий failure rate, медленное выполнение)

## См. также

- `examples/logger-demo` - Демо различных логеров
- `internal/infrastructure/monitoring/observer.go` - Observer pattern
- `internal/infrastructure/monitoring/clickhouse_logger.go` - ClickHouse интеграция
