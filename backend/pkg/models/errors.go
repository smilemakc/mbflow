// Package models defines the public domain models and error types for MBFlow.
package models

import "errors"

// Common error types for MBFlow SDK.
var (
	// Client errors
	ErrClientClosed = errors.New("client is closed")

	// Workflow errors
	ErrInvalidWorkflowID = errors.New("invalid workflow ID")
	ErrWorkflowNotFound  = errors.New("workflow not found")
	ErrWorkflowExists    = errors.New("workflow already exists")
	ErrInvalidWorkflow   = errors.New("invalid workflow")
	ErrCyclicDependency  = errors.New("cyclic dependency detected")
	ErrOrphanedNodes     = errors.New("orphaned nodes detected")
	ErrInvalidNodeType   = errors.New("invalid node type")
	ErrNodeNotFound      = errors.New("node not found")
	ErrEdgeNotFound      = errors.New("edge not found")
	ErrInvalidEdge       = errors.New("invalid edge")

	// Execution errors
	ErrInvalidExecutionID  = errors.New("invalid execution ID")
	ErrExecutionNotFound   = errors.New("execution not found")
	ErrExecutionFailed     = errors.New("execution failed")
	ErrExecutionCancelled  = errors.New("execution cancelled")
	ErrExecutionTimeout    = errors.New("execution timeout")
	ErrNodeExecutionFailed = errors.New("node execution failed")
	ErrInvalidInput        = errors.New("invalid input")
	ErrInvalidOutput       = errors.New("invalid output")

	// Trigger errors
	ErrInvalidTriggerID     = errors.New("invalid trigger ID")
	ErrTriggerNotFound      = errors.New("trigger not found")
	ErrInvalidTriggerType   = errors.New("invalid trigger type")
	ErrInvalidTriggerConfig = errors.New("invalid trigger configuration")
	ErrTriggerDisabled      = errors.New("trigger is disabled")

	// Executor errors
	ErrExecutorNotFound = errors.New("executor not found")
	ErrExecutorFailed   = errors.New("executor failed")
	ErrInvalidConfig    = errors.New("invalid configuration")

	// Authorization errors
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
	ErrRoleNotFound       = errors.New("role not found")
	ErrInvalidRole        = errors.New("invalid role")
	ErrPermissionDenied   = errors.New("permission denied")

	// Validation errors
	ErrValidationFailed = errors.New("validation failed")
	ErrRequired         = errors.New("required field is missing")

	// Billing and resource errors
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrAccountNotFound       = errors.New("account not found")
	ErrAccountInactive       = errors.New("account is inactive")
	ErrAccountSuspended      = errors.New("account is suspended")
	ErrAccountClosed         = errors.New("account is closed")
	ErrResourceNotFound      = errors.New("resource not found")
	ErrResourceLimitExceeded = errors.New("resource limit exceeded")
	ErrStorageLimitExceeded  = errors.New("storage limit exceeded")
	ErrTransactionNotFound   = errors.New("transaction not found")
	ErrTransactionFailed     = errors.New("transaction failed")
	ErrDuplicateTransaction  = errors.New("duplicate transaction")
	ErrPricingPlanNotFound   = errors.New("pricing plan not found")
	ErrInvalidResourceType   = errors.New("invalid resource type")
	ErrInvalidID             = errors.New("invalid ID format")

	// Rental key errors
	ErrRentalKeyNotFound         = errors.New("rental key not found")
	ErrRentalKeySuspended        = errors.New("rental key is suspended")
	ErrDailyLimitExceeded        = errors.New("daily request limit exceeded")
	ErrMonthlyTokenLimitExceeded = errors.New("monthly token limit exceeded")
	ErrRentalKeyAccessDenied     = errors.New("rental key access denied")
)

// WorkflowError represents an error that occurred during workflow operations.
type WorkflowError struct {
	WorkflowID string
	Operation  string
	Err        error
}

func (e *WorkflowError) Error() string {
	return "workflow " + e.WorkflowID + " " + e.Operation + ": " + e.Err.Error()
}

func (e *WorkflowError) Unwrap() error {
	return e.Err
}

// ExecutionError represents an error that occurred during execution.
type ExecutionError struct {
	ExecutionID string
	NodeID      string
	Err         error
}

func (e *ExecutionError) Error() string {
	msg := "execution " + e.ExecutionID
	if e.NodeID != "" {
		msg += " node " + e.NodeID
	}
	msg += ": " + e.Err.Error()
	return msg
}

func (e *ExecutionError) Unwrap() error {
	return e.Err
}

// ValidationError represents a validation error with details.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ValidationErrors represents multiple validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	return e[0].Error()
}

// AuthError represents an authentication or authorization error.
type AuthError struct {
	UserID string
	Action string
	Err    error
}

func (e *AuthError) Error() string {
	msg := "auth error"
	if e.UserID != "" {
		msg += " for user " + e.UserID
	}
	if e.Action != "" {
		msg += " during " + e.Action
	}
	msg += ": " + e.Err.Error()
	return msg
}

func (e *AuthError) Unwrap() error {
	return e.Err
}
