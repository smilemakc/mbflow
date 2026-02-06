package grpc

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/pkg/models"
)

// --- mapError tests ---

func TestMapError_ShouldReturnNil_WhenErrorIsNil(t *testing.T) {
	result := mapError(nil)

	assert.Nil(t, result)
}

func TestMapError_ShouldReturnInvalidArgument_WhenOperationErrorHTTP400(t *testing.T) {
	opErr := &serviceapi.OperationError{
		Code:       "VALIDATION_FAILED",
		Message:    "name is required",
		HTTPStatus: http.StatusBadRequest,
	}

	result := mapError(opErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "name is required", st.Message())
}

func TestMapError_ShouldReturnUnauthenticated_WhenOperationErrorHTTP401(t *testing.T) {
	opErr := &serviceapi.OperationError{
		Code:       "UNAUTHORIZED",
		Message:    "invalid token",
		HTTPStatus: http.StatusUnauthorized,
	}

	result := mapError(opErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "invalid token", st.Message())
}

func TestMapError_ShouldReturnPermissionDenied_WhenOperationErrorHTTP403(t *testing.T) {
	opErr := &serviceapi.OperationError{
		Code:       "FORBIDDEN",
		Message:    "access denied",
		HTTPStatus: http.StatusForbidden,
	}

	result := mapError(opErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	assert.Equal(t, "access denied", st.Message())
}

func TestMapError_ShouldReturnNotFound_WhenOperationErrorHTTP404(t *testing.T) {
	opErr := &serviceapi.OperationError{
		Code:       "NOT_FOUND",
		Message:    "resource does not exist",
		HTTPStatus: http.StatusNotFound,
	}

	result := mapError(opErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "resource does not exist", st.Message())
}

func TestMapError_ShouldReturnAlreadyExists_WhenOperationErrorHTTP409(t *testing.T) {
	opErr := &serviceapi.OperationError{
		Code:       "CONFLICT",
		Message:    "resource already exists",
		HTTPStatus: http.StatusConflict,
	}

	result := mapError(opErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, "resource already exists", st.Message())
}

func TestMapError_ShouldReturnResourceExhausted_WhenOperationErrorHTTP429(t *testing.T) {
	opErr := &serviceapi.OperationError{
		Code:       "RATE_LIMITED",
		Message:    "too many requests",
		HTTPStatus: http.StatusTooManyRequests,
	}

	result := mapError(opErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
	assert.Equal(t, "too many requests", st.Message())
}

func TestMapError_ShouldReturnUnimplemented_WhenOperationErrorHTTP501(t *testing.T) {
	opErr := serviceapi.NewNotImplementedError("operation not supported")

	result := mapError(opErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, st.Code())
	assert.Equal(t, "operation not supported", st.Message())
}

func TestMapError_ShouldReturnNotFound_WhenWorkflowNotFound(t *testing.T) {
	result := mapError(models.ErrWorkflowNotFound)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "workflow not found", st.Message())
}

func TestMapError_ShouldReturnNotFound_WhenExecutionNotFound(t *testing.T) {
	result := mapError(models.ErrExecutionNotFound)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "execution not found", st.Message())
}

func TestMapError_ShouldReturnNotFound_WhenTriggerNotFound(t *testing.T) {
	result := mapError(models.ErrTriggerNotFound)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "trigger not found", st.Message())
}

func TestMapError_ShouldReturnNotFound_WhenResourceNotFound(t *testing.T) {
	result := mapError(models.ErrResourceNotFound)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "resource not found", st.Message())
}

func TestMapError_ShouldReturnInvalidArgument_WhenInvalidID(t *testing.T) {
	result := mapError(models.ErrInvalidID)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "invalid ID format", st.Message())
}

func TestMapError_ShouldReturnUnauthenticated_WhenUnauthorized(t *testing.T) {
	result := mapError(models.ErrUnauthorized)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "authentication required", st.Message())
}

func TestMapError_ShouldReturnPermissionDenied_WhenForbidden(t *testing.T) {
	result := mapError(models.ErrForbidden)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	assert.Equal(t, "access denied", st.Message())
}

func TestMapError_ShouldReturnAlreadyExists_WhenWorkflowExists(t *testing.T) {
	result := mapError(models.ErrWorkflowExists)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, "workflow already exists", st.Message())
}

func TestMapError_ShouldReturnInvalidArgument_WhenValidationFailed(t *testing.T) {
	result := mapError(models.ErrValidationFailed)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "validation failed", st.Message())
}

func TestMapError_ShouldReturnInternal_WhenUnknownError(t *testing.T) {
	unknownErr := errors.New("something unexpected happened")

	result := mapError(unknownErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal error", st.Message())
}

func TestMapError_ShouldReturnNotFound_WhenWrappedWorkflowNotFound(t *testing.T) {
	wrapped := fmt.Errorf("failed to load: %w", models.ErrWorkflowNotFound)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "workflow not found", st.Message())
}

func TestMapError_ShouldReturnNotFound_WhenWrappedExecutionNotFound(t *testing.T) {
	wrapped := fmt.Errorf("operation failed: %w", models.ErrExecutionNotFound)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "execution not found", st.Message())
}

func TestMapError_ShouldReturnUnauthenticated_WhenWrappedUnauthorized(t *testing.T) {
	wrapped := fmt.Errorf("auth check: %w", models.ErrUnauthorized)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "authentication required", st.Message())
}

func TestMapError_ShouldReturnPermissionDenied_WhenWrappedForbidden(t *testing.T) {
	wrapped := fmt.Errorf("access check: %w", models.ErrForbidden)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	assert.Equal(t, "access denied", st.Message())
}

func TestMapError_ShouldReturnAlreadyExists_WhenWrappedWorkflowExists(t *testing.T) {
	wrapped := fmt.Errorf("create failed: %w", models.ErrWorkflowExists)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, "workflow already exists", st.Message())
}

func TestMapError_ShouldReturnInvalidArgument_WhenWrappedInvalidID(t *testing.T) {
	wrapped := fmt.Errorf("parse error: %w", models.ErrInvalidID)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "invalid ID format", st.Message())
}

func TestMapError_ShouldReturnNotFound_WhenWrappedTriggerNotFound(t *testing.T) {
	wrapped := fmt.Errorf("trigger lookup: %w", models.ErrTriggerNotFound)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "trigger not found", st.Message())
}

func TestMapError_ShouldReturnNotFound_WhenWrappedResourceNotFound(t *testing.T) {
	wrapped := fmt.Errorf("resource query: %w", models.ErrResourceNotFound)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "resource not found", st.Message())
}

func TestMapError_ShouldReturnInvalidArgument_WhenWrappedValidationFailed(t *testing.T) {
	wrapped := fmt.Errorf("input check: %w", models.ErrValidationFailed)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "validation failed", st.Message())
}

func TestMapError_ShouldReturnInternal_WhenWrappedUnknownError(t *testing.T) {
	inner := errors.New("database connection lost")
	wrapped := fmt.Errorf("query failed: %w", inner)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal error", st.Message())
}

func TestMapError_ShouldReturnCorrectCode_WhenWrappedOperationError(t *testing.T) {
	opErr := &serviceapi.OperationError{
		Code:       "BAD_REQUEST",
		Message:    "invalid parameter",
		HTTPStatus: http.StatusBadRequest,
	}
	wrapped := fmt.Errorf("validation layer: %w", opErr)

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "invalid parameter", st.Message())
}

func TestMapError_ShouldReturnInternal_WhenDoubleWrappedUnknownError(t *testing.T) {
	inner := errors.New("disk full")
	wrapped := fmt.Errorf("write: %w", fmt.Errorf("io: %w", inner))

	result := mapError(wrapped)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal error", st.Message())
}

// --- httpStatusToGRPCCode tests ---

func TestHttpStatusToGRPCCode_ShouldReturnOK_WhenHTTP200(t *testing.T) {
	assert.Equal(t, codes.OK, httpStatusToGRPCCode(http.StatusOK))
}

func TestHttpStatusToGRPCCode_ShouldReturnOK_WhenHTTP201(t *testing.T) {
	assert.Equal(t, codes.OK, httpStatusToGRPCCode(http.StatusCreated))
}

func TestHttpStatusToGRPCCode_ShouldReturnOK_WhenHTTP204(t *testing.T) {
	assert.Equal(t, codes.OK, httpStatusToGRPCCode(http.StatusNoContent))
}

func TestHttpStatusToGRPCCode_ShouldReturnOK_WhenHTTP299(t *testing.T) {
	assert.Equal(t, codes.OK, httpStatusToGRPCCode(299))
}

func TestHttpStatusToGRPCCode_ShouldReturnInvalidArgument_WhenHTTP400(t *testing.T) {
	assert.Equal(t, codes.InvalidArgument, httpStatusToGRPCCode(http.StatusBadRequest))
}

func TestHttpStatusToGRPCCode_ShouldReturnUnauthenticated_WhenHTTP401(t *testing.T) {
	assert.Equal(t, codes.Unauthenticated, httpStatusToGRPCCode(http.StatusUnauthorized))
}

func TestHttpStatusToGRPCCode_ShouldReturnPermissionDenied_WhenHTTP403(t *testing.T) {
	assert.Equal(t, codes.PermissionDenied, httpStatusToGRPCCode(http.StatusForbidden))
}

func TestHttpStatusToGRPCCode_ShouldReturnNotFound_WhenHTTP404(t *testing.T) {
	assert.Equal(t, codes.NotFound, httpStatusToGRPCCode(http.StatusNotFound))
}

func TestHttpStatusToGRPCCode_ShouldReturnAlreadyExists_WhenHTTP409(t *testing.T) {
	assert.Equal(t, codes.AlreadyExists, httpStatusToGRPCCode(http.StatusConflict))
}

func TestHttpStatusToGRPCCode_ShouldReturnResourceExhausted_WhenHTTP429(t *testing.T) {
	assert.Equal(t, codes.ResourceExhausted, httpStatusToGRPCCode(http.StatusTooManyRequests))
}

func TestHttpStatusToGRPCCode_ShouldReturnUnimplemented_WhenHTTP501(t *testing.T) {
	assert.Equal(t, codes.Unimplemented, httpStatusToGRPCCode(http.StatusNotImplemented))
}

func TestHttpStatusToGRPCCode_ShouldReturnInternal_WhenHTTP500(t *testing.T) {
	assert.Equal(t, codes.Internal, httpStatusToGRPCCode(http.StatusInternalServerError))
}

func TestHttpStatusToGRPCCode_ShouldReturnInternal_WhenHTTP502(t *testing.T) {
	assert.Equal(t, codes.Internal, httpStatusToGRPCCode(http.StatusBadGateway))
}

func TestHttpStatusToGRPCCode_ShouldReturnInternal_WhenHTTP503(t *testing.T) {
	assert.Equal(t, codes.Internal, httpStatusToGRPCCode(http.StatusServiceUnavailable))
}

func TestHttpStatusToGRPCCode_ShouldReturnInternal_WhenUnknownStatusCode(t *testing.T) {
	assert.Equal(t, codes.Internal, httpStatusToGRPCCode(999))
}

func TestHttpStatusToGRPCCode_ShouldReturnInternal_WhenHTTP100(t *testing.T) {
	assert.Equal(t, codes.Internal, httpStatusToGRPCCode(http.StatusContinue))
}

func TestHttpStatusToGRPCCode_ShouldReturnInternal_WhenHTTP300(t *testing.T) {
	assert.Equal(t, codes.Internal, httpStatusToGRPCCode(http.StatusMultipleChoices))
}

func TestHttpStatusToGRPCCode_ShouldReturnInternal_WhenHTTP422(t *testing.T) {
	assert.Equal(t, codes.Internal, httpStatusToGRPCCode(http.StatusUnprocessableEntity))
}

func TestHttpStatusToGRPCCode_ShouldReturnInternal_WhenHTTP0(t *testing.T) {
	assert.Equal(t, codes.Internal, httpStatusToGRPCCode(0))
}

func TestHttpStatusToGRPCCode_ShouldReturnInternal_WhenNegativeStatus(t *testing.T) {
	assert.Equal(t, codes.Internal, httpStatusToGRPCCode(-1))
}

// --- Table-driven tests for comprehensive coverage ---

func TestMapError_ShouldMapAllSentinelErrors_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
		expectedMsg  string
	}{
		{"WorkflowNotFound", models.ErrWorkflowNotFound, codes.NotFound, "workflow not found"},
		{"ExecutionNotFound", models.ErrExecutionNotFound, codes.NotFound, "execution not found"},
		{"TriggerNotFound", models.ErrTriggerNotFound, codes.NotFound, "trigger not found"},
		{"ResourceNotFound", models.ErrResourceNotFound, codes.NotFound, "resource not found"},
		{"InvalidID", models.ErrInvalidID, codes.InvalidArgument, "invalid ID format"},
		{"Unauthorized", models.ErrUnauthorized, codes.Unauthenticated, "authentication required"},
		{"Forbidden", models.ErrForbidden, codes.PermissionDenied, "access denied"},
		{"WorkflowExists", models.ErrWorkflowExists, codes.AlreadyExists, "workflow already exists"},
		{"ValidationFailed", models.ErrValidationFailed, codes.InvalidArgument, "validation failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapError(tt.err)

			require.NotNil(t, result)
			st, ok := status.FromError(result)
			require.True(t, ok)
			assert.Equal(t, tt.expectedCode, st.Code())
			assert.Equal(t, tt.expectedMsg, st.Message())
		})
	}
}

func TestHttpStatusToGRPCCode_ShouldMapAllStatuses_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		httpStatus   int
		expectedCode codes.Code
	}{
		{"200 OK", 200, codes.OK},
		{"201 Created", 201, codes.OK},
		{"204 NoContent", 204, codes.OK},
		{"299 boundary", 299, codes.OK},
		{"400 BadRequest", 400, codes.InvalidArgument},
		{"401 Unauthorized", 401, codes.Unauthenticated},
		{"403 Forbidden", 403, codes.PermissionDenied},
		{"404 NotFound", 404, codes.NotFound},
		{"409 Conflict", 409, codes.AlreadyExists},
		{"429 TooManyRequests", 429, codes.ResourceExhausted},
		{"501 NotImplemented", 501, codes.Unimplemented},
		{"500 InternalServerError", 500, codes.Internal},
		{"502 BadGateway", 502, codes.Internal},
		{"503 ServiceUnavailable", 503, codes.Internal},
		{"300 redirect", 300, codes.Internal},
		{"100 informational", 100, codes.Internal},
		{"0 zero", 0, codes.Internal},
		{"-1 negative", -1, codes.Internal},
		{"999 unknown", 999, codes.Internal},
		{"422 unprocessable", 422, codes.Internal},
		{"405 method not allowed", 405, codes.Internal},
		{"408 request timeout", 408, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := httpStatusToGRPCCode(tt.httpStatus)

			assert.Equal(t, tt.expectedCode, result)
		})
	}
}

func TestMapError_ShouldPreferOperationError_WhenWrappedOverSentinel(t *testing.T) {
	// OperationError takes priority over sentinel errors via errors.As
	opErr := &serviceapi.OperationError{
		Code:       "CUSTOM",
		Message:    "custom message",
		HTTPStatus: http.StatusConflict,
	}

	result := mapError(opErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, "custom message", st.Message())
}

func TestMapError_ShouldReturnNil_WhenOperationErrorHTTP200(t *testing.T) {
	// gRPC's status.Errorf with codes.OK returns nil because OK is not an error
	opErr := &serviceapi.OperationError{
		Code:       "SUCCESS",
		Message:    "all good",
		HTTPStatus: http.StatusOK,
	}

	result := mapError(opErr)

	assert.Nil(t, result)
}

func TestMapError_ShouldReturnInternal_WhenOperationErrorHTTP500(t *testing.T) {
	opErr := &serviceapi.OperationError{
		Code:       "INTERNAL",
		Message:    "server broke",
		HTTPStatus: http.StatusInternalServerError,
	}

	result := mapError(opErr)

	require.NotNil(t, result)
	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "server broke", st.Message())
}
