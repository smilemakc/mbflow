# Transform Executor Documentation

The Transform Executor provides powerful data transformation capabilities for MBFlow workflows. It supports multiple transformation types including expressions, jq filters, and templates.

## Overview

The Transform Executor allows you to transform data between nodes in a workflow using:
- **Passthrough**: Pass data unchanged
- **Template**: String template substitution with `{{env.var}}` and `{{input.field}}`
- **Expression**: Powerful expression language using [expr-lang](https://github.com/expr-lang/expr)
- **JQ**: JSON query and transformation using [gojq](https://github.com/itchyny/gojq)

## Configuration

### Basic Structure

```json
{
  "type": "transform",
  "config": {
    "type": "passthrough|template|expression|jq",
    // type-specific configuration...
  }
}
```

## Transformation Types

### 1. Passthrough

Simply passes the input to the output without modification.

**Configuration:**
```json
{
  "type": "passthrough"
}
```

**Example:**
```go
input: {"name": "John", "age": 30}
output: {"name": "John", "age": 30}
```

### 2. Template

String template substitution using `{{env.var}}` and `{{input.field}}` syntax.

**Configuration:**
```json
{
  "type": "template",
  "template": "Hello {{env.name}}!"
}
```

**Template Variables:**
- `{{env.varName}}` - Workflow or execution variables
- `{{input.fieldName}}` - Parent node output fields
- `{{input.nested.path}}` - Nested field access

**Example:**
```json
// Configuration
{
  "type": "template",
  "template": "Welcome {{input.firstName}} {{input.lastName}}!"
}

// Input
{"firstName": "John", "lastName": "Doe"}

// Output
"Welcome John Doe!"
```

### 3. Expression

Powerful expression language for calculations, conditionals, and data manipulation.

**Configuration:**
```json
{
  "type": "expression",
  "expression": "input.price * input.quantity"
}
```

**Expression Features:**
- Arithmetic: `+`, `-`, `*`, `/`, `%`
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
- Logical: `&&`, `||`, `!`
- Ternary: `condition ? valueIfTrue : valueIfFalse`
- String operations: `+` for concatenation
- Field access: `input.field`, `input.nested.field`

**Examples:**

#### Simple Calculation
```json
{
  "type": "expression",
  "expression": "input.price * 1.2"
}
// Input: {"price": 100}
// Output: 120
```

#### Complex Calculation
```json
{
  "type": "expression",
  "expression": "(input.price * input.quantity) * (1 + input.taxRate)"
}
// Input: {"price": 50, "quantity": 3, "taxRate": 0.2}
// Output: 180 (50 * 3 * 1.2)
```

#### String Manipulation
```json
{
  "type": "expression",
  "expression": "input.firstName + \" \" + input.lastName"
}
// Input: {"firstName": "John", "lastName": "Doe"}
// Output: "John Doe"
```

#### Conditional Logic
```json
{
  "type": "expression",
  "expression": "input.age >= 18 ? \"adult\" : \"minor\""
}
// Input: {"age": 25}
// Output: "adult"
```

#### With Template Variables
```json
{
  "type": "expression",
  "expression": "input.price * {{env.discountRate}}"
}
// Workflow vars: {"discountRate": 0.9}
// Input: {"price": 100}
// Output: 90
```

### 4. JQ (JSON Query)

Full jq query language support for complex JSON transformations.

**Configuration:**
```json
{
  "type": "jq",
  "filter": ".name"
}
```

**JQ Features:**
- Field selection: `.field`, `.nested.field`
- Array operations: `.array[]`, `.array[0]`, `.array[-1]`
- Filtering: `select()`, `map()`, `filter()`
- Object construction: `{key: .value}`
- Array construction: `[.items[] | .name]`
- Pipes: `expr1 | expr2 | expr3`

**Examples:**

#### Simple Field Access
```json
{
  "type": "jq",
  "filter": ".name"
}
// Input: {"name": "John", "email": "john@example.com"}
// Output: "John"
```

#### Nested Access
```json
{
  "type": "jq",
  "filter": ".user.profile.email"
}
// Input: {"user": {"profile": {"email": "user@example.com"}}}
// Output: "user@example.com"
```

#### Array Filtering
```json
{
  "type": "jq",
  "filter": ".items[] | select(.price > 50)"
}
// Input: {"items": [{"name": "Item1", "price": 30}, {"name": "Item2", "price": 75}]}
// Output: {"name": "Item2", "price": 75}
```

#### Array Mapping
```json
{
  "type": "jq",
  "filter": "[.items[] | .name]"
}
// Input: {"items": [{"name": "A"}, {"name": "B"}, {"name": "C"}]}
// Output: ["A", "B", "C"]
```

#### Object Construction
```json
{
  "type": "jq",
  "filter": "{fullName: (.firstName + \" \" + .lastName), contact: .email}"
}
// Input: {"firstName": "John", "lastName": "Doe", "email": "john@example.com"}
// Output: {"fullName": "John Doe", "contact": "john@example.com"}
```

#### With Template Variables
```json
{
  "type": "jq",
  "filter": ".items[] | select(.category == \"{{env.targetCategory}}\")"
}
// Workflow vars: {"targetCategory": "electronics"}
// Input: {"items": [{"name": "Phone", "category": "electronics"}]}
// Output: {"name": "Phone", "category": "electronics"}
```

## Template Engine Integration

All transformation types support template variable substitution through the `TemplateExecutorWrapper`:

```go
// Create executor
exec := builtin.NewTransformExecutor()

// Create template engine with variables
varCtx := template.NewVariableContext()
varCtx.WorkflowVars = map[string]interface{}{
    "apiUrl": "https://api.example.com",
    "version": "v1",
}
varCtx.ExecutionVars = map[string]interface{}{
    "userId": "12345",
}
varCtx.InputVars = map[string]interface{}{
    "requestId": "req-001",
}

engine := template.NewEngine(varCtx, template.TemplateOptions{
    StrictMode: false, // Set to true to fail on missing variables
})

// Wrap executor
wrappedExec := executor.NewTemplateExecutorWrapper(exec, engine)

// Use wrapped executor
config := map[string]interface{}{
    "type": "expression",
    "expression": "\"{{env.apiUrl}}/{{env.version}}/users/{{env.userId}}\"",
}

result, err := wrappedExec.Execute(ctx, config, input)
// Result: "https://api.example.com/v1/users/12345"
```

## Variable Precedence

Variables are resolved in this order:
1. **Execution Variables** (highest priority) - Runtime variables that override workflow vars
2. **Workflow Variables** - Variables defined in workflow
3. **Input Variables** - Output from parent node

Example:
```go
varCtx.WorkflowVars = map[string]interface{}{
    "discount": "10",
}
varCtx.ExecutionVars = map[string]interface{}{
    "discount": "15", // This overrides workflow variable
}

// Template: "{{env.discount}}" resolves to "15"
```

## Strict Mode

When strict mode is enabled, missing template variables cause execution to fail:

```go
engine := template.NewEngine(varCtx, template.TemplateOptions{
    StrictMode: true, // Fail on missing variables
})

// This will fail if 'missingVar' doesn't exist
config := map[string]interface{}{
    "type": "template",
    "template": "{{env.missingVar}}",
}
```

## Complete Workflow Example

```go
// Node 1: Calculate total price
config1 := map[string]interface{}{
    "type": "expression",
    "expression": "input.price * input.quantity",
}
// Input: {"price": 100, "quantity": 5}
// Output: 500

// Node 2: Apply discount using template vars
config2 := map[string]interface{}{
    "type": "jq",
    "filter": "{total: ., discount: {{env.discountPct}}, finalPrice: (. * (100 - {{env.discountPct}}) / 100)}",
}
// Workflow vars: {"discountPct": 15}
// Input: 500
// Output: {"total": 500, "discount": 15, "finalPrice": 425}

// Node 3: Format final message
config3 := map[string]interface{}{
    "type": "template",
    "template": "Order total: ${{input.finalPrice}} ({{input.discount}}% discount applied)",
}
// Input: {"total": 500, "discount": 15, "finalPrice": 425}
// Output: "Order total: $425 (15% discount applied)"
```

## Error Handling

### Expression Errors
```go
// Invalid expression syntax
{
    "type": "expression",
    "expression": "invalid syntax {{"
}
// Error: "failed to compile expression: ..."

// Runtime error
{
    "type": "expression",
    "expression": "input.nonexistent.field"
}
// Error: "failed to execute expression: ..."
```

### JQ Errors
```go
// Invalid jq syntax
{
    "type": "jq",
    "filter": ".invalid["
}
// Error: "failed to parse jq filter: ..."

// No output
{
    "type": "jq",
    "filter": ".items[] | select(.price > 1000)"
}
// If no items match, error: "jq filter produced no output"
```

### Template Errors (Strict Mode)
```go
{
    "type": "template",
    "template": "{{env.missingVariable}}"
}
// With StrictMode=true: Error: "variable not found: env.missingVariable"
// With StrictMode=false: Output: ""
```

## Validation

The Transform Executor validates configuration before execution:

```go
exec := builtin.NewTransformExecutor()

config := map[string]interface{}{
    "type": "expression",
    // Missing "expression" field
}

err := exec.Validate(config)
// Error: "expression is required for expression transformation"
```

Required fields by type:
- **passthrough**: None
- **template**: `template` (string)
- **expression**: `expression` (string)
- **jq**: `filter` (string)

## Best Practices

1. **Use the right tool:**
   - Simple field selection → jq
   - Calculations → expression
   - String formatting → template
   - No transformation → passthrough

2. **Template variables:**
   - Use `{{env.var}}` for workflow/execution config
   - Use `{{input.field}}` for dynamic data from previous nodes

3. **Complex transformations:**
   - Chain multiple transform nodes
   - Keep individual transformations simple and focused

4. **Error handling:**
   - Enable strict mode in production for early error detection
   - Validate expressions/filters before deployment

5. **Performance:**
   - Expressions are compiled once and cached
   - JQ queries are compiled for efficiency
   - Avoid complex jq filters on large datasets

## Testing

Comprehensive test suite included in `transform_test.go`:

```bash
# Run all transform tests
go test -v ./pkg/executor/builtin -run TestTransform

# Run specific test
go test -v ./pkg/executor/builtin -run TestTransformExecutor_Expression_Simple
```

## References

- [expr-lang documentation](https://expr-lang.org/)
- [gojq documentation](https://github.com/itchyny/gojq)
- [jq manual](https://stedolan.github.io/jq/manual/)
- [MBFlow Template Engine](TEMPLATE_ENGINE.md)
