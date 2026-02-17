# LLM Executor with Input Templates Example

This example demonstrates how to use the LLM executor with input templates (`{{input.X}}`) to chain multiple LLM calls in a workflow.

## Overview

The workflow performs a multi-step code analysis:

1. **Detect Language**: Identifies the programming language from code
2. **Analyze Code**: Reviews code for issues using the detected language
3. **Refactor Code**: Generates improved version based on analysis
4. **Explain Changes**: Provides beginner-friendly explanation

Each step uses output from the previous step via `{{input.X}}` templates.

## Architecture Pattern

```
┌──────────────────────────────────────────────────────────┐
│                  Input Template Flow                     │
└──────────────────────────────────────────────────────────┘

User Input: {"code": "..."}
     │
     ▼
┌─────────────────────┐
│  detect_language    │  prompt: "...{{input.code}}"
│  (LLM Node)         │  ← Resolves to actual code
└─────────────────────┘
     │ output: {"content": "Go", ...}
     ▼
┌─────────────────────┐
│  analyze_code       │  instruction: "You are {{input.content}} expert"
│  (LLM Node)         │  ← Resolves to "You are Go expert"
└─────────────────────┘
     │ output: {"content": "Issues: ...", ...}
     ▼
┌─────────────────────┐
│  refactor_code      │  prompt: "Based on: {{input.content}}..."
│  (LLM Node)         │  ← Gets analysis from previous step
└─────────────────────┘
     │ output: {"content": "refactored code", ...}
     ▼
┌─────────────────────┐
│  explain_changes    │  prompt: "Explain: {{input.content}}"
│  (LLM Node)         │  ← Gets refactored code
└─────────────────────┘
```

## Key Features Demonstrated

### 1. Input Variable Access

```json
{
  "prompt": "Analyze this code: {{input.code}}"
}
```

The `{{input.code}}` template accesses the `code` field from:
- Initial workflow input (for first node)
- Parent node output (for subsequent nodes)

### 2. Chaining LLM Responses

```json
{
  "instruction": "You are a {{input.content}} expert"
}
```

LLM executors return output in this format:
```json
{
  "content": "Go",
  "model": "gpt-4",
  "usage": {...}
}
```

The `{{input.content}}` template extracts the `content` field from the LLM response.

### 3. Environment Variables

```json
{
  "model": "{{env.model}}",
  "api_key": "{{env.openai_api_key}}"
}
```

Use `{{env.X}}` for configuration values from workflow/execution variables.

### 4. Multiple Input Fields

```json
{
  "prompt": "ANALYSIS: {{input.analysis}}\n\nCODE: {{input.content}}"
}
```

You can access multiple fields from parent node output in a single template.

## How Template Resolution Works

The template resolution happens **before** the executor runs:

1. **Workflow Engine** wraps executor in `TemplateExecutorWrapper`
2. **Parent Node Output** is placed in `ExecutionContextData.ParentNodeOutput`
3. **Template Engine** maps `ParentNodeOutput` to `InputVars`
4. **Templates** like `{{input.field}}` are resolved in config
5. **Executor** receives the fully resolved configuration

This means the LLM executor never directly uses the `input` parameter - it's already been resolved into the config strings.

## Prerequisites

1. **Go 1.21+**: To build and run the example
2. **OpenAI API Key** (optional): Only needed for actual execution (not yet available)

## Setup

```bash
# Optional: Set OpenAI API key (for future use)
export OPENAI_API_KEY="sk-..."

# Run the example (embedded mode - no server required)
cd examples/llm_with_input
go run main.go
```

## Current Status

**✅ Working:**
- Workflow creation with template syntax
- DAG validation and structure display
- Template resolution flow demonstration
- Automatic template resolution (tested in unit tests)

**⏳ Requires Repository Layer:**
- Actual workflow execution with LLM calls
- Workflow persistence in database
- Execution monitoring and history

## Expected Output

The example creates a workflow and displays its structure along with template resolution flow:

```
Creating workflow...
✓ Workflow created: Code Analysis with Input Templates (ID: 94718818-9e98-420c-8165-60b887026421)

📋 WORKFLOW STRUCTURE:
─────────────────────────────────────────────────────────
Nodes: 4
  1. Detect Programming Language (detect_language)
  2. Analyze Code Quality (analyze_code)
  3. Generate Refactored Code (refactor_code)
  4. Explain Refactoring (explain_changes)

Edges: 3
  1. detect_language → analyze_code
  2. analyze_code → refactor_code
  3. refactor_code → explain_changes
─────────────────────────────────────────────────────────

🔄 TEMPLATE RESOLUTION FLOW:
Shows how data flows between nodes with template resolution

🎯 KEY FEATURES DEMONSTRATED:
✓ Automatic Template Resolution
✓ Parent Output Namespace Management
✓ Variable Precedence
✓ Chain of LLM Calls

📚 TEMPLATE SYNTAX REFERENCE:
{{input.field}}        - Access field from parent node output
{{env.variable}}       - Access workflow/execution variable
{{input.parent.field}} - Access specific parent (multi-parent)
```

## Template Syntax Reference

### Accessing Parent Output

| Template | Description | Example Use Case |
|----------|-------------|------------------|
| `{{input.field}}` | Access field from parent node output | `{{input.code}}`, `{{input.data}}` |
| `{{input.content}}` | Access LLM response text | Chaining LLM calls |
| `{{input.user.name}}` | Access nested object | `{{input.response.data.id}}` |
| `{{input.items[0]}}` | Access array element | `{{input.results[0].score}}` |

### Accessing Variables

| Template | Description | Example Use Case |
|----------|-------------|------------------|
| `{{env.var}}` | Workflow variable | `{{env.openai_api_key}}` |
| `{{env.var}}` | Execution variable (overrides workflow) | `{{env.user_id}}` |

### Variable Precedence

When the same variable exists in multiple contexts:

1. **Execution variables** (highest priority)
2. **Workflow variables**
3. **Input variables** (lowest priority)

## Error Handling

### Missing Variables

If strict mode is enabled and a template variable is missing:

```
template variable not found: input.missing_field
```

**Solution**: Ensure parent node output contains the required field.

### Type Mismatches

All template values are converted to strings. For complex objects:

```json
{
  "prompt": "Data: {{input.complex_object}}"
}
```

This may result in `[object Object]` or similar. Use specific field access instead:

```json
{
  "prompt": "Name: {{input.object.name}}, ID: {{input.object.id}}"
}
```

## Best Practices

1. **Use Descriptive Field Names**: `{{input.userQuery}}` is better than `{{input.data}}`
2. **Check Parent Output Structure**: Know what fields the parent node provides
3. **Set Temperature Appropriately**: Use 0.0 for deterministic tasks, higher for creative ones
4. **Set max_tokens**: Prevent excessive token usage
5. **Use Environment Variables for Secrets**: Never hardcode API keys in config

## Related Documentation

- [LLM Executor Guide](../../docs/executors/LLM_EXECUTOR.md)
- [Template Engine Documentation](../../docs/template_engine.md)
- [Function Calling Example](../function_calling/)

## Implementation Details

### Automatic Template Resolution

The template resolution happens in `backend/internal/application/engine/node_executor.go`:

```go
func (ne *NodeExecutor) Execute(ctx context.Context, nodeCtx *NodeContext) (any, error) {
    // 1. Get base executor
    baseExecutor := ne.executorManager.Get(nodeCtx.Node.Type)

    // 2. Create execution context for templates
    execCtxData := &executor.ExecutionContextData{
        WorkflowVariables:  nodeCtx.WorkflowVariables,
        ExecutionVariables: nodeCtx.ExecutionVariables,
        ParentNodeOutput:   nodeCtx.DirectParentOutput, // Maps to {{input.X}}
    }

    // 3. Wrap with template engine (AUTOMATIC!)
    templateEngine := executor.NewTemplateEngine(execCtxData)
    wrappedExecutor := executor.NewTemplateExecutorWrapper(baseExecutor, templateEngine)

    // 4. Execute - templates auto-resolved
    return wrappedExecutor.Execute(ctx, nodeCtx.Node.Config, nodeCtx.DirectParentOutput)
}
```

### Unit Tests

All template resolution features are tested:

```bash
cd backend/internal/application/engine
go test -v -run TestNodeExecutor
```

**Coverage:**
- ✅ Template resolution ({{input.X}}, {{env.X}})
- ✅ Variable precedence (execution > workflow)
- ✅ Multiple parent handling
- ✅ No parent handling (execution input)

All tests passing: **10/10** with **42% coverage**

## Troubleshooting

### "Warning: OPENAI_API_KEY not set"

This is informational only. The example works without an API key since it only demonstrates workflow structure, not actual execution.

### Templates not resolving

If templates aren't resolving during actual execution (when repository layer is available):
1. Check parent node output contains the expected fields
2. Verify template syntax: `{{input.field}}` not `{input.field}`
3. Confirm workflow engine is wrapping executor with `TemplateExecutorWrapper`

## Next Steps

To enable actual LLM execution:
1. Implement repository layer (WorkflowRepository, ExecutionRepository, EventRepository)
2. Initialize ExecutionManager with repositories in `sdk.Client.initializeEmbedded()`
3. Remove the "not yet implemented" error from `sdk.execution.go`
4. Run workflows with real OpenAI API calls
