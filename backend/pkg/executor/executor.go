// Package executor provides the executor interface and registry for node execution.
//
// Executors are responsible for executing individual nodes in a workflow.
// Each node type has a corresponding executor that implements the Executor interface.
//
// Built-in executors include:
//   - HTTP: Makes HTTP requests (GET, POST, PUT, DELETE)
//   - LLM: Integrates with LLM providers (OpenAI, Anthropic)
//   - Transform: Transforms data using expressions
//   - Conditional: Evaluates conditions and routes execution
//   - Merge: Combines outputs from multiple nodes
//
// Custom executors can be registered at runtime using the Manager.
package executor

import (
	"context"
	"fmt"
)

// Executor is the interface that all node executors must implement.
// It defines the contract for executing a node and validating its configuration.
type Executor interface {
	// Execute executes the node with the given configuration and input.
	// It returns the output data or an error if execution fails.
	Execute(ctx context.Context, config map[string]any, input any) (any, error)

	// Validate validates the node configuration.
	// It returns an error if the configuration is invalid.
	Validate(config map[string]any) error
}

// Manager manages the registration and retrieval of executors.
// It provides a central registry for all executor types.
type Manager interface {
	// Register registers an executor for a specific node type.
	// If an executor for the type already exists, it will be replaced.
	Register(nodeType string, executor Executor) error

	// Get retrieves an executor by node type.
	// Returns an error if the executor is not found.
	Get(nodeType string) (Executor, error)

	// Has checks if an executor is registered for the given node type.
	Has(nodeType string) bool

	// List returns a list of all registered executor types.
	List() []string

	// Unregister removes an executor for a specific node type.
	Unregister(nodeType string) error
}

// ExecutorFunc is an adapter to allow the use of ordinary functions as Executors.
// If f is a function with the appropriate signature, ExecutorFunc(f) is an Executor
// that calls f.
type ExecutorFunc struct {
	ExecuteFn  func(ctx context.Context, config map[string]any, input any) (any, error)
	ValidateFn func(config map[string]any) error
}

// Execute calls the ExecuteFn function.
func (f *ExecutorFunc) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	return f.ExecuteFn(ctx, config, input)
}

// Validate calls the ValidateFn function.
func (f *ExecutorFunc) Validate(config map[string]any) error {
	if f.ValidateFn == nil {
		return nil
	}
	return f.ValidateFn(config)
}

// ExecutionContext provides additional context for executor execution.
type ExecutionContext struct {
	ExecutionID string
	NodeID      string
	WorkflowID  string
	Metadata    map[string]any
}

// NewExecutorFunc creates a new ExecutorFunc with the given functions.
func NewExecutorFunc(
	executeFn func(ctx context.Context, config map[string]any, input any) (any, error),
	validateFn func(config map[string]any) error,
) Executor {
	return &ExecutorFunc{
		ExecuteFn:  executeFn,
		ValidateFn: validateFn,
	}
}

// BaseExecutor provides common functionality for executors.
type BaseExecutor struct {
	NodeType string
}

// NewBaseExecutor creates a new BaseExecutor.
func NewBaseExecutor(nodeType string) *BaseExecutor {
	return &BaseExecutor{
		NodeType: nodeType,
	}
}

// ValidateRequired validates that required fields are present in the configuration.
func (b *BaseExecutor) ValidateRequired(config map[string]any, fields ...string) error {
	for _, field := range fields {
		if _, ok := config[field]; !ok {
			return fmt.Errorf("required field missing: %s", field)
		}
	}
	return nil
}

// GetString safely retrieves a string value from config.
func (b *BaseExecutor) GetString(config map[string]any, key string) (string, error) {
	val, ok := config[key]
	if !ok {
		return "", fmt.Errorf("field not found: %s", key)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("field %s is not a string", key)
	}

	return str, nil
}

// GetStringDefault safely retrieves a string value from config with a default.
func (b *BaseExecutor) GetStringDefault(config map[string]any, key, defaultValue string) string {
	val, ok := config[key]
	if !ok {
		return defaultValue
	}

	str, ok := val.(string)
	if !ok {
		return defaultValue
	}

	return str
}

// GetInt safely retrieves an int value from config.
func (b *BaseExecutor) GetInt(config map[string]any, key string) (int, error) {
	val, ok := config[key]
	if !ok {
		return 0, fmt.Errorf("field not found: %s", key)
	}

	// Handle both float64 (from JSON) and int
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("field %s is not a number", key)
	}
}

// GetIntDefault safely retrieves an int value from config with a default.
func (b *BaseExecutor) GetIntDefault(config map[string]any, key string, defaultValue int) int {
	val, ok := config[key]
	if !ok {
		return defaultValue
	}

	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return defaultValue
	}
}

// GetBool safely retrieves a bool value from config.
func (b *BaseExecutor) GetBool(config map[string]any, key string) (bool, error) {
	val, ok := config[key]
	if !ok {
		return false, fmt.Errorf("field not found: %s", key)
	}

	boolVal, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("field %s is not a boolean", key)
	}

	return boolVal, nil
}

// GetBoolDefault safely retrieves a bool value from config with a default.
func (b *BaseExecutor) GetBoolDefault(config map[string]any, key string, defaultValue bool) bool {
	val, ok := config[key]
	if !ok {
		return defaultValue
	}

	boolVal, ok := val.(bool)
	if !ok {
		return defaultValue
	}

	return boolVal
}

// GetMap safely retrieves a map value from config.
func (b *BaseExecutor) GetMap(config map[string]any, key string) (map[string]any, error) {
	val, ok := config[key]
	if !ok {
		return nil, fmt.Errorf("field not found: %s", key)
	}

	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field %s is not a map", key)
	}

	return m, nil
}
