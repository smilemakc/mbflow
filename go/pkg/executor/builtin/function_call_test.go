package builtin

import (
	"context"
	"fmt"
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctionCallExecutor_Execute_DirectInput(t *testing.T) {
	executor := NewFunctionCallExecutor()

	// Register a test function
	executor.RegisterFunction("test_add", func(args map[string]any) (any, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return a + b, nil
	})

	config := map[string]any{
		"function_name": "test_add",
		"arguments":     `{"a": 5, "b": 3}`,
	}

	result, err := executor.Execute(context.Background(), config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	output, ok := result.(*models.FunctionCallOutput)
	require.True(t, ok)

	assert.True(t, output.Success)
	assert.Equal(t, "test_add", output.FunctionName)
	assert.Equal(t, 8.0, output.Result)
	assert.Empty(t, output.Error)
}

func TestFunctionCallExecutor_Execute_FromLLMToolCalls(t *testing.T) {
	executor := NewFunctionCallExecutor()

	// Register a test function
	executor.RegisterFunction("get_weather", func(args map[string]any) (any, error) {
		location, _ := args["location"].(string)
		return map[string]any{
			"location":    location,
			"temperature": 22,
			"condition":   "sunny",
		}, nil
	})

	// Simulate input from LLM executor with tool_calls
	input := map[string]any{
		"tool_calls": []any{
			map[string]any{
				"id":   "call-123",
				"type": "function",
				"function": map[string]any{
					"name":      "get_weather",
					"arguments": `{"location": "London"}`,
				},
			},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]any{}, input)
	require.NoError(t, err)
	require.NotNil(t, result)

	output, ok := result.(*models.FunctionCallOutput)
	require.True(t, ok)

	assert.True(t, output.Success)
	assert.Equal(t, "get_weather", output.FunctionName)
	assert.Equal(t, "call-123", output.ToolCallID)

	weatherResult, ok := output.Result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "London", weatherResult["location"])
	assert.Equal(t, 22, weatherResult["temperature"])
}

func TestFunctionCallExecutor_Execute_FunctionNotFound(t *testing.T) {
	executor := NewFunctionCallExecutor()

	config := map[string]any{
		"function_name": "nonexistent_function",
		"arguments":     `{}`,
	}

	result, err := executor.Execute(context.Background(), config, nil)
	require.NoError(t, err) // Should not error, but return error output
	require.NotNil(t, result)

	output, ok := result.(*models.FunctionCallOutput)
	require.True(t, ok)

	assert.False(t, output.Success)
	assert.Contains(t, output.Error, "function not found")
}

func TestFunctionCallExecutor_Execute_FunctionError(t *testing.T) {
	executor := NewFunctionCallExecutor()

	// Register a function that returns an error
	executor.RegisterFunction("error_function", func(args map[string]any) (any, error) {
		return nil, fmt.Errorf("intentional error")
	})

	config := map[string]any{
		"function_name": "error_function",
		"arguments":     `{}`,
	}

	result, err := executor.Execute(context.Background(), config, nil)
	require.NoError(t, err) // Should not error, but return error output
	require.NotNil(t, result)

	output, ok := result.(*models.FunctionCallOutput)
	require.True(t, ok)

	assert.False(t, output.Success)
	assert.Contains(t, output.Error, "intentional error")
}

func TestFunctionCallExecutor_Execute_InvalidJSON(t *testing.T) {
	executor := NewFunctionCallExecutor()

	executor.RegisterFunction("test_func", func(args map[string]any) (any, error) {
		return "ok", nil
	})

	config := map[string]any{
		"function_name": "test_func",
		"arguments":     `{invalid json}`,
	}

	_, err := executor.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse function arguments")
}

func TestFunctionCallExecutor_BuiltInFunctions(t *testing.T) {
	executor := NewFunctionCallExecutor()

	t.Run("get_current_time", func(t *testing.T) {
		config := map[string]any{
			"function_name": "get_current_time",
			"arguments":     `{"format": "unix"}`,
		}

		result, err := executor.Execute(context.Background(), config, nil)
		require.NoError(t, err)

		output, ok := result.(*models.FunctionCallOutput)
		require.True(t, ok)
		assert.True(t, output.Success)

		timestamp, ok := output.Result.(int64)
		require.True(t, ok)
		assert.Greater(t, timestamp, int64(0))
	})

	t.Run("get_current_date", func(t *testing.T) {
		config := map[string]any{
			"function_name": "get_current_date",
			"arguments":     `{}`,
		}

		result, err := executor.Execute(context.Background(), config, nil)
		require.NoError(t, err)

		output, ok := result.(*models.FunctionCallOutput)
		require.True(t, ok)
		assert.True(t, output.Success)

		date, ok := output.Result.(string)
		require.True(t, ok)
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, date)
	})

	t.Run("json_parse", func(t *testing.T) {
		config := map[string]any{
			"function_name": "json_parse",
			"arguments":     `{"json": "{\"name\":\"John\",\"age\":30}"}`,
		}

		result, err := executor.Execute(context.Background(), config, nil)
		require.NoError(t, err)

		output, ok := result.(*models.FunctionCallOutput)
		require.True(t, ok)
		assert.True(t, output.Success)

		parsed, ok := output.Result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "John", parsed["name"])
		assert.Equal(t, float64(30), parsed["age"])
	})

	t.Run("json_stringify", func(t *testing.T) {
		config := map[string]any{
			"function_name": "json_stringify",
			"arguments":     `{"value": {"name": "John", "age": 30}}`,
		}

		result, err := executor.Execute(context.Background(), config, nil)
		require.NoError(t, err)

		output, ok := result.(*models.FunctionCallOutput)
		require.True(t, ok)
		assert.True(t, output.Success)

		jsonStr, ok := output.Result.(string)
		require.True(t, ok)
		assert.Contains(t, jsonStr, "John")
		assert.Contains(t, jsonStr, "30")
	})

	t.Run("get_weather", func(t *testing.T) {
		config := map[string]any{
			"function_name": "get_weather",
			"arguments":     `{"location": "London", "unit": "celsius"}`,
		}

		result, err := executor.Execute(context.Background(), config, nil)
		require.NoError(t, err)

		output, ok := result.(*models.FunctionCallOutput)
		require.True(t, ok)
		assert.True(t, output.Success)

		weather, ok := output.Result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "London", weather["location"])
		assert.Equal(t, "celsius", weather["unit"])
		assert.NotNil(t, weather["temperature"])
		assert.NotNil(t, weather["condition"])
	})
}

func TestFunctionCallExecutor_RegisterAndUnregister(t *testing.T) {
	executor := NewFunctionCallExecutor()

	// Register a function
	executor.RegisterFunction("custom_func", func(args map[string]any) (any, error) {
		return "custom result", nil
	})

	// Verify it's registered
	functions := executor.ListFunctions()
	assert.Contains(t, functions, "custom_func")

	// Execute the function
	config := map[string]any{
		"function_name": "custom_func",
		"arguments":     `{}`,
	}

	result, err := executor.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	output, ok := result.(*models.FunctionCallOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "custom result", output.Result)

	// Unregister the function
	executor.UnregisterFunction("custom_func")

	// Verify it's no longer registered
	functions = executor.ListFunctions()
	assert.NotContains(t, functions, "custom_func")

	// Try to execute - should fail
	result, err = executor.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	output, ok = result.(*models.FunctionCallOutput)
	require.True(t, ok)
	assert.False(t, output.Success)
	assert.Contains(t, output.Error, "function not found")
}

func TestFunctionCallExecutor_Validate(t *testing.T) {
	executor := NewFunctionCallExecutor()

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name: "valid config with function_name",
			config: map[string]any{
				"function_name": "test_func",
				"arguments":     `{}`,
			},
			wantErr: false,
		},
		{
			name: "empty function_name",
			config: map[string]any{
				"function_name": "",
			},
			wantErr: true,
		},
		{
			name:    "empty config",
			config:  map[string]any{},
			wantErr: false, // Config can be empty, function_name can come from input
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFunctionCallExecutor_ParseInput_Formats(t *testing.T) {
	executor := NewFunctionCallExecutor()

	tests := []struct {
		name         string
		config       map[string]any
		input        any
		expectName   string
		expectArgs   string
		expectToolID string
		expectError  bool
	}{
		{
			name: "config only",
			config: map[string]any{
				"function_name": "func1",
				"arguments":     `{"a": 1}`,
				"tool_call_id":  "tool-1",
			},
			input:        nil,
			expectName:   "func1",
			expectArgs:   `{"a": 1}`,
			expectToolID: "tool-1",
		},
		{
			name:   "input map format",
			config: map[string]any{},
			input: map[string]any{
				"function_name": "func2",
				"arguments":     `{"b": 2}`,
				"tool_call_id":  "tool-2",
			},
			expectName:   "func2",
			expectArgs:   `{"b": 2}`,
			expectToolID: "tool-2",
		},
		{
			name:   "tool call format",
			config: map[string]any{},
			input: map[string]any{
				"id":   "tool-3",
				"type": "function",
				"function": map[string]any{
					"name":      "func3",
					"arguments": `{"c": 3}`,
				},
			},
			expectName:   "func3",
			expectArgs:   `{"c": 3}`,
			expectToolID: "tool-3",
		},
		{
			name:   "tool_calls array",
			config: map[string]any{},
			input: map[string]any{
				"tool_calls": []any{
					map[string]any{
						"id":   "tool-4",
						"type": "function",
						"function": map[string]any{
							"name":      "func4",
							"arguments": `{"d": 4}`,
						},
					},
				},
			},
			expectName:   "func4",
			expectArgs:   `{"d": 4}`,
			expectToolID: "tool-4",
		},
		{
			name: "config overrides input",
			config: map[string]any{
				"function_name": "config_func",
				"arguments":     `{"override": true}`,
			},
			input: map[string]any{
				"function_name": "input_func",
				"arguments":     `{"override": false}`,
			},
			expectName: "config_func",
			expectArgs: `{"override": true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcInput, err := executor.parseInput(tt.config, tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectName, funcInput.FunctionName)
			assert.Equal(t, tt.expectArgs, funcInput.Arguments)

			if tt.expectToolID != "" {
				assert.Equal(t, tt.expectToolID, funcInput.ToolCallID)
			}
		})
	}
}

func TestFunctionCallInput_ParseArguments(t *testing.T) {
	tests := []struct {
		name        string
		input       *models.FunctionCallInput
		expectArgs  map[string]any
		expectError bool
	}{
		{
			name: "valid JSON",
			input: &models.FunctionCallInput{
				FunctionName: "test",
				Arguments:    `{"name": "John", "age": 30}`,
			},
			expectArgs: map[string]any{
				"name": "John",
				"age":  float64(30),
			},
			expectError: false,
		},
		{
			name: "empty object",
			input: &models.FunctionCallInput{
				FunctionName: "test",
				Arguments:    `{}`,
			},
			expectArgs:  map[string]any{},
			expectError: false,
		},
		{
			name: "invalid JSON",
			input: &models.FunctionCallInput{
				FunctionName: "test",
				Arguments:    `{invalid}`,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := tt.input.ParseArguments()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectArgs, args)
		})
	}
}

func TestFunctionRegistry(t *testing.T) {
	registry := models.NewFunctionRegistry()

	// Test Register and Get
	handler := func(args map[string]any) (any, error) {
		return "test result", nil
	}

	registry.Register("test_func", handler)

	retrievedHandler, ok := registry.Get("test_func")
	assert.True(t, ok)
	assert.NotNil(t, retrievedHandler)

	result, err := retrievedHandler(map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "test result", result)

	// Test Has
	assert.True(t, registry.Has("test_func"))
	assert.False(t, registry.Has("nonexistent"))

	// Test List
	functions := registry.List()
	assert.Contains(t, functions, "test_func")

	// Test Unregister
	registry.Unregister("test_func")
	assert.False(t, registry.Has("test_func"))
}
