package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// MakeRequest helper function to make HTTP requests to the test server
func MakeRequest(t *testing.T, router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	return w
}

// MakeRequestRaw makes a request with raw string body
func MakeRequestRaw(t *testing.T, router *gin.Engine, method, path, rawBody string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(rawBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	return w
}

// MakeRequestWithHeaders makes a request with custom headers
func MakeRequestWithHeaders(t *testing.T, router *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

// ParseResponse helper function to parse JSON response directly into result.
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder, result interface{}) {
	t.Helper()

	body := w.Body.Bytes()
	err := json.Unmarshal(body, result)
	require.NoError(t, err, "Failed to parse response: %s", string(body))
}

// ParseListResponse parses a list response with flat {data: [...], total, limit, offset} structure.
// Stores the data array in result and returns a map with total/limit/offset.
func ParseListResponse(t *testing.T, w *httptest.ResponseRecorder, result interface{}) map[string]interface{} {
	t.Helper()

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Total  *int            `json:"total"`
		Limit  *int            `json:"limit"`
		Offset *int            `json:"offset"`
	}
	body := w.Body.Bytes()
	err := json.Unmarshal(body, &envelope)
	require.NoError(t, err, "Failed to parse list response: %s", string(body))
	require.NotNil(t, envelope.Data, "Response missing 'data' field: %s", string(body))

	err = json.Unmarshal(envelope.Data, result)
	require.NoError(t, err, "Failed to parse list data: %s", string(body))

	meta := map[string]interface{}{}
	if envelope.Total != nil {
		meta["total"] = float64(*envelope.Total)
	}
	if envelope.Limit != nil {
		meta["limit"] = float64(*envelope.Limit)
	}
	if envelope.Offset != nil {
		meta["offset"] = float64(*envelope.Offset)
	}
	return meta
}

// AssertJSONResponse asserts the response status code and parses JSON
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, result interface{}) {
	t.Helper()

	require.Equal(t, expectedStatus, w.Code, "Unexpected status code. Response: %s", w.Body.String())

	if result != nil && w.Code >= 200 && w.Code < 300 {
		ParseResponse(t, w, result)
	}
}

// AssertErrorResponse asserts an error response with expected status and message
// Supports both legacy format (field "error") and new APIError format (field "message")
func AssertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	require.Equal(t, expectedStatus, w.Code, "Unexpected status code")

	var errorResp map[string]interface{}
	ParseResponse(t, w, &errorResp)

	if expectedMessage != "" {
		// Support both new APIError format (message) and legacy format (error)
		message, ok := errorResp["message"]
		if !ok {
			message = errorResp["error"]
		}
		require.Contains(t, message, expectedMessage,
			"Error message doesn't contain expected text")
	}
}

// MockHTTPServer creates a simple mock HTTP server for testing
func MockHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// MockJSONServer creates a mock server that returns JSON responses
func MockJSONServer(t *testing.T, responses map[string]interface{}) *httptest.Server {
	t.Helper()

	return MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Default response
		response := map[string]interface{}{
			"success": true,
			"data":    "mock response",
		}

		// Check if we have a specific response for this path
		if pathResponse, ok := responses[r.URL.Path]; ok {
			response = pathResponse.(map[string]interface{})
		}

		json.NewEncoder(w).Encode(response)
	})
}

// MockErrorServer creates a mock server that returns errors
func MockErrorServer(t *testing.T, statusCode int, errorMessage string) *httptest.Server {
	t.Helper()

	return MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		response := map[string]interface{}{
			"error": errorMessage,
		}

		json.NewEncoder(w).Encode(response)
	})
}

// AssertWorkflowCreated asserts that a workflow was created successfully
func AssertWorkflowCreated(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	var result map[string]interface{}
	AssertJSONResponse(t, w, http.StatusCreated, &result)
	require.NotEmpty(t, result["id"], "Workflow ID should not be empty")
	return result
}

// AssertWorkflowDeleted asserts that a workflow was deleted successfully
func AssertWorkflowDeleted(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	require.Equal(t, http.StatusNoContent, w.Code, "Expected 204 No Content")
}

// AssertExecutionStarted asserts that an execution was started successfully
func AssertExecutionStarted(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	var result map[string]interface{}
	AssertJSONResponse(t, w, http.StatusOK, &result)
	require.NotEmpty(t, result["id"], "Execution ID should not be empty")
	return result
}

// MockAuthMiddleware creates a middleware that sets user context for testing
// Use this to add authentication context to test routes
func MockAuthMiddleware(userID string, isAdmin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("is_admin", isAdmin)
		c.Next()
	}
}
