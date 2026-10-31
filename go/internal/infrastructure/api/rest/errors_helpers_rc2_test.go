package rest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/auth"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func TestRC2_APIError_ErrorReturnsMessage(t *testing.T) {
	t.Parallel()
	err := NewAPIError("X", "boom message", http.StatusTeapot)
	assert.Equal(t, "boom message", err.Error())

	var target *APIError
	wrapped := fmt.Errorf("wrap: %w", err)
	require.True(t, errors.As(wrapped, &target))
	assert.Same(t, err, TranslateError(wrapped), "an APIError is passed through unchanged")
}

func TestRC2_TranslateError_Table(t *testing.T) {
	t.Parallel()

	assert.Nil(t, TranslateError(nil))

	cases := []struct {
		err    error
		code   string
		status int
	}{
		{&serviceapi.OperationError{Code: "OP_CODE", Message: "op", HTTPStatus: 418}, "OP_CODE", 418},
		{models.ErrWorkflowNotFound, "WORKFLOW_NOT_FOUND", 404},
		{models.ErrExecutionNotFound, "EXECUTION_NOT_FOUND", 404},
		{models.ErrTriggerNotFound, "TRIGGER_NOT_FOUND", 404},
		{models.ErrNodeNotFound, "NODE_NOT_FOUND", 404},
		{models.ErrEdgeNotFound, "EDGE_NOT_FOUND", 404},
		{models.ErrResourceNotFound, "RESOURCE_NOT_FOUND", 404},
		{models.ErrAccountNotFound, "ACCOUNT_NOT_FOUND", 404},
		{models.ErrRentalKeyNotFound, "RENTAL_KEY_NOT_FOUND", 404},
		{models.ErrUserNotFound, "USER_NOT_FOUND", 404},
		{models.ErrRoleNotFound, "ROLE_NOT_FOUND", 404},
		{models.ErrAPIClientNotFound, "API_CLIENT_NOT_FOUND", 404},
		{models.ErrInvalidAPISpec, "INVALID_API_SPEC", 400},
		{models.ErrInvalidWorkflowID, "INVALID_WORKFLOW_ID", 400},
		{models.ErrInvalidExecutionID, "INVALID_EXECUTION_ID", 400},
		{models.ErrInvalidTriggerID, "INVALID_TRIGGER_ID", 400},
		{models.ErrInvalidID, "INVALID_ID", 400},
		{models.ErrCyclicDependency, "CYCLIC_DEPENDENCY", 400},
		{models.ErrOrphanedNodes, "ORPHANED_NODES", 400},
		{models.ErrInvalidNodeType, "INVALID_NODE_TYPE", 400},
		{models.ErrInvalidEdge, "INVALID_EDGE", 400},
		{models.ErrInvalidWorkflow, "INVALID_WORKFLOW", 400},
		{models.ErrInvalidTriggerType, "INVALID_TRIGGER_TYPE", 400},
		{models.ErrInvalidTriggerConfig, "INVALID_TRIGGER_CONFIG", 400},
		{models.ErrInvalidConfig, "INVALID_CONFIG", 400},
		{models.ErrInvalidInput, "INVALID_INPUT", 400},
		{models.ErrInvalidResourceType, "INVALID_RESOURCE_TYPE", 400},
		{models.ErrWorkflowExists, "WORKFLOW_EXISTS", 409},
		{models.ErrUserExists, "USER_EXISTS", 409},
		{models.ErrUnauthorized, "UNAUTHORIZED", 401},
		{models.ErrForbidden, "FORBIDDEN", 403},
		{models.ErrInvalidCredentials, "INVALID_CREDENTIALS", 401},
		{models.ErrInvalidToken, "INVALID_TOKEN", 401},
		{models.ErrTokenExpired, "TOKEN_EXPIRED", 401},
		{models.ErrSessionExpired, "SESSION_EXPIRED", 401},
		{models.ErrPermissionDenied, "PERMISSION_DENIED", 403},
		{models.ErrInsufficientBalance, "INSUFFICIENT_BALANCE", 402},
		{models.ErrAccountInactive, "ACCOUNT_INACTIVE", 403},
		{models.ErrAccountSuspended, "ACCOUNT_SUSPENDED", 403},
		{models.ErrAccountClosed, "ACCOUNT_CLOSED", 403},
		{models.ErrResourceLimitExceeded, "RESOURCE_LIMIT_EXCEEDED", 403},
		{models.ErrStorageLimitExceeded, "STORAGE_LIMIT_EXCEEDED", 403},
		{models.ErrDailyLimitExceeded, "DAILY_LIMIT_EXCEEDED", 429},
		{models.ErrMonthlyTokenLimitExceeded, "MONTHLY_TOKEN_LIMIT_EXCEEDED", 429},
		{models.ErrTriggerDisabled, "TRIGGER_DISABLED", 400},
		{models.ErrRentalKeySuspended, "RENTAL_KEY_SUSPENDED", 403},
		{models.ErrRentalKeyAccessDenied, "RENTAL_KEY_ACCESS_DENIED", 403},
		{models.ErrValidationFailed, "VALIDATION_FAILED", 400},
		{auth.ErrUserNotFound, "USER_NOT_FOUND", 404},
		{auth.ErrEmailAlreadyTaken, "EMAIL_ALREADY_TAKEN", 409},
		{auth.ErrUsernameAlreadyTaken, "USERNAME_ALREADY_TAKEN", 409},
		{auth.ErrInvalidCredentials, "INVALID_CREDENTIALS", 401},
		{auth.ErrAccountLocked, "ACCOUNT_LOCKED", 403},
		{auth.ErrAccountInactive, "ACCOUNT_INACTIVE", 403},
		{auth.ErrInvalidRefreshToken, "INVALID_REFRESH_TOKEN", 401},
		{auth.ErrRefreshTokenExpired, "REFRESH_TOKEN_EXPIRED", 401},
		{auth.ErrRegistrationDisabled, "REGISTRATION_DISABLED", 403},
		{auth.ErrRoleNotFound, "ROLE_NOT_FOUND", 404},
		{auth.ErrGRPCProviderNotConfigured, "GRPC_NOT_CONFIGURED", 503},
		{auth.ErrGRPCLoginFailed, "GRPC_LOGIN_FAILED", 502},
		{auth.ErrGRPCTokenValidationFailed, "GRPC_TOKEN_VALIDATION_FAILED", 502},
		{auth.ErrGRPCUserFetchFailed, "GRPC_USER_FETCH_FAILED", 502},
		{auth.ErrGRPCUserCreateFailed, "GRPC_USER_CREATE_FAILED", 502},
		{auth.ErrRefreshNotSupported, "REFRESH_NOT_SUPPORTED", 501},
		{auth.ErrCallbackNotSupported, "CALLBACK_NOT_SUPPORTED", 501},
		{auth.ErrNoProvidersAvailable, "NO_AUTH_PROVIDERS", 503},
		{auth.ErrAllProvidersFailed, "ALL_PROVIDERS_FAILED", 503},
		{sql.ErrNoRows, "NOT_FOUND", 404},
		{errors.New("thing was Not Found here"), "NOT_FOUND", 404},
		{errors.New("pg: no rows in result set"), "NOT_FOUND", 404},
		{&auth.PasswordError{Message: "too weak"}, "INVALID_PASSWORD", 400},
		{errors.New("something exploded"), "INTERNAL_ERROR", 500},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.code+"/"+tc.err.Error(), func(t *testing.T) {
			// Wrapping must not change the translation.
			got := TranslateError(fmt.Errorf("ctx: %w", tc.err))
			require.NotNil(t, got)
			assert.Equal(t, tc.code, got.Code)
			assert.Equal(t, tc.status, got.HTTPStatus)
		})
	}

	grpc := TranslateError(fmt.Errorf("dial: %w", auth.ErrGRPCLoginFailed))
	assert.Contains(t, grpc.Details["original_error"], "dial")
}

func TestRC2_TranslateError_ValidationErrors(t *testing.T) {
	t.Parallel()

	single := TranslateError(&models.ValidationError{Field: "name", Message: "name is bad"})
	assert.Equal(t, "VALIDATION_ERROR", single.Code)
	assert.Equal(t, "name is bad", single.Message)
	assert.Equal(t, "name", single.Details["field"])

	multi := TranslateError(models.ValidationErrors{
		{Field: "a", Message: "a is bad"},
		{Field: "b", Message: "b is bad"},
	})
	assert.Equal(t, "VALIDATION_FAILED", multi.Code)
	assert.Equal(t, "a is bad", multi.Message)
	assert.Equal(t, http.StatusBadRequest, multi.HTTPStatus)

	empty := TranslateError(models.ValidationErrors{})
	assert.Equal(t, "VALIDATION_FAILED", empty.Code)
	assert.Equal(t, "Multiple validation errors", empty.Message)
}

// rc2Serve runs a single handler for one request and returns the recorder.
func rc2Serve(t *testing.T, method, route, path string, body []byte, handler gin.HandlerFunc, pre ...gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers := append(append([]gin.HandlerFunc{}, pre...), handler)
	r.Handle(method, route, handlers...)
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func rc2DecodeMap(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m), w.Body.String())
	return m
}

func TestRC2_BindJSON_ValidationMessages(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name  string `json:"name" binding:"required"`
		ID    string `json:"id" binding:"omitempty,uuid"`
		Short string `json:"short" binding:"omitempty,min=3"`
		Long  string `json:"long" binding:"omitempty,max=2"`
		Mail  string `json:"mail" binding:"omitempty,email"`
	}
	handler := func(c *gin.Context) {
		var p payload
		if err := bindJSON(c, &p); err != nil {
			return
		}
		c.JSON(http.StatusOK, p)
	}

	w := rc2Serve(t, http.MethodPost, "/x", "/x", []byte(`{"id":"nope","short":"a","long":"abcd","mail":"bad"}`), handler)
	require.Equal(t, http.StatusBadRequest, w.Code)
	msg := rc2DecodeMap(t, w)["message"].(string)
	assert.Contains(t, msg, "name is required")
	assert.Contains(t, msg, "id must be a valid UUID")
	assert.Contains(t, msg, "short must be at least 3 characters")
	assert.Contains(t, msg, "long must be at most 2 characters")
	assert.Contains(t, msg, "mail is invalid")

	w = rc2Serve(t, http.MethodPost, "/x", "/x", []byte(`{not json`), handler)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_JSON", rc2DecodeMap(t, w)["code"])

	w = rc2Serve(t, http.MethodPost, "/x", "/x", []byte(`{"name":"ok"}`), handler)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRC2_BindJSON_TooLargeBody(t *testing.T) {
	t.Parallel()

	limiter := NewBodySizeMiddleware(nil, 16)
	handler := func(c *gin.Context) {
		var p map[string]any
		if err := bindJSON(c, &p); err != nil {
			return
		}
		c.JSON(http.StatusOK, p)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/x", limiter.LimitBodySize(), handler)

	// Declared oversized body is rejected before the handler runs.
	big := `{"k":"` + strings.Repeat("a", 64) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(big))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Equal(t, "PAYLOAD_TOO_LARGE", rc2DecodeMap(t, w)["code"])

	// Unknown length (chunked) body is cut off by MaxBytesReader and
	// reported by bindJSON.
	req = httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(big))
	req.ContentLength = -1
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Equal(t, "PAYLOAD_TOO_LARGE", rc2DecodeMap(t, w)["code"])

	// Small body passes.
	req = httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"k":1}`))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRC2_GetParamAndParamFirst(t *testing.T) {
	t.Parallel()

	handler := func(c *gin.Context) {
		v, ok := getParam(c, "id")
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": v, "first": paramFirst(c, "missing", "id"), "none": paramFirst(c, "missing")})
	}
	w := rc2Serve(t, http.MethodGet, "/items/:id", "/items/abc", nil, handler)
	require.Equal(t, http.StatusOK, w.Code)
	body := rc2DecodeMap(t, w)
	assert.Equal(t, "abc", body["id"])
	assert.Equal(t, "abc", body["first"])
	assert.Equal(t, "", body["none"])

	// Route without the parameter: getParam reports it as missing, with the
	// request ID in the details.
	setReqID := func(c *gin.Context) { c.Set(ContextKeyRequestID, "req-1"); c.Next() }
	w = rc2Serve(t, http.MethodGet, "/items", "/items", nil, handler, setReqID)
	require.Equal(t, http.StatusBadRequest, w.Code)
	body = rc2DecodeMap(t, w)
	assert.Equal(t, "MISSING_PARAMETER", body["code"])
	assert.Equal(t, "id is required", body["message"])
	assert.Equal(t, "req-1", body["details"].(map[string]any)["request_id"])
}

func TestRC2_QueryHelpersAndRespondSuccess(t *testing.T) {
	t.Parallel()

	handler := func(c *gin.Context) {
		if c.Query("meta") == "1" {
			respondSuccess(c, http.StatusOK, []int{1}, &listMeta{Total: 9, Limit: 2, Offset: 3})
			return
		}
		respondSuccess(c, http.StatusOK, gin.H{
			"q":   getQuery(c, "q", "dflt"),
			"n":   getQueryInt(c, "n", 7),
			"bad": parseIntQuery("x", 5),
		}, nil)
	}
	w := rc2Serve(t, http.MethodGet, "/q", "/q?n=abc", nil, handler)
	body := rc2DecodeMap(t, w)
	assert.Equal(t, "dflt", body["q"])
	assert.Equal(t, float64(7), body["n"])
	assert.Equal(t, float64(5), body["bad"])

	w = rc2Serve(t, http.MethodGet, "/q", "/q?q=v&n=3", nil, handler)
	body = rc2DecodeMap(t, w)
	assert.Equal(t, "v", body["q"])
	assert.Equal(t, float64(3), body["n"])

	w = rc2Serve(t, http.MethodGet, "/q", "/q?meta=1", nil, handler)
	body = rc2DecodeMap(t, w)
	assert.Equal(t, float64(9), body["total"])
	assert.Equal(t, float64(2), body["limit"])
	assert.Equal(t, float64(3), body["offset"])
}
