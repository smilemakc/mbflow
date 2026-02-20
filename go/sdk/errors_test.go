package mbflow_test

import (
	"errors"
	"fmt"
	"testing"

	mbflow "github.com/smilemakc/mbflow/go/sdk"
)

func TestAPIErrorIs(t *testing.T) {
	err := &mbflow.APIError{
		StatusCode: 404,
		Code:       "not_found",
		Message:    "workflow not found",
	}
	wrapped := fmt.Errorf("operation failed: %w", err)

	if !errors.Is(wrapped, mbflow.ErrNotFound) {
		t.Error("expected errors.Is(wrapped, ErrNotFound) to be true")
	}

	var apiErr *mbflow.APIError
	if !errors.As(wrapped, &apiErr) {
		t.Fatal("expected errors.As to work")
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestAPIErrorString(t *testing.T) {
	err := &mbflow.APIError{StatusCode: 429, Code: "rate_limit", Message: "too many requests"}
	s := err.Error()
	if s == "" {
		t.Error("Error() returned empty string")
	}
}

func TestAPIErrorUnwrap_AllStatuses(t *testing.T) {
	tests := []struct {
		status   int
		sentinel error
	}{
		{404, mbflow.ErrNotFound},
		{409, mbflow.ErrConflict},
		{401, mbflow.ErrUnauthorized},
		{403, mbflow.ErrForbidden},
		{422, mbflow.ErrValidation},
		{429, mbflow.ErrRateLimit},
		{408, mbflow.ErrTimeout},
		{500, mbflow.ErrInternal},
	}
	for _, tt := range tests {
		err := &mbflow.APIError{StatusCode: tt.status, Code: "test", Message: "test"}
		if !errors.Is(err, tt.sentinel) {
			t.Errorf("status %d: expected Is(%v) to be true", tt.status, tt.sentinel)
		}
	}
}
