# Telegram Executor

## Обзор

Telegram Executor - это встроенный исполнитель для отправки сообщений и медиа-файлов через Telegram Bot API. Он
поддерживает отправку текстовых сообщений с различными форматами разметки, а также фото, документов, аудио и видео
файлов.

## Возможности

- **Текстовые сообщения** с поддержкой форматирования (Markdown, MarkdownV2, HTML)
- **Медиа-файлы**: фото, документы, аудио, видео
- **Гибкая работа с файлами**: base64, URL, file_id
- **Дополнительные опции**: отключение уведомлений, защита контента, ответы на сообщения
- **Автоматическое разрешение шаблонов** через Template Engine
- **Таймауты и обработка ошибок**

## Конфигурация

### Обязательные параметры

| Параметр       | Тип    | Описание                                                     |
|----------------|--------|--------------------------------------------------------------|
| `bot_token`    | string | Токен бота от @BotFather (формат: `<bot_id>:<token>`)        |
| `chat_id`      | string | ID чата (ID пользователя, группы или @channel)               |
| `message_type` | string | Тип сообщения: `text`, `photo`, `document`, `audio`, `video` |

### Параметры для текстовых сообщений

| Параметр                   | Тип    | Описание                                               |
|----------------------------|--------|--------------------------------------------------------|
| `text`                     | string | Текст сообщения (обязательно для `message_type=text`)  |
| `parse_mode`               | string | Режим форматирования: `Markdown`, `MarkdownV2`, `HTML` |
| `disable_web_page_preview` | bool   | Отключить предварительный просмотр ссылок              |

### Параметры для медиа-сообщений

| Параметр      | Тип    | Описание                                             |
|---------------|--------|------------------------------------------------------|
| `file_source` | string | Источник файла: `base64`, `url`, `file_id`           |
| `file_data`   | string | Данные файла (base64 строка, URL или file_id)        |
| `file_name`   | string | Имя файла (опционально, используется для документов) |
| `text`        | string | Подпись к медиа-файлу (caption)                      |

### Дополнительные параметры

| Параметр               | Тип  | По умолчанию | Описание                           |
|------------------------|------|--------------|------------------------------------|
| `disable_notification` | bool | false        | Отправить без звука уведомления    |
| `protect_content`      | bool | false        | Защитить от пересылки/сохранения   |
| `reply_to_message_id`  | int  | 0            | Ответить на конкретное сообщение   |
| `message_thread_id`    | int  | 0            | ID темы для форумов                |
| `timeout`              | int  | 30           | Таймаут запроса в секундах (1-300) |

## Примеры использования

### 1. Простое текстовое сообщение

```json
{
  "id": "notify_user",
  "type": "telegram",
  "config": {
    "bot_token": "{{env.telegram_bot_token}}",
    "chat_id": "{{env.telegram_chat_id}}",
    "message_type": "text",
    "text": "Hello, World!"
  }
}
```

### 2. Форматированное сообщение с Markdown

```json
{
  "id": "workflow_status",
  "type": "telegram",
  "config": {
    "bot_token": "{{env.telegram_bot_token}}",
    "chat_id": "{{env.telegram_chat_id}}",
    "message_type": "text",
    "text": "*Workflow completed!*\n\nStatus: `success`\nDuration: {{input.duration}}ms",
    "parse_mode": "Markdown",
    "disable_notification": true
  }
}
```

### 3. HTML форматирование

```json
{
  "id": "html_message",
  "type": "telegram",
  "config": {
    "bot_token": "{{env.telegram_bot_token}}",
    "chat_id": "-1001234567890",
    "message_type": "text",
    "text": "<b>Alert!</b>\n\nUser <code>{{input.username}}</code> registered.\n<a href='{{input.profile_url}}'>View Profile</a>",
    "parse_mode": "HTML"
  }
}
```

### 4. Отправка фото по URL

```json
{
  "id": "send_photo",
  "type": "telegram",
  "config": {
    "bot_token": "{{env.telegram_bot_token}}",
    "chat_id": "{{env.telegram_chat_id}}",
    "message_type": "photo",
    "file_source": "url",
    "file_data": "https://example.com/image.jpg",
    "text": "Check out this image!"
  }
}
```

### 5. Отправка документа из base64

```json
{
  "id": "send_report",
  "type": "telegram",
  "config": {
    "bot_token": "{{env.telegram_bot_token}}",
    "chat_id": "{{env.telegram_chat_id}}",
    "message_type": "document",
    "file_source": "base64",
    "file_data": "{{input.pdf_base64}}",
    "file_name": "monthly_report.pdf",
    "text": "Monthly report for {{input.month}}",
    "parse_mode": "Markdown"
  }
}
```

### 6. Переиспользование file_id

После отправки медиа-файла, Telegram возвращает `file_id`, который можно использовать для повторной отправки без
повторной загрузки:

```json
{
  "id": "resend_photo",
  "type": "telegram",
  "config": {
    "bot_token": "{{env.telegram_bot_token}}",
    "chat_id": "{{env.telegram_chat_id}}",
    "message_type": "photo",
    "file_source": "file_id",
    "file_data": "AgACAgIAAxkBAAIC..."
  }
}
```

### 7. Ответ на сообщение

```json
{
  "id": "reply_message",
  "type": "telegram",
  "config": {
    "bot_token": "{{env.telegram_bot_token}}",
    "chat_id": "{{env.telegram_chat_id}}",
    "message_type": "text",
    "text": "This is a reply to your message",
    "reply_to_message_id": 12345
  }
}
```

## Выходные данные

### Успешный ответ (текстовое сообщение)

```json
{
  "success": true,
  "message_id": 42,
  "chat_id": -1001234567890,
  "date": 1234567890,
  "message_type": "text",
  "text": "Hello, World!",
  "duration_ms": 234
}
```

### Успешный ответ (медиа-сообщение)

```json
{
  "success": true,
  "message_id": 43,
  "chat_id": -1001234567890,
  "date": 1234567890,
  "message_type": "photo",
  "caption": "Check out this image!",
  "file_id": "AgACAgIAAxkBAAIC...",
  "file_unique_id": "AQADAgAD...",
  "file_size": 12345,
  "duration_ms": 456
}
```

### Ответ с ошибкой

```json
{
  "success": false,
  "message_type": "text",
  "error": "Forbidden: bot was blocked by the user",
  "error_code": 403,
  "duration_ms": 123
}
```

## Обработка ошибок

Telegram API может возвращать различные коды ошибок:

| Код  | Описание                                              |
|------|-------------------------------------------------------|
| 400  | Неверные параметры запроса                            |
| 401  | Неверный токен бота                                   |
| 403  | Бот заблокирован пользователем или нет доступа к чату |
| 404  | Чат не найден                                         |
| 429  | Превышен лимит запросов (rate limit)                  |
| 500+ | Ошибка сервера Telegram                               |

Все ошибки возвращаются в поле `error` с кодом в `error_code`. Workflow может обрабатывать эти ошибки в последующих
нодах.

## Использование Template Engine

Telegram executor автоматически поддерживает шаблоны для всех строковых параметров:

### Переменные окружения

```json
{
  "bot_token": "{{env.telegram_bot_token}}",
  "chat_id": "{{env.telegram_chat_id}}"
}
```

### Выходные данные родительских нод

```json
{
  "text": "User {{input.username}} registered at {{input.timestamp}}"
}
```

### Вложенные объекты

```json
{
  "text": "Order #{{input.order.id}} for {{input.order.customer.name}}\nTotal: ${{input.order.total}}"
}
```

### Массивы

```json
{
  "text": "First item: {{input.items[0].name}}"
}
```

## Примеры Workflow

### Уведомление о завершении workflow

```mermaid
---
config:
  layout: elk
---
flowchart LR
    A[HTTP: Fetch Data] --> B[Transform: Process]
    B --> C[Telegram: Notify Success]
%% Node type styles
    classDef httpNode fill: #e1f5ff, stroke: #01579b, stroke-width: 2px
    classDef transformNode fill: #fff3e0, stroke: #e65100, stroke-width: 2px
    classDef telegramNode fill: #e8f5e9, stroke: #1b5e20, stroke-width: 2px
    class A httpNode
    class B transformNode
    class C telegramNode
```

```json
{
  "nodes": [
    {
      "id": "fetch_data",
      "type": "http",
      "config": {
        "method": "GET",
        "url": "https://api.example.com/data"
      }
    },
    {
      "id": "process",
      "type": "transform",
      "config": {
        "type": "expression",
        "expression": "{'count': len(input.body.items)}"
      }
    },
    {
      "id": "notify",
      "type": "telegram",
      "config": {
        "bot_token": "{{env.telegram_bot_token}}",
        "chat_id": "{{env.telegram_chat_id}}",
        "message_type": "text",
        "text": "Data processed: {{input.count}} items",
        "parse_mode": "Markdown"
      }
    }
  ],
  "edges": [
    {
      "from": "fetch_data",
      "to": "process"
    },
    {
      "from": "process",
      "to": "notify"
    }
  ]
}
```

### Отправка отчета с документом

```mermaid
---
config:
  layout: elk
---
flowchart LR
    A[HTTP: Generate Report] --> B[Transform: Encode Base64]
    B --> C[Telegram: Send Document]
%% Node type styles
    classDef httpNode fill: #e1f5ff, stroke: #01579b, stroke-width: 2px
    classDef transformNode fill: #fff3e0, stroke: #e65100, stroke-width: 2px
    classDef telegramNode fill: #e8f5e9, stroke: #1b5e20, stroke-width: 2px
    class A httpNode
    class B transformNode
    class C telegramNode
```

```json
{
  "nodes": [
    {
      "id": "generate_report",
      "type": "http",
      "config": {
        "method": "POST",
        "url": "https://api.example.com/reports/generate",
        "body": {
          "type": "monthly",
          "month": "{{env.current_month}}"
        }
      }
    },
    {
      "id": "encode_pdf",
      "type": "transform",
      "config": {
        "type": "expression",
        "expression": "{'pdf_base64': base64.b64encode(input.body.pdf_bytes).decode()}"
      }
    },
    {
      "id": "send_report",
      "type": "telegram",
      "config": {
        "bot_token": "{{env.telegram_bot_token}}",
        "chat_id": "{{env.telegram_chat_id}}",
        "message_type": "document",
        "file_source": "base64",
        "file_data": "{{input.pdf_base64}}",
        "file_name": "report_{{env.current_month}}.pdf",
        "text": "Monthly Report - {{env.current_month}}"
      }
    }
  ],
  "edges": [
    {
      "from": "generate_report",
      "to": "encode_pdf"
    },
    {
      "from": "encode_pdf",
      "to": "send_report"
    }
  ]
}
```

### Условная отправка уведомлений

```mermaid
---
config:
  layout: elk
---
flowchart TB
    A[HTTP: Check Status] --> B{Transform: Is Error?}
    B -->|Yes| C[Telegram: Error Alert]
    B -->|No| D[Telegram: Success Message]
%% Node type styles
    classDef httpNode fill: #e1f5ff, stroke: #01579b, stroke-width: 2px
    classDef transformNode fill: #fff3e0, stroke: #e65100, stroke-width: 2px
    classDef telegramNode fill: #e8f5e9, stroke: #1b5e20, stroke-width: 2px
    class A httpNode
    class B transformNode
    class C, D telegramNode
```

## Получение Bot Token

1. Откройте Telegram и найдите бота [@BotFather](https://t.me/botfather)
2. Отправьте команду `/newbot`
3. Следуйте инструкциям для создания бота
4. Сохраните полученный токен в формате `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`
5. Добавьте токен в переменные окружения workflow: `telegram_bot_token`

## Получение Chat ID

### Для личных чатов:

1. Напишите боту любое сообщение
2. Откройте в браузере: `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
3. Найдите `"chat":{"id":123456789}` - это ваш chat_id

### Для групп:

1. Добавьте бота в группу
2. Отправьте любое сообщение в группу
3. Откройте в браузере: `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
4. Найдите `"chat":{"id":-1001234567890}` - это chat_id группы (с минусом)

### Для каналов:

1. Добавьте бота как администратора канала
2. Chat ID канала - это `@channel_username` или числовой ID начинающийся с `-100`

## Лимиты Telegram API

- **Текстовые сообщения**: до 4096 символов
- **Фото**: до 10 МБ (для base64) или 20 МБ (для URL)
- **Документы**: до 50 МБ
- **Аудио/Видео**: до 50 МБ
- **Rate Limit**: 30 сообщений в секунду для одного бота

## Советы и рекомендации

1. **Используйте переменные окружения** для хранения токенов и chat_id
2. **Тестируйте форматирование** - разные parse_mode имеют разный синтаксис
3. **Сохраняйте file_id** после первой отправки для экономии трафика
4. **Обрабатывайте ошибки** в последующих нодах workflow
5. **Используйте disable_notification** для неважных уведомлений
6. **Для больших файлов** предпочитайте URL вместо base64

## См. также

- [Telegram Bot API Documentation](https://core.telegram.org/bots/api)
- [MBFlow Template Engine](../TEMPLATE_ENGINE.md)
- [Transform Executor](TRANSFORM_EXECUTOR.md)
- [HTTP Executor](HTTP_EXECUTOR.md)