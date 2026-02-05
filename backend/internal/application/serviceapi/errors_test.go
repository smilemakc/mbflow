package serviceapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- OperationError ---

func TestOperationError_Error_ShouldReturnMessage(t *testing.T) {
	opErr := &OperationError{
		Code:       "TEST_ERROR",
		Message:    "something went wrong",
		HTTPStatus: http.StatusBadRequest,
	}

	assert.Equal(t, "something went wrong", opErr.Error())
}

func TestOperationError_Error_ShouldReturnEmptyString_WhenMessageIsEmpty(t *testing.T) {
	opErr := &OperationError{
		Code:       "EMPTY",
		Message:    "",
		HTTPStatus: http.StatusInternalServerError,
	}

	assert.Equal(t, "", opErr.Error())
}

// --- NewValidationError ---

func TestNewValidationError_ShouldReturnBadRequest(t *testing.T) {
	err := NewValidationError("FIELD_REQUIRED", "name is required")

	assert.Equal(t, "FIELD_REQUIRED", err.Code)
	assert.Equal(t, "name is required", err.Message)
	assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
}

func TestNewValidationError_ShouldImplementErrorInterface(t *testing.T) {
	var err error = NewValidationError("CODE", "msg")

	assert.Equal(t, "msg", err.Error())
}

// --- NewNotImplementedError ---

func TestNewNotImplementedError_ShouldReturn501(t *testing.T) {
	err := NewNotImplementedError("feature not available")

	assert.Equal(t, "NOT_IMPLEMENTED", err.Code)
	assert.Equal(t, "feature not available", err.Message)
	assert.Equal(t, http.StatusNotImplemented, err.HTTPStatus)
}

func TestNewNotImplementedError_ShouldImplementErrorInterface(t *testing.T) {
	var err error = NewNotImplementedError("not yet")

	assert.Equal(t, "not yet", err.Error())
}
