package rest

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/smilemakc/mbflow/internal/application/auth"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/pkg/models"
)

type APIError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
}

func (e *APIError) Error() string {
	return e.Message
}

func NewAPIError(code, message string, httpStatus int) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func NewAPIErrorWithDetails(code, message string, httpStatus int, details map[string]interface{}) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: httpStatus,
	}
}

var (
	ErrBadRequest          = NewAPIError("BAD_REQUEST", "Invalid request", http.StatusBadRequest)
	ErrUnauthorized        = NewAPIError("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
	ErrForbidden           = NewAPIError("FORBIDDEN", "Access denied", http.StatusForbidden)
	ErrNotFound            = NewAPIError("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrConflict            = NewAPIError("CONFLICT", "Resource conflict", http.StatusConflict)
	ErrValidationFailed    = NewAPIError("VALIDATION_FAILED", "Validation failed", http.StatusBadRequest)
	ErrInternalServer      = NewAPIError("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrTooManyRequests     = NewAPIError("RATE_LIMIT_EXCEEDED", "Too many requests", http.StatusTooManyRequests)
	ErrInvalidJSON         = NewAPIError("INVALID_JSON", "Invalid JSON in request body", http.StatusBadRequest)
	ErrMissingParameter    = NewAPIError("MISSING_PARAMETER", "Required parameter is missing", http.StatusBadRequest)
	ErrInvalidParameter    = NewAPIError("INVALID_PARAMETER", "Invalid parameter value", http.StatusBadRequest)
	ErrInvalidID           = NewAPIError("INVALID_ID", "Invalid ID format", http.StatusBadRequest)
	ErrTokenExpired        = NewAPIError("TOKEN_EXPIRED", "Token has expired", http.StatusUnauthorized)
	ErrInvalidToken        = NewAPIError("INVALID_TOKEN", "Invalid token", http.StatusUnauthorized)
	ErrInsufficientBalance = NewAPIError("INSUFFICIENT_BALANCE", "Insufficient account balance", http.StatusPaymentRequired)
)

func TranslateError(err error) *APIError {
	if err == nil {
		return nil
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}

	var opErr *serviceapi.OperationError
	if errors.As(err, &opErr) {
		return NewAPIError(opErr.Code, opErr.Message, opErr.HTTPStatus)
	}

	switch {
	case errors.Is(err, models.ErrWorkflowNotFound):
		return NewAPIError("WORKFLOW_NOT_FOUND", "Workflow not found", http.StatusNotFound)
	case errors.Is(err, models.ErrExecutionNotFound):
		return NewAPIError("EXECUTION_NOT_FOUND", "Execution not found", http.StatusNotFound)
	case errors.Is(err, models.ErrTriggerNotFound):
		return NewAPIError("TRIGGER_NOT_FOUND", "Trigger not found", http.StatusNotFound)
	case errors.Is(err, models.ErrNodeNotFound):
		return NewAPIError("NODE_NOT_FOUND", "Node not found", http.StatusNotFound)
	case errors.Is(err, models.ErrEdgeNotFound):
		return NewAPIError("EDGE_NOT_FOUND", "Edge not found", http.StatusNotFound)
	case errors.Is(err, models.ErrResourceNotFound):
		return NewAPIError("RESOURCE_NOT_FOUND", "Resource not found", http.StatusNotFound)
	case errors.Is(err, models.ErrAccountNotFound):
		return NewAPIError("ACCOUNT_NOT_FOUND", "Account not found", http.StatusNotFound)
	case errors.Is(err, models.ErrRentalKeyNotFound):
		return NewAPIError("RENTAL_KEY_NOT_FOUND", "Rental key not found", http.StatusNotFound)
	case errors.Is(err, models.ErrUserNotFound):
		return NewAPIError("USER_NOT_FOUND", "User not found", http.StatusNotFound)
	case errors.Is(err, models.ErrRoleNotFound):
		return NewAPIError("ROLE_NOT_FOUND", "Role not found", http.StatusNotFound)

	case errors.Is(err, models.ErrInvalidWorkflowID):
		return NewAPIError("INVALID_WORKFLOW_ID", "Invalid workflow ID format", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidExecutionID):
		return NewAPIError("INVALID_EXECUTION_ID", "Invalid execution ID format", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidTriggerID):
		return NewAPIError("INVALID_TRIGGER_ID", "Invalid trigger ID format", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidID):
		return NewAPIError("INVALID_ID", "Invalid ID format", http.StatusBadRequest)

	case errors.Is(err, models.ErrCyclicDependency):
		return NewAPIError("CYCLIC_DEPENDENCY", "Workflow contains cyclic dependencies", http.StatusBadRequest)
	case errors.Is(err, models.ErrOrphanedNodes):
		return NewAPIError("ORPHANED_NODES", "Workflow contains orphaned nodes", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidNodeType):
		return NewAPIError("INVALID_NODE_TYPE", "Invalid node type", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidEdge):
		return NewAPIError("INVALID_EDGE", "Invalid edge configuration", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidWorkflow):
		return NewAPIError("INVALID_WORKFLOW", "Invalid workflow structure", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidTriggerType):
		return NewAPIError("INVALID_TRIGGER_TYPE", "Invalid trigger type", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidTriggerConfig):
		return NewAPIError("INVALID_TRIGGER_CONFIG", "Invalid trigger configuration", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidConfig):
		return NewAPIError("INVALID_CONFIG", "Invalid configuration", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidInput):
		return NewAPIError("INVALID_INPUT", "Invalid input data", http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidResourceType):
		return NewAPIError("INVALID_RESOURCE_TYPE", "Invalid resource type", http.StatusBadRequest)

	case errors.Is(err, models.ErrWorkflowExists):
		return NewAPIError("WORKFLOW_EXISTS", "Workflow already exists", http.StatusConflict)
	case errors.Is(err, models.ErrUserExists):
		return NewAPIError("USER_EXISTS", "User already exists", http.StatusConflict)

	case errors.Is(err, models.ErrUnauthorized):
		return NewAPIError("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
	case errors.Is(err, models.ErrForbidden):
		return NewAPIError("FORBIDDEN", "Access denied", http.StatusForbidden)
	case errors.Is(err, models.ErrInvalidCredentials):
		return NewAPIError("INVALID_CREDENTIALS", "Invalid credentials", http.StatusUnauthorized)
	case errors.Is(err, models.ErrInvalidToken):
		return NewAPIError("INVALID_TOKEN", "Invalid token", http.StatusUnauthorized)
	case errors.Is(err, models.ErrTokenExpired):
		return NewAPIError("TOKEN_EXPIRED", "Token has expired", http.StatusUnauthorized)
	case errors.Is(err, models.ErrSessionExpired):
		return NewAPIError("SESSION_EXPIRED", "Session has expired", http.StatusUnauthorized)
	case errors.Is(err, models.ErrPermissionDenied):
		return NewAPIError("PERMISSION_DENIED", "Permission denied", http.StatusForbidden)

	case errors.Is(err, models.ErrInsufficientBalance):
		return NewAPIError("INSUFFICIENT_BALANCE", "Insufficient account balance", http.StatusPaymentRequired)
	case errors.Is(err, models.ErrAccountInactive):
		return NewAPIError("ACCOUNT_INACTIVE", "Account is inactive", http.StatusForbidden)
	case errors.Is(err, models.ErrAccountSuspended):
		return NewAPIError("ACCOUNT_SUSPENDED", "Account is suspended", http.StatusForbidden)
	case errors.Is(err, models.ErrAccountClosed):
		return NewAPIError("ACCOUNT_CLOSED", "Account is closed", http.StatusForbidden)
	case errors.Is(err, models.ErrResourceLimitExceeded):
		return NewAPIError("RESOURCE_LIMIT_EXCEEDED", "Resource limit exceeded", http.StatusForbidden)
	case errors.Is(err, models.ErrStorageLimitExceeded):
		return NewAPIError("STORAGE_LIMIT_EXCEEDED", "Storage limit exceeded", http.StatusForbidden)
	case errors.Is(err, models.ErrDailyLimitExceeded):
		return NewAPIError("DAILY_LIMIT_EXCEEDED", "Daily request limit exceeded", http.StatusTooManyRequests)
	case errors.Is(err, models.ErrMonthlyTokenLimitExceeded):
		return NewAPIError("MONTHLY_TOKEN_LIMIT_EXCEEDED", "Monthly token limit exceeded", http.StatusTooManyRequests)

	case errors.Is(err, models.ErrTriggerDisabled):
		return NewAPIError("TRIGGER_DISABLED", "Trigger is disabled", http.StatusBadRequest)
	case errors.Is(err, models.ErrRentalKeySuspended):
		return NewAPIError("RENTAL_KEY_SUSPENDED", "Rental key is suspended", http.StatusForbidden)
	case errors.Is(err, models.ErrRentalKeyAccessDenied):
		return NewAPIError("RENTAL_KEY_ACCESS_DENIED", "Rental key access denied", http.StatusForbidden)

	case errors.Is(err, models.ErrValidationFailed):
		return NewAPIError("VALIDATION_FAILED", "Validation failed", http.StatusBadRequest)

	case errors.Is(err, auth.ErrUserNotFound):
		return NewAPIError("USER_NOT_FOUND", "User not found", http.StatusNotFound)
	case errors.Is(err, auth.ErrEmailAlreadyTaken):
		return NewAPIError("EMAIL_ALREADY_TAKEN", "Email is already taken", http.StatusConflict)
	case errors.Is(err, auth.ErrUsernameAlreadyTaken):
		return NewAPIError("USERNAME_ALREADY_TAKEN", "Username is already taken", http.StatusConflict)
	case errors.Is(err, auth.ErrInvalidCredentials):
		return NewAPIError("INVALID_CREDENTIALS", "Invalid credentials", http.StatusUnauthorized)
	case errors.Is(err, auth.ErrAccountLocked):
		return NewAPIError("ACCOUNT_LOCKED", "Account is locked", http.StatusForbidden)
	case errors.Is(err, auth.ErrAccountInactive):
		return NewAPIError("ACCOUNT_INACTIVE", "Account is inactive", http.StatusForbidden)
	case errors.Is(err, auth.ErrInvalidRefreshToken):
		return NewAPIError("INVALID_REFRESH_TOKEN", "Invalid refresh token", http.StatusUnauthorized)
	case errors.Is(err, auth.ErrRefreshTokenExpired):
		return NewAPIError("REFRESH_TOKEN_EXPIRED", "Refresh token has expired", http.StatusUnauthorized)
	case errors.Is(err, auth.ErrRegistrationDisabled):
		return NewAPIError("REGISTRATION_DISABLED", "Registration is disabled", http.StatusForbidden)
	case errors.Is(err, auth.ErrRoleNotFound):
		return NewAPIError("ROLE_NOT_FOUND", "Role not found", http.StatusNotFound)

	// gRPC provider errors
	case errors.Is(err, auth.ErrGRPCProviderNotConfigured):
		return NewAPIError("GRPC_NOT_CONFIGURED", "gRPC authentication provider is not configured", http.StatusServiceUnavailable)
	case errors.Is(err, auth.ErrGRPCLoginFailed):
		return NewAPIErrorWithDetails("GRPC_LOGIN_FAILED", "Login via gRPC auth-gateway failed", http.StatusBadGateway, map[string]interface{}{
			"original_error": err.Error(),
		})
	case errors.Is(err, auth.ErrGRPCTokenValidationFailed):
		return NewAPIErrorWithDetails("GRPC_TOKEN_VALIDATION_FAILED", "Token validation via gRPC auth-gateway failed", http.StatusBadGateway, map[string]interface{}{
			"original_error": err.Error(),
		})
	case errors.Is(err, auth.ErrGRPCUserFetchFailed):
		return NewAPIErrorWithDetails("GRPC_USER_FETCH_FAILED", "User fetch via gRPC auth-gateway failed", http.StatusBadGateway, map[string]interface{}{
			"original_error": err.Error(),
		})
	case errors.Is(err, auth.ErrGRPCUserCreateFailed):
		return NewAPIErrorWithDetails("GRPC_USER_CREATE_FAILED", "User creation via gRPC auth-gateway failed", http.StatusBadGateway, map[string]interface{}{
			"original_error": err.Error(),
		})
	case errors.Is(err, auth.ErrRefreshNotSupported):
		return NewAPIError("REFRESH_NOT_SUPPORTED", "Refresh token not supported via gRPC proxy", http.StatusNotImplemented)
	case errors.Is(err, auth.ErrCallbackNotSupported):
		return NewAPIError("CALLBACK_NOT_SUPPORTED", "OAuth callback not supported via gRPC proxy", http.StatusNotImplemented)
	case errors.Is(err, auth.ErrNoProvidersAvailable):
		return NewAPIError("NO_AUTH_PROVIDERS", "No authentication providers available", http.StatusServiceUnavailable)
	case errors.Is(err, auth.ErrAllProvidersFailed):
		return NewAPIErrorWithDetails("ALL_PROVIDERS_FAILED", "All authentication providers failed", http.StatusServiceUnavailable, map[string]interface{}{
			"original_error": err.Error(),
		})

	// Database-level not found (when repository doesn't wrap sql.ErrNoRows)
	case errors.Is(err, sql.ErrNoRows):
		return NewAPIError("NOT_FOUND", "Resource not found", http.StatusNotFound)
	}

	// Check for string patterns in error message as fallback
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "no rows") || strings.Contains(errMsg, "not found") {
		return NewAPIError("NOT_FOUND", "Resource not found", http.StatusNotFound)
	}

	// Check for custom error types in default block
	{
		var passwordErr *auth.PasswordError
		if errors.As(err, &passwordErr) {
			return NewAPIError("INVALID_PASSWORD", passwordErr.Error(), http.StatusBadRequest)
		}

		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			return NewAPIErrorWithDetails(
				"VALIDATION_ERROR",
				validationErr.Message,
				http.StatusBadRequest,
				map[string]interface{}{
					"field": validationErr.Field,
				},
			)
		}

		var validationErrs models.ValidationErrors
		if errors.As(err, &validationErrs) {
			details := make(map[string]interface{})
			for i, ve := range validationErrs {
				details[ve.Field] = ve.Message
				if i == 0 {
					return NewAPIErrorWithDetails("VALIDATION_FAILED", ve.Message, http.StatusBadRequest, details)
				}
			}
			return NewAPIErrorWithDetails("VALIDATION_FAILED", "Multiple validation errors", http.StatusBadRequest, details)
		}
	}

	return NewAPIError("INTERNAL_ERROR", "An unexpected error occurred", http.StatusInternalServerError)
}
