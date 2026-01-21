# MBFlow YAML Workflow Configuration

This directory contains example YAML workflow configurations for MBFlow.

## YAML Format Specification v1.0

```yaml
# MBFlow Workflow Configuration v1.0
metadata:
  name: "Workflow Name"           # Required: workflow name
  description: "Description"      # Optional: workflow description
  version: 1                      # Optional: version number (default: 1)
  tags: [ "tag1", "tag2" ]          # Optional: workflow tags

variables: # Optional: workflow variables
  api_key: "{{env.API_KEY}}"
  base_url: "https://api.example.com"

nodes: # Required: list of workflow nodes
  - id: node_1                    # Required: unique node identifier
    name: "Node Name"             # Required: display name
    type: http                    # Required: executor type
    config: # Optional: node-specific configuration
      url: "{{variables.base_url}}/endpoint"
      method: "GET"
    position: # Optional: position for UI rendering
      x: 100
      y: 100

edges: # Optional: connections between nodes
  - id: e1                        # Required: unique edge identifier
    from: node_1                  # Required: source node ID
    to: node_2                    # Required: target node ID
    condition: "data.status == 'ok'"  # Optional: conditional expression

trigger: # Optional: workflow trigger
  name: "Trigger Name"            # Required: trigger name
  type: cron                      # Required: cron|webhook|event|manual|interval
  enabled: true                   # Optional: enable state (default: true)
  config: # Required: trigger-specific configuration
    schedule: "0 9 * * *"
```

## Available Node Types

### Core Executors

| Type          | Description                                 |
|---------------|---------------------------------------------|
| `http`        | Make HTTP requests to external APIs         |
| `transform`   | Transform data using JSONPath/expressions   |
| `llm`         | AI/LLM processing (OpenAI, Anthropic, etc.) |
| `conditional` | Conditional branching based on expressions  |
| `merge`       | Merge data from multiple inputs             |

### Integration Executors

| Type                | Description                        |
|---------------------|------------------------------------|
| `telegram`          | Send messages via Telegram Bot API |
| `telegram_download` | Download files from Telegram       |
| `telegram_parse`    | Parse Telegram updates             |
| `telegram_callback` | Handle Telegram callback queries   |
| `rss_parser`        | Parse RSS/Atom feeds               |
| `google_sheets`     | Read/write Google Sheets           |
| `google_drive`      | Upload/download from Google Drive  |

### Data Adapters

| Type              | Description                      |
|-------------------|----------------------------------|
| `csv_to_json`     | Convert CSV to JSON              |
| `string_to_json`  | Parse string as JSON             |
| `json_to_string`  | Serialize JSON to string         |
| `base64_to_bytes` | Decode base64 to bytes           |
| `bytes_to_base64` | Encode bytes to base64           |
| `bytes_to_json`   | Parse bytes as JSON              |
| `html_clean`      | Clean and extract text from HTML |

### Utility Executors

| Type            | Description              |
|-----------------|--------------------------|
| `function_call` | Execute custom functions |

## Trigger Types

| Type       | Description      | Config Example               |
|------------|------------------|------------------------------|
| `manual`   | Manual execution | `{}`                         |
| `cron`     | Cron schedule    | `schedule: "0 9 * * *"`      |
| `webhook`  | HTTP webhook     | `path: "/hooks/my-hook"`     |
| `event`    | Event-driven     | `event_type: "user.created"` |
| `interval` | Fixed interval   | `interval: "5m"`             |

## Node Configuration Examples

### HTTP Node

```yaml
- id: http_request
  name: "Fetch Data"
  type: http
  config:
    url: "https://api.example.com/data"
    method: "GET"
    headers:
      Authorization: "Bearer {{variables.api_key}}"
    timeout: 30
```

### LLM Node

```yaml
- id: ai_analysis
  name: "AI Analysis"
  type: llm
  config:
    provider: "openai"
    model: "gpt-4"
    prompt: "Analyze the following data: {{input.data}}"
    max_tokens: 1000
    temperature: 0.7
```

### Transform Node

```yaml
- id: transform_data
  name: "Transform Data"
  type: transform
  config:
    mapping:
      title: "$.data.title"
      summary: "$.data.description"
    output_format: "json"
```

### Conditional Node

```yaml
- id: check_condition
  name: "Check Status"
  type: conditional
  config:
    expression: "data.status == 'success'"
    true_output: "success_branch"
    false_output: "error_branch"
```

### Telegram Node

```yaml
- id: send_telegram
  name: "Send Notification"
  type: telegram
  config:
    bot_token: "{{variables.telegram_bot_token}}"
    chat_id: "{{variables.telegram_chat_id}}"
    message: "New update: {{input.title}}"
    parse_mode: "HTML"
```

### Google Sheets Node

```yaml
- id: write_sheets
  name: "Write to Sheets"
  type: google_sheets
  config:
    operation: "append"
    spreadsheet_id: "{{variables.spreadsheet_id}}"
    range: "Sheet1!A:D"
    values: "{{input.rows}}"
```

## Import API

### File Upload (multipart/form-data)

```bash
curl -X POST http://localhost:8585/api/v1/workflows/import \
  -F "file=@workflow.yaml"
```

### Raw YAML Body

```bash
curl -X POST http://localhost:8585/api/v1/workflows/import \
  -H "Content-Type: application/x-yaml" \
  -d @workflow.yaml
```

### Response

```json
{
  "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My Workflow",
  "status": "draft",
  "nodes_count": 5,
  "edges_count": 4,
  "trigger_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

## Export API

### Export as YAML

```bash
curl http://localhost:8585/api/v1/workflows/{id}/export?format=yaml
```

### Export as JSON

```bash
curl http://localhost:8585/api/v1/workflows/{id}/export?format=json
```

## Example Workflows

| File                         | Description                                      |
|------------------------------|--------------------------------------------------|
| `rss_ai_telegram.yaml`       | RSS feed → AI analysis → Telegram notifications  |
| `webhook_conditional.yaml`   | Webhook with conditional routing                 |
| `scheduled_reports.yaml`     | Cron-triggered reports (sales data → AI → email) |
| `data_pipeline.yaml`         | CSV → JSON → Google Sheets ETL                   |
| `ai_content_generation.yaml` | AI content generation with web search            |
| `user_onboarding.yaml`       | User onboarding automation                       |
| `file_processing.yaml`       | File download → process → storage                |
