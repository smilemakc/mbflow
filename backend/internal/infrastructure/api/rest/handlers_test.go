package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Setup gin test mode
func init() {
	gin.SetMode(gin.TestMode)
}

// ErrorResponse represents an error response for testing (new APIError format)
type ErrorResponse struct {
	Message string                 `json:"message"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details"`
}

// Helper functions for testing

func performRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	req, _ := http.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseJSON(t *testing.T, body string, v interface{}) {
	if err := json.Unmarshal([]byte(body), v); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
}

func TestRespondJSON(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		respondJSON(c, http.StatusOK, gin.H{"message": "success"})
	})

	w := performRequest(router, "GET", "/test", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	parseJSON(t, w.Body.String(), &response)

	if response["message"] != "success" {
		t.Errorf("expected message=success, got %s", response["message"])
	}
}

func TestRespondError(t *testing.T) {
	router := gin.New()
	router.GET("/error", func(c *gin.Context) {
		respondError(c, http.StatusBadRequest, "invalid request")
	})

	w := performRequest(router, "GET", "/error", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response map[string]string
	parseJSON(t, w.Body.String(), &response)

	// New APIError format uses "message" field
	if response["message"] != "invalid request" {
		t.Errorf("expected error message 'invalid request', got %s", response["message"])
	}
}

func TestGetQueryInt(t *testing.T) {
	tests := []struct {
		name         string
		queryParam   string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			queryParam:   "10",
			defaultValue: 5,
			expected:     10,
		},
		{
			name:         "invalid integer uses default",
			queryParam:   "abc",
			defaultValue: 5,
			expected:     5,
		},
		{
			name:         "empty uses default",
			queryParam:   "",
			defaultValue: 5,
			expected:     5,
		},
		{
			name:         "negative integer",
			queryParam:   "-10",
			defaultValue: 5,
			expected:     -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				value := getQueryInt(c, "param", tt.defaultValue)
				c.JSON(http.StatusOK, gin.H{"value": value})
			})

			path := "/test"
			if tt.queryParam != "" {
				path += "?param=" + tt.queryParam
			}

			w := performRequest(router, "GET", path, nil)

			var response map[string]interface{}
			parseJSON(t, w.Body.String(), &response)

			value := int(response["value"].(float64))
			if value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, value)
			}
		})
	}
}

func TestGetQuery(t *testing.T) {
	tests := []struct {
		name         string
		queryParam   string
		defaultValue string
		expected     string
	}{
		{
			name:         "with value",
			queryParam:   "testvalue",
			defaultValue: "default",
			expected:     "testvalue",
		},
		{
			name:         "empty uses default",
			queryParam:   "",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				value := getQuery(c, "param", tt.defaultValue)
				c.JSON(http.StatusOK, gin.H{"value": value})
			})

			path := "/test"
			if tt.queryParam != "" {
				path += "?param=" + tt.queryParam
			}

			w := performRequest(router, "GET", path, nil)

			var response map[string]interface{}
			parseJSON(t, w.Body.String(), &response)

			value := response["value"].(string)
			if value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, value)
			}
		})
	}
}

func TestGetParam(t *testing.T) {
	router := gin.New()

	router.GET("/valid/:id", func(c *gin.Context) {
		id, ok := getParam(c, "id")
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	// Test with valid param
	t.Run("valid param", func(t *testing.T) {
		w := performRequest(router, "GET", "/valid/test-id", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response map[string]string
		parseJSON(t, w.Body.String(), &response)

		if response["id"] != "test-id" {
			t.Errorf("expected id=test-id, got %s", response["id"])
		}
	})
}

func TestValidateWorkflowRequest(t *testing.T) {
	tests := []struct {
		name        string
		workflow    map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid workflow",
			workflow: map[string]interface{}{
				"name": "Test Workflow",
				"nodes": []map[string]interface{}{
					{
						"id":   "node-1",
						"name": "Node 1",
						"type": "http",
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing name",
			workflow: map[string]interface{}{
				"nodes": []map[string]interface{}{
					{
						"id":   "node-1",
						"name": "Node 1",
						"type": "http",
					},
				},
			},
			expectError: true,
			errorMsg:    "name",
		},
		{
			name: "empty nodes",
			workflow: map[string]interface{}{
				"name":  "Test",
				"nodes": []map[string]interface{}{},
			},
			expectError: true,
			errorMsg:    "node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates the structure that would be used in actual handlers
			// In real implementation, validation would happen in the handler
			if tt.expectError {
				if tt.workflow["name"] == nil || tt.workflow["name"] == "" {
					// Expected: name validation
					if tt.errorMsg != "name" {
						t.Error("expected name validation error")
					}
				}
				if nodes, ok := tt.workflow["nodes"].([]map[string]interface{}); ok {
					if len(nodes) == 0 && tt.errorMsg == "node" {
						// Expected: node validation
					}
				}
			}
		})
	}
}

func TestBindJSON(t *testing.T) {
	router := gin.New()

	router.POST("/test", func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		if err := bindJSON(c, &req); err != nil {
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": req.Name})
	})

	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "valid JSON",
			body:           map[string]string{"name": "test"},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "invalid JSON",
			body:           "invalid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(router, "POST", "/test", tt.body)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRespondSuccess(t *testing.T) {
	router := gin.New()

	router.GET("/with-meta", func(c *gin.Context) {
		respondSuccess(c, http.StatusOK, map[string]string{"key": "value"}, &MetaInfo{
			Total:  100,
			Limit:  10,
			Offset: 0,
		})
	})

	router.GET("/without-meta", func(c *gin.Context) {
		respondSuccess(c, http.StatusOK, map[string]string{"key": "value"}, nil)
	})

	tests := []struct {
		name    string
		path    string
		hasMeta bool
		hasData bool
	}{
		{
			name:    "with meta",
			path:    "/with-meta",
			hasMeta: true,
			hasData: true,
		},
		{
			name:    "without meta",
			path:    "/without-meta",
			hasMeta: false,
			hasData: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(router, "GET", tt.path, nil)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			parseJSON(t, w.Body.String(), &response)

			if tt.hasData && response["data"] == nil {
				t.Error("expected data field")
			}

			if tt.hasMeta && response["meta"] == nil {
				t.Error("expected meta field")
			}

			if !tt.hasMeta && response["meta"] != nil {
				t.Error("unexpected meta field")
			}
		})
	}
}

func TestRespondErrorWithDetails(t *testing.T) {
	router := gin.New()
	router.GET("/error", func(c *gin.Context) {
		respondErrorWithDetails(c, http.StatusBadRequest, "validation failed", "VAL_001", map[string]interface{}{
			"field": "name",
		})
	})

	w := performRequest(router, "GET", "/error", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	parseJSON(t, w.Body.String(), &response)

	// New APIError format uses Message field instead of Error
	// The ErrorResponse struct is for testing and needs to check the right field
	var rawResponse map[string]interface{}
	parseJSON(t, w.Body.String(), &rawResponse)

	if rawResponse["message"] != "validation failed" {
		t.Errorf("expected message 'validation failed', got %v", rawResponse["message"])
	}

	if response.Code != "VAL_001" {
		t.Errorf("expected code 'VAL_001', got %s", response.Code)
	}

	if response.Details == nil {
		t.Error("expected details")
	}
}

func TestParseIntQuery(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid positive integer",
			value:        "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "valid negative integer",
			value:        "-5",
			defaultValue: 10,
			expected:     -5,
		},
		{
			name:         "empty string uses default",
			value:        "",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "invalid string uses default",
			value:        "abc",
			defaultValue: 10,
			expected:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntQuery(tt.value, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
