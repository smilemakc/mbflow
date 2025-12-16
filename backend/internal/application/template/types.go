// Package template provides a template engine for resolving variables in node configurations.
//
// The template engine supports the following syntax:
//   - {{env.varName}} - Access environment/workflow variables
//   - {{input.fieldName}} - Access output from parent node
//   - {{resource.alias}} - Access workflow resource by alias
//   - {{resource.alias.field}} - Access specific field in resource
//
// Variable resolution follows a specific precedence:
//  1. Execution variables (highest priority, override workflow vars)
//  2. Workflow variables
//  3. Input variables (from parent node output)
//
// The engine supports both strict and non-strict modes:
//   - Strict mode: Missing variables cause execution to fail with an error
//   - Non-strict mode: Missing variables are replaced with empty string or kept as placeholder
package template

import (
	"errors"
	"fmt"
)

// VariableContext holds all variables available for template resolution.
// Variables are resolved with the following precedence:
//  1. ExecutionVars (runtime variables, highest priority)
//  2. WorkflowVars (workflow-level variables)
//  3. InputVars (parent node output, lowest priority)
type VariableContext struct {
	// WorkflowVars contains workflow-level variables from the workflow definition
	WorkflowVars map[string]interface{}

	// ExecutionVars contains runtime variables that override workflow variables
	ExecutionVars map[string]interface{}

	// InputVars contains variables from parent node outputs
	InputVars map[string]interface{}

	// ResourceVars contains workflow resources indexed by alias
	// Each resource is a map with fields: id, type, name, config, etc.
	ResourceVars map[string]interface{}
}

// NewVariableContext creates a new variable context with the given variables.
func NewVariableContext() *VariableContext {
	return &VariableContext{
		WorkflowVars:  make(map[string]interface{}),
		ExecutionVars: make(map[string]interface{}),
		InputVars:     make(map[string]interface{}),
		ResourceVars:  make(map[string]interface{}),
	}
}

// GetEnvVariable retrieves an environment variable with proper precedence.
// Execution variables override workflow variables.
func (c *VariableContext) GetEnvVariable(name string) (interface{}, bool) {
	// Check execution vars first (highest priority)
	if val, ok := c.ExecutionVars[name]; ok {
		return val, true
	}

	// Check workflow vars
	if val, ok := c.WorkflowVars[name]; ok {
		return val, true
	}

	return nil, false
}

// GetInputVariable retrieves an input variable from parent node output.
func (c *VariableContext) GetInputVariable(name string) (interface{}, bool) {
	val, ok := c.InputVars[name]
	return val, ok
}

// GetResourceVariable retrieves a resource by alias.
func (c *VariableContext) GetResourceVariable(alias string) (interface{}, bool) {
	if c.ResourceVars == nil {
		return nil, false
	}
	val, ok := c.ResourceVars[alias]
	return val, ok
}

// TemplateOptions configures template resolution behavior.
type TemplateOptions struct {
	// StrictMode determines error handling for missing variables
	// When true, missing variables cause an error
	// When false, missing variables are handled gracefully
	StrictMode bool

	// PlaceholderOnMissing keeps the original placeholder when variable is missing
	// Only applies when StrictMode is false
	// If false, replaces with empty string instead
	PlaceholderOnMissing bool
}

// DefaultOptions returns the default template options.
func DefaultOptions() TemplateOptions {
	return TemplateOptions{
		StrictMode:           false,
		PlaceholderOnMissing: false,
	}
}

// TemplateError represents an error that occurred during template resolution.
type TemplateError struct {
	Template string
	Variable string
	Path     string
	Err      error
}

// Error implements the error interface.
func (e *TemplateError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("template error in '%s': failed to resolve '{{%s.%s}}': %v",
			e.Template, e.Variable, e.Path, e.Err)
	}
	return fmt.Sprintf("template error in '%s': failed to resolve '{{%s}}': %v",
		e.Template, e.Variable, e.Err)
}

// Unwrap returns the underlying error.
func (e *TemplateError) Unwrap() error {
	return e.Err
}

// Common errors
var (
	ErrVariableNotFound  = errors.New("variable not found")
	ErrInvalidPath       = errors.New("invalid path")
	ErrInvalidTemplate   = errors.New("invalid template syntax")
	ErrTypeNotSupported  = errors.New("type not supported for path traversal")
	ErrArrayIndexInvalid = errors.New("invalid array index")
	ErrArrayOutOfBounds  = errors.New("array index out of bounds")
)
