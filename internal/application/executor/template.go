package executor

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
)

// TemplateProcessor provides centralized template processing for node configurations
type TemplateProcessor struct {
	evaluator *ConditionEvaluator // Reuse existing expr-lang infrastructure
	mu        sync.RWMutex
	debug     bool

	// Compiled regex patterns
	simpleVarPattern *regexp.Regexp // {{variable}}
	exprPattern      *regexp.Regexp // ${expression}
}

// TemplateConfig holds configuration for template processing
type TemplateConfig struct {
	StrictMode bool     // true = fail on missing vars, false = leave placeholder
	Fields     []string // Specific fields to template (empty = all strings)
}

// NewTemplateProcessor creates a new template processor
func NewTemplateProcessor(evaluator *ConditionEvaluator) *TemplateProcessor {
	return &TemplateProcessor{
		evaluator:        evaluator,
		debug:            false,
		simpleVarPattern: regexp.MustCompile(`\{\{([^}]+)\}\}`), // {{variable}}
		exprPattern:      regexp.MustCompile(`\$\{([^}]+)\}`),   // ${expression}
	}
}

// SetDebug enables or disables debug logging
func (tp *TemplateProcessor) SetDebug(debug bool) {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.debug = debug
}

// Process processes any value type recursively
func (tp *TemplateProcessor) Process(
	value any,
	variables map[string]any,
	config TemplateConfig,
) (any, error) {
	switch v := value.(type) {
	case string:
		return tp.processString(v, variables, config)
	case map[string]any:
		return tp.processMap(v, variables, config)
	case []any:
		return tp.processSlice(v, variables, config)
	default:
		// Non-templatable types return as-is
		return value, nil
	}
}

// ProcessMap processes a map recursively, optionally filtering by fields
func (tp *TemplateProcessor) ProcessMap(
	m map[string]any,
	variables map[string]any,
	config TemplateConfig,
) (map[string]any, error) {
	result := make(map[string]any, len(m))

	for key, value := range m {
		// Skip if fields specified and this field is not in the list
		if len(config.Fields) > 0 && !containsString(config.Fields, key) {
			result[key] = value
			continue
		}

		processed, err := tp.Process(value, variables, config)
		if err != nil {
			return nil, fmt.Errorf("failed to process field '%s': %w", key, err)
		}
		result[key] = processed
	}

	return result, nil
}

// processString handles template processing for string values
func (tp *TemplateProcessor) processString(
	s string,
	vars map[string]any,
	cfg TemplateConfig,
) (string, error) {
	// Early termination if no template patterns found
	if !strings.Contains(s, "{{") && !strings.Contains(s, "${") {
		return s, nil
	}

	result := s

	// Step 1: Process ${expression} patterns FIRST (for composition support)
	exprMatches := tp.exprPattern.FindAllStringSubmatch(result, -1)
	for _, match := range exprMatches {
		if len(match) < 2 {
			continue
		}

		placeholder := match[0] // ${expression}
		expression := match[1]  // expression
		value, err := tp.evaluateExpression(expression, vars)

		if err != nil {
			if cfg.StrictMode {
				return "", fmt.Errorf("expression '${%s}' failed: %w", expression, err)
			}
			// Lenient mode: leave placeholder unchanged
			if tp.debug {
				fmt.Printf("[TemplateProcessor] Expression evaluation failed (lenient mode): ${%s}: %v\n", expression, err)
			}
			continue
		}

		// Replace placeholder with evaluated value
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}

	// Step 2: Process {{variable}} patterns SECOND
	varMatches := tp.simpleVarPattern.FindAllStringSubmatch(result, -1)
	for _, match := range varMatches {
		if len(match) < 2 {
			continue
		}

		placeholder := match[0]                // {{variable}}
		varPath := strings.TrimSpace(match[1]) // variable
		value := getNestedValue(vars, varPath)

		if value == nil {
			if cfg.StrictMode {
				return "", fmt.Errorf("variable '{{%s}}' not found", varPath)
			}
			// Lenient mode: leave placeholder unchanged
			if tp.debug {
				fmt.Printf("[TemplateProcessor] Variable not found (lenient mode): {{%s}}\n", varPath)
			}
			continue
		}

		// Replace placeholder with variable value
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}

	return result, nil
}

// processMap handles template processing for map values
func (tp *TemplateProcessor) processMap(
	m map[string]any,
	vars map[string]any,
	cfg TemplateConfig,
) (map[string]any, error) {
	result := make(map[string]any, len(m))

	for key, value := range m {
		processed, err := tp.Process(value, vars, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to process map key '%s': %w", key, err)
		}
		result[key] = processed
	}

	return result, nil
}

// processSlice handles template processing for slice values
func (tp *TemplateProcessor) processSlice(
	slice []any,
	vars map[string]any,
	cfg TemplateConfig,
) ([]any, error) {
	result := make([]any, len(slice))

	for i, value := range slice {
		processed, err := tp.Process(value, vars, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to process slice index %d: %w", i, err)
		}
		result[i] = processed
	}

	return result, nil
}

// evaluateExpression evaluates an expr-lang expression
func (tp *TemplateProcessor) evaluateExpression(
	expression string,
	vars map[string]any,
) (any, error) {
	// Normalize variables (reuse existing pattern from ConditionEvaluator)
	normalizedVars := normalizeVariables(vars)

	// Compile and execute expression
	program, err := expr.Compile(expression, expr.Env(normalizedVars), expr.AsAny())
	if err != nil {
		// Try without Env for more flexibility
		program, err = expr.Compile(expression, expr.AsAny())
		if err != nil {
			return nil, fmt.Errorf("failed to compile expression: %w", err)
		}
	}

	result, err := expr.Run(program, normalizedVars)
	if err != nil {
		return nil, fmt.Errorf("failed to execute expression: %w", err)
	}

	return result, nil
}

// containsString checks if a string slice contains a value
func containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
