package serviceapi

import "net/http"

// OperationError represents a business logic error with a code and HTTP status.
type OperationError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *OperationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error (400 Bad Request).
func NewValidationError(code, message string) *OperationError {
	return &OperationError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewNotImplementedError creates a new not-implemented error (501).
func NewNotImplementedError(message string) *OperationError {
	return &OperationError{
		Code:       "NOT_IMPLEMENTED",
		Message:    message,
		HTTPStatus: http.StatusNotImplemented,
	}
}

// NewNotFoundError creates a new not-found error (404 Not Found).
func NewNotFoundError(code, message string) *OperationError {
	return &OperationError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusNotFound,
	}
}

// NewConflictError creates a new conflict error (409 Conflict), for requests
// that are invalid in the resource's current state.
func NewConflictError(code, message string) *OperationError {
	return &OperationError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}
