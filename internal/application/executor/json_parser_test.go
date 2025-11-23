package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONParserExecutor_ParseValidJSON(t *testing.T) {
	executor := NewJSONParserExecutor()
	assert.Equal(t, "json-parser", executor.Type())

	// Create execution context
	state := NewExecutionState("test-exec", "test-workflow")
	execCtx := NewExecutionContext(context.Background(), state)

	// Set a JSON string variable
	jsonStr := `{"score": 10, "pass": true, "issues": []}`
	execCtx.SetVariable("quality_score", jsonStr)

	// Execute parser
	config := map[string]any{
		"input_key": "quality_score",
	}

	result, err := executor.Execute(context.Background(), execCtx, "node-1", config)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check result
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "success", resultMap["status"])

	// Check that variable was parsed
	parsedValue, ok := execCtx.GetVariable("quality_score")
	require.True(t, ok)

	parsedMap, ok := parsedValue.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(10), parsedMap["score"])
	assert.Equal(t, true, parsedMap["pass"])
}

func TestJSONParserExecutor_NestedAccess(t *testing.T) {
	executor := NewJSONParserExecutor()

	// Create execution context
	state := NewExecutionState("test-exec", "test-workflow")
	execCtx := NewExecutionContext(context.Background(), state)

	// Set a JSON string variable
	jsonStr := `{"user": {"name": "John", "email": "john@example.com"}, "status": "active"}`
	execCtx.SetVariable("user_data", jsonStr)

	// Execute parser
	config := map[string]any{
		"input_key":  "user_data",
		"output_key": "parsed_user",
	}

	result, err := executor.Execute(context.Background(), execCtx, "node-1", config)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check that we can access nested values
	allVars := execCtx.GetAllVariables()
	userName := getNestedValue(allVars, "parsed_user.user.name")
	assert.Equal(t, "John", userName)

	userEmail := getNestedValue(allVars, "parsed_user.user.email")
	assert.Equal(t, "john@example.com", userEmail)
}

func TestJSONParserExecutor_InvalidJSON_FailOnError(t *testing.T) {
	executor := NewJSONParserExecutor()

	// Create execution context
	state := NewExecutionState("test-exec", "test-workflow")
	execCtx := NewExecutionContext(context.Background(), state)

	// Set an invalid JSON string
	execCtx.SetVariable("bad_json", "not a json string")

	// Execute parser with fail_on_error=true (default)
	config := map[string]any{
		"input_key": "bad_json",
	}

	result, err := executor.Execute(context.Background(), execCtx, "node-1", config)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestJSONParserExecutor_InvalidJSON_NoFailOnError(t *testing.T) {
	executor := NewJSONParserExecutor()

	// Create execution context
	state := NewExecutionState("test-exec", "test-workflow")
	execCtx := NewExecutionContext(context.Background(), state)

	// Set an invalid JSON string
	execCtx.SetVariable("bad_json", "not a json string")

	// Execute parser with fail_on_error=false
	config := map[string]any{
		"input_key":     "bad_json",
		"fail_on_error": false,
	}

	result, err := executor.Execute(context.Background(), execCtx, "node-1", config)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check result
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "parse_error", resultMap["status"])
	assert.True(t, resultMap["passthrough"].(bool))

	// Original value should be preserved
	value, ok := execCtx.GetVariable("bad_json")
	require.True(t, ok)
	assert.Equal(t, "not a json string", value)
}

func TestJSONParserExecutor_AlreadyParsed(t *testing.T) {
	executor := NewJSONParserExecutor()

	// Create execution context
	state := NewExecutionState("test-exec", "test-workflow")
	execCtx := NewExecutionContext(context.Background(), state)

	// Set an already parsed object
	parsedObj := map[string]interface{}{
		"key": "value",
	}
	execCtx.SetVariable("already_parsed", parsedObj)

	// Execute parser
	config := map[string]any{
		"input_key": "already_parsed",
	}

	result, err := executor.Execute(context.Background(), execCtx, "node-1", config)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check result
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "passthrough", resultMap["status"])
	assert.True(t, resultMap["already_parsed"].(bool))

	// Value should be unchanged
	value, ok := execCtx.GetVariable("already_parsed")
	require.True(t, ok)
	assert.Equal(t, parsedObj, value)
}

func TestJSONParserExecutor_DifferentOutputKey(t *testing.T) {
	executor := NewJSONParserExecutor()

	// Create execution context
	state := NewExecutionState("test-exec", "test-workflow")
	execCtx := NewExecutionContext(context.Background(), state)

	// Set a JSON string variable
	jsonStr := `{"name": "Test"}`
	execCtx.SetVariable("input_json", jsonStr)

	// Execute parser with different output key
	config := map[string]any{
		"input_key":  "input_json",
		"output_key": "parsed_json",
	}

	result, err := executor.Execute(context.Background(), execCtx, "node-1", config)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Original should still be a string
	originalValue, ok := execCtx.GetVariable("input_json")
	require.True(t, ok)
	assert.Equal(t, jsonStr, originalValue)

	// Parsed should be an object
	parsedValue, ok := execCtx.GetVariable("parsed_json")
	require.True(t, ok)
	parsedMap, ok := parsedValue.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Test", parsedMap["name"])
}

func TestJSONParserExecutor_ParseArray(t *testing.T) {
	executor := NewJSONParserExecutor()

	// Create execution context
	state := NewExecutionState("test-exec", "test-workflow")
	execCtx := NewExecutionContext(context.Background(), state)

	// Set a JSON array string
	jsonStr := `[1, 2, 3, 4, 5]`
	execCtx.SetVariable("numbers", jsonStr)

	// Execute parser
	config := map[string]any{
		"input_key": "numbers",
	}

	result, err := executor.Execute(context.Background(), execCtx, "node-1", config)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check that variable was parsed as array
	parsedValue, ok := execCtx.GetVariable("numbers")
	require.True(t, ok)

	parsedArray, ok := parsedValue.([]interface{})
	require.True(t, ok)
	assert.Len(t, parsedArray, 5)
	assert.Equal(t, float64(1), parsedArray[0])
	assert.Equal(t, float64(5), parsedArray[4])
}

func TestJSONParserExecutor_MissingInputKey(t *testing.T) {
	executor := NewJSONParserExecutor()

	// Create execution context
	state := NewExecutionState("test-exec", "test-workflow")
	execCtx := NewExecutionContext(context.Background(), state)

	// Execute parser without setting the input variable
	config := map[string]any{
		"input_key": "missing_key",
	}

	result, err := executor.Execute(context.Background(), execCtx, "node-1", config)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}
