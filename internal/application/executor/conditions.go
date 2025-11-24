package executor

import (
	"fmt"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/smilemakc/mbflow/internal/domain"
)

// ConditionEvaluator provides centralized condition evaluation with caching
type ConditionEvaluator struct {
	mu sync.RWMutex

	// Cache for compiled expressions
	compiledCache map[string]*vm.Program

	// Cache for evaluation results (per execution)
	resultCache map[string]bool

	// Enable/disable caching
	enableCache bool

	// Debug mode for detailed logging
	debug bool
}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator(enableCache bool) *ConditionEvaluator {
	return &ConditionEvaluator{
		compiledCache: make(map[string]*vm.Program),
		resultCache:   make(map[string]bool),
		enableCache:   enableCache,
		debug:         false,
	}
}

// SetDebug enables or disables debug logging
func (ce *ConditionEvaluator) SetDebug(debug bool) {
	ce.debug = debug
}

// ClearResultCache clears the result cache (should be called per execution)
func (ce *ConditionEvaluator) ClearResultCache() {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.resultCache = make(map[string]bool)
}

// Evaluate evaluates a condition expression against variables
func (ce *ConditionEvaluator) Evaluate(condition string, variables map[string]any) (bool, error) {
	if condition == "" {
		return false, domain.NewDomainError(
			domain.ErrCodeInvalidInput,
			"condition cannot be empty",
			nil,
		)
	}

	// Check result cache
	if ce.enableCache {
		cacheKey := ce.makeResultCacheKey(condition, variables)
		ce.mu.RLock()
		result, cached := ce.resultCache[cacheKey]
		ce.mu.RUnlock()

		if cached {
			return result, nil
		}
	}

	// Normalize string values
	normalizedVars := normalizeVariables(variables)

	// Get or compile program
	program, err := ce.getCompiledProgram(condition)
	if err != nil {
		return false, err
	}

	// Execute program
	result, err := expr.Run(program, normalizedVars)
	if err != nil {
		return ce.handleEvaluationError(condition, normalizedVars, err)
	}

	// Convert to boolean
	resultBool, ok := result.(bool)
	if !ok {
		return false, domain.NewDomainError(
			domain.ErrCodeInvalidType,
			fmt.Sprintf("condition '%s' did not return boolean, got %T", condition, result),
			nil,
		)
	}

	// Cache result
	if ce.enableCache {
		cacheKey := ce.makeResultCacheKey(condition, variables)
		ce.mu.Lock()
		ce.resultCache[cacheKey] = resultBool
		ce.mu.Unlock()
	}

	return resultBool, nil
}

// EvaluateEdge evaluates a conditional edge
func (ce *ConditionEvaluator) EvaluateEdge(
	edge domain.Edge,
	variables *domain.VariableSet,
) (bool, error) {
	if edge.Type() != domain.EdgeTypeConditional {
		// Non-conditional edges are always active
		return true, nil
	}

	// Get condition from edge config
	config := edge.Config()
	conditionRaw, ok := config["condition"]
	if !ok {
		return false, domain.NewDomainError(
			domain.ErrCodeValidationFailed,
			fmt.Sprintf("conditional edge %s has no condition", edge.ID()),
			nil,
		)
	}

	condition, ok := conditionRaw.(string)
	if !ok {
		return false, domain.NewDomainError(
			domain.ErrCodeInvalidType,
			fmt.Sprintf("condition for edge %s is not a string", edge.ID()),
			nil,
		)
	}

	// Evaluate condition
	return ce.Evaluate(condition, variables.All())
}

// getCompiledProgram gets a compiled program from cache or compiles it
func (ce *ConditionEvaluator) getCompiledProgram(condition string) (*vm.Program, error) {
	// Check cache
	ce.mu.RLock()
	program, cached := ce.compiledCache[condition]
	ce.mu.RUnlock()

	if cached {
		return program, nil
	}

	// Compile with environment that allows map keys as variables
	envType := map[string]interface{}{}
	compiledProgram, err := expr.Compile(condition, expr.Env(envType), expr.AsBool())
	if err != nil {
		// Try without Env for more flexibility
		compiledProgram, err = expr.Compile(condition, expr.AsBool())
		if err != nil {
			return nil, domain.NewDomainError(
				domain.ErrCodeInvalidInput,
				fmt.Sprintf("failed to compile condition '%s'", condition),
				err,
			)
		}
	}

	// Cache compiled program
	ce.mu.Lock()
	ce.compiledCache[condition] = compiledProgram
	ce.mu.Unlock()

	return compiledProgram, nil
}

// handleEvaluationError handles errors during expression evaluation
func (ce *ConditionEvaluator) handleEvaluationError(
	condition string,
	variables map[string]any,
	err error,
) (bool, error) {
	errMsg := err.Error()

	// Check for missing/undefined variables
	if ce.isVariableNotFoundError(errMsg) {
		// Variable doesn't exist yet - condition is false (graceful handling)
		if ce.debug {
			fmt.Printf("[ConditionEvaluator] Variable not yet available for condition '%s': %v\n", condition, err)
		}
		return false, nil
	}

	// For other errors, return error with context
	varInfo := ce.formatVariablesForError(variables)
	return false, domain.NewDomainError(
		domain.ErrCodeInvalidInput,
		fmt.Sprintf("failed to evaluate condition '%s'%s", condition, varInfo),
		err,
	)
}

// isVariableNotFoundError checks if error is due to missing variable
func (ce *ConditionEvaluator) isVariableNotFoundError(errMsg string) bool {
	patterns := []string{
		"cannot fetch",
		"undefined",
		"unknown name",
		"nil pointer",
		"not found",
	}

	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(errMsg), pattern) {
			return true
		}
	}

	return false
}

// formatVariablesForError creates a formatted string of variables for error messages
func (ce *ConditionEvaluator) formatVariablesForError(variables map[string]any) string {
	if len(variables) == 0 {
		return " (no variables available)"
	}

	var varInfo []string
	for k, v := range variables {
		// Only include simple string values for readability
		if strVal, ok := v.(string); ok && len(strVal) < 100 {
			varInfo = append(varInfo, fmt.Sprintf("%s=%q", k, strVal))
		} else {
			varInfo = append(varInfo, fmt.Sprintf("%s=<%T>", k, v))
		}

		// Limit number of variables in error message
		if len(varInfo) >= 10 {
			varInfo = append(varInfo, "...")
			break
		}
	}

	if len(varInfo) > 0 {
		return fmt.Sprintf(" with variables [%s]", strings.Join(varInfo, ", "))
	}

	return ""
}

// makeResultCacheKey creates a cache key for result caching
func (ce *ConditionEvaluator) makeResultCacheKey(condition string, variables map[string]any) string {
	// Simple hash: condition + sorted variable names and values
	// For production, consider using a proper hash function
	var parts []string
	parts = append(parts, condition)

	// Add variable values that affect the condition
	// This is simplified - a full implementation would extract variables used in the condition
	for k, v := range variables {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	return strings.Join(parts, "|")
}

// normalizeVariables normalizes variable values for consistent evaluation
func normalizeVariables(variables map[string]any) map[string]any {
	normalized := make(map[string]any, len(variables))

	for k, v := range variables {
		normalized[k] = normalizeValue(v)
	}

	return normalized
}

// normalizeValue normalizes a single value
func normalizeValue(value any) any {
	switch v := value.(type) {
	case string:
		// Trim whitespace from strings
		return strings.TrimSpace(v)

	case map[string]any:
		// Recursively normalize maps
		normalized := make(map[string]any, len(v))
		for k, val := range v {
			normalized[k] = normalizeValue(val)
		}
		return normalized

	case []any:
		// Recursively normalize slices
		normalized := make([]any, len(v))
		for i, val := range v {
			normalized[i] = normalizeValue(val)
		}
		return normalized

	default:
		return v
	}
}

// BatchEvaluate evaluates multiple conditions at once (useful for optimization)
func (ce *ConditionEvaluator) BatchEvaluate(
	conditions map[string]string, // key -> condition
	variables map[string]any,
) (map[string]bool, error) {
	results := make(map[string]bool, len(conditions))

	for key, condition := range conditions {
		result, err := ce.Evaluate(condition, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate condition for key '%s': %w", key, err)
		}
		results[key] = result
	}

	return results, nil
}

// GetCacheStats returns cache statistics
func (ce *ConditionEvaluator) GetCacheStats() map[string]int {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	return map[string]int{
		"compiled_cache_size": len(ce.compiledCache),
		"result_cache_size":   len(ce.resultCache),
	}
}
