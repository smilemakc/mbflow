# LLM Executor

LLM Executor provides integration with Large Language Model providers (OpenAI, Anthropic) for executing AI-powered tasks within workflows.

## Table of Contents

- [Overview](#overview)
- [Configuration](#configuration)
- [Input Parameter Usage](#input-parameter-usage)
- [Template Resolution](#template-resolution)
- [Supported Providers](#supported-providers)
- [Examples](#examples)

## Overview

The LLM Executor supports:
- Multiple LLM providers (OpenAI, Anthropic)
- Standard Chat Completions API
- OpenAI Responses API for structured outputs
- Function calling and tool usage
- Structured outputs with JSON Schema
- Vision models with image inputs
- RAG with vector stores
- Reasoning models (o3-mini, etc.)

## Configuration

### Basic Configuration

```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "prompt": "Analyze this data: {{input.data}}",
    "instruction": "You are a helpful data analyst",
    "temperature": 0.7,
    "max_tokens": 1000
  }
}
```

### Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | Yes | LLM provider: `openai`, `openai_responses`, `anthropic` |
| `model` | string | Yes | Model name (e.g., `gpt-4`, `gpt-3.5-turbo`, `claude-3-sonnet`) |
| `api_key` | string | Yes | API key for the provider |
| `prompt` | string | Yes | User message/prompt |
| `instruction` | string | No | System message (instruction for the model) |
| `temperature` | float | No | Sampling temperature (0-2, default varies by model) |
| `max_tokens` | int | No | Maximum tokens to generate |
| `top_p` | float | No | Nucleus sampling parameter (0-1) |
| `frequency_penalty` | float | No | Frequency penalty (-2 to 2) |
| `presence_penalty` | float | No | Presence penalty (-2 to 2) |
| `stop_sequences` | []string | No | Stop sequences for generation |
| `tools` | []object | No | Function tools available to the model |
| `response_format` | object | No | Structured output format |
| `use_input_directly` | bool | No | Pass input parameter directly to LLM (useful for Responses API) |

### Provider-Specific Fields

#### OpenAI Provider
- `base_url` - Custom API endpoint (default: https://api.openai.com/v1)
- `org_id` - Organization ID for multi-tenant setups

#### Responses API (`openai_responses`)
- `input` - Structured input for the conversation (string or array of message objects)
- `instructions` - Alternative to `instruction` field
- `background` - Process request in background (bool)
- `hosted_tools` - Built-in OpenAI tools (web_search, file_search, code_interpreter)
- `max_tool_calls` - Limit number of tool call iterations
- `store` - Whether to store the response (default: true)
- `previous_response_id` - Continue from previous conversation

#### Reasoning Models
- `reasoning.effort` - Reasoning effort level (`low`, `medium`, `high`)

## Input Parameter Usage

The `input` parameter contains complete output from parent nodes and can be used in three ways:

### 1. Template-Based Access (Recommended)

Use templates to extract specific fields from parent node output:

```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "prompt": "Summarize this article:\n\nTitle: {{input.title}}\nContent: {{input.content}}"
  }
}
```

**When to use:** Most common use case. When you need specific fields from parent output.

### 2. Direct Input Pass-Through

Pass the entire input object to the LLM request (useful for Responses API):

```json
{
  "type": "llm",
  "config": {
    "provider": "openai_responses",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "instructions": "Process the conversation history",
    "use_input_directly": true
  }
}
```

**When to use:**
- When using Responses API with structured conversation history
- When parent node output is already in the correct format for LLM input
- When you want to pass complex nested structures without flattening

**Input format for Responses API:**
```json
{
  "messages": [
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hi there!"},
    {"role": "user", "content": "How are you?"}
  ]
}
```

### 3. Manual Config Field

Explicitly set `input` field in config (takes precedence over `use_input_directly`):

```json
{
  "type": "llm",
  "config": {
    "provider": "openai_responses",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "instructions": "Continue this conversation",
    "input": "{{input}}"
  }
}
```

**When to use:** When you need full control over input serialization.

### Priority Order

1. Explicit `config.input` field (if set in config)
2. `use_input_directly` flag (if true)
3. Template resolution in `prompt` field (default behavior)

## Template Resolution

Templates in config are automatically resolved before execution:

### Available Template Variables

- `{{env.varName}}` - Workflow/execution variables
- `{{input.fieldName}}` - Parent node outputs
- `{{input.nested.path}}` - Nested field access
- `{{input.items[0].name}}` - Array element access

### Example: Multi-Step Processing

**Workflow:**
```
1. HTTP Request → Fetch user data
2. Transform → Extract relevant fields
3. LLM → Analyze user behavior
```

**Parent node output:**
```json
{
  "user": {
    "id": 123,
    "name": "John Doe",
    "purchases": [
      {"product": "Book", "amount": 29.99},
      {"product": "Coffee", "amount": 4.50}
    ]
  }
}
```

**LLM node config:**
```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "prompt": "Analyze purchase behavior for user {{input.user.name}} (ID: {{input.user.id}}). Recent purchases: {{input.user.purchases}}"
  }
}
```

## Supported Providers

### OpenAI (Chat Completions)

Provider ID: `openai`

Supported models:
- GPT-4 family: `gpt-4`, `gpt-4-turbo`, `gpt-4-turbo-preview`
- GPT-3.5: `gpt-3.5-turbo`
- Vision models: `gpt-4-vision-preview`

### OpenAI (Responses API)

Provider ID: `openai_responses`

Responses API provides structured conversation management with built-in tools.

Supported models: Same as Chat Completions

Special features:
- Structured input/output
- Built-in hosted tools (web search, file search, code interpreter)
- Background processing for long-running tasks
- Response storage and continuation

### Anthropic (Coming Soon)

Provider ID: `anthropic`

Supported models:
- Claude 3 family: `claude-3-opus`, `claude-3-sonnet`, `claude-3-haiku`

## Examples

### Example 1: Simple Text Analysis

```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-3.5-turbo",
    "api_key": "{{env.openai_api_key}}",
    "prompt": "Is this feedback positive or negative? {{input.feedback_text}}",
    "temperature": 0.3,
    "max_tokens": 50
  }
}
```

### Example 2: Structured Output with JSON Schema

```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "prompt": "Extract key information from this text: {{input.text}}",
    "response_format": {
      "type": "json_schema",
      "json_schema": {
        "name": "extraction_result",
        "strict": true,
        "schema": {
          "type": "object",
          "properties": {
            "summary": {"type": "string"},
            "sentiment": {"type": "string", "enum": ["positive", "negative", "neutral"]},
            "key_points": {"type": "array", "items": {"type": "string"}}
          },
          "required": ["summary", "sentiment", "key_points"],
          "additionalProperties": false
        }
      }
    }
  }
}
```

### Example 3: Function Calling

```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "prompt": "What's the weather in {{input.city}}?",
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get current weather for a city",
          "parameters": {
            "type": "object",
            "properties": {
              "city": {"type": "string", "description": "City name"},
              "units": {"type": "string", "enum": ["celsius", "fahrenheit"]}
            },
            "required": ["city"]
          }
        }
      }
    ]
  }
}
```

### Example 4: Vision Model

```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4-vision-preview",
    "api_key": "{{env.openai_api_key}}",
    "prompt": "Describe what you see in this image",
    "image_url": ["{{input.image_url}}"],
    "max_tokens": 300
  }
}
```

### Example 5: Responses API with Direct Input

```json
{
  "type": "llm",
  "config": {
    "provider": "openai_responses",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "instructions": "You are a helpful customer service assistant",
    "use_input_directly": true,
    "hosted_tools": [
      {
        "type": "web_search_preview",
        "domains": ["docs.example.com"],
        "search_context_size": "medium"
      }
    ],
    "max_tool_calls": 5
  }
}
```

**Expected input format:**
```json
{
  "messages": [
    {"role": "user", "content": "I need help with my order #12345"}
  ]
}
```

### Example 6: Reasoning Model (o3-mini)

```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "o3-mini",
    "api_key": "{{env.openai_api_key}}",
    "prompt": "Solve this complex problem: {{input.problem}}",
    "reasoning": {
      "effort": "high"
    }
  }
}
```

### Example 7: RAG with Vector Store

```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "prompt": "Answer this question using the knowledge base: {{input.question}}",
    "vector_store_id": "{{env.vector_store_id}}",
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "retrieval",
          "description": "Search the knowledge base"
        }
      }
    ]
  }
}
```

### Example 8: Continuing Conversations

```json
{
  "type": "llm",
  "config": {
    "provider": "openai_responses",
    "model": "gpt-4",
    "api_key": "{{env.openai_api_key}}",
    "instructions": "Continue the conversation naturally",
    "previous_response_id": "{{input.last_response_id}}",
    "input": "{{input.user_message}}"
  }
}
```

## Output Format

The executor returns a map with the following fields:

```json
{
  "content": "Generated text response",
  "response_id": "resp_abc123",
  "model": "gpt-4",
  "finish_reason": "stop",
  "created_at": 1234567890,
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150
  },
  "tool_calls": [...],
  "metadata": {...}
}
```

### Responses API Output

For `openai_responses` provider, additional fields:

```json
{
  "status": "completed",
  "output_items": [
    {
      "id": "item_abc",
      "type": "message",
      "role": "assistant",
      "content": [
        {"type": "text", "text": "Response text"}
      ]
    }
  ]
}
```

## Error Handling

The executor returns descriptive errors for common issues:

- `failed to parse LLM config` - Invalid configuration
- `provider not found` - Unsupported or unregistered provider
- `LLM execution failed` - API error or network issue
- `function not found` - Tool/function not available (for function calling)

Use workflow-level retry policies to handle transient failures.

## Best Practices

1. **Use templates for most cases** - Template resolution is the most common and straightforward approach
2. **Use `use_input_directly` for Responses API** - When working with conversation history
3. **Set appropriate timeouts** - LLM calls can be slow, especially with reasoning models
4. **Handle rate limits** - Implement retry logic with exponential backoff
5. **Monitor token usage** - Track usage in the output for cost optimization
6. **Use structured outputs** - JSON Schema ensures consistent, parseable responses
7. **Test with cheaper models first** - Use `gpt-3.5-turbo` for development
8. **Store API keys in variables** - Never hardcode keys, use `{{env.api_key}}`

## Related Documentation

- [Template Engine](../TEMPLATE_ENGINE.md)
- [Transform Executor](../TRANSFORM_EXECUTOR.md)
- [Function Call Executor](./FUNCTION_CALL_EXECUTOR.md)
- [Node Executor Implementation](../internal/node_executor.md)