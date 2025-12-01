# MBFlow Executors Documentation

This directory contains comprehensive documentation for MBFlow's built-in executors.

## Available Executors

### 1. LLM Executor (`llm`)

Integrates Large Language Models (OpenAI, Anthropic) into workflows.

**Features:**
- Text generation and completion
- Function calling / tool use
- Structured outputs with JSON Schema
- Multimodal support (vision, images)
- Vector store integration (RAG)
- Template variable support

**Documentation:**
- [Complete LLM Executor Guide](./LLM_EXECUTOR.md)
- [Function Calling Guide](./FUNCTION_CALLING.md)
- [Workflow Examples](./llm_workflow_examples.md)

**Quick Example:**
```json
{
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "prompt": "Explain {{input.topic}} in simple terms",
    "max_tokens": 200
  }
}
```

### 2. Function Call Executor (`function_call`)

Executes functions called by LLMs, enabling tool use and external API integration.

**Features:**
- Built-in utility functions
- Custom function registration
- Seamless LLM integration
- Error handling
- Multiple input formats

**Built-in Functions:**
- `get_current_time` - Get timestamps in various formats
- `get_current_date` - Get current date
- `http_request` - Make HTTP API calls
- `json_parse` - Parse JSON strings
- `json_stringify` - Convert to JSON
- `get_weather` - Mock weather data (example)

**Documentation:**
- [Function Calling Guide](./FUNCTION_CALLING.md)

**Quick Example:**
```json
{
  "type": "function_call",
  "config": {
    "function_name": "{{input.tool_calls[0].function.name}}",
    "arguments": "{{input.tool_calls[0].function.arguments}}"
  }
}
```

### 3. HTTP Executor (`http`)

Makes HTTP requests to external APIs.

**Quick Example:**
```json
{
  "type": "http",
  "config": {
    "method": "POST",
    "url": "https://api.example.com/data",
    "headers": {"Content-Type": "application/json"},
    "body": {"key": "{{input.value}}"}
  }
}
```

### 4. Transform Executor (`transform`)

Transforms data using expressions, templates, or JQ queries.

**Quick Example:**
```json
{
  "type": "transform",
  "config": {
    "type": "expression",
    "expression": "input.price * (1 - input.discount/100)"
  }
}
```

## Common Patterns

### LLM + Function Calling

```
┌─────────┐      ┌──────────────┐      ┌─────────┐
│ LLM Node│─────▶│Function Call │─────▶│LLM Node │
│         │      │  Executor    │      │         │
└─────────┘      └──────────────┘      └─────────┘
   Detects          Executes           Formats
   need for         function           result
   function
```

### Data Pipeline

```
┌─────────┐      ┌──────────┐      ┌─────────┐
│HTTP Node│─────▶│Transform │─────▶│LLM Node │
│         │      │  Node    │      │         │
└─────────┘      └──────────┘      └─────────┘
 Fetch data      Extract/         Analyze/
                 filter           summarize
```

## Environment Variables

### OpenAI (LLM Executor)

```bash
export OPENAI_API_KEY="sk-..."
export OPENAI_ORG_ID="org-..."         # Optional
export OPENAI_BASE_URL="..."           # Optional, custom endpoint
```

## Registration

Built-in executors are registered via the `RegisterBuiltins` function:

```go
import (
    "github.com/smilemakc/mbflow/pkg/executor"
    "github.com/smilemakc/mbflow/pkg/executor/builtin"
)

func main() {
    manager := executor.NewManager()
    builtin.RegisterBuiltins(manager)

    // Now llm, function_call, http, transform are available
}
```

## File Structure

```
backend/
├── pkg/
│   ├── executor/
│   │   ├── executor.go           # Executor interface
│   │   ├── registry.go           # Executor registry
│   │   └── builtin/
│   │       ├── register.go       # Built-in registration
│   │       ├── llm.go            # LLM executor
│   │       ├── llm_openai.go     # OpenAI provider
│   │       ├── function_call.go  # Function call executor
│   │       ├── http.go           # HTTP executor
│   │       └── transform.go      # Transform executor
│   └── models/
│       ├── llm.go                # LLM models
│       └── function_call.go      # Function call models
└── docs/
    └── executors/
        ├── LLM_EXECUTOR.md       # LLM executor guide
        ├── FUNCTION_CALLING.md   # Function calling guide
        └── llm_workflow_examples.md  # Example workflows
```

## Testing

All executors have comprehensive test coverage:

```bash
# Run all executor tests
go test ./pkg/executor/builtin/...

# Run specific tests
go test -v ./pkg/executor/builtin/ -run TestLLMExecutor
go test -v ./pkg/executor/builtin/ -run TestFunctionCall
```

## Custom Executors

You can create custom executors by implementing the `Executor` interface:

```go
type MyExecutor struct {
    *executor.BaseExecutor
}

func (e *MyExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
    // Your execution logic
    return result, nil
}

func (e *MyExecutor) Validate(config map[string]interface{}) error {
    // Validation logic
    return nil
}

// Register it
manager.Register("my_executor", &MyExecutor{
    BaseExecutor: executor.NewBaseExecutor("my_executor"),
})
```

## Next Steps

1. **Start with basics:** [LLM Executor Guide](./LLM_EXECUTOR.md)
2. **Add tool use:** [Function Calling Guide](./FUNCTION_CALLING.md)
3. **See examples:** [Workflow Examples](./llm_workflow_examples.md)
4. **Build workflows:** Combine executors for complex pipelines

## Support

For issues, questions, or contributions:
- GitHub Issues: https://github.com/smilemakc/mbflow
- Documentation: /docs
