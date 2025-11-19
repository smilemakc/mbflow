package errors

import (
	"fmt"
)

// ExecutionError represents an error that occurred during workflow execution.
// This is the base error type for all execution-related errors.
type ExecutionError struct {
	// WorkflowID is the ID of the workflow being executed
	WorkflowID string
	// ExecutionID is the ID of the execution instance
	ExecutionID string
	// NodeID is the ID of the node where the error occurred (if applicable)
	NodeID string
	// Message is the error message
	Message string
	// Cause is the underlying error that caused this error
	Cause error
	// Retryable indicates whether this error can be retried
	Retryable bool
}

// Error implements the error interface.
func (e *ExecutionError) Error() string {
	if e.NodeID != "" {
		return fmt.Sprintf("execution error in workflow %s (execution %s) at node %s: %s",
			e.WorkflowID, e.ExecutionID, e.NodeID, e.Message)
	}
	return fmt.Sprintf("execution error in workflow %s (execution %s): %s",
		e.WorkflowID, e.ExecutionID, e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// NodeExecutionError represents an error that occurred during node execution.
type NodeExecutionError struct {
	ExecutionError
	// NodeType is the type of the node that failed
	NodeType string
	// AttemptNumber is the attempt number (for retries)
	AttemptNumber int
}

// Error implements the error interface.
func (e *NodeExecutionError) Error() string {
	return fmt.Sprintf("node execution error [%s] (attempt %d): %s",
		e.NodeType, e.AttemptNumber, e.ExecutionError.Error())
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}

// StateError represents an error related to execution state management.
type StateError struct {
	ExecutionID string
	Message     string
	Cause       error
}

// Error implements the error interface.
func (e *StateError) Error() string {
	return fmt.Sprintf("state error for execution %s: %s", e.ExecutionID, e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *StateError) Unwrap() error {
	return e.Cause
}

// ConfigurationError represents a configuration error.
type ConfigurationError struct {
	Component string
	Message   string
}

// Error implements the error interface.
func (e *ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error in %s: %s", e.Component, e.Message)
}

// NewExecutionError creates a new ExecutionError.
func NewExecutionError(workflowID, executionID, nodeID, message string, cause error, retryable bool) *ExecutionError {
	return &ExecutionError{
		WorkflowID:  workflowID,
		ExecutionID: executionID,
		NodeID:      nodeID,
		Message:     message,
		Cause:       cause,
		Retryable:   retryable,
	}
}

// NewNodeExecutionError creates a new NodeExecutionError.
func NewNodeExecutionError(workflowID, executionID, nodeID, nodeType string, attemptNumber int, message string, cause error, retryable bool) *NodeExecutionError {
	return &NodeExecutionError{
		ExecutionError: ExecutionError{
			WorkflowID:  workflowID,
			ExecutionID: executionID,
			NodeID:      nodeID,
			Message:     message,
			Cause:       cause,
			Retryable:   retryable,
		},
		NodeType:      nodeType,
		AttemptNumber: attemptNumber,
	}
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewStateError creates a new StateError.
func NewStateError(executionID, message string, cause error) *StateError {
	return &StateError{
		ExecutionID: executionID,
		Message:     message,
		Cause:       cause,
	}
}

// NewConfigurationError creates a new ConfigurationError.
func NewConfigurationError(component, message string) *ConfigurationError {
	return &ConfigurationError{
		Component: component,
		Message:   message,
	}
}

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	if execErr, ok := err.(*ExecutionError); ok {
		return execErr.Retryable
	}
	if nodeErr, ok := err.(*NodeExecutionError); ok {
		return nodeErr.Retryable
	}
	return false
}
