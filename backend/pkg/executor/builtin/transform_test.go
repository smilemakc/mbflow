package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/smilemakc/mbflow/internal/application/template"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformExecutor_Passthrough(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type": "passthrough",
	}

	input := map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
	}

	result, err := exec.Execute(context.Background(), config, input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestTransformExecutor_Template(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":     "template",
		"template": "Hello {{env.name}}!",
	}

	// Create template engine with variables
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars = map[string]interface{}{
		"name": "World",
	}

	engine := template.NewEngine(varCtx, template.TemplateOptions{})
	wrappedExec := executor.NewTemplateExecutorWrapper(exec, engine)

	result, err := wrappedExec.Execute(context.Background(), config, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

func TestTransformExecutor_Expression_Simple(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":       "expression",
		"expression": "input.price * 2",
	}

	input := map[string]interface{}{
		"price": 100,
	}

	result, err := exec.Execute(context.Background(), config, input)
	require.NoError(t, err)
	assert.Equal(t, 200, result)
}

func TestTransformExecutor_Expression_ComplexCalculation(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":       "expression",
		"expression": "(input.price * input.quantity) * (1 + input.taxRate)",
	}

	input := map[string]interface{}{
		"price":    50.0,
		"quantity": 3.0,
		"taxRate":  0.2, // 20% tax
	}

	result, err := exec.Execute(context.Background(), config, input)
	require.NoError(t, err)
	assert.Equal(t, 180.0, result) // (50 * 3) * 1.2 = 180
}

func TestTransformExecutor_Expression_StringManipulation(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":       "expression",
		"expression": `input.firstName + " " + input.lastName`,
	}

	input := map[string]interface{}{
		"firstName": "John",
		"lastName":  "Doe",
	}

	result, err := exec.Execute(context.Background(), config, input)
	require.NoError(t, err)
	assert.Equal(t, "John Doe", result)
}

func TestTransformExecutor_Expression_Conditional(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":       "expression",
		"expression": `input.age >= 18 ? "adult" : "minor"`,
	}

	t.Run("Adult", func(t *testing.T) {
		input := map[string]interface{}{"age": 25}
		result, err := exec.Execute(context.Background(), config, input)
		require.NoError(t, err)
		assert.Equal(t, "adult", result)
	})

	t.Run("Minor", func(t *testing.T) {
		input := map[string]interface{}{"age": 15}
		result, err := exec.Execute(context.Background(), config, input)
		require.NoError(t, err)
		assert.Equal(t, "minor", result)
	})
}

func TestTransformExecutor_Expression_WithTemplates(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":       "expression",
		"expression": "input.price * {{env.multiplier}}",
	}

	// Create template engine
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars = map[string]interface{}{
		"multiplier": "2",
	}

	engine := template.NewEngine(varCtx, template.TemplateOptions{})
	wrappedExec := executor.NewTemplateExecutorWrapper(exec, engine)

	input := map[string]interface{}{
		"price": 100,
	}

	result, err := wrappedExec.Execute(context.Background(), config, input)
	require.NoError(t, err)
	assert.Equal(t, 200, result)
}

func TestTransformExecutor_JQ_Simple(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":   "jq",
		"filter": ".name",
	}

	input := map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
	}

	result, err := exec.Execute(context.Background(), config, input)
	require.NoError(t, err)
	assert.Equal(t, "John", result)
}

func TestTransformExecutor_JQ_NestedAccess(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":   "jq",
		"filter": ".user.profile.email",
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"email": "user@example.com",
				"name":  "User Name",
			},
		},
	}

	result, err := exec.Execute(context.Background(), config, input)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", result)
}

func TestTransformExecutor_JQ_ArrayFilter(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":   "jq",
		"filter": ".items[] | select(.price > 50)",
	}

	input := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "Item1", "price": 30},
			map[string]interface{}{"name": "Item2", "price": 75},
			map[string]interface{}{"name": "Item3", "price": 100},
		},
	}

	result, err := exec.Execute(context.Background(), config, input)
	require.NoError(t, err)

	// First matching item
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Item2", resultMap["name"])
	// Price can be either int or json.Number depending on parsing
	price := resultMap["price"]
	if priceNum, ok := price.(json.Number); ok {
		assert.Equal(t, "75", priceNum.String())
	} else {
		assert.Equal(t, 75, price)
	}
}

func TestTransformExecutor_JQ_Map(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":   "jq",
		"filter": "[.items[] | .name]",
	}

	input := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "Item1", "price": 30},
			map[string]interface{}{"name": "Item2", "price": 75},
			map[string]interface{}{"name": "Item3", "price": 100},
		},
	}

	result, err := exec.Execute(context.Background(), config, input)
	require.NoError(t, err)

	resultSlice, ok := result.([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"Item1", "Item2", "Item3"}, resultSlice)
}

func TestTransformExecutor_JQ_Construction(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":   "jq",
		"filter": `{fullName: (.firstName + " " + .lastName), contact: .email}`,
	}

	input := map[string]interface{}{
		"firstName": "John",
		"lastName":  "Doe",
		"email":     "john.doe@example.com",
	}

	result, err := exec.Execute(context.Background(), config, input)
	require.NoError(t, err)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "John Doe", resultMap["fullName"])
	assert.Equal(t, "john.doe@example.com", resultMap["contact"])
}

func TestTransformExecutor_JQ_WithTemplates(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":   "jq",
		"filter": ".items[] | select(.category == \"{{env.targetCategory}}\")",
	}

	// Create template engine
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars = map[string]interface{}{
		"targetCategory": "electronics",
	}

	engine := template.NewEngine(varCtx, template.TemplateOptions{})
	wrappedExec := executor.NewTemplateExecutorWrapper(exec, engine)

	input := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "Phone", "category": "electronics"},
			map[string]interface{}{"name": "Shirt", "category": "clothing"},
			map[string]interface{}{"name": "Laptop", "category": "electronics"},
		},
	}

	result, err := wrappedExec.Execute(context.Background(), config, input)
	require.NoError(t, err)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Phone", resultMap["name"])
	assert.Equal(t, "electronics", resultMap["category"])
}

func TestTransformExecutor_JQ_WithStringInput(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":   "jq",
		"filter": ".user.name",
	}

	// JSON string as input
	inputJSON := `{"user": {"name": "Alice", "age": 30}}`

	result, err := exec.Execute(context.Background(), config, inputJSON)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result)
}

func TestTransformExecutor_CompleteWorkflow(t *testing.T) {
	// Simulate a complete workflow with multiple transform nodes
	exec := NewTransformExecutor()

	// Setup template context
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars = map[string]interface{}{
		"baseUrl":     "https://api.example.com",
		"apiVersion":  "v1",
		"discountPct": "10",
	}
	varCtx.ExecutionVars = map[string]interface{}{
		"discountPct": "15", // Override workflow variable
	}

	engine := template.NewEngine(varCtx, template.TemplateOptions{})
	wrappedExec := executor.NewTemplateExecutorWrapper(exec, engine)

	// Node 1: Use expression to calculate total
	config1 := map[string]interface{}{
		"type":       "expression",
		"expression": "input.price * input.quantity",
	}

	input1 := map[string]interface{}{
		"price":    100,
		"quantity": 5,
	}

	result1, err := wrappedExec.Execute(context.Background(), config1, input1)
	require.NoError(t, err)
	assert.Equal(t, 500, result1)

	// Node 2: Use jq to transform with template substitution
	config2 := map[string]interface{}{
		"type":   "jq",
		"filter": `{total: ., discount: {{env.discountPct}}, finalPrice: (. * (100 - {{env.discountPct}}) / 100)}`,
	}

	result2, err := wrappedExec.Execute(context.Background(), config2, result1)
	require.NoError(t, err)

	resultMap, ok := result2.(map[string]interface{})
	require.True(t, ok)
	// Values can be int or json.Number
	total := resultMap["total"]
	if totalNum, ok := total.(json.Number); ok {
		assert.Equal(t, "500", totalNum.String())
	} else {
		assert.Equal(t, 500, total)
	}
	discount := resultMap["discount"]
	if discountNum, ok := discount.(json.Number); ok {
		assert.Equal(t, "15", discountNum.String())
	} else {
		assert.Equal(t, 15, discount)
	}
	finalPrice := resultMap["finalPrice"]
	if finalPriceNum, ok := finalPrice.(json.Number); ok {
		assert.Equal(t, "425", finalPriceNum.String())
	} else {
		assert.Equal(t, 425, finalPrice)
	}
}

func TestTransformExecutor_StrictMode(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":       "expression",
		"expression": "input.price * {{env.multiplier}}",
	}

	// Strict mode enabled - missing variable should fail
	varCtx := template.NewVariableContext()
	engine := template.NewEngine(varCtx, template.TemplateOptions{
		StrictMode: true,
	})
	wrappedExec := executor.NewTemplateExecutorWrapper(exec, engine)

	input := map[string]interface{}{
		"price": 100,
	}

	_, err := wrappedExec.Execute(context.Background(), config, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "variable not found")
}

func TestTransformExecutor_Validation(t *testing.T) {
	exec := NewTransformExecutor()

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid passthrough",
			config: map[string]interface{}{
				"type": "passthrough",
			},
			wantErr: false,
		},
		{
			name: "Valid template",
			config: map[string]interface{}{
				"type":     "template",
				"template": "Hello {{env.name}}",
			},
			wantErr: false,
		},
		{
			name: "Template without template field",
			config: map[string]interface{}{
				"type": "template",
			},
			wantErr: true,
			errMsg:  "template is required",
		},
		{
			name: "Valid expression",
			config: map[string]interface{}{
				"type":       "expression",
				"expression": "input.x + input.y",
			},
			wantErr: false,
		},
		{
			name: "Expression without expression field",
			config: map[string]interface{}{
				"type": "expression",
			},
			wantErr: true,
			errMsg:  "expression is required",
		},
		{
			name: "Valid jq",
			config: map[string]interface{}{
				"type":   "jq",
				"filter": ".name",
			},
			wantErr: false,
		},
		{
			name: "JQ without filter field",
			config: map[string]interface{}{
				"type": "jq",
			},
			wantErr: true,
			errMsg:  "filter is required",
		},
		{
			name: "Invalid type",
			config: map[string]interface{}{
				"type": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid transformation type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exec.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransformExecutor_Expression_ErrorHandling(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":       "expression",
		"expression": "input.invalid.nested.access",
	}

	input := map[string]interface{}{
		"valid": "data",
	}

	_, err := exec.Execute(context.Background(), config, input)
	assert.Error(t, err)
	// Error can occur during compile or execute
	assert.True(t,
		strings.Contains(err.Error(), "failed to compile expression") ||
			strings.Contains(err.Error(), "failed to execute expression"),
		"expected compilation or execution error, got: %v", err)
}

func TestTransformExecutor_JQ_ErrorHandling(t *testing.T) {
	exec := NewTransformExecutor()

	config := map[string]interface{}{
		"type":   "jq",
		"filter": "invalid syntax {{",
	}

	input := map[string]interface{}{
		"name": "test",
	}

	_, err := exec.Execute(context.Background(), config, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse jq filter")
}
