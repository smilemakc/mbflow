# Function Calling Guide

This guide explains how to use the LLM Executor's function calling feature with the FunctionCallExecutor.

## Overview

Function calling allows LLMs to:
1. Detect when a function should be called
2. Extract arguments from natural language
3. Return structured function call requests
4. Integrate external APIs and tools

## Architecture

```
┌─────────┐      ┌─────────────┐      ┌──────────────────┐
│ LLM Node│─────▶│ Detects need│─────▶│ FunctionCall Node│
│         │      │ for function│      │                  │
└─────────┘      └─────────────┘      └──────────────────┘
                                              │
                                              ▼
                                      ┌───────────────┐
                                      │ Execute Func  │
                                      │ Return Result │
                                      └───────────────┘
```

## Basic Workflow

### Step 1: Define Functions in LLM Node

```json
{
  "id": "llm1",
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
              "location": {"type": "string", "description": "City name"},
              "unit": {"type": "string", "enum": ["celsius", "fahrenheit"]}
            },
            "required": ["location"]
          }
        }
      }
    ]
  }
}
```

### Step 2: Handle Function Call

```json
{
  "id": "func1",
  "type": "function_call",
  "config": {
    "function_name": "{{input.tool_calls[0].function.name}}",
    "arguments": "{{input.tool_calls[0].function.arguments}}",
    "tool_call_id": "{{input.tool_calls[0].id}}"
  }
}
```

### Step 3: Connect Nodes

```json
{
  "edges": [
    {"from": "llm1", "to": "func1"}
  ]
}
```

## Built-in Functions

The FunctionCallExecutor includes several built-in functions:

### get_current_time

Get current timestamp in various formats.

```json
{
  "function_name": "get_current_time",
  "arguments": "{\"format\": \"unix\"}"
}
```

**Parameters:**
- `format` (string): "RFC3339", "unix", "iso8601", or custom Go time format

### get_current_date

Get current date in YYYY-MM-DD format.

```json
{
  "function_name": "get_current_date",
  "arguments": "{}"
}
```

### http_request

Make HTTP requests.

```json
{
  "function_name": "http_request",
  "arguments": "{\"method\": \"GET\", \"url\": \"https://api.example.com/data\"}"
}
```

**Parameters:**
- `method` (string): HTTP method
- `url` (string, required): Request URL
- `body` (object): Request body
- `headers` (object): HTTP headers

### json_parse

Parse JSON string to object.

```json
{
  "function_name": "json_parse",
  "arguments": "{\"json\": \"{\\\"name\\\":\\\"John\\\"}\"}"
}
```

### json_stringify

Convert object to JSON string.

```json
{
  "function_name": "json_stringify",
  "arguments": "{\"value\": {\"name\": \"John\"}, \"pretty\": true}"
}
```

### get_weather (Mock)

Example weather function (returns mock data).

```json
{
  "function_name": "get_weather",
  "arguments": "{\"location\": \"London\", \"unit\": \"celsius\"}"
}
```

## Custom Functions

Register custom functions at runtime:

```go
import (
    "github.com/smilemakc/mbflow/pkg/executor/builtin"
    "github.com/smilemakc/mbflow/pkg/models"
)

// Get the function call executor
manager := executor.NewManager()
builtin.RegisterBuiltins(manager)

funcExec, _ := manager.Get("function_call")
functionCallExec := funcExec.(*builtin.FunctionCallExecutor)

// Register custom function
functionCallExec.RegisterFunction("calculate_discount", func(args map[string]interface{}) (interface{}, error) {
    price := args["price"].(float64)
    discount := args["discount_percent"].(float64)

    finalPrice := price * (1 - discount/100)

    return map[string]interface{}{
        "original_price": price,
        "discount": discount,
        "final_price": finalPrice,
        "savings": price - finalPrice,
    }, nil
})
```

## Complete Workflow Example

Here's a complete multi-step workflow with function calling:

```json
{
  "name": "Weather Assistant Workflow",
  "nodes": [
    {
      "id": "llm1",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "instruction": "You are a weather assistant. Use the get_weather function to answer questions.",
        "prompt": "{{input.userMessage}}",
        "tools": [
          {
            "type": "function",
            "function": {
              "name": "get_weather",
              "description": "Get current weather for a location",
              "parameters": {
                "type": "object",
                "properties": {
                  "location": {"type": "string"},
                  "unit": {"type": "string", "enum": ["celsius", "fahrenheit"]}
                },
                "required": ["location"]
              }
            }
          }
        ]
      }
    },
    {
      "id": "func1",
      "type": "function_call",
      "config": {
        "function_name": "{{input.tool_calls[0].function.name}}",
        "arguments": "{{input.tool_calls[0].function.arguments}}"
      }
    },
    {
      "id": "llm2",
      "type": "llm",
      "config": {
        "provider": "openai",
        "model": "gpt-4",
        "prompt": "Format this weather data nicely: {{input.result}}"
      }
    }
  ],
  "edges": [
    {"from": "llm1", "to": "func1"},
    {"from": "func1", "to": "llm2"}
  ]
}
```

## Advanced Patterns

### Multiple Function Calls

Handle workflows where LLM makes multiple function calls:

```json
{
  "id": "func_handler",
  "type": "function_call",
  "config": {
    "function_name": "{{input.tool_calls[0].function.name}}",
    "arguments": "{{input.tool_calls[0].function.arguments}}"
  }
}
```

For multiple calls, add a loop or use conditional routing.

### Conditional Function Execution

Use conditional nodes to route based on function name:

```json
{
  "id": "router",
  "type": "conditional",
  "config": {
    "condition": "{{input.tool_calls[0].function.name}} == 'get_weather'"
  }
}
```

### Error Handling

Function call executor returns errors gracefully:

```json
{
  "result": null,
  "function_name": "nonexistent",
  "success": false,
  "error": "function not found: nonexistent"
}
```

Check `success` field before using `result`.

## Best Practices

1. **Clear Function Descriptions**: Write detailed, unambiguous function descriptions
2. **Strict Parameters**: Use JSON Schema with `required` fields
3. **Error Handling**: Always check function call output `success` field
4. **Idempotency**: Make functions idempotent when possible
5. **Timeouts**: Set appropriate timeouts for external API calls
6. **Logging**: Log function executions for debugging

## Troubleshooting

### LLM Doesn't Call Function

- Ensure function description clearly explains when to use it
- Make prompt explicitly mention the function's purpose
- Check that parameters schema is valid JSON Schema

### Function Call Fails

- Validate arguments JSON is properly formatted
- Check function is registered in FunctionCallExecutor
- Review function handler for errors

### Wrong Arguments Extracted

- Improve parameter descriptions
- Add examples in function description
- Use `enum` for constrained values

## Next Steps

- [LLM Executor Guide](./LLM_EXECUTOR.md)
- [Workflow Examples](./llm_workflow_examples.md)
