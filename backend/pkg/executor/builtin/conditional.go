package builtin

import (
	"context"
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/smilemakc/mbflow/pkg/executor"
)

// ConditionalExecutor evaluates conditions and routes execution.
type ConditionalExecutor struct {
	*executor.BaseExecutor
}

// NewConditionalExecutor creates a new conditional executor.
func NewConditionalExecutor() *ConditionalExecutor {
	return &ConditionalExecutor{
		BaseExecutor: executor.NewBaseExecutor("conditional"),
	}
}

// Execute executes the conditional logic.
func (e *ConditionalExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	// Get condition type
	conditionType := e.GetStringDefault(config, "condition_type", "expression")

	switch conditionType {
	case "expression":
		// Get expression string
		exprStr, err := e.GetString(config, "condition")
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

		// Ensure output is boolean
		if val, ok := output.(bool); ok {
			return val, nil
		}

		return nil, fmt.Errorf("expression result is not a boolean: %v", output)

	default:
		return nil, fmt.Errorf("unknown condition type: %s", conditionType)
	}
}

// Validate validates the conditional executor configuration.
func (e *ConditionalExecutor) Validate(config map[string]any) error {
	conditionType := e.GetStringDefault(config, "condition_type", "expression")

	validTypes := map[string]bool{
		"expression": true,
	}

	if !validTypes[conditionType] {
		return fmt.Errorf("invalid condition type: %s", conditionType)
	}

	// Type-specific validation
	switch conditionType {
	case "expression":
		if _, err := e.GetString(config, "expression"); err != nil {
			return fmt.Errorf("expression is required for expression condition")
		}
	}

	return nil
}
