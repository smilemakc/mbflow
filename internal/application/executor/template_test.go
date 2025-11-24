package executor

import (
	"testing"
)

func TestProcessSimpleVariable(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"name": "John",
		"age":  30,
	}

	result, err := tp.processString("Hello {{name}}, you are {{age}} years old", vars, TemplateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello John, you are 30 years old"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessExpression(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"score": 85,
	}

	result, err := tp.processString("Priority: ${score > 80 ? 'high' : 'low'}", vars, TemplateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Priority: high"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessNestedAccess(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"user": map[string]any{
			"contact": map[string]any{
				"email": "john@example.com",
			},
		},
	}

	result, err := tp.processString("Email: {{user.contact.email}}", vars, TemplateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Email: john@example.com"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessMixedSyntax(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"base_price": 100,
		"customer": map[string]any{
			"name": "Alice",
		},
	}

	result, err := tp.processString("Price: ${base_price + 10} for {{customer.name}}", vars, TemplateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Price: 110 for Alice"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStrictModeWithMissingVariable(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{}

	config := TemplateConfig{
		StrictMode: true,
	}

	_, err := tp.processString("Hello {{missing}}", vars, config)
	if err == nil {
		t.Fatal("expected error for missing variable in strict mode")
	}
}

func TestLenientModeWithMissingVariable(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{}

	config := TemplateConfig{
		StrictMode: false,
	}

	result, err := tp.processString("Hello {{missing}}", vars, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello {{missing}}"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessMap(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"name":   "Bob",
		"status": "active",
	}

	input := map[string]any{
		"prompt":  "Hello {{name}}",
		"message": "Status: {{status}}",
		"number":  42,
	}

	result, err := tp.processMap(input, vars, TemplateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["prompt"] != "Hello Bob" {
		t.Errorf("expected 'Hello Bob', got %q", result["prompt"])
	}
	if result["message"] != "Status: active" {
		t.Errorf("expected 'Status: active', got %q", result["message"])
	}
	if result["number"] != 42 {
		t.Errorf("expected 42, got %v", result["number"])
	}
}

func TestProcessSlice(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"name": "Charlie",
	}

	input := []any{
		"Hello {{name}}",
		"Welcome",
		123,
	}

	result, err := tp.processSlice(input, vars, TemplateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	if result[0] != "Hello Charlie" {
		t.Errorf("expected 'Hello Charlie', got %q", result[0])
	}
	if result[1] != "Welcome" {
		t.Errorf("expected 'Welcome', got %q", result[1])
	}
	if result[2] != 123 {
		t.Errorf("expected 123, got %v", result[2])
	}
}

func TestConditionalExpression(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"difficulty": 7,
	}

	result, err := tp.processString("Level: ${difficulty > 5 ? 'advanced' : 'beginner'}", vars, TemplateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Level: advanced"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestArithmeticExpression(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"quantity": 5,
		"price":    10.0,
	}

	result, err := tp.processString("Total: ${quantity * price}", vars, TemplateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Total: 50"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFieldSpecificTemplating(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"user_id": 123,
	}

	input := map[string]any{
		"url":          "https://api.example.com/users/${user_id}",
		"static_field": "{{not_templated}}",
	}

	config := TemplateConfig{
		Fields: []string{"url"}, // Only template the URL field
	}

	result, err := tp.ProcessMap(input, vars, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["url"] != "https://api.example.com/users/123" {
		t.Errorf("expected 'https://api.example.com/users/123', got %q", result["url"])
	}
	if result["static_field"] != "{{not_templated}}" {
		t.Errorf("expected '{{not_templated}}', got %q", result["static_field"])
	}
}

func TestEdgeCases(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	tests := []struct {
		name     string
		input    string
		vars     map[string]any
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			vars:     map[string]any{},
			expected: "",
		},
		{
			name:     "No template patterns",
			input:    "Hello World",
			vars:     map[string]any{},
			expected: "Hello World",
		},
		{
			name:     "Multiple same variable",
			input:    "{{name}} and {{name}}",
			vars:     map[string]any{"name": "Test"},
			expected: "Test and Test",
		},
		{
			name:     "Spaces in variable",
			input:    "{{ name }}",
			vars:     map[string]any{"name": "Test"},
			expected: "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tp.processString(tt.input, tt.vars, TemplateConfig{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRecursiveMapProcessing(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"topic": "AI",
	}

	input := map[string]any{
		"config": map[string]any{
			"prompt": "Write about {{topic}}",
			"nested": map[string]any{
				"field": "Topic: {{topic}}",
			},
		},
	}

	result, err := tp.processMap(input, vars, TemplateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config := result["config"].(map[string]any)
	if config["prompt"] != "Write about AI" {
		t.Errorf("expected 'Write about AI', got %q", config["prompt"])
	}

	nested := config["nested"].(map[string]any)
	if nested["field"] != "Topic: AI" {
		t.Errorf("expected 'Topic: AI', got %q", nested["field"])
	}
}

func TestStrictModeWithExpressionError(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"score": 50,
	}

	config := TemplateConfig{
		StrictMode: true,
	}

	// Reference undefined variable in expression
	_, err := tp.processString("Result: ${undefined_var + 10}", vars, config)
	if err == nil {
		t.Fatal("expected error for undefined variable in expression")
	}
}

func TestLenientModeWithExpressionError(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{
		"score": 50,
	}

	config := TemplateConfig{
		StrictMode: false,
	}

	// Reference undefined variable in expression
	result, err := tp.processString("Result: ${undefined_var + 10}", vars, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Result: ${undefined_var + 10}"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessNonStringTypes(t *testing.T) {
	evaluator := NewConditionEvaluator(true)
	tp := NewTemplateProcessor(evaluator)

	vars := map[string]any{}

	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "Integer",
			input:    42,
			expected: 42,
		},
		{
			name:     "Float",
			input:    3.14,
			expected: 3.14,
		},
		{
			name:     "Boolean",
			input:    true,
			expected: true,
		},
		{
			name:     "Nil",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tp.Process(tt.input, vars, TemplateConfig{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
