package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/itchyny/gojq"
	"github.com/smilemakc/mbflow/pkg/executor"
)

// TransformExecutor transforms data using expressions or templates.
type TransformExecutor struct {
	*executor.BaseExecutor
}

// NewTransformExecutor creates a new transform executor.
func NewTransformExecutor() *TransformExecutor {
	return &TransformExecutor{
		BaseExecutor: executor.NewBaseExecutor("transform"),
	}
}

// Execute executes a data transformation.
func (e *TransformExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	// Get transformation type
	transformType := e.GetStringDefault(config, "type", "passthrough")

	switch transformType {
	case "passthrough":
		return input, nil

	case "template":
		// Get template string
		tmpl, err := e.GetString(config, "template")
		if err != nil {
			return nil, err
		}

		// Note: Template resolution is handled by the TemplateExecutorWrapper
		// The template string in config has already been resolved
		// We just return the result
		return tmpl, nil

	case "expression":
		// Get expression string
		exprStr, err := e.GetString(config, "expression")
		if err != nil {
			return nil, err
		}

		// Prepare environment for expression evaluation
		env := map[string]any{
			"input": input,
		}

		// Compile expression with environment
		program, err := expr.Compile(exprStr, expr.Env(env))
		if err != nil {
			return nil, fmt.Errorf("failed to compile expression: %w", err)
		}

		// Execute expression
		output, err := expr.Run(program, env)
		if err != nil {
			return nil, fmt.Errorf("failed to execute expression: %w", err)
		}

		return output, nil

	case "jq":
		// Get jq filter string
		filterStr, err := e.GetString(config, "filter")
		if err != nil {
			return nil, err
		}

		// Parse jq query
		query, err := gojq.Parse(filterStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse jq filter: %w", err)
		}

		// Compile jq query
		code, err := gojq.Compile(query)
		if err != nil {
			return nil, fmt.Errorf("failed to compile jq filter: %w", err)
		}

		// Convert input to any if needed
		var inputData any
		switch v := input.(type) {
		case string:
			// Try to parse as JSON
			if err := json.Unmarshal([]byte(v), &inputData); err != nil {
				// If not JSON, use as-is
				inputData = v
			}
		case []byte:
			// Try to parse as JSON
			if err := json.Unmarshal(v, &inputData); err != nil {
				// If not JSON, convert to string
				inputData = string(v)
			}
		default:
			inputData = v
		}

		// Execute jq filter
		iter := code.Run(inputData)
		v, ok := iter.Next()
		if !ok {
			return nil, fmt.Errorf("jq filter produced no output")
		}

		// Check for errors
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("jq filter execution error: %w", err)
		}

		return v, nil

	default:
		return nil, fmt.Errorf("unknown transformation type: %s", transformType)
	}
}

// Validate validates the transform executor configuration.
func (e *TransformExecutor) Validate(config map[string]any) error {
	transformType := e.GetStringDefault(config, "type", "passthrough")

	validTypes := map[string]bool{
		"passthrough": true,
		"template":    true,
		"expression":  true,
		"jq":          true,
	}

	if !validTypes[transformType] {
		return fmt.Errorf("invalid transformation type: %s", transformType)
	}

	// Type-specific validation
	switch transformType {
	case "template":
		if _, err := e.GetString(config, "template"); err != nil {
			return fmt.Errorf("template is required for template transformation")
		}

	case "expression":
		if _, err := e.GetString(config, "expression"); err != nil {
			return fmt.Errorf("expression is required for expression transformation")
		}

	case "jq":
		if _, err := e.GetString(config, "filter"); err != nil {
			return fmt.Errorf("filter is required for jq transformation")
		}
	}

	return nil
}
