// Package builder provides fluent, type-safe workflow construction for MBFlow.
//
// The builder API offers a more ergonomic way to create workflows compared to
// manual struct initialization, with compile-time type safety, early validation,
// and IDE autocomplete support.
//
// # Basic Usage
//
// Create a simple HTTP workflow:
//
//	workflow := builder.NewWorkflow("Fetch User Data",
//	    builder.WithVariable("api_base", "https://api.example.com"),
//	).AddNode(
//	    builder.NewHTTPGetNode(
//	        "fetch",
//	        "Fetch User",
//	        "{{env.api_base}}/users/{{input.user_id}}",
//	    ),
//	).MustBuild()
//
// # Node Types
//
// The builder supports all built-in node types with convenience constructors:
//
// HTTP Nodes:
//   - NewHTTPGetNode(id, name, url, opts...)
//   - NewHTTPPostNode(id, name, url, body, opts...)
//   - NewHTTPPutNode(id, name, url, body, opts...)
//   - NewHTTPDeleteNode(id, name, url, opts...)
//   - NewHTTPPatchNode(id, name, url, body, opts...)
//
// LLM Nodes:
//   - NewOpenAINode(id, name, model, prompt, opts...)
//   - NewAnthropicNode(id, name, model, prompt, opts...)
//
// Transform Nodes:
//   - NewPassthroughNode(id, name, opts...)
//   - NewExpressionNode(id, name, expr, opts...)
//   - NewJQNode(id, name, filter, opts...)
//   - NewTemplateNode(id, name, template, opts...)
//
// # Connecting Nodes
//
// Use the Connect() method to create edges between nodes:
//
//	workflow := builder.NewWorkflow("Pipeline").
//	    AddNode(builder.NewHTTPGetNode("fetch", "Fetch", "https://api.example.com/data")).
//	    AddNode(builder.NewJQNode("transform", "Transform", `.[] | {id, name}`)).
//	    AddNode(builder.NewHTTPPostNode("load", "Load", "https://api.example.com/results", nil)).
//	    Connect("fetch", "transform").
//	    Connect("transform", "load").
//	    MustBuild()
//
// # Conditional Edges
//
// Use conditional edge helpers for branching logic:
//
//	workflow.Connect("check", "success", builder.WhenTrue("output.success"))
//	workflow.Connect("check", "failure", builder.WhenFalse("output.success"))
//	workflow.Connect("check", "error", builder.WhenEqual("output.status", "error"))
//
// # Positioning
//
// Position nodes using several strategies:
//
// Absolute positioning:
//
//	builder.NewHTTPGetNode("node1", "Node 1", "url",
//	    builder.WithPosition(100, 200),
//	)
//
// Grid layout:
//
//	builder.NewHTTPGetNode("node1", "Node 1", "url",
//	    builder.GridPosition(0, 0),  // Row 0, Col 0 -> (0, 0)
//	)
//
// Auto-layout:
//
//	builder.NewWorkflow("My Workflow",
//	    builder.WithAutoLayout(),  // Automatically positions nodes
//	).AddNode(...).AddNode(...).MustBuild()
//
// # Workflow Options
//
// Configure workflows with functional options:
//
//	workflow := builder.NewWorkflow("Production Workflow",
//	    builder.WithDescription("A production workflow"),
//	    builder.WithStatus(models.WorkflowStatusActive),
//	    builder.WithTags("production", "critical"),
//	    builder.WithVariable("api_key", "secret"),
//	    builder.WithMetadata("author", "John Doe"),
//	    builder.WithAutoLayout(),
//	    builder.WithStrictValidation(),
//	).AddNode(...).MustBuild()
//
// # Node Options
//
// HTTP node options:
//   - HTTPMethod(method) - GET, POST, PUT, DELETE, etc.
//   - HTTPURL(url) - Request URL
//   - HTTPBody(body) - Request body
//   - HTTPHeaders(headers) - All headers
//   - HTTPHeader(key, value) - Single header
//   - HTTPTimeout(duration) - Request timeout
//
// LLM node options:
//   - LLMProvider(provider) - openai, anthropic
//   - LLMModel(model) - Model name
//   - LLMPrompt(prompt) - Prompt template
//   - LLMAPIKey(key) - API key
//   - LLMTemperature(temp) - Temperature (0-2, validated)
//   - LLMMaxTokens(tokens) - Max tokens
//   - LLMTopP(topP) - Top-p sampling (0-1, validated)
//   - LLMSystemPrompt(prompt) - System prompt
//   - LLMJSONMode() - Enable JSON response mode
//
// Transform node options:
//   - TransformType(type) - passthrough, expression, jq, template
//   - TransformExpression(expr) - expr-lang expression
//   - TransformJQ(filter) - JQ filter
//   - TransformTemplate(tmpl) - Template string
//
// Generic node options:
//   - WithNodeDescription(desc) - Node description
//   - WithPosition(x, y) - Absolute position
//   - GridPosition(row, col) - Grid position
//   - WithNodeMetadata(key, value) - Node metadata
//   - WithConfig(config) - Raw config map (escape hatch)
//   - WithConfigValue(key, value) - Single config value
//
// # Error Handling
//
// Use Build() for error handling:
//
//	workflow, err := builder.NewWorkflow("Test").
//	    AddNode(...).
//	    Build()
//	if err != nil {
//	    return err
//	}
//
// Or use MustBuild() for tests/examples (panics on error):
//
//	workflow := builder.NewWorkflow("Test").
//	    AddNode(...).
//	    MustBuild()
//
// # Validation
//
// The builder validates at multiple levels:
//
//  1. Option-level: Type constraints (e.g., temperature 0-2)
//  2. Build-level: Structure validation (required fields, DAG)
//  3. SDK-level: Runtime validation (executor availability)
//
// Enable strict validation to check all configs upfront:
//
//	workflow := builder.NewWorkflow("Test",
//	    builder.WithStrictValidation(),
//	).AddNode(...).Build()
//
// # Migration
//
// See MIGRATION.md for detailed migration guide from raw structs to builders.
//
// # Examples
//
// See examples_test.go for complete working examples.
package builder
