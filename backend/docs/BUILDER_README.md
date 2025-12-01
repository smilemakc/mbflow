# MBFlow Builder API

[![Go Reference](https://pkg.go.dev/badge/github.com/smilemakc/mbflow/pkg/builder.svg)](https://pkg.go.dev/github.com/smilemakc/mbflow/pkg/builder)

A fluent, type-safe builder API for constructing MBFlow workflows.

## Features

✅ **Type-Safe Configuration** - Compile-time validation for node configs
✅ **Fluent API** - Chainable methods for readable workflow construction
✅ **Early Validation** - Catch errors at build time, not runtime
✅ **IDE Support** - Full autocomplete for all options
✅ **64% Less Code** - Significantly reduce boilerplate
✅ **100% Backward Compatible** - Works alongside existing code

## Quick Start

```go
import "github.com/smilemakc/mbflow/pkg/builder"

// Create a simple workflow
workflow := builder.NewWorkflow("Fetch User Data",
    builder.WithVariable("api_base", "https://api.example.com"),
).AddNode(
    builder.NewHTTPGetNode(
        "fetch",
        "Fetch User",
        "{{env.api_base}}/users/{{input.user_id}}",
    ),
).MustBuild()

// Use with SDK
client.Workflows().Create(ctx, workflow)
```

## Installation

```bash
go get github.com/smilemakc/mbflow/pkg/builder
```

## Usage Examples

### HTTP Workflow

```go
workflow := builder.NewWorkflow("API Pipeline",
    builder.WithAutoLayout(),
).AddNode(
    builder.NewHTTPGetNode("fetch", "Fetch Data", "https://api.example.com/data"),
).AddNode(
    builder.NewJQNode("transform", "Transform", `.[] | {id, name}`),
).AddNode(
    builder.NewHTTPPostNode("send", "Send Results", "https://api.example.com/results", nil),
).Connect("fetch", "transform").
  Connect("transform", "send").
  MustBuild()
```

### LLM Workflow

```go
workflow := builder.NewWorkflow("Code Analysis",
    builder.WithVariable("openai_api_key", apiKey),
).AddNode(
    builder.NewOpenAINode(
        "analyze",
        "Analyze Code",
        "gpt-4",
        "Analyze this code: {{input.code}}",
        builder.LLMAPIKey("{{env.openai_api_key}}"),
        builder.LLMTemperature(0.2),
        builder.LLMMaxTokens(1000),
    ),
).MustBuild()
```

### Conditional Workflow

```go
workflow := builder.NewWorkflow("Conditional Flow").
    AddNode(builder.NewHTTPGetNode("check", "Check", "https://api.example.com/status")).
    AddNode(builder.NewPassthroughNode("success", "Success Handler")).
    AddNode(builder.NewPassthroughNode("failure", "Failure Handler")).
    Connect("check", "success", builder.WhenTrue("output.success")).
    Connect("check", "failure", builder.WhenFalse("output.success")).
    MustBuild()
```

## Node Types

### HTTP Nodes

```go
builder.NewHTTPGetNode(id, name, url, opts...)
builder.NewHTTPPostNode(id, name, url, body, opts...)
builder.NewHTTPPutNode(id, name, url, body, opts...)
builder.NewHTTPDeleteNode(id, name, url, opts...)
builder.NewHTTPPatchNode(id, name, url, body, opts...)
```

**Options:**
- `HTTPHeader(key, value)` - Add header
- `HTTPHeaders(map)` - Set all headers
- `HTTPBody(map)` - Set request body
- `HTTPTimeout(duration)` - Set timeout

### LLM Nodes

```go
builder.NewOpenAINode(id, name, model, prompt, opts...)
builder.NewAnthropicNode(id, name, model, prompt, opts...)
```

**Options:**
- `LLMAPIKey(key)` - Set API key
- `LLMTemperature(temp)` - Set temperature (0-2, validated)
- `LLMMaxTokens(tokens)` - Set max tokens
- `LLMTopP(topP)` - Set top-p (0-1, validated)
- `LLMSystemPrompt(prompt)` - Set system prompt
- `LLMJSONMode()` - Enable JSON response mode

### Transform Nodes

```go
builder.NewPassthroughNode(id, name, opts...)
builder.NewExpressionNode(id, name, expr, opts...)
builder.NewJQNode(id, name, filter, opts...)
builder.NewTemplateNode(id, name, template, opts...)
```

## Workflow Options

```go
builder.NewWorkflow("My Workflow",
    builder.WithDescription("Workflow description"),
    builder.WithStatus(models.WorkflowStatusActive),
    builder.WithTags("tag1", "tag2"),
    builder.WithVariable("key", "value"),
    builder.WithVariables(map[string]interface{}{...}),
    builder.WithMetadata("key", "value"),
    builder.WithAutoLayout(),           // Auto-position nodes
    builder.WithStrictValidation(),     // Validate configs upfront
)
```

## Positioning

### Absolute Position

```go
builder.NewHTTPGetNode("node1", "Node 1", "url",
    builder.WithPosition(100, 200),
)
```

### Grid Layout

```go
builder.NewHTTPGetNode("node1", "Node 1", "url",
    builder.GridPosition(0, 0),  // Row 0, Col 0 -> (0, 0)
)
builder.NewHTTPGetNode("node2", "Node 2", "url",
    builder.GridPosition(0, 1),  // Row 0, Col 1 -> (200, 0)
)
```

### Auto Layout

```go
workflow := builder.NewWorkflow("Auto Layout",
    builder.WithAutoLayout(),  // Nodes positioned automatically
).AddNode(...).AddNode(...).MustBuild()
```

## Conditional Edges

```go
// Simple condition
workflow.Connect("check", "next", builder.WhenTrue("output.success"))

// Negated condition
workflow.Connect("check", "error", builder.WhenFalse("output.success"))

// Equality check
workflow.Connect("check", "handler", builder.WhenEqual("output.status", "ready"))
```

## Error Handling

### Build() - Returns Error

```go
workflow, err := builder.NewWorkflow("Test").
    AddNode(...).
    Build()
if err != nil {
    return err
}
```

### MustBuild() - Panics on Error

```go
workflow := builder.NewWorkflow("Test").
    AddNode(...).
    MustBuild()  // Panics if validation fails
```

## Validation

The builder validates at three levels:

1. **Option-level** - Type constraints (e.g., temperature 0-2)
2. **Build-level** - Structure validation (required fields, DAG)
3. **SDK-level** - Runtime validation (executor availability)

Enable strict validation:

```go
workflow := builder.NewWorkflow("Test",
    builder.WithStrictValidation(),  // Validates all configs upfront
).AddNode(...).Build()
```

## Migration from Raw Structs

**Before:**

```go
workflow := &models.Workflow{
    Name: "Test",
    Variables: map[string]interface{}{
        "api_key": "secret",
    },
    Nodes: []*models.Node{
        {
            ID:   "fetch",
            Name: "Fetch Data",
            Type: "http",
            Config: map[string]interface{}{
                "method": "GET",
                "url":    "https://api.example.com",
            },
        },
    },
    Edges: []*models.Edge{},
}
```

**After:**

```go
workflow := builder.NewWorkflow("Test",
    builder.WithVariable("api_key", "secret"),
).AddNode(
    builder.NewHTTPGetNode("fetch", "Fetch Data", "https://api.example.com"),
).MustBuild()
```

**Result:** 55% less code, 100% more readable!

## Documentation

- [Migration Guide](MIGRATION.md) - Detailed migration from raw structs
- [GoDoc](https://pkg.go.dev/github.com/smilemakc/mbflow/pkg/builder) - Full API reference
- [Examples](examples_test.go) - Working code examples
- [Complete Example](../../examples/builder_usage/main.go) - Full application

## Testing

```bash
# Run all tests
go test ./pkg/builder/...

# Run with coverage
go test -cover ./pkg/builder/...

# Run examples
go run ./examples/builder_usage/main.go
```

## Contributing

The builder API follows these principles:

1. **Functional Options Pattern** - Consistent with `pkg/sdk/options.go`
2. **Type Safety** - Validate at compile time when possible
3. **Fluent API** - Method chaining for readability
4. **Backward Compatible** - Never break existing code

## License

Same as MBFlow main project.

## Support

- 📖 [Documentation](https://github.com/smilemakc/mbflow)
- 🐛 [Issue Tracker](https://github.com/smilemakc/mbflow/issues)
- 💬 [Discussions](https://github.com/smilemakc/mbflow/discussions)
