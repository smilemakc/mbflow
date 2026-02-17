package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== FunctionCallInput Tests ====================

func TestFunctionCallInput_ParseArguments_Success(t *testing.T) {
	input := &FunctionCallInput{
		FunctionName: "get_weather",
		Arguments:    `{"location":"Paris","units":"celsius"}`,
		ToolCallID:   "call_123",
	}

	args, err := input.ParseArguments()
	require.NoError(t, err)
	assert.NotNil(t, args)
	assert.Equal(t, "Paris", args["location"])
	assert.Equal(t, "celsius", args["units"])
}

func TestFunctionCallInput_ParseArguments_EmptyObject(t *testing.T) {
	input := &FunctionCallInput{
		FunctionName: "no_args_function",
		Arguments:    `{}`,
	}

	args, err := input.ParseArguments()
	require.NoError(t, err)
	assert.NotNil(t, args)
	assert.Empty(t, args)
}

func TestFunctionCallInput_ParseArguments_ComplexArguments(t *testing.T) {
	input := &FunctionCallInput{
		FunctionName: "complex_function",
		Arguments:    `{"user":{"name":"John","age":30},"items":[1,2,3],"enabled":true}`,
	}

	args, err := input.ParseArguments()
	require.NoError(t, err)
	assert.NotNil(t, args)

	// Verify nested object
	user, ok := args["user"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "John", user["name"])
	assert.Equal(t, float64(30), user["age"]) // JSON numbers are float64

	// Verify array
	items, ok := args["items"].([]any)
	assert.True(t, ok)
	assert.Len(t, items, 3)

	// Verify boolean
	enabled, ok := args["enabled"].(bool)
	assert.True(t, ok)
	assert.True(t, enabled)
}

func TestFunctionCallInput_ParseArguments_InvalidJSON(t *testing.T) {
	input := &FunctionCallInput{
		FunctionName: "test_function",
		Arguments:    `{invalid json}`,
	}

	args, err := input.ParseArguments()
	assert.Error(t, err)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), "failed to parse function arguments")
}

func TestFunctionCallInput_ParseArguments_EmptyString(t *testing.T) {
	input := &FunctionCallInput{
		FunctionName: "test_function",
		Arguments:    "",
	}

	args, err := input.ParseArguments()
	assert.Error(t, err)
	assert.Nil(t, args)
}

func TestFunctionCallInput_ParseArguments_NonObjectJSON(t *testing.T) {
	// JSON array instead of object - should fail
	input := &FunctionCallInput{
		FunctionName: "test_function",
		Arguments:    `[1,2,3]`,
	}

	args, err := input.ParseArguments()
	assert.Error(t, err)
	assert.Nil(t, args)
}

func TestFunctionCallInput_JSONMarshaling(t *testing.T) {
	input := &FunctionCallInput{
		FunctionName: "get_weather",
		Arguments:    `{"location":"Paris"}`,
		ToolCallID:   "call_123",
		Metadata: map[string]any{
			"source": "llm",
		},
	}

	data, err := json.Marshal(input)
	require.NoError(t, err)

	var unmarshaled FunctionCallInput
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, input.FunctionName, unmarshaled.FunctionName)
	assert.Equal(t, input.Arguments, unmarshaled.Arguments)
	assert.Equal(t, input.ToolCallID, unmarshaled.ToolCallID)
	assert.NotNil(t, unmarshaled.Metadata)
}

// ==================== FunctionCallOutput Tests ====================

func TestFunctionCallOutput_Success(t *testing.T) {
	output := &FunctionCallOutput{
		Result: map[string]any{
			"temperature": 22,
			"conditions":  "sunny",
		},
		FunctionName: "get_weather",
		ToolCallID:   "call_123",
		Success:      true,
		Error:        "",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var unmarshaled FunctionCallOutput
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, output.FunctionName, unmarshaled.FunctionName)
	assert.Equal(t, output.ToolCallID, unmarshaled.ToolCallID)
	assert.True(t, unmarshaled.Success)
	assert.Empty(t, unmarshaled.Error)
	assert.NotNil(t, unmarshaled.Result)
}

func TestFunctionCallOutput_Error(t *testing.T) {
	output := &FunctionCallOutput{
		Result:       nil,
		FunctionName: "search_database",
		ToolCallID:   "call_456",
		Success:      false,
		Error:        "Database connection timeout",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var unmarshaled FunctionCallOutput
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, output.FunctionName, unmarshaled.FunctionName)
	assert.False(t, unmarshaled.Success)
	assert.Equal(t, "Database connection timeout", unmarshaled.Error)
	assert.Nil(t, unmarshaled.Result)
}

func TestFunctionCallOutput_WithMetadata(t *testing.T) {
	output := &FunctionCallOutput{
		Result:       map[string]any{"status": "ok"},
		FunctionName: "http_request",
		ToolCallID:   "call_789",
		Success:      true,
		Metadata: map[string]any{
			"duration_ms": 150,
			"cache_hit":   false,
		},
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var unmarshaled FunctionCallOutput
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.NotNil(t, unmarshaled.Metadata)
	assert.Equal(t, float64(150), unmarshaled.Metadata["duration_ms"])
	assert.Equal(t, false, unmarshaled.Metadata["cache_hit"])
}

// ==================== FunctionCallConfig Tests ====================

func TestFunctionCallConfig_JSONMarshaling(t *testing.T) {
	config := &FunctionCallConfig{
		FunctionName: "get_weather",
		Arguments:    `{"location":"Paris"}`,
		ToolCallID:   "call_123",
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled FunctionCallConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.FunctionName, unmarshaled.FunctionName)
	assert.Equal(t, config.Arguments, unmarshaled.Arguments)
	assert.Equal(t, config.ToolCallID, unmarshaled.ToolCallID)
}

// ==================== FunctionRegistry Tests ====================

func TestNewFunctionRegistry_Success(t *testing.T) {
	registry := NewFunctionRegistry()

	require.NotNil(t, registry)
	assert.NotNil(t, registry.handlers)
	assert.Empty(t, registry.handlers)
}

func TestFunctionRegistry_Register(t *testing.T) {
	registry := NewFunctionRegistry()

	handler := func(args map[string]any) (any, error) {
		return "result", nil
	}

	registry.Register("test_function", handler)

	assert.True(t, registry.Has("test_function"))
}

func TestFunctionRegistry_Get_Existing(t *testing.T) {
	registry := NewFunctionRegistry()

	handler := func(args map[string]any) (any, error) {
		return "test result", nil
	}

	registry.Register("test_function", handler)

	retrieved, ok := registry.Get("test_function")
	assert.True(t, ok)
	assert.NotNil(t, retrieved)

	// Test the handler works
	result, err := retrieved(map[string]any{"key": "value"})
	require.NoError(t, err)
	assert.Equal(t, "test result", result)
}

func TestFunctionRegistry_Get_NonExisting(t *testing.T) {
	registry := NewFunctionRegistry()

	handler, ok := registry.Get("non_existent")
	assert.False(t, ok)
	assert.Nil(t, handler)
}

func TestFunctionRegistry_Has_Existing(t *testing.T) {
	registry := NewFunctionRegistry()

	handler := func(args map[string]any) (any, error) {
		return nil, nil
	}

	registry.Register("test_function", handler)

	assert.True(t, registry.Has("test_function"))
}

func TestFunctionRegistry_Has_NonExisting(t *testing.T) {
	registry := NewFunctionRegistry()

	assert.False(t, registry.Has("non_existent"))
}

func TestFunctionRegistry_List_Empty(t *testing.T) {
	registry := NewFunctionRegistry()

	names := registry.List()
	assert.NotNil(t, names)
	assert.Empty(t, names)
}

func TestFunctionRegistry_List_MultipleFunctions(t *testing.T) {
	registry := NewFunctionRegistry()

	handler := func(args map[string]any) (any, error) {
		return nil, nil
	}

	registry.Register("function1", handler)
	registry.Register("function2", handler)
	registry.Register("function3", handler)

	names := registry.List()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "function1")
	assert.Contains(t, names, "function2")
	assert.Contains(t, names, "function3")
}

func TestFunctionRegistry_Unregister_Existing(t *testing.T) {
	registry := NewFunctionRegistry()

	handler := func(args map[string]any) (any, error) {
		return nil, nil
	}

	registry.Register("test_function", handler)
	assert.True(t, registry.Has("test_function"))

	registry.Unregister("test_function")
	assert.False(t, registry.Has("test_function"))
}

func TestFunctionRegistry_Unregister_NonExisting(t *testing.T) {
	registry := NewFunctionRegistry()

	// Should not panic when unregistering non-existent function
	registry.Unregister("non_existent")

	// Verify registry is still usable
	handler := func(args map[string]any) (any, error) {
		return nil, nil
	}
	registry.Register("test", handler)
	assert.True(t, registry.Has("test"))
}

func TestFunctionRegistry_OverwriteRegistration(t *testing.T) {
	registry := NewFunctionRegistry()

	handler1 := func(args map[string]any) (any, error) {
		return "result1", nil
	}

	handler2 := func(args map[string]any) (any, error) {
		return "result2", nil
	}

	registry.Register("test_function", handler1)
	registry.Register("test_function", handler2) // Overwrite

	retrieved, ok := registry.Get("test_function")
	assert.True(t, ok)

	result, err := retrieved(nil)
	require.NoError(t, err)
	assert.Equal(t, "result2", result) // Should be the second handler
}

// ==================== FunctionHandler Tests ====================

func TestFunctionHandler_Success(t *testing.T) {
	handler := FunctionHandler(func(args map[string]any) (any, error) {
		location := args["location"].(string)
		return map[string]any{
			"location":    location,
			"temperature": 22,
		}, nil
	})

	result, err := handler(map[string]any{
		"location": "Paris",
	})

	require.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]any)
	assert.Equal(t, "Paris", resultMap["location"])
	assert.Equal(t, 22, resultMap["temperature"])
}

func TestFunctionHandler_WithError(t *testing.T) {
	handler := FunctionHandler(func(args map[string]any) (any, error) {
		return nil, assert.AnError
	})

	result, err := handler(map[string]any{})

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== Complex Integration Tests ====================

func TestFunctionCall_CompleteWorkflow(t *testing.T) {
	// 1. Create registry
	registry := NewFunctionRegistry()

	// 2. Register functions
	registry.Register("get_weather", func(args map[string]any) (any, error) {
		location := args["location"].(string)
		return map[string]any{
			"location":    location,
			"temperature": 22,
			"conditions":  "sunny",
		}, nil
	})

	registry.Register("calculate_sum", func(args map[string]any) (any, error) {
		numbers := args["numbers"].([]any)
		sum := 0.0
		for _, num := range numbers {
			sum += num.(float64)
		}
		return sum, nil
	})

	// 3. Create function call input
	input := &FunctionCallInput{
		FunctionName: "get_weather",
		Arguments:    `{"location":"Paris"}`,
		ToolCallID:   "call_123",
	}

	// 4. Parse arguments
	args, err := input.ParseArguments()
	require.NoError(t, err)

	// 5. Get handler from registry
	handler, ok := registry.Get(input.FunctionName)
	assert.True(t, ok)

	// 6. Execute function
	result, err := handler(args)
	require.NoError(t, err)

	// 7. Create output
	output := &FunctionCallOutput{
		Result:       result,
		FunctionName: input.FunctionName,
		ToolCallID:   input.ToolCallID,
		Success:      true,
	}

	// 8. Verify output
	assert.Equal(t, "get_weather", output.FunctionName)
	assert.Equal(t, "call_123", output.ToolCallID)
	assert.True(t, output.Success)

	resultMap := output.Result.(map[string]any)
	assert.Equal(t, "Paris", resultMap["location"])
	assert.Equal(t, 22, resultMap["temperature"])

	// 9. Test second function
	input2 := &FunctionCallInput{
		FunctionName: "calculate_sum",
		Arguments:    `{"numbers":[1,2,3,4,5]}`,
		ToolCallID:   "call_456",
	}

	args2, err := input2.ParseArguments()
	require.NoError(t, err)

	handler2, ok := registry.Get(input2.FunctionName)
	assert.True(t, ok)

	result2, err := handler2(args2)
	require.NoError(t, err)
	assert.Equal(t, float64(15), result2)
}

func TestFunctionCall_ErrorHandling(t *testing.T) {
	registry := NewFunctionRegistry()

	// Register function that returns error
	registry.Register("failing_function", func(args map[string]any) (any, error) {
		return nil, assert.AnError
	})

	input := &FunctionCallInput{
		FunctionName: "failing_function",
		Arguments:    `{}`,
		ToolCallID:   "call_error",
	}

	args, err := input.ParseArguments()
	require.NoError(t, err)

	handler, ok := registry.Get(input.FunctionName)
	assert.True(t, ok)

	result, err := handler(args)
	assert.Error(t, err)
	assert.Nil(t, result)

	// Create error output
	output := &FunctionCallOutput{
		Result:       nil,
		FunctionName: input.FunctionName,
		ToolCallID:   input.ToolCallID,
		Success:      false,
		Error:        err.Error(),
	}

	assert.False(t, output.Success)
	assert.NotEmpty(t, output.Error)
}

func TestFunctionCall_MultipleRegistrations(t *testing.T) {
	registry := NewFunctionRegistry()

	// Register 10 functions
	for i := 0; i < 10; i++ {
		funcName := "function_" + string(rune('0'+i))
		registry.Register(funcName, func(args map[string]any) (any, error) {
			return "result", nil
		})
	}

	// Verify all registered
	names := registry.List()
	assert.Len(t, names, 10)

	// Unregister half
	for i := 0; i < 5; i++ {
		funcName := "function_" + string(rune('0'+i))
		registry.Unregister(funcName)
	}

	// Verify remaining count
	names = registry.List()
	assert.Len(t, names, 5)
}
