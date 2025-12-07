package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/smilemakc/mbflow/internal/application/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutorForWrapper is a simple mock executor for testing the wrapper
type mockExecutorForWrapper struct {
	executeFunc  func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error)
	validateFunc func(config map[string]interface{}) error
}

func (m *mockExecutorForWrapper) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, config, input)
	}
	return config, nil
}

func (m *mockExecutorForWrapper) Validate(config map[string]interface{}) error {
	if m.validateFunc != nil {
		return m.validateFunc(config)
	}
	return nil
}

func TestNewTemplateExecutorWrapper_WithEngine(t *testing.T) {
	mockExec := &mockExecutorForWrapper{}
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars = map[string]interface{}{"name": "test"}
	engine := template.NewEngine(varCtx, template.TemplateOptions{})

	wrapped := NewTemplateExecutorWrapper(mockExec, engine)

	require.NotNil(t, wrapped)
	wrapper, ok := wrapped.(*TemplateExecutorWrapper)
	require.True(t, ok, "should return TemplateExecutorWrapper")
	assert.Equal(t, mockExec, wrapper.executor)
	assert.Equal(t, engine, wrapper.engine)
}

func TestNewTemplateExecutorWrapper_WithoutEngine(t *testing.T) {
	mockExec := &mockExecutorForWrapper{}

	wrapped := NewTemplateExecutorWrapper(mockExec, nil)

	require.NotNil(t, wrapped)
	assert.Equal(t, mockExec, wrapped, "should return original executor when engine is nil")
}

func TestTemplateExecutorWrapper_Execute_Success(t *testing.T) {
	// Setup template engine with variables
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars = map[string]interface{}{
		"apiKey":  "secret-key-123",
		"baseURL": "https://api.example.com",
	}
	varCtx.InputVars = map[string]interface{}{
		"userId": "user-456",
	}
	engine := template.NewEngine(varCtx, template.TemplateOptions{})

	// Create mock executor that captures resolved config
	var capturedConfig map[string]interface{}
	mockExec := &mockExecutorForWrapper{
		executeFunc: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			capturedConfig = config
			return map[string]interface{}{"success": true}, nil
		},
	}

	wrapper := NewTemplateExecutorWrapper(mockExec, engine)

	// Execute with template config
	config := map[string]interface{}{
		"url": "{{env.baseURL}}/users/{{input.userId}}",
		"headers": map[string]interface{}{
			"Authorization": "Bearer {{env.apiKey}}",
		},
	}

	result, err := wrapper.Execute(context.Background(), config, nil)

	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"success": true}, result)

	// Verify templates were resolved
	require.NotNil(t, capturedConfig)
	assert.Equal(t, "https://api.example.com/users/user-456", capturedConfig["url"])
	headers, ok := capturedConfig["headers"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Bearer secret-key-123", headers["Authorization"])
}

func TestTemplateExecutorWrapper_Execute_TemplateResolutionError(t *testing.T) {
	// Setup engine with strict mode
	varCtx := template.NewVariableContext()
	engine := template.NewEngine(varCtx, template.TemplateOptions{
		StrictMode: true,
	})

	mockExec := &mockExecutorForWrapper{}
	wrapper := NewTemplateExecutorWrapper(mockExec, engine)

	// Config with undefined variable
	config := map[string]interface{}{
		"value": "{{env.undefinedVar}}",
	}

	result, err := wrapper.Execute(context.Background(), config, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "undefined")
}

func TestTemplateExecutorWrapper_Execute_ExecutorError(t *testing.T) {
	varCtx := template.NewVariableContext()
	engine := template.NewEngine(varCtx, template.TemplateOptions{})

	// Mock executor that returns error
	expectedErr := errors.New("executor failed")
	mockExec := &mockExecutorForWrapper{
		executeFunc: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			return nil, expectedErr
		},
	}

	wrapper := NewTemplateExecutorWrapper(mockExec, engine)

	config := map[string]interface{}{"key": "value"}
	result, err := wrapper.Execute(context.Background(), config, nil)

	assert.Equal(t, expectedErr, err)
	assert.Nil(t, result)
}

func TestTemplateExecutorWrapper_Validate_Success(t *testing.T) {
	varCtx := template.NewVariableContext()
	engine := template.NewEngine(varCtx, template.TemplateOptions{})

	mockExec := &mockExecutorForWrapper{
		validateFunc: func(config map[string]interface{}) error {
			if config["required"] == nil {
				return errors.New("required field missing")
			}
			return nil
		},
	}

	wrapper := NewTemplateExecutorWrapper(mockExec, engine)

	// Valid config
	err := wrapper.Validate(map[string]interface{}{"required": "value"})
	assert.NoError(t, err)

	// Invalid config
	err = wrapper.Validate(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required field missing")
}

func TestGetExecutionContext_Exists(t *testing.T) {
	execData := &ExecutionContextData{
		WorkflowVariables:  map[string]interface{}{"key": "value"},
		ExecutionVariables: map[string]interface{}{"exec": "data"},
		ParentNodeOutput:   map[string]interface{}{"output": "result"},
		StrictMode:         true,
	}

	ctx := WithExecutionContext(context.Background(), execData)

	retrieved, ok := GetExecutionContext(ctx)
	require.True(t, ok)
	require.NotNil(t, retrieved)
	assert.Equal(t, execData, retrieved)
	assert.Equal(t, "value", retrieved.WorkflowVariables["key"])
	assert.Equal(t, "data", retrieved.ExecutionVariables["exec"])
	assert.Equal(t, "result", retrieved.ParentNodeOutput["output"])
	assert.True(t, retrieved.StrictMode)
}

func TestGetExecutionContext_NotExists(t *testing.T) {
	ctx := context.Background()

	retrieved, ok := GetExecutionContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, retrieved)
}

func TestWithExecutionContext(t *testing.T) {
	execData := &ExecutionContextData{
		WorkflowVariables:  map[string]interface{}{"workflow": "var"},
		ExecutionVariables: map[string]interface{}{"execution": "var"},
		ParentNodeOutput:   map[string]interface{}{"parent": "output"},
		StrictMode:         false,
	}

	ctx := WithExecutionContext(context.Background(), execData)

	// Verify context contains the data
	value := ctx.Value(ExecutionContextKey{})
	require.NotNil(t, value)

	data, ok := value.(*ExecutionContextData)
	require.True(t, ok)
	assert.Equal(t, execData, data)
}

func TestNewTemplateEngine(t *testing.T) {
	execCtx := &ExecutionContextData{
		WorkflowVariables: map[string]interface{}{
			"apiKey":  "test-key",
			"timeout": 30,
		},
		ExecutionVariables: map[string]interface{}{
			"executionID": "exec-123",
			"attempt":     1,
		},
		ParentNodeOutput: map[string]interface{}{
			"userId": "user-456",
			"status": "success",
		},
		StrictMode: true,
	}

	engine := NewTemplateEngine(execCtx)

	require.NotNil(t, engine)

	// Test that engine can resolve templates with all variable types
	config := map[string]interface{}{
		"workflow_var":  "{{env.apiKey}}",
		"execution_var": "{{env.executionID}}",
		"input_var":     "{{input.userId}}",
		"timeout":       "{{env.timeout}}",
	}

	resolved, err := engine.ResolveConfig(config)
	require.NoError(t, err)

	assert.Equal(t, "test-key", resolved["workflow_var"])
	assert.Equal(t, "exec-123", resolved["execution_var"])
	assert.Equal(t, "user-456", resolved["input_var"])
	// Note: template resolution converts values to strings
	assert.Equal(t, "30", resolved["timeout"])
}

func TestNewTemplateEngine_WithStrictMode(t *testing.T) {
	execCtx := &ExecutionContextData{
		WorkflowVariables:  map[string]interface{}{"defined": "value"},
		ExecutionVariables: map[string]interface{}{},
		ParentNodeOutput:   map[string]interface{}{},
		StrictMode:         true,
	}

	engine := NewTemplateEngine(execCtx)

	// Try to resolve undefined variable in strict mode
	config := map[string]interface{}{
		"value": "{{env.undefinedVar}}",
	}

	_, err := engine.ResolveConfig(config)
	assert.Error(t, err, "should error on undefined variable in strict mode")
	assert.Contains(t, err.Error(), "undefined")
}

func TestNewTemplateEngine_VariablePrecedence(t *testing.T) {
	// Test that execution vars override workflow vars
	execCtx := &ExecutionContextData{
		WorkflowVariables: map[string]interface{}{
			"apiKey":  "workflow-key",
			"timeout": 30,
		},
		ExecutionVariables: map[string]interface{}{
			"apiKey": "execution-key", // Should override workflow var
		},
		ParentNodeOutput: map[string]interface{}{},
		StrictMode:       false,
	}

	engine := NewTemplateEngine(execCtx)

	config := map[string]interface{}{
		"key":     "{{env.apiKey}}",
		"timeout": "{{env.timeout}}",
	}

	resolved, err := engine.ResolveConfig(config)
	require.NoError(t, err)

	// Execution var should take precedence
	assert.Equal(t, "execution-key", resolved["key"])
	// Note: template resolution converts values to strings
	assert.Equal(t, "30", resolved["timeout"])
}

func TestTemplateExecutorWrapper_Execute_ComplexTemplates(t *testing.T) {
	// Test with nested structures and multiple template types
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars = map[string]interface{}{
		"baseURL": "https://api.example.com",
		"version": "v1",
	}
	varCtx.InputVars = map[string]interface{}{
		"user": map[string]interface{}{
			"id":   "123",
			"name": "John",
		},
		"items": []interface{}{"item1", "item2"},
	}
	engine := template.NewEngine(varCtx, template.TemplateOptions{})

	var capturedConfig map[string]interface{}
	mockExec := &mockExecutorForWrapper{
		executeFunc: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
			capturedConfig = config
			return "ok", nil
		},
	}

	wrapper := NewTemplateExecutorWrapper(mockExec, engine)

	config := map[string]interface{}{
		"url": "{{env.baseURL}}/{{env.version}}/users/{{input.user.id}}",
		"body": map[string]interface{}{
			"name": "{{input.user.name}}",
			"items": []interface{}{
				"{{input.items[0]}}",
				"{{input.items[1]}}",
			},
		},
	}

	result, err := wrapper.Execute(context.Background(), config, nil)

	require.NoError(t, err)
	assert.Equal(t, "ok", result)

	// Verify complex template resolution
	assert.Equal(t, "https://api.example.com/v1/users/123", capturedConfig["url"])

	body, ok := capturedConfig["body"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "John", body["name"])

	items, ok := body["items"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "item1", items[0])
	assert.Equal(t, "item2", items[1])
}
