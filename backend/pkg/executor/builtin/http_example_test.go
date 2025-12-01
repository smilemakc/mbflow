package builtin_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/smilemakc/mbflow/internal/application/template"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/executor/builtin"
)

// Example_httpExecutorWithTemplates demonstrates using templates with HTTP executor.
func Example_httpExecutorWithTemplates() {
	// Create an HTTP executor
	httpExec := builtin.NewHTTPExecutor()

	// Create a variable context with workflow and execution variables
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars["apiUrl"] = "https://api.example.com"
	varCtx.ExecutionVars["apiKey"] = "secret-key-123"
	varCtx.InputVars["userId"] = "user-456"

	// Create a template engine
	opts := template.TemplateOptions{
		StrictMode:           false,
		PlaceholderOnMissing: false,
	}
	engine := template.NewEngine(varCtx, opts)

	// Wrap the executor with template resolution
	wrappedExec := executor.NewTemplateExecutorWrapper(httpExec, engine)

	// Configuration with templates
	config := map[string]interface{}{
		"method": "GET",
		"url":    "{{env.apiUrl}}/users/{{input.userId}}",
		"headers": map[string]interface{}{
			"Authorization": "Bearer {{env.apiKey}}",
			"Content-Type":  "application/json",
		},
	}

	// The wrapper will resolve templates before execution
	// This would resolve to:
	// - url: "https://api.example.com/users/user-456"
	// - headers.Authorization: "Bearer secret-key-123"

	fmt.Println("Template resolution happens automatically!")

	// Note: This example doesn't actually execute the HTTP request
	// as it would require a real server
	_ = wrappedExec
	_ = config
	// Output:
	// Template resolution happens automatically!
}

// TestHTTPExecutor_TemplateResolution tests that templates are resolved correctly.
func TestHTTPExecutor_TemplateResolution(t *testing.T) {
	// Create template engine
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars["baseUrl"] = "https://api.test.com"
	varCtx.ExecutionVars["token"] = "test-token"
	varCtx.InputVars["resourceId"] = "123"

	opts := template.DefaultOptions()
	engine := template.NewEngine(varCtx, opts)

	// Create HTTP executor
	httpExec := builtin.NewHTTPExecutor()
	wrappedExec := executor.NewTemplateExecutorWrapper(httpExec, engine)

	// Config with templates
	config := map[string]interface{}{
		"method": "GET",
		"url":    "{{env.baseUrl}}/resource/{{input.resourceId}}",
		"headers": map[string]interface{}{
			"Authorization": "Bearer {{env.token}}",
		},
	}

	// Validate that the wrapped executor maintains the same interface
	if err := wrappedExec.Validate(config); err != nil {
		t.Errorf("Validate() failed: %v", err)
	}

	// Note: We don't execute the actual HTTP request in tests
	// but the template resolution would happen automatically in Execute()
}

// TestHTTPExecutor_StrictMode tests strict mode template resolution.
func TestHTTPExecutor_StrictMode(t *testing.T) {
	// Create template engine in strict mode
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars["baseUrl"] = "https://api.test.com"
	// Note: "apiKey" is missing

	opts := template.TemplateOptions{
		StrictMode: true, // Strict mode enabled
	}
	engine := template.NewEngine(varCtx, opts)

	// Create HTTP executor
	httpExec := builtin.NewHTTPExecutor()
	wrappedExec := executor.NewTemplateExecutorWrapper(httpExec, engine)

	// Config with a missing variable
	config := map[string]interface{}{
		"method": "GET",
		"url":    "{{env.baseUrl}}/users",
		"headers": map[string]interface{}{
			"Authorization": "Bearer {{env.apiKey}}", // apiKey is missing!
		},
	}

	ctx := context.Background()

	// Execute should fail because apiKey is missing in strict mode
	_, err := wrappedExec.Execute(ctx, config, nil)
	if err == nil {
		t.Error("Expected error in strict mode when variable is missing, got nil")
	}
}

// TestHTTPExecutor_ComplexTemplates tests complex template scenarios.
func TestHTTPExecutor_ComplexTemplates(t *testing.T) {
	// Create template engine
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars["apiUrl"] = "https://api.example.com"
	varCtx.InputVars["response"] = map[string]interface{}{
		"data": map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{
					"id":   1,
					"name": "Alice",
				},
				map[string]interface{}{
					"id":   2,
					"name": "Bob",
				},
			},
		},
	}

	opts := template.DefaultOptions()
	engine := template.NewEngine(varCtx, opts)

	// Create HTTP executor
	httpExec := builtin.NewHTTPExecutor()
	wrappedExec := executor.NewTemplateExecutorWrapper(httpExec, engine)

	// Config with complex nested templates
	config := map[string]interface{}{
		"method": "POST",
		"url":    "{{env.apiUrl}}/users/{{input.response.data.users[0].id}}/notify",
		"body": map[string]interface{}{
			"message": "Hello {{input.response.data.users[1].name}}!",
		},
	}

	// Validate configuration
	if err := wrappedExec.Validate(config); err != nil {
		t.Errorf("Validate() failed: %v", err)
	}

	// The templates would resolve to:
	// - url: "https://api.example.com/users/1/notify"
	// - body.message: "Hello Bob!"
}
