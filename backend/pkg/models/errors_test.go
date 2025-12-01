package models

import (
	"errors"
	"testing"
)

func TestWorkflowError(t *testing.T) {
	baseErr := errors.New("something went wrong")
	wfErr := &WorkflowError{
		WorkflowID: "wf-123",
		Operation:  "create",
		Err:        baseErr,
	}

	// Test Error() method
	expectedMsg := "workflow wf-123 create: something went wrong"
	if wfErr.Error() != expectedMsg {
		t.Errorf("Error() = %s, want %s", wfErr.Error(), expectedMsg)
	}

	// Test Unwrap() method
	if unwrapped := wfErr.Unwrap(); unwrapped != baseErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
	}

	// Test errors.Is()
	if !errors.Is(wfErr, baseErr) {
		t.Error("errors.Is() should return true for wrapped error")
	}
}

func TestExecutionError(t *testing.T) {
	baseErr := errors.New("execution failed")

	tests := []struct {
		name        string
		execErr     *ExecutionError
		expectedMsg string
	}{
		{
			name: "with node ID",
			execErr: &ExecutionError{
				ExecutionID: "exec-123",
				NodeID:      "node-456",
				Err:         baseErr,
			},
			expectedMsg: "execution exec-123 node node-456: execution failed",
		},
		{
			name: "without node ID",
			execErr: &ExecutionError{
				ExecutionID: "exec-123",
				Err:         baseErr,
			},
			expectedMsg: "execution exec-123: execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Error() method
			if tt.execErr.Error() != tt.expectedMsg {
				t.Errorf("Error() = %s, want %s", tt.execErr.Error(), tt.expectedMsg)
			}

			// Test Unwrap() method
			if unwrapped := tt.execErr.Unwrap(); unwrapped != baseErr {
				t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
			}

			// Test errors.Is()
			if !errors.Is(tt.execErr, baseErr) {
				t.Error("errors.Is() should return true for wrapped error")
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	valErr := &ValidationError{
		Field:   "name",
		Message: "name is required",
	}

	expectedMsg := "name: name is required"
	if valErr.Error() != expectedMsg {
		t.Errorf("Error() = %s, want %s", valErr.Error(), expectedMsg)
	}
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		errors      ValidationErrors
		expectedMsg string
	}{
		{
			name: "single error",
			errors: ValidationErrors{
				{Field: "name", Message: "name is required"},
			},
			expectedMsg: "name: name is required",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				{Field: "name", Message: "name is required"},
				{Field: "type", Message: "type is invalid"},
			},
			expectedMsg: "name: name is required", // Should return first error
		},
		{
			name:        "no errors",
			errors:      ValidationErrors{},
			expectedMsg: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.errors.Error() != tt.expectedMsg {
				t.Errorf("Error() = %s, want %s", tt.errors.Error(), tt.expectedMsg)
			}
		})
	}
}

func TestCommonErrors(t *testing.T) {
	// Test that common errors are defined
	commonErrors := []error{
		ErrClientClosed,
		ErrInvalidWorkflowID,
		ErrWorkflowNotFound,
		ErrWorkflowExists,
		ErrInvalidWorkflow,
		ErrCyclicDependency,
		ErrOrphanedNodes,
		ErrInvalidNodeType,
		ErrNodeNotFound,
		ErrEdgeNotFound,
		ErrInvalidEdge,
		ErrInvalidExecutionID,
		ErrExecutionNotFound,
		ErrExecutionFailed,
		ErrExecutionCancelled,
		ErrExecutionTimeout,
		ErrNodeExecutionFailed,
		ErrInvalidInput,
		ErrInvalidOutput,
		ErrInvalidTriggerID,
		ErrTriggerNotFound,
		ErrInvalidTriggerType,
		ErrInvalidTriggerConfig,
		ErrTriggerDisabled,
		ErrExecutorNotFound,
		ErrExecutorFailed,
		ErrInvalidConfig,
		ErrUnauthorized,
		ErrForbidden,
		ErrValidationFailed,
		ErrRequired,
	}

	for _, err := range commonErrors {
		if err == nil {
			t.Error("common error is nil")
		}
		if err.Error() == "" {
			t.Error("common error has empty message")
		}
	}
}

func TestErrorWrapping(t *testing.T) {
	baseErr := ErrWorkflowNotFound

	wfErr := &WorkflowError{
		WorkflowID: "wf-123",
		Operation:  "get",
		Err:        baseErr,
	}

	// Test errors.Is() with wrapped errors
	if !errors.Is(wfErr, ErrWorkflowNotFound) {
		t.Error("errors.Is() should work with WorkflowError")
	}

	execErr := &ExecutionError{
		ExecutionID: "exec-123",
		Err:         ErrExecutionFailed,
	}

	if !errors.Is(execErr, ErrExecutionFailed) {
		t.Error("errors.Is() should work with ExecutionError")
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"client closed", ErrClientClosed, "client is closed"},
		{"workflow not found", ErrWorkflowNotFound, "workflow not found"},
		{"node not found", ErrNodeNotFound, "node not found"},
		{"edge not found", ErrEdgeNotFound, "edge not found"},
		{"execution failed", ErrExecutionFailed, "execution failed"},
		{"executor not found", ErrExecutorNotFound, "executor not found"},
		{"validation failed", ErrValidationFailed, "validation failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Error message = %s, want %s", tt.err.Error(), tt.expected)
			}
		})
	}
}
