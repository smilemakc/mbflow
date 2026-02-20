# Template Engine Package

## Overview

This package provides a robust template engine for variable substitution in MBFlow workflow configurations. It supports nested paths, array indexing, and configurable error handling.

## Package Structure

```
template/
├── types.go         # Core types, errors, and interfaces
├── resolver.go      # Variable resolution logic
├── engine.go        # Main template engine
├── engine_test.go   # Comprehensive test suite
└── README.md        # This file
```

## Core Components

### Types (`types.go`)

#### VariableContext

Holds all variables available for template resolution with proper precedence:

```go
type VariableContext struct {
    WorkflowVars  map[string]any // Workflow-level variables
    ExecutionVars map[string]any // Runtime variables (override workflow)
    InputVars     map[string]any // Parent node output
}
```

#### TemplateOptions

Configures template resolution behavior:

```go
type TemplateOptions struct {
    StrictMode           bool // Error on missing variables
    PlaceholderOnMissing bool // Keep placeholder when variable missing
}
```

#### Error Types

```go
var (
    ErrVariableNotFound   = errors.New("variable not found")
    ErrInvalidPath        = errors.New("invalid path")
    ErrInvalidTemplate    = errors.New("invalid template syntax")
    ErrTypeNotSupported   = errors.New("type not supported for path traversal")
    ErrArrayIndexInvalid  = errors.New("invalid array index")
    ErrArrayOutOfBounds   = errors.New("array index out of bounds")
)
```

### Resolver (`resolver.go`)

Handles variable resolution with support for:
- Nested object paths (`user.profile.email`)
- Array indexing (`items[0].name`)
- Multi-dimensional arrays (`matrix[0][1]`)
- Type conversions

```go
type Resolver struct {
    context *VariableContext
    options TemplateOptions
}
```

### Engine (`engine.go`)

Main template engine that:
- Parses template strings
- Resolves variables recursively
- Handles complex data structures
- Provides utility functions

```go
type Engine struct {
    resolver *Resolver
    options  TemplateOptions
}
```

## API Reference

### Creating an Engine

```go
// Create variable context
ctx := NewVariableContext()
ctx.WorkflowVars["apiUrl"] = "https://api.example.com"
ctx.ExecutionVars["apiKey"] = "secret-123"
ctx.InputVars["userId"] = "user-456"

// Create engine with options
opts := TemplateOptions{
    StrictMode: false,
    PlaceholderOnMissing: false,
}
engine := NewEngine(ctx, opts)

// Or use defaults
engine := NewEngineWithDefaults(ctx)
```

### Resolving Templates

#### String Resolution

```go
result, err := engine.ResolveString("Hello {{input.userId}}")
// result: "Hello user-456"
```

#### Data Structure Resolution

```go
config := map[string]any{
    "url": "{{env.apiUrl}}/users/{{input.userId}}",
    "headers": map[string]any{
        "Authorization": "Bearer {{env.apiKey}}",
    },
}

resolved, err := engine.Resolve(config)
// resolved: map with all templates resolved
```

#### Config Resolution

```go
resolved, err := engine.ResolveConfig(config)
// Convenience method for map[string]any
```

### Utility Functions

#### Check for Templates

```go
hasTemplates := HasTemplates("Hello {{env.name}}")
// hasTemplates: true
```

#### Extract Variables

```go
vars := ExtractVariables("{{input.greeting}} {{env.name}}")
// vars: ["input.greeting", "env.name"]
```

#### Validate Template

```go
err := ValidateTemplate("{{env.apiUrl}}/users")
// err: nil (valid template)

err := ValidateTemplate("{{unknown.var}}")
// err: unknown variable type 'unknown'
```

## Template Syntax

### Basic Format

```
{{type.path.to.value}}
```

### Supported Variable Types

- `env` - Environment/workflow variables
- `input` - Parent node output

### Path Expressions

```go
// Simple field
"{{env.apiKey}}"

// Nested field
"{{input.user.profile.email}}"

// Array element
"{{input.items[0]}}"

// Array element field
"{{input.users[1].name}}"

// Nested array
"{{input.data.results[0].items[2].id}}"

// Multi-dimensional array
"{{input.matrix[0][1]}}"
```

## Variable Resolution Precedence

When resolving `{{env.varName}}`:

1. Check `ExecutionVars` (highest priority)
2. Check `WorkflowVars` (lower priority)
3. Return error if not found (strict mode) or empty string (non-strict)

## Error Handling

### Strict Mode

```go
opts := TemplateOptions{StrictMode: true}
engine := NewEngine(ctx, opts)

_, err := engine.ResolveString("{{env.missing}}")
// err: variable not found: {{env.missing}}
```

### Non-Strict Mode (Default)

```go
opts := TemplateOptions{StrictMode: false}
engine := NewEngine(ctx, opts)

result, err := engine.ResolveString("{{env.missing}}")
// result: "" (empty string)
// err: nil
```

### Placeholder Preservation

```go
opts := TemplateOptions{
    StrictMode: false,
    PlaceholderOnMissing: true,
}
engine := NewEngine(ctx, opts)

result, err := engine.ResolveString("{{env.missing}}")
// result: "{{env.missing}}" (kept as-is)
// err: nil
```

## Type Conversion

Values are automatically converted to strings in templates:

```go
ctx.InputVars["number"] = 42
ctx.InputVars["float"] = 3.14
ctx.InputVars["bool"] = true
ctx.InputVars["object"] = map[string]any{"key": "value"}

engine.ResolveString("{{input.number}}")  // "42"
engine.ResolveString("{{input.float}}")   // "3.14"
engine.ResolveString("{{input.bool}}")    // "true"
engine.ResolveString("{{input.object}}")  // {"key":"value"}
```

## Examples

### Simple Substitution

```go
ctx := NewVariableContext()
ctx.WorkflowVars["name"] = "World"

engine := NewEngineWithDefaults(ctx)
result, _ := engine.ResolveString("Hello {{env.name}}!")
// result: "Hello World!"
```

### Nested Path

```go
ctx := NewVariableContext()
ctx.InputVars["user"] = map[string]any{
    "profile": map[string]any{
        "email": "user@example.com",
    },
}

engine := NewEngineWithDefaults(ctx)
result, _ := engine.ResolveString("Email: {{input.user.profile.email}}")
// result: "Email: user@example.com"
```

### Array Access

```go
ctx := NewVariableContext()
ctx.InputVars["items"] = []any{
    map[string]any{"name": "Item1"},
    map[string]any{"name": "Item2"},
}

engine := NewEngineWithDefaults(ctx)
result, _ := engine.ResolveString("First: {{input.items[0].name}}")
// result: "First: Item1"
```

### Variable Precedence

```go
ctx := NewVariableContext()
ctx.WorkflowVars["apiKey"] = "workflow-key"
ctx.ExecutionVars["apiKey"] = "execution-key"

engine := NewEngineWithDefaults(ctx)
result, _ := engine.ResolveString("Key: {{env.apiKey}}")
// result: "Key: execution-key" (execution overrides workflow)
```

### Complex Data Structure

```go
ctx := NewVariableContext()
ctx.WorkflowVars["apiUrl"] = "https://api.example.com"
ctx.InputVars["userId"] = "123"

config := map[string]any{
    "url": "{{env.apiUrl}}/users/{{input.userId}}",
    "headers": map[string]any{
        "Content-Type": "application/json",
    },
    "params": []any{
        "{{input.userId}}",
        "active",
    },
}

engine := NewEngineWithDefaults(ctx)
resolved, _ := engine.Resolve(config)

// resolved:
// {
//   "url": "https://api.example.com/users/123",
//   "headers": {"Content-Type": "application/json"},
//   "params": ["123", "active"]
// }
```

## Performance

### Complexity

- **String Resolution**: O(n) where n = number of templates
- **Path Traversal**: O(d) where d = depth of nested path
- **Array Indexing**: O(1) for single index, O(k) for k indices

### Memory

- Minimal overhead for template parsing
- No caching (templates resolved fresh each time)
- Resolved values replace original structures

## Testing

### Run Tests

```bash
go test ./internal/application/template/
```

### Test Coverage

The test suite (`engine_test.go`) covers:

- ✅ Simple string substitution
- ✅ Nested path resolution
- ✅ Array access and indexing
- ✅ Variable precedence
- ✅ Strict mode behavior
- ✅ Placeholder preservation
- ✅ Map resolution
- ✅ Slice resolution
- ✅ Type conversion
- ✅ Template detection
- ✅ Variable extraction
- ✅ Template validation
- ✅ Complex real-world scenarios

## Thread Safety

The template engine is **not thread-safe** by design. Each execution should create its own engine instance with its own variable context.

```go
// Good: New engine per execution
for _, execution := range executions {
    ctx := createContextForExecution(execution)
    engine := NewEngineWithDefaults(ctx)
    // ... use engine
}

// Bad: Shared engine across goroutines
// engine := NewEngineWithDefaults(ctx)
// for _, execution := range executions {
//     go func(e Execution) {
//         engine.Resolve(e.config) // NOT SAFE
//     }(execution)
// }
```

## Limitations

1. **No Expression Evaluation**: Templates only do variable substitution, not arithmetic or logic
2. **No Default Values**: Cannot specify fallback values for missing variables
3. **No Filters**: No support for value transformation (e.g., `lowercase`, `trim`)
4. **No Conditionals**: Cannot use if/else logic in templates
5. **String-Only Output**: All values converted to strings in string templates

These may be addressed in future versions.

## Design Decisions

### Why Separate Resolver?

The `Resolver` is separate from `Engine` to:
- Allow reuse of resolution logic
- Make testing easier
- Support future alternative resolution strategies

### Why No Caching?

Templates are not cached because:
- Variable values change between executions
- Caching complexity outweighs benefits
- Memory footprint would grow unbounded
- Resolution is already fast enough

### Why Strict Mode Optional?

Different workflows have different requirements:
- **Production APIs**: Use strict mode to fail fast
- **Data Pipelines**: Use non-strict to handle missing data gracefully
- **Development**: Use non-strict for flexibility

## Integration

### With Executors

Use `TemplateExecutorWrapper` to automatically resolve templates:

```go
import (
    "github.com/smilemakc/mbflow/go/pkg/executor"
    "github.com/smilemakc/mbflow/go/internal/application/template"
)

// Create base executor
httpExec := builtin.NewHTTPExecutor()

// Create template engine
ctx := template.NewVariableContext()
// ... populate ctx
engine := template.NewEngineWithDefaults(ctx)

// Wrap executor
wrappedExec := executor.NewTemplateExecutorWrapper(httpExec, engine)

// Use wrapped executor (templates resolved automatically)
output, err := wrappedExec.Execute(ctx, config, input)
```

### With Workflow Engine

The workflow execution engine should:

1. Create `VariableContext` from workflow and execution
2. Create `Engine` with appropriate options
3. Wrap each executor with `TemplateExecutorWrapper`
4. Execute nodes (templates resolved automatically)

## Troubleshooting

### Templates Not Resolving

**Symptom**: Templates appear as `{{env.var}}` in output

**Solution**: Ensure you're using the template engine:
```go
// Missing this step
wrappedExec := executor.NewTemplateExecutorWrapper(exec, engine)
```

### Variable Not Found

**Symptom**: Error "variable not found" in strict mode

**Solution**:
1. Check variable name spelling
2. Verify variable is in correct map (WorkflowVars vs ExecutionVars)
3. Use non-strict mode if variable is optional

### Array Index Out of Bounds

**Symptom**: Error "array index out of bounds"

**Solution**:
1. Verify array has enough elements
2. Add conditional logic to check array length
3. Use non-strict mode to handle gracefully

### Path Resolution Fails

**Symptom**: Empty result or "field not found" error

**Solution**:
1. Print the input data structure
2. Verify the path is correct
3. Check for typos in field names (case-sensitive)

## Contributing

When contributing to this package:

1. Add tests for new features
2. Maintain backward compatibility
3. Update this README
4. Follow Go coding standards
5. Add benchmarks for performance-critical changes

## Future Work

Potential enhancements:

- [ ] Expression language (arithmetic, comparisons)
- [ ] Filter functions
- [ ] Default values syntax
- [ ] Conditional templates
- [ ] Template caching
- [ ] Custom type converters
- [ ] Template inheritance
