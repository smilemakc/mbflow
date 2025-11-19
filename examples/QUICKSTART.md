# Quick Start Guide - Complex Workflow Examples

## 🎯 Цель

Эти примеры демонстрируют сложные workflow с ветвлениями и обработкой данных, используя запросы к OpenAI для получения информации и формирования следующих запросов.

## 📦 Что создано

### 4 полноценных примера workflow

1. **AI Content Pipeline** - Генерация контента с проверкой качества
2. **Customer Support AI** - Автоматизация поддержки клиентов
3. **Data Analysis & Reporting** - Анализ данных и отчетность
4. **Code Review & Refactoring** - Автоматическое ревью кода

## 🚀 Быстрый старт

### Запуск примеров

```bash
# Перейти в директорию проекта
cd /Users/balashov/PycharmProjects/mbflow

# Запустить любой пример:

# Пример 1: AI Content Pipeline
cd examples/ai-content-pipeline && go run main.go

# Пример 2: Customer Support AI
cd examples/customer-support-ai && go run main.go

# Пример 3: Data Analysis & Reporting
cd examples/data-analysis-reporting && go run main.go

# Пример 4: Code Review & Refactoring
cd examples/code-review-refactoring && go run main.go
```

### Что вы увидите

Каждый пример выведет:

- ✅ ID созданного workflow
- ✅ Сводку (количество узлов и связей)
- ✅ Структуру workflow с описанием логики
- ✅ Конфигурацию триггера
- ✅ Список всех узлов с типами
- ✅ Список всех связей с типами

## 📚 Документация

### Основные документы

1. **README.md** - Полная документация на английском
   - Описание каждого примера
   - Демонстрируемые паттерны
   - Типы узлов и связей
   - Инструкции по расширению

2. **ПРИМЕРЫ.md** - Детальная документация на русском
   - Описание всех примеров
   - Ключевые концепции
   - Статистика
   - Обучающая ценность

3. **DIAGRAMS.md** - Визуальные диаграммы
   - Mermaid диаграммы для каждого workflow
   - Паттерны ветвления
   - Паттерны параллельной обработки
   - Паттерны обратной связи

4. **SUMMARY.md** - Итоговая сводка
   - Статистика по всем примерам
   - Демонстрируемые паттерны
   - Интеграция с OpenAI
   - Следующие шаги

### YAML спецификации

Для некоторых примеров созданы YAML файлы:

- `ai-content-pipeline/workflow.yaml`
- `customer-support-ai/workflow.yaml`

## 🎓 Ключевые концепции

### 1. Ветвление на основе AI-анализа

```go
// Анализ качества
nodeAnalyzeQuality := mbflow.NewNode(
    uuid.NewString(),
    workflowID,
    "openai-completion",
    "Analyze Content Quality",
    map[string]any{
        "model": "gpt-4",
        "prompt": "Rate quality as 'high', 'medium', or 'low'",
        "output_key": "quality_rating",
    },
)

// Маршрутизация по результату
nodeRouter := mbflow.NewNode(
    uuid.NewString(),
    workflowID,
    "conditional-router",
    "Route Based on Quality",
    map[string]any{
        "input_key": "quality_rating",
        "routes": map[string]string{
            "high":   "continue",
            "medium": "enhance",
            "low":    "regenerate",
        },
    },
)
```

### 2. Формирование следующего запроса на основе предыдущего

```go
// Первый запрос - генерация
nodeGenerate := mbflow.NewNode(
    uuid.NewString(),
    workflowID,
    "openai-completion",
    "Generate Content",
    map[string]any{
        "prompt": "Write about {{topic}}",
        "output_key": "content",
    },
)

// Второй запрос - использует результат первого
nodeAnalyze := mbflow.NewNode(
    uuid.NewString(),
    workflowID,
    "openai-completion",
    "Analyze Content",
    map[string]any{
        "prompt": "Analyze this content: {{content}}",
        "output_key": "analysis",
    },
)

// Третий запрос - использует результаты обоих
nodeImprove := mbflow.NewNode(
    uuid.NewString(),
    workflowID,
    "openai-completion",
    "Improve Content",
    map[string]any{
        "prompt": "Improve content based on analysis:\nContent: {{content}}\nAnalysis: {{analysis}}",
        "output_key": "improved_content",
    },
)
```

### 3. Параллельная обработка

```go
// Запуск параллельных переводов
edge1 := mbflow.NewEdge(id1, workflowID, mergeNode.ID(), translateES.ID(), "parallel", nil)
edge2 := mbflow.NewEdge(id2, workflowID, mergeNode.ID(), translateFR.ID(), "parallel", nil)
edge3 := mbflow.NewEdge(id3, workflowID, mergeNode.ID(), translateDE.ID(), "parallel", nil)

// Синхронизация результатов
joinEdge1 := mbflow.NewEdge(id4, workflowID, translateES.ID(), aggregate.ID(), "join", nil)
joinEdge2 := mbflow.NewEdge(id5, workflowID, translateFR.ID(), aggregate.ID(), "join", nil)
joinEdge3 := mbflow.NewEdge(id6, workflowID, translateDE.ID(), aggregate.ID(), "join", nil)
```

### 4. Циклы обратной связи

```go
// Генерация
edge1 := mbflow.NewEdge(id1, workflowID, generate.ID(), check.ID(), "direct", nil)

// Проверка качества
edge2 := mbflow.NewEdge(id2, workflowID, check.ID(), router.ID(), "direct", nil)

// Если качество низкое - возврат к генерации
edge3 := mbflow.NewEdge(
    id3, 
    workflowID, 
    router.ID(), 
    generate.ID(), 
    "conditional", 
    map[string]any{"condition": "quality == 'low'", "retry": true},
)
```

## 📊 Статистика примеров

| Пример | Узлов | Связей | Ветвлений | Параллельных веток | Циклов |
|--------|-------|--------|-----------|-------------------|--------|
| AI Content Pipeline | 15 | 19 | 3 | 4 | 1 |
| Customer Support AI | 18 | 25 | 4 | 3 | 1 |
| Data Analysis | 22 | 28 | 2 | 5 | 0 |
| Code Review | 22 | 30 | 5 | 3 | 1 |
| **ИТОГО** | **77** | **102** | **14** | **15** | **4** |

## 🔍 Структура файлов

```
examples/
├── ai-content-pipeline/
│   ├── main.go              # Полная реализация
│   └── workflow.yaml        # YAML спецификация
├── customer-support-ai/
│   ├── main.go
│   └── workflow.yaml
├── data-analysis-reporting/
│   └── main.go
├── code-review-refactoring/
│   └── main.go
├── README.md                # Документация (EN)
├── ПРИМЕРЫ.md               # Документация (RU)
├── DIAGRAMS.md              # Визуальные диаграммы
├── SUMMARY.md               # Итоговая сводка
└── QUICKSTART.md            # Этот файл
```

## 💡 Примеры использования

### Пример 1: Простое ветвление

```go
// Анализ → Маршрутизация → Разные действия
Generate → Analyze → Router → {High → Publish, Low → Improve}
```

### Пример 2: Вложенное ветвление

```go
// Классификация → Тип → Критичность → Действие
Classify → TypeRouter → {
    Billing → FetchAccount → CriticalityRouter,
    Technical → CriticalityRouter
}
```

### Пример 3: Параллельная обработка с синхронизацией

```go
// Разделение → Параллельная обработка → Объединение
Split → {Task1, Task2, Task3} → Join → Continue
```

### Пример 4: Цикл с условием выхода

```go
// Генерация → Проверка → (если плохо) → Улучшение → Проверка
Generate → Check → {Pass → Continue, Fail → Improve → Check}
```

## 🎯 Следующие шаги

1. **Изучите примеры** - запустите каждый и посмотрите вывод
2. **Прочитайте код** - изучите реализацию в `main.go`
3. **Посмотрите диаграммы** - визуализация в `DIAGRAMS.md`
4. **Адаптируйте под себя** - используйте как шаблоны

## 📖 Дополнительные ресурсы

- Основная документация проекта: `/Users/balashov/PycharmProjects/mbflow/README.md`
- Примеры использования API: `/Users/balashov/PycharmProjects/mbflow/examples/basic/`
- Публичный API: `/Users/balashov/PycharmProjects/mbflow/mbflow.go`

## ✅ Проверка работоспособности

Запустите тест:

```bash
cd /Users/balashov/PycharmProjects/mbflow/examples/ai-content-pipeline
go run main.go
```

Ожидаемый вывод:

```
Created workflow: AI Content Pipeline with Branching (ID: ...)

=== Workflow Summary ===
Workflow: AI Content Pipeline with Branching
Nodes: 15
Edges: 19

=== Workflow Structure ===
1. Generate Initial Content (OpenAI)
2. Analyze Content Quality (OpenAI)
3. Route Based on Quality:
   - High Quality → Merge → Continue
   - Medium Quality → Enhance Content → Merge → Continue
   - Low Quality → Regenerate → Re-analyze (loop)
...
```

## 🎉 Готово

Теперь у вас есть 4 полноценных примера сложных workflow с:

- ✅ Ветвлениями на основе AI-анализа
- ✅ Параллельной обработкой данных
- ✅ Циклами обратной связи
- ✅ Формированием следующих запросов на основе предыдущих
- ✅ Полной документацией
- ✅ Визуальными диаграммами

Используйте их как основу для своих проектов!
