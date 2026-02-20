package builtin

import (
	"context"
	"reflect"
	"testing"
)

func TestMergeExecutor_Execute_StrategyAll(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "all",
	}

	input := map[string]any{
		"parent1": map[string]any{"data": "value1"},
		"parent2": map[string]any{"data": "value2"},
	}

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Strategy "all" passes through the input
	if !reflect.DeepEqual(result, input) {
		t.Errorf("Expected result to equal input, got: %v", result)
	}
}

func TestMergeExecutor_Execute_StrategyAny(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "any",
	}

	input := map[string]any{
		"parent1": map[string]any{"data": "value1"},
	}

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Strategy "any" passes through the input
	if !reflect.DeepEqual(result, input) {
		t.Errorf("Expected result to equal input, got: %v", result)
	}
}

func TestMergeExecutor_Execute_DefaultStrategy(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		// No merge_strategy specified, should default to "all"
	}

	input := map[string]any{
		"data": "value",
	}

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Default strategy "all" passes through the input
	if !reflect.DeepEqual(result, input) {
		t.Errorf("Expected result to equal input, got: %v", result)
	}
}

func TestMergeExecutor_Execute_UnknownStrategy(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "unknown_strategy",
	}

	input := map[string]any{
		"data": "value",
	}

	_, err := executor.Execute(context.Background(), config, input)
	if err == nil {
		t.Error("Expected error for unknown strategy, got nil")
	}

	expectedMsg := "unknown merge strategy"
	if err != nil && len(err.Error()) > 0 {
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to start with '%s', got: %v", expectedMsg, err)
		}
	}
}

func TestMergeExecutor_Execute_WithArrayInput(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "all",
	}

	input := []any{
		map[string]any{"id": 1},
		map[string]any{"id": 2},
	}

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Strategy "all" passes through the input
	if !reflect.DeepEqual(result, input) {
		t.Errorf("Expected result to equal input, got: %v", result)
	}
}

func TestMergeExecutor_Execute_WithStringInput(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "all",
	}

	input := "simple string input"

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Strategy "all" passes through the input
	if !reflect.DeepEqual(result, input) {
		t.Errorf("Expected result to equal input, got: %v", result)
	}
}

func TestMergeExecutor_Execute_WithNilInput(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "all",
	}

	var input any = nil

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Strategy "all" passes through the input
	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}
}

func TestMergeExecutor_Validate_StrategyAll(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "all",
	}

	err := executor.Validate(config)
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}

func TestMergeExecutor_Validate_StrategyAny(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "any",
	}

	err := executor.Validate(config)
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}

func TestMergeExecutor_Validate_DefaultStrategy(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		// No merge_strategy, defaults to "all"
	}

	err := executor.Validate(config)
	if err != nil {
		t.Errorf("Expected valid config with default strategy, got error: %v", err)
	}
}

func TestMergeExecutor_Validate_InvalidStrategy(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "invalid_strategy",
	}

	err := executor.Validate(config)
	if err == nil {
		t.Error("Expected error for invalid strategy, got nil")
	}

	expectedMsg := "invalid merge strategy"
	if err != nil && len(err.Error()) > 0 {
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to start with '%s', got: %v", expectedMsg, err)
		}
	}
}

func TestMergeExecutor_Execute_ComplexNestedInput(t *testing.T) {
	executor := NewMergeExecutor()

	config := map[string]any{
		"merge_strategy": "all",
	}

	input := map[string]any{
		"parent1": map[string]any{
			"nested": map[string]any{
				"value": 123,
			},
		},
		"parent2": []any{
			map[string]any{"id": 1},
			map[string]any{"id": 2},
		},
		"parent3": "simple value",
	}

	result, err := executor.Execute(context.Background(), config, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Strategy "all" passes through the input
	if !reflect.DeepEqual(result, input) {
		t.Errorf("Expected result to equal input, got: %v", result)
	}
}
