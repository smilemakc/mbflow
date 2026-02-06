package engine

// SimpleConditionEvaluator implements ConditionEvaluator with basic string matching.
// Used by standalone executor where expr-lang is not needed.
type SimpleConditionEvaluator struct{}

// NewSimpleConditionEvaluator creates a new SimpleConditionEvaluator.
func NewSimpleConditionEvaluator() *SimpleConditionEvaluator {
	return &SimpleConditionEvaluator{}
}

// Evaluate evaluates a simple condition expression.
// Supports "true", "false", and presence check (output != nil).
func (e *SimpleConditionEvaluator) Evaluate(condition string, nodeOutput interface{}) (bool, error) {
	if condition == "" || condition == "true" {
		return true, nil
	}
	if condition == "false" {
		return false, nil
	}

	// Basic condition support - check if output exists
	if nodeOutput != nil {
		return true, nil
	}

	return false, nil
}
