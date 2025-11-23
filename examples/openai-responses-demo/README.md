# OpenAI Responses API Demo

Демонстрация использования OpenAI Responses API для получения структурированных JSON ответов.

## Описание

Этот пример показывает, как использовать executor `openai-responses` для работы с OpenAI Responses API:

1. **Структурированные выходные данные**: Использование `response_format` с JSON schema для валидации ответов
2. **Настройка параметров**: Конфигурирование temperature, top_p, frequency_penalty, presence_penalty
3. **Обработка JSON**: Автоматический парсинг и использование структурированных данных в workflow

## Структура workflow

```
1. Extract Product Info (openai-responses)
   ↓
2. Generate Recommendation (openai-responses)
```

### Node 1: Extract Product Info
- Извлекает структурированную информацию о продукте из текстового описания
- Использует JSON schema для определения структуры выходных данных
- Возвращает объект с полями: name, category, specifications, price_range, target_audience

### Node 2: Generate Recommendation
- Генерирует персонализированную рекомендацию на основе структурированных данных
- Возвращает объект с полями: recommendation_text, pros, cons, rating, best_for

## Запуск

```bash
# Установите переменную окружения с API ключом OpenAI
export OPENAI_API_KEY="your-api-key-here"

# Запустите пример
cd examples/openai-responses-demo
go run main.go

# Или с кастомным описанием продукта
go run main.go -description "Gaming mouse with RGB lighting, 16000 DPI, wireless, 70-hour battery life"
```

## Пример вывода

```json
📦 Extracted Product Information:
{
  "name": "High-Performance Laptop",
  "category": "Laptops",
  "specifications": {
    "processor": "Intel i7",
    "ram": "16GB",
    "storage": "512GB SSD",
    "display": "15.6 inch"
  },
  "price_range": "$800-$1200",
  "target_audience": "Professionals and power users"
}

💡 Product Recommendation:
{
  "recommendation_text": "This high-performance laptop is excellent for...",
  "pros": [
    "Powerful Intel i7 processor",
    "Ample 16GB RAM for multitasking",
    "Fast SSD storage"
  ],
  "cons": [
    "May be expensive for casual users",
    "Battery life could be better"
  ],
  "rating": 8.5,
  "best_for": "Professional work, software development, content creation"
}
```

## Конфигурация

### Базовые параметры
- `model`: Модель OpenAI (по умолчанию: "gpt-4o")
- `prompt`: Текст промпта с поддержкой подстановки переменных через `{{variable}}`
- `output_key`: Ключ для сохранения результата в контексте выполнения

### Параметры генерации
- `max_tokens`: Максимальное количество токенов в ответе
- `temperature`: Контролирует случайность (0.0-2.0)
- `top_p`: Nucleus sampling (0.0-1.0)
- `frequency_penalty`: Штраф за повторения (-2.0 до 2.0)
- `presence_penalty`: Штраф за повторение тем (-2.0 до 2.0)
- `stop`: Последовательности остановки

### Response Format (JSON Schema)
- `type`: "json_schema" для структурированных выходных данных
- `json_schema`: Объект с описанием схемы
  - `name`: Имя схемы
  - `schema`: JSON Schema спецификация
  - `strict`: true для строгой валидации

## Пример JSON Schema

```json
{
  "type": "json_schema",
  "json_schema": {
    "name": "product_info",
    "schema": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string",
          "description": "Product name"
        },
        "category": {
          "type": "string",
          "description": "Product category"
        }
      },
      "required": ["name", "category"],
      "additionalProperties": false
    },
    "strict": true
  }
}
```

## API Key

API ключ может быть предоставлен тремя способами (в порядке приоритета):

1. В конфигурации ноды: `"api_key": "sk-..."`
2. Через переменную контекста выполнения: `openai_api_key` или `OPENAI_API_KEY`
3. При создании executor: `OpenAIAPIKey` в `EngineConfig`

## Отличия от openai-completion

| Параметр | openai-completion | openai-responses |
|----------|-------------------|------------------|
| Structured Output | ❌ | ✅ |
| JSON Schema | ❌ | ✅ |
| top_p | ❌ | ✅ |
| frequency_penalty | ❌ | ✅ |
| presence_penalty | ❌ | ✅ |
| stop sequences | ❌ | ✅ |
| Automatic JSON parsing | ❌ | ✅ |

## Ссылки

- [OpenAI API Documentation](https://platform.openai.com/docs/api-reference/chat)
- [JSON Schema](https://json-schema.org/)
- [Structured Outputs Guide](https://platform.openai.com/docs/guides/structured-outputs)
