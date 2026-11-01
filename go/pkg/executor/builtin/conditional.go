package builtin

import (
	"context"
	"fmt"

	"github.com/expr-lang/expr"

	"github.com/smilemakc/mbflow/go/pkg/executor"
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

// Execute evaluates the condition. The output is the node's input passed
// through (so nodes after the branch still see the data) with the boolean
// outcome under "result", which selects the "true"/"false" edges. A non-map
// input is returned under "input".
func (e *ConditionalExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	conditionType := e.GetStringDefault(config, "condition_type", "expression")

	switch conditionType {
	case "expression":
		exprStr := conditionExpression(config)
		if exprStr == "" {
			return nil, fmt.Errorf("condition is required for expression condition")
		}

		env := map[string]any{
			"input": input,
		}

		program, err := expr.Compile(exprStr, expr.Env(env))
		if err != nil {
			return nil, fmt.Errorf("failed to compile expression: %w", err)
		}

		output, err := expr.Run(program, env)
		if err != nil {
			return nil, fmt.Errorf("failed to execute expression: %w", err)
		}

		result, ok := output.(bool)
		if !ok {
			return nil, fmt.Errorf("expression result is not a boolean: %v", output)
		}
		return conditionalOutput(input, result), nil

	default:
		return nil, fmt.Errorf("unknown condition type: %s", conditionType)
	}
}

// conditionExpression returns the condition, accepting "condition" (used by
// the UI) and "expression" (used by older workflows and examples).
func conditionExpression(config map[string]any) string {
	if s, ok := config["condition"].(string); ok && s != "" {
		return s
	}
	s, _ := config["expression"].(string)
	return s
}

// conditionalOutput passes the input through with the outcome under "result".
func conditionalOutput(input any, result bool) map[string]any {
	out := map[string]any{}
	if m, ok := input.(map[string]any); ok {
		for k, v := range m {
			out[k] = v
		}
	} else if input != nil {
		out["input"] = input
	}
	out["result"] = result
	return out
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
		if conditionExpression(config) == "" {
			return fmt.Errorf("condition is required for expression condition")
		}
	}

	return nil
}
