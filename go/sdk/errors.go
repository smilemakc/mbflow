package mbflow

import (
	"errors"
	"fmt"
)

// Sentinel errors for common API error conditions.
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrValidation   = errors.New("validation error")
	ErrRateLimit    = errors.New("rate limit exceeded")
	ErrTimeout      = errors.New("timeout")
	ErrInternal     = errors.New("internal server error")
)

// APIError represents a structured error response from the MBFlow server.
type APIError struct {
	StatusCode int            `json:"status_code"`
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("mbflow: %s (HTTP %d, code=%s)", e.Message, e.StatusCode, e.Code)
}

// Unwrap returns the matching sentinel error based on HTTP status code.
func (e *APIError) Unwrap() error {
	switch e.StatusCode {
	case 404:
		return ErrNotFound
	case 409:
		return ErrConflict
	case 401:
		return ErrUnauthorized
	case 403:
		return ErrForbidden
	case 422:
		return ErrValidation
	case 429:
		return ErrRateLimit
	case 408:
		return ErrTimeout
	default:
		return ErrInternal
	}
}
