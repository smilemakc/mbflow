# Template Engine Documentation

## Overview

The MBFlow template engine provides powerful variable substitution capabilities for workflow node configurations. It allows you to reference workflow variables, execution variables, and parent node outputs using a simple template syntax.

## Features

- **Simple Syntax**: `{{env.varName}}` and `{{input.fieldName}}`
- **Nested Path Support**: Access nested fields with dot notation (e.g., `{{input.user.profile.email}}`)
- **Array Access**: Access array elements with bracket notation (e.g., `{{input.items[0].name}}`)
- **Variable Precedence**: Execution variables override workflow variables
- **Strict Mode**: Optional strict validation for missing variables
- **Type Conversion**: Automatic conversion of values to strings in templates
- **Deep Resolution**: Templates are resolved in all data structures (maps, slices, nested objects)

## Template Syntax

### Basic Syntax

Templates use double curly braces `{{ }}` with a variable type and path:

```
{{type.path.to.value}}
```

### Variable Types

#### 1. Environment Variables (`env`)

Access workflow-level and execution-level variables:

```
{{env.apiUrl}}
{{env.apiKey}}
{{env.config.timeout}}
```

**Variable Precedence:**
- Execution variables (highest priority)
- Workflow variables (lower priority)

#### 2. Input Variables (`input`)

Access output from parent node:

```
{{input.userId}}
{{input.response.data}}
{{input.user.email}}
```

### Path Expressions

#### Dot Notation (Nested Fields)

Access nested object fields using dots:

```
{{input.user.profile.email}}
{{env.database.connection.host}}
```

#### Array Indexing

Access array elements using bracket notation:

```
{{input.items[0]}}
{{input.users[0].name}}
{{input.data.results[2].id}}
```

#### Chained Indexing

Support for multi-dimensional arrays:

```
{{input.matrix[0][1]}}
{{input.data[0][2][3]}}
```

## Configuration

### Workflow Variables

Define variables at the workflow level:

```json
{
  "id": "wf-123",
  "name": "API Integration Workflow",
  "variables": {
    "apiUrl": "https://api.example.com",
    "timeout": 30,
    "retryCount": 3
  }
}
```

### Execution Variables

Override workflow variables at execution time:

```json
{
  "workflowId": "wf-123",
  "variables": {
    "apiUrl": "https://api.staging.example.com",
    "apiKey": "exec-specific-key"
  },
  "strictMode": true
}
```

### Strict Mode

Control error handling for missing variables:

```json
{
  "strictMode": true  // Execution fails if any variable is missing
}
```

```json
{
  "strictMode": false // Missing variables are replaced with empty string (default)
}
```

## Usage Examples

### HTTP Node with Templates

```json
{
  "type": "http",
  "config": {
    "method": "GET",
    "url": "{{env.apiUrl}}/users/{{input.userId}}",
    "headers": {
      "Authorization": "Bearer {{env.apiKey}}",
      "Content-Type": "application/json"
    }
  }
}
```

**Resolution (assuming variables):**
- `env.apiUrl = "https://api.example.com"`
- `env.apiKey = "secret-123"`
- `input.userId = "user-456"`

**Result:**
```json
{
  "method": "GET",
  "url": "https://api.example.com/users/user-456",
  "headers": {
    "Authorization": "Bearer secret-123",
    "Content-Type": "application/json"
  }
}
```

### Complex Nested Example

Given this input from a parent node:

```json
{
  "response": {
    "status": 200,
    "data": {
      "users": [
        {"id": 1, "name": "Alice", "email": "alice@example.com"},
        {"id": 2, "name": "Bob", "email": "bob@example.com"}
      ]
    }
  }
}
```

You can use templates like:

```json
{
  "type": "http",
  "config": {
    "method": "POST",
    "url": "{{env.apiUrl}}/notify/{{input.response.data.users[0].id}}",
    "body": {
      "recipient": "{{input.response.data.users[1].email}}",
      "message": "Hello {{input.response.data.users[0].name}}!"
    }
  }
}
```

**Resolves to:**
```json
{
  "method": "POST",
  "url": "https://api.example.com/notify/1",
  "body": {
    "recipient": "bob@example.com",
    "message": "Hello Alice!"
  }
}
```

### Transform Node with Templates

```json
{
  "type": "transform",
  "config": {
    "type": "template",
    "template": "User {{input.user.name}} ({{input.user.email}}) signed up at {{env.baseUrl}}"
  }
}
```

## Implementation Details

### Architecture

The template system consists of three main components:

1. **Template Engine** (`internal/application/template/engine.go`)
   - Main entry point for template resolution
   - Handles string and data structure resolution

2. **Variable Resolver** (`internal/application/template/resolver.go`)
   - Resolves individual variable references
   - Handles nested paths and array indexing

3. **Executor Wrapper** (`pkg/executor/template_wrapper.go`)
   - Transparently wraps executors
   - Automatically resolves templates before execution

### Variable Resolution Flow

```
1. Node execution starts
2. TemplateExecutorWrapper intercepts
3. Template engine resolves all templates in config
4. Resolved config passed to actual executor
5. Executor executes with resolved values
```

### Type Conversion

The template engine automatically converts values to strings:

| Type    | Conversion                    |
|---------|-------------------------------|
| string  | As-is                         |
| number  | "42", "3.14"                  |
| boolean | "true", "false"               |
| object  | JSON string                   |
| array   | JSON string                   |
| null    | "" (empty string)             |

## Error Handling

### Strict Mode

**Enabled (`strictMode: true`):**
- Missing variables cause execution to fail
- Error message includes variable name and path
- Execution stops immediately

**Disabled (`strictMode: false`, default):**
- Missing variables are replaced with empty string
- Execution continues
- No error is raised

### Common Errors

```go
// Variable not found
template error: failed to resolve '{{env.missingVar}}': variable not found

// Invalid template syntax
template error: invalid variable reference 'env' (expected format: {{type.path}})

// Invalid path
template error: failed to resolve '{{input.user.invalid}}': field 'invalid' not found

// Array out of bounds
template error: failed to resolve '{{input.items[10]}}': array index out of bounds: index 10, length 3
```

## Best Practices

### 1. Use Meaningful Variable Names

```json
// Good
{
  "variables": {
    "apiUrl": "https://api.example.com",
    "authToken": "secret-key"
  }
}

// Avoid
{
  "variables": {
    "url1": "https://api.example.com",
    "t": "secret-key"
  }
}
```

### 2. Enable Strict Mode for Critical Workflows

```json
{
  "strictMode": true  // Fail fast on missing variables
}
```

### 3. Validate Templates Before Execution

```go
import "github.com/smilemakc/mbflow/internal/application/template"

err := template.ValidateTemplate("{{env.apiUrl}}/users")
if err != nil {
    // Handle validation error
}
```

### 4. Use Workflow Variables for Constants

```json
{
  "variables": {
    "apiUrl": "https://api.example.com",
    "maxRetries": 3,
    "timeout": 30
  }
}
```

### 5. Use Execution Variables for Secrets

```json
{
  "variables": {
    "apiKey": "execution-specific-key",
    "dbPassword": "secure-password"
  }
}
```

## API Reference

### Template Engine API

```go
import "github.com/smilemakc/mbflow/internal/application/template"

// Create variable context
ctx := template.NewVariableContext()
ctx.WorkflowVars["apiUrl"] = "https://api.example.com"
ctx.ExecutionVars["apiKey"] = "secret-123"
ctx.InputVars["userId"] = "user-456"

// Create engine
opts := template.TemplateOptions{
    StrictMode: false,
    PlaceholderOnMissing: false,
}
engine := template.NewEngine(ctx, opts)

// Resolve string template
result, err := engine.ResolveString("Hello {{input.userId}}")
// result: "Hello user-456"

// Resolve data structure
config := map[string]interface{}{
    "url": "{{env.apiUrl}}/users/{{input.userId}}",
}
resolved, err := engine.Resolve(config)
```

### Executor Wrapper API

```go
import (
    "github.com/smilemakc/mbflow/pkg/executor"
    "github.com/smilemakc/mbflow/pkg/executor/builtin"
)

// Create executor
httpExec := builtin.NewHTTPExecutor()

// Wrap with template engine
wrappedExec := executor.NewTemplateExecutorWrapper(httpExec, engine)

// Execute (templates are resolved automatically)
output, err := wrappedExec.Execute(ctx, config, input)
```

## Testing

### Unit Tests

Run template engine tests:

```bash
go test ./internal/application/template/
```

### Test Coverage

- Simple string substitution
- Nested path resolution
- Array indexing
- Variable precedence
- Strict mode behavior
- Complex scenarios with real workflow data

## Performance Considerations

1. **Template Resolution Complexity**: O(n) where n is the number of template placeholders
2. **Path Traversal**: O(d) where d is the depth of nested path
3. **Caching**: Templates are resolved on each execution (no caching)
4. **Memory**: Minimal overhead, resolved values replace placeholders

## Migration Guide

### From Manual String Replacement

**Before:**
```go
url := fmt.Sprintf("%s/users/%s", apiUrl, userId)
```

**After:**
```json
{
  "url": "{{env.apiUrl}}/users/{{input.userId}}"
}
```

### From Environment Variables

**Before:**
```go
apiKey := os.Getenv("API_KEY")
```

**After:**
```json
{
  "variables": {
    "apiKey": "value-from-env"
  }
}
```

## Troubleshooting

### Templates Not Resolving

**Problem:** Templates appear as `{{env.var}}` in output

**Solution:** Ensure TemplateExecutorWrapper is used:
```go
wrappedExec := executor.NewTemplateExecutorWrapper(exec, engine)
```

### Variable Not Found Errors

**Problem:** "variable not found" error in strict mode

**Solution:**
1. Check variable name spelling
2. Verify variable is set in workflow or execution
3. Use non-strict mode for optional variables

### Array Index Out of Bounds

**Problem:** Array index exceeds array length

**Solution:**
1. Verify array length before accessing
2. Use conditional logic to check array size
3. Handle missing data gracefully

## Future Enhancements

Planned features:

- [ ] Expression language support (arithmetic, comparisons)
- [ ] Filter functions (e.g., `{{input.email | lowercase}}`)
- [ ] Conditional templates (e.g., `{{if env.debug}}debug mode{{end}}`)
- [ ] Default values (e.g., `{{env.apiUrl | default "https://api.example.com"}}`)
- [ ] Template caching for performance
- [ ] Template pre-compilation
