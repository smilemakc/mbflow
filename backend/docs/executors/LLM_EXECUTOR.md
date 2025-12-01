# LLM Executor Guide

The LLM Executor provides comprehensive integration with Large Language Model providers (currently OpenAI, with
extensibility for others like Anthropic).

## Table of Contents

- [Overview](#overview)
- [Setup](#setup)
- [Basic Usage](#basic-usage)
- [Configuration Options](#configuration-options)
- [Features](#features)
    - [Basic Text Generation](#basic-text-generation)
    - [System Instructions](#system-instructions)
    - [Function Calling](#function-calling)
    - [Structured Outputs](#structured-outputs)
    - [Multimodal (Vision)](#multimodal-vision)
    - [Vector Store Integration](#vector-store-integration)
- [Provider-Specific Details](#provider-specific-details)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The LLM Executor (`type: llm`) enables workflows to leverage large language models for:

- Text generation and completion
- Question answering
- Data extraction and transformation
- Function calling / tool use
- Structured JSON output
- Vision/multimodal tasks
- RAG (Retrieval-Augmented Generation) with vector stores

## Setup

### Environment Variables

#### OpenAI Provider

```bash
# Required
export OPENAI_API_KEY="sk-..."

# Optional
export OPENAI_ORG_ID="org-..."        # Organization ID
export OPENAI_BASE_URL="..."          # Custom base URL (default: https://api.openai.com/v1)
```

### Workflow Registration

Built-in executors are registered automatically when you create a manager:

```go
import (
"github.com/smilemakc/mbflow/pkg/executor"
"github.com/smilemakc/mbflow/pkg/executor/builtin"
)

manager := executor.NewManager()
builtin.RegisterBuiltins(manager) // Registers llm, function_call, http, transform
```

## Basic Usage

### Minimal Configuration

```json
{
  "id": "llm1",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "Explain quantum computing in simple terms"
  }
}
```

### With System Instruction

```json
{
  "id": "llm1",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "instruction": "You are a helpful assistant that explains technical concepts clearly.",
    "prompt": "What is Docker?",
    "max_tokens": 500,
    "temperature": 0.7
  }
}
```

## Using Input from Previous Nodes

LLM executor supports templates to access output from parent nodes using `{{input.field}}` syntax. This enables chaining
LLM calls and building complex multi-step workflows.

### How It Works

The template resolution happens **before** the executor runs:

1. Workflow engine wraps the executor in `TemplateExecutorWrapper`
2. Output from the parent node is placed in `ExecutionContextData.ParentNodeOutput`
3. Template engine maps `ParentNodeOutput` to `InputVars`
4. Templates like `{{input.field}}` are resolved in the config
5. Executor receives the resolved configuration

This pattern allows seamless data flow between workflow nodes without requiring manual data passing.

### Simple Example: Chaining LLM Calls

**Node 1: Extract Information**

```json
{
  "id": "extract_topic",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "Extract the main topic from this document: {{input.document}}",
    "temperature": 0.0
  }
}
```

**Node 2: Generate Summary**

```json
{
  "id": "generate_summary",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "Write a detailed summary about: {{input.content}}",
    "instruction": "Focus on {{input.topic}} aspects. Keep it concise.",
    "max_tokens": 300
  }
}
```

In this workflow, Node 2 receives the output from Node 1 as `input`, and the templates `{{input.content}}` and
`{{input.topic}}` are automatically resolved from the LLM response.

### Complex Input Examples

**Nested Objects:**

```json
{
  "id": "user_analysis",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "Analyze user {{input.user.profile.name}} (ID: {{input.user.id}}) from {{input.user.profile.location}}"
  }
}
```

If the input is:

```json
{
  "user": {
    "id": "12345",
    "profile": {
      "name": "Alice Smith",
      "location": "San Francisco"
    }
  }
}
```

The resolved prompt becomes:

```
Analyze user Alice Smith (ID: 12345) from San Francisco
```

**Arrays:**

```json
{
  "id": "compare_items",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "Compare {{input.items[0].name}} and {{input.items[1].name}}. Which is better for {{input.useCase}}?",
    "temperature": 0.3
  }
}
```

**Combining Environment and Input Variables:**

```json
{
  "id": "smart_assistant",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "{{env.preferred_model}}",
    "instruction": "You are a {{env.assistant_role}} expert.",
    "prompt": "Help with this {{input.taskType}}: {{input.taskDescription}}",
    "temperature": 0.7
  }
}
```

This example shows:

- `{{env.X}}`: Workflow/execution variables for configuration
- `{{input.X}}`: Dynamic data from parent node output

### Multi-Step Code Analysis Workflow

**Step 1: Code Review**

```json
{
  "id": "code_review",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "instruction": "You are an expert code reviewer focusing on security and best practices.",
    "prompt": "Review this {{input.language}} code:\n\n{{input.code}}\n\nProvide security issues and suggestions.",
    "temperature": 0.2
  }
}
```

**Step 2: Generate Refactored Version**

```json
{
  "id": "refactor_code",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "instruction": "You are a code refactoring expert.",
    "prompt": "Based on this review:\n{{input.content}}\n\nRefactor the original code to address all issues.",
    "temperature": 0.1,
    "max_tokens": 1000
  }
}
```

**Step 3: Explain Changes**

```json
{
  "id": "explain_changes",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-3.5-turbo",
    "prompt": "Explain the changes made in this refactoring:\n{{input.content}}\n\nMake it understandable for junior developers.",
    "temperature": 0.5,
    "max_tokens": 500
  }
}
```

### Important Notes

- **Template Resolution Timing**: Templates are resolved by `TemplateExecutorWrapper` **before** the executor runs
- **Input Parameter**: The `input` parameter contains the complete output from the parent node
- **Variable Types**:
    - `{{env.X}}`: Workflow/execution variables (configuration, API keys, settings)
    - `{{input.X}}`: Parent node output (dynamic data flow)
- **Strict Mode**: If enabled, templates with missing variables will cause validation errors
- **Type Safety**: All template values are converted to strings during resolution

### Programmatic Usage with SDK

When using the SDK, the workflow engine automatically creates the wrapper:

```go
import (
"context"
"github.com/smilemakc/mbflow/pkg/executor"
"github.com/smilemakc/mbflow/pkg/executor/builtin"
"github.com/smilemakc/mbflow/internal/application/template"
)

// Create executor
llmExec := builtin.NewLLMExecutor()

// Output from previous node
previousNodeOutput := map[string]interface{}{
"code": "func main() { ... }",
"language": "Go",
}

// Create execution context
execCtx := &executor.ExecutionContextData{
WorkflowVariables: map[string]interface{}{
"openai_api_key": "sk-...",
},
ParentNodeOutput: previousNodeOutput,
StrictMode: true,
}

// Create template engine and wrapper
engine := executor.NewTemplateEngine(execCtx)
wrappedExec := executor.NewTemplateExecutorWrapper(llmExec, engine)

// Configuration with templates
config := map[string]interface{}{
"provider": "openai",
"model": "gpt-4",
"prompt": "Review this {{input.language}} code:\n{{input.code}}",
}

// Execute (templates are automatically resolved)
result, err := wrappedExec.Execute(context.Background(), config, previousNodeOutput)
```

## Configuration Options

### Required Fields

| Field      | Type   | Description                                     |
|------------|--------|-------------------------------------------------|
| `provider` | string | LLM provider (`"openai"`, `"anthropic"`)        |
| `model`    | string | Model name (e.g., `"gpt-4"`, `"gpt-3.5-turbo"`) |
| `prompt`   | string | User prompt/question                            |

### Optional Fields

| Field                  | Type    | Default | Description                              |
|------------------------|---------|---------|------------------------------------------|
| `instruction`          | string  | -       | System message to set context/behavior   |
| `max_tokens`           | integer | -       | Maximum tokens in response               |
| `temperature`          | float   | 1.0     | Randomness (0.0-2.0)                     |
| `top_p`                | float   | 1.0     | Nucleus sampling (0.0-1.0)               |
| `frequency_penalty`    | float   | 0.0     | Penalize frequent tokens (-2.0 to 2.0)   |
| `presence_penalty`     | float   | 0.0     | Penalize present tokens (-2.0 to 2.0)    |
| `stop_sequences`       | array   | -       | Stop generation at these strings         |
| `vector_store_id`      | string  | -       | OpenAI vector store ID for RAG           |
| `image_url`            | array   | -       | Image URLs for vision models             |
| `image_id`             | array   | -       | OpenAI file IDs for images               |
| `file_id`              | array   | -       | OpenAI file IDs for documents            |
| `tools`                | array   | -       | Function definitions for tool calling    |
| `response_format`      | object  | -       | Structured output format                 |
| `previous_response_id` | string  | -       | Continue conversation from this response |

## Features

### Basic Text Generation

Simple question-answering:

```json
{
  "id": "summarize",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "Summarize this article: {{input.article}}",
    "max_tokens": 150
  }
}
```

### System Instructions

Control the LLM's behavior and persona:

```json
{
  "id": "code_reviewer",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "instruction": "You are an expert code reviewer. Focus on security, performance, and best practices.",
    "prompt": "Review this code:\n{{input.code}}",
    "temperature": 0.3
  }
}
```

### Function Calling

Define functions the LLM can call:

```json
{
  "id": "weather_assistant",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "What's the weather in {{input.city}}?",
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get current weather for a location",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {
                "type": "string",
                "description": "City name"
              },
              "unit": {
                "type": "string",
                "enum": [
                  "celsius",
                  "fahrenheit"
                ]
              }
            },
            "required": [
              "location"
            ]
          }
        }
      }
    ]
  }
}
```

**Output format when function is called:**

```json
{
  "content": "",
  "finish_reason": "tool_calls",
  "tool_calls": [
    {
      "id": "call_abc123",
      "type": "function",
      "function": {
        "name": "get_weather",
        "arguments": "{\"location\":\"London\",\"unit\":\"celsius\"}"
      }
    }
  ]
}
```

See [FUNCTION_CALLING.md](./FUNCTION_CALLING.md) for complete workflows.

### Structured Outputs

Ensure LLM returns valid JSON conforming to a schema:

#### JSON Object Mode

```json
{
  "id": "extract_info",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "Extract user information: {{input.text}}",
    "response_format": {
      "type": "json_object"
    }
  }
}
```

#### JSON Schema Mode (Strict)

```json
{
  "id": "extract_structured",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "Extract contact info from: {{input.email_body}}",
    "response_format": {
      "type": "json_schema",
      "json_schema": {
        "name": "contact_info",
        "description": "Contact information extraction",
        "strict": true,
        "schema": {
          "type": "object",
          "properties": {
            "name": {
              "type": "string"
            },
            "email": {
              "type": "string"
            },
            "phone": {
              "type": "string"
            },
            "company": {
              "type": "string"
            }
          },
          "required": [
            "name",
            "email"
          ],
          "additionalProperties": false
        }
      }
    }
  }
}
```

### Multimodal (Vision)

Analyze images with vision models:

```json
{
  "id": "image_analysis",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4-vision-preview",
    "prompt": "Describe what's in this image in detail",
    "image_url": [
      "{{input.imageUrl}}"
    ],
    "max_tokens": 500
  }
}
```

Multiple images:

```json
{
  "id": "compare_images",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4-vision-preview",
    "prompt": "Compare these two images and highlight the differences",
    "image_url": [
      "{{input.image1Url}}",
      "{{input.image2Url}}"
    ]
  }
}
```

### Vector Store Integration

Use OpenAI vector stores for RAG:

```json
{
  "id": "knowledge_query",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "{{input.question}}",
    "vector_store_id": "vs_abc123xyz",
    "instruction": "Answer using the knowledge base. Cite sources when possible."
  }
}
```

**Note:** Vector store integration requires the Assistants API. The basic chat completions endpoint does not directly
support vector stores. You'll need to implement RAG manually or use the Assistants API.

## Provider-Specific Details

### OpenAI

**Supported Models:**

- `gpt-4` - Most capable, best for complex tasks
- `gpt-4-turbo` - Faster, cheaper than GPT-4
- `gpt-3.5-turbo` - Fast, good for simple tasks
- `gpt-4-vision-preview` - Multimodal (text + images)
- `gpt-4o` - Latest optimized model

**Model Selection Guidelines:**

- Use `gpt-4` for: complex reasoning, code generation, analysis
- Use `gpt-3.5-turbo` for: simple Q&A, classification, summarization
- Use `gpt-4-vision-preview` for: image analysis, OCR, visual Q&A

## Error Handling

### Common Errors

**Invalid API Key:**

```
LLM error (openai): Incorrect API key provided
```

Solution: Check `OPENAI_API_KEY` environment variable

**Rate Limiting:**

```
LLM error (openai): Rate limit exceeded
```

Solution: Implement retry logic or reduce request frequency

**Token Limit Exceeded:**

```
LLM error (openai): This model's maximum context length is 4096 tokens
```

Solution: Reduce `prompt` length or use a model with larger context window

**Invalid Configuration:**

```
required field missing: provider
```

Solution: Ensure all required fields are provided

### Retry Strategy

The LLM executor does not implement automatic retries. Use MBFlow's built-in retry mechanism:

```json
{
  "id": "llm1",
  "type": "llm",
  "config": {
    "..."
  },
  "retry": {
    "max_attempts": 3,
    "backoff_seconds": 2
  }
}
```

## Best Practices

### 1. Use Appropriate Temperature

```json
// Deterministic tasks (data extraction, classification)
{
  "temperature": 0.0
}

// Creative tasks (writing, brainstorming)
{
  "temperature": 0.8
}

// Balanced (general Q&A)
{
  "temperature": 0.7
}
```

### 2. Set Token Limits

Always set `max_tokens` to prevent excessive costs:

```json
{
  "max_tokens": 500
  // Adjust based on expected response length
}
```

### 3. Use System Instructions

System instructions improve consistency:

```json
{
  "instruction": "You are a helpful assistant. Keep responses concise and factual."
}
```

### 4. Template Variables

Leverage template substitution for dynamic prompts:

```json
{
  "prompt": "Analyze this {{input.dataType}}: {{input.data}}",
  "instruction": "You are a {{env.expertRole}} expert."
}
```

### 5. Structured Outputs for Data Extraction

Use JSON schema mode for reliable parsing:

```json
{
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "name": "extracted_data",
      "strict": true,
      "schema": {
        "..."
      }
    }
  }
}
```

### 6. Cost Optimization

- Use `gpt-3.5-turbo` when possible
- Set appropriate `max_tokens`
- Cache responses for repeated queries
- Use `stop_sequences` to prevent unnecessary generation

## Troubleshooting

### Issue: Empty Response

**Symptoms:** `content` field is empty

**Causes:**

1. Function call was triggered (`finish_reason: "tool_calls"`)
2. Content filter activated
3. Stop sequence matched immediately

**Solution:**

- Check `finish_reason` field
- Review `tool_calls` array
- Verify `stop_sequences` configuration

### Issue: Inconsistent Outputs

**Symptoms:** Same prompt produces different results

**Causes:**

- High `temperature` setting
- Non-deterministic model behavior

**Solution:**

- Set `temperature: 0.0` for consistent results
- Use `seed` parameter (if supported)
- Use structured outputs with JSON schema

### Issue: Slow Response Times

**Symptoms:** Long execution times

**Causes:**

- Large `max_tokens` value
- Complex prompts
- Model selection (GPT-4 slower than GPT-3.5-turbo)

**Solution:**

- Reduce `max_tokens`
- Simplify prompts
- Use faster models when appropriate
- Implement timeout handling

### Issue: Function Calls Not Working

**Symptoms:** LLM doesn't call defined functions

**Causes:**

- Function descriptions unclear
- Parameters not well-defined
- Prompt doesn't indicate need for function

**Solution:**

- Improve function `description` field
- Use clear parameter names and descriptions
- Explicitly mention function capabilities in prompt

## Response Format

All LLM executors return a standardized response:

```json
{
  "content": "The assistant's response text",
  "response_id": "resp_abc123",
  "model": "gpt-4",
  "usage": {
    "prompt_tokens": 50,
    "completion_tokens": 100,
    "total_tokens": 150
  },
  "tool_calls": [
    {
      "id": "call_xyz",
      "type": "function",
      "function": {
        "name": "function_name",
        "arguments": "{\"arg1\":\"value1\"}"
      }
    }
  ],
  "finish_reason": "stop",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Fields:**

- `content`: Generated text (empty if function was called)
- `response_id`: Unique identifier for this response
- `model`: Model that generated the response
- `usage`: Token consumption statistics
- `tool_calls`: Function calls (if any)
- `finish_reason`: Why generation stopped (`"stop"`, `"length"`, `"tool_calls"`, `"content_filter"`)
- `created_at`: Timestamp

## Next Steps

- [Function Calling Guide](./FUNCTION_CALLING.md) - Complete guide to using function calling
- [Workflow Examples](./llm_workflow_examples.md) - Real-world workflow examples
- [API Reference](../api/) - Full API documentation
