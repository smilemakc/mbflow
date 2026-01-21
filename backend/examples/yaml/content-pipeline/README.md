# LLM Content Pipeline - Modular Workflows

Модульная архитектура контент-пайплайна из 8 независимых workflow.

## Архитектура

```
┌─────────────────────────────────────────────────────────────────────┐
│                    ORCHESTRATOR (main entry point)                  │
│                 orchestrator.yaml - координация всех этапов         │
└─────────────────────────────────────────────────────────────────────┘
                                   │
       ┌───────────────────────────┼───────────────────────────┐
       ▼                           ▼                           ▼
┌──────────────┐          ┌──────────────┐          ┌──────────────┐
│   PLANNER    │          │    BRIEF     │          │     RAG      │
│   AGENT      │─────────▶│   BUILDER    │─────────▶│   EVIDENCE   │
│ 01_planner   │          │  02_brief    │          │   03_rag     │
└──────────────┘          └──────────────┘          └──────────────┘
                                                            │
       ┌────────────────────────────────────────────────────┘
       ▼
┌──────────────┐          ┌──────────────┐          ┌──────────────┐
│  GENERATOR   │          │  COMPLIANCE  │          │    ROUTER    │
│   AGENT      │─────────▶│   EDITOR     │─────────▶│  (risk-based)│
│ 04_generator │          │ 05_compliance│          │  06_router   │
└──────────────┘          └──────────────┘          └──────────────┘
                                                            │
       ┌────────────────────────────────────────────────────┘
       ▼
┌──────────────┐          ┌──────────────┐
│   CHANNEL    │          │  ANALYTICS   │
│   ADAPTER    │─────────▶│   AGENT      │
│  07_adapter  │          │ 08_analytics │
└──────────────┘          └──────────────┘
```

## Файлы

| # | Файл | Описание | Триггер |
|---|------|----------|---------|
| 0 | `00_orchestrator.yaml` | Главный координатор | cron/webhook/manual |
| 1 | `01_planner.yaml` | Стратегическое планирование | sub-workflow |
| 2 | `02_brief_builder.yaml` | Автоматический бриф | sub-workflow |
| 3 | `03_rag_evidence.yaml` | RAG: сбор доказательств | sub-workflow |
| 4 | `04_content_generator.yaml` | Генерация вариантов | sub-workflow |
| 5 | `05_compliance_editor.yaml` | Compliance + Risk scoring | sub-workflow |
| 6 | `06_workflow_router.yaml` | Risk-based маршрутизация | sub-workflow |
| 7 | `07_channel_adapter.yaml` | Адаптация под платформы | sub-workflow |
| 8 | `08_analytics.yaml` | Аналитика и замыкание цикла | sub-workflow/cron |

## Переменные окружения

```bash
# LLM
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4-turbo
OPENAI_MODEL_CHEAP=gpt-3.5-turbo

# RAG Backend
RAG_ENDPOINT=https://api.yourrag.com
RAG_API_KEY=...

# Brand
BRAND_ID=uuid
BRAND_TOV="professional, friendly, expert"

# Channels
TELEGRAM_BOT_TOKEN=...
TELEGRAM_CHANNEL_ID=@channel

# Budget
MAX_TOKENS_PER_ITEM=50000
MAX_COST_PER_ITEM=2.0
```

## Использование

### Запуск полного цикла
```bash
curl -X POST http://localhost:8585/api/v1/workflows/import \
  -F "file=@00_orchestrator.yaml"

# Запуск
curl -X POST http://localhost:8585/api/v1/executions \
  -H "Content-Type: application/json" \
  -d '{"workflow_id": "...", "input": {"brand_id": "..."}}'
```

### Запуск отдельного этапа
```bash
# Только генерация вариантов
curl -X POST http://localhost:8585/api/v1/executions \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "04_content_generator",
    "input": {
      "brief": {...},
      "evidence_bundle": {...}
    }
  }'
```

## Risk Levels

| Level | Критерии | Действие |
|-------|----------|----------|
| **Low** | Нейтральный контент, нет claims | Auto-approve |
| **Medium** | Soft claims, эмоциональные триггеры | Marketer review (24h) |
| **High** | Medical/legal claims, конкуренты | Expert review (48h) |
| **Hard Stop** | Claims без evidence | Manual only |
