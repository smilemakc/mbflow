package builtin

import (
	"context"
	"testing"
)

func TestConditionalExecutor_Execute_Success_True(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "expression",
		"condition":      "input.score >= 80",
	}

	input := map[string]any{
		"score": 85,
	}

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	boolResult, ok := conditionalResult(result)
	if !ok {
		t.Errorf("Expected a map with a bool result, got: %T", result)
	} else if !boolResult {
		t.Errorf("Expected true, got false")
	}
}

func TestConditionalExecutor_Execute_Success_False(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "expression",
		"condition":      "input.score >= 80",
	}

	input := map[string]any{
		"score": 50,
	}

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	boolResult, ok := conditionalResult(result)
	if !ok {
		t.Errorf("Expected a map with a bool result, got: %T", result)
	} else if boolResult {
		t.Errorf("Expected false, got true")
	}
}

func TestConditionalExecutor_Execute_CompilationError(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "expression",
		"condition":      "input.score >= && 80", // Invalid syntax
	}

	input := map[string]any{
		"score": 50,
	}

	_, err := executor.Execute(context.Background(), config, input)
	if err == nil {
		t.Error("Expected compilation error, got nil")
	}

	expectedMsg := "failed to compile expression"
	if err != nil && len(err.Error()) > 0 {
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to start with '%s', got: %v", expectedMsg, err)
		}
	}
}

func TestConditionalExecutor_Execute_RuntimeError(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "expression",
		"condition":      "input.score >= 80", // score doesn't exist in input
	}

	input := map[string]any{
		"data": "value",
	}

	_, err := executor.Execute(context.Background(), config, input)
	if err == nil {
		t.Error("Expected runtime error, got nil")
	}

	expectedMsg := "failed to execute expression"
	if err != nil && len(err.Error()) > 0 {
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to start with '%s', got: %v", expectedMsg, err)
		}
	}
}

func TestConditionalExecutor_Execute_NonBooleanResult(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "expression",
		"condition":      "input.score", // Returns number, not bool
	}

	input := map[string]any{
		"score": 50,
	}

	_, err := executor.Execute(context.Background(), config, input)
	if err == nil {
		t.Error("Expected error for non-boolean result, got nil")
	}

	expectedMsg := "expression result is not a boolean"
	if err != nil && len(err.Error()) > 0 {
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to start with '%s', got: %v", expectedMsg, err)
		}
	}
}

func TestConditionalExecutor_Execute_MissingCondition(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "expression",
		// Missing "condition" field
	}

	input := map[string]any{
		"score": 50,
	}

	_, err := executor.Execute(context.Background(), config, input)
	if err == nil {
		t.Error("Expected error for missing condition, got nil")
	}
}

func TestConditionalExecutor_Execute_UnknownConditionType(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "unknown_type",
		"condition":      "input.score >= 80",
	}

	input := map[string]any{
		"score": 50,
	}

	_, err := executor.Execute(context.Background(), config, input)
	if err == nil {
		t.Error("Expected error for unknown condition type, got nil")
	}

	expectedMsg := "unknown condition type"
	if err != nil && len(err.Error()) > 0 {
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to start with '%s', got: %v", expectedMsg, err)
		}
	}
}

func TestConditionalExecutor_Execute_DefaultConditionType(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		// No condition_type specified, should default to "expression"
		"condition": "input.value == true",
	}

	input := map[string]any{
		"value": true,
	}

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	boolResult, ok := conditionalResult(result)
	if !ok {
		t.Errorf("Expected a map with a bool result, got: %T", result)
	} else if !boolResult {
		t.Errorf("Expected true, got false")
	}
}

func TestConditionalExecutor_Execute_ComplexExpression(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "expression",
		"condition":      "input.score >= 50 && input.score < 80 && input.status == 'active'",
	}

	input := map[string]any{
		"score":  60,
		"status": "active",
	}

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	boolResult, ok := conditionalResult(result)
	if !ok {
		t.Errorf("Expected a map with a bool result, got: %T", result)
	} else if !boolResult {
		t.Errorf("Expected true, got false")
	}
}

func TestConditionalExecutor_Validate_Success(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "expression",
		"expression":     "input.value == true",
	}

	err := executor.Validate(config)
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}

func TestConditionalExecutor_Validate_InvalidConditionType(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "invalid_type",
		"expression":     "input.value == true",
	}

	err := executor.Validate(config)
	if err == nil {
		t.Error("Expected error for invalid condition type, got nil")
	}

	expectedMsg := "invalid condition type"
	if err != nil && len(err.Error()) > 0 {
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to start with '%s', got: %v", expectedMsg, err)
		}
	}
}

func TestConditionalExecutor_Validate_MissingExpression(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		"condition_type": "expression",
		// Missing "expression" field
	}

	err := executor.Validate(config)
	if err == nil {
		t.Error("Expected error for missing expression, got nil")
	}

	expectedMsg := "condition is required"
	if err != nil && len(err.Error()) > 0 {
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to start with '%s', got: %v", expectedMsg, err)
		}
	}
}

func TestConditionalExecutor_Validate_DefaultConditionType(t *testing.T) {
	executor := NewConditionalExecutor()

	config := map[string]any{
		// No condition_type, defaults to "expression"
		"expression": "input.value == true",
	}

	err := executor.Validate(config)
	if err != nil {
		t.Errorf("Expected valid config with default condition type, got error: %v", err)
	}
}

// conditionalResult extracts the boolean outcome from the executor output.
func conditionalResult(out any) (bool, bool) {
	m, ok := out.(map[string]any)
	if !ok {
		return false, false
	}
	b, ok := m["result"].(bool)
	return b, ok
}

func TestConditionalExecutor_PassesInputThrough(t *testing.T) {
	executor := NewConditionalExecutor()

	out, err := executor.Execute(context.Background(), map[string]any{"condition": "input.score > 1"},
		map[string]any{"score": 5, "name": "x"})
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	if m["score"] != 5 || m["name"] != "x" || m["result"] != true {
		t.Errorf("unexpected output: %v", m)
	}

	out, err = executor.Execute(context.Background(), map[string]any{"expression": "input > 1"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	m = out.(map[string]any)
	if m["input"] != 3 || m["result"] != true {
		t.Errorf("unexpected output for scalar input: %v", m)
	}

	out, err = executor.Execute(context.Background(), map[string]any{"condition": "true"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.(map[string]any)) != 1 {
		t.Errorf("nil input should only produce result: %v", out)
	}

	if _, err := executor.Execute(context.Background(), map[string]any{}, nil); err == nil {
		t.Error("expected an error without a condition")
	}
	if err := executor.Validate(map[string]any{"condition": "true"}); err != nil {
		t.Errorf("condition key should validate: %v", err)
	}
	if err := executor.Validate(map[string]any{"expression": "true"}); err != nil {
		t.Errorf("expression key should validate: %v", err)
	}
}
