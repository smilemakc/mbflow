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

	if boolResult, ok := result.(bool); !ok {
		t.Errorf("Expected bool result, got: %T", result)
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

	if boolResult, ok := result.(bool); !ok {
		t.Errorf("Expected bool result, got: %T", result)
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

	if boolResult, ok := result.(bool); !ok {
		t.Errorf("Expected bool result, got: %T", result)
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

	if boolResult, ok := result.(bool); !ok {
		t.Errorf("Expected bool result, got: %T", result)
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

	expectedMsg := "expression is required"
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
