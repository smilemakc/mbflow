package rest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/executor/builtin"
	"github.com/smilemakc/mbflow/go/testutil"
)

func setupValidationTest(t *testing.T) (*gin.Engine, string, func()) {
	t.Helper()

	db, cleanup := testutil.SetupTestTx(t)
	workflowRepo := storage.NewWorkflowRepository(db)
	log := logger.New(config.LoggingConfig{Level: "error", Format: "text"})

	executorManager := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executorManager))
	require.NoError(t, builtin.RegisterAdapters(executorManager))

	ops := &serviceapi.Operations{
		WorkflowRepo:    workflowRepo,
		ExecutorManager: executorManager,
		Logger:          log,
	}
	handlers := NewWorkflowHandlers(ops, log)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/workflows", handlers.HandleCreateWorkflow)
	api.PUT("/workflows/:workflow_id", handlers.HandleUpdateWorkflow)

	createReq := map[string]any{"name": "Validation Test Workflow"}
	w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, w.Code)
	var created map[string]any
	testutil.ParseResponse(t, w, &created)
	workflowID := created["id"].(string)

	return router, workflowID, cleanup
}

// TestValidateNodes_ValidTypes tests that all registered executor types are valid
func TestValidateNodes_ValidTypes(t *testing.T) {
	t.Parallel()
	validTypes := []string{
		"http", "transform", "llm", "conditional", "merge",
		"html_clean", "rss_parser", "base64_to_bytes",
	}

	for _, nodeType := range validTypes {
		t.Run("valid_"+nodeType, func(t *testing.T) {
			router, workflowID, cleanup := setupValidationTest(t)
			defer cleanup()

			updateReq := map[string]any{
				"name": "Updated Workflow",
				"nodes": []map[string]any{
					{
						"id":   "node-1",
						"name": "Test Node",
						"type": nodeType,
					},
				},
			}

			w := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), updateReq)
			assert.Equal(t, http.StatusOK, w.Code, "Type %s should be valid. Response: %s", nodeType, w.Body.String())
		})
	}
}

// TestValidateNodes_UIOnlyTypes tests that UI-only types are valid
func TestValidateNodes_UIOnlyTypes(t *testing.T) {
	t.Parallel()
	router, workflowID, cleanup := setupValidationTest(t)
	defer cleanup()

	updateReq := map[string]any{
		"name": "Updated Workflow",
		"nodes": []map[string]any{
			{
				"id":   "comment-1",
				"name": "Comment Node",
				"type": "comment",
			},
		},
	}

	w := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), updateReq)
	assert.Equal(t, http.StatusOK, w.Code, "Comment type should be valid (UI-only). Response: %s", w.Body.String())
}

// TestValidateNodes_InvalidType tests that unregistered types are rejected
func TestValidateNodes_InvalidType(t *testing.T) {
	t.Parallel()
	router, workflowID, cleanup := setupValidationTest(t)
	defer cleanup()

	updateReq := map[string]any{
		"name": "Updated Workflow",
		"nodes": []map[string]any{
			{
				"id":   "node-1",
				"name": "Invalid Node",
				"type": "nonexistent_type",
			},
		},
	}

	w := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), updateReq)
	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "invalid type 'nonexistent_type'")
}

// TestValidateNodes_MixedTypes tests validation with both valid and invalid types
func TestValidateNodes_MixedTypes(t *testing.T) {
	t.Parallel()
	router, workflowID, cleanup := setupValidationTest(t)
	defer cleanup()

	validNodes := []map[string]any{
		{"id": "node-1", "name": "HTTP Node", "type": "http"},
		{"id": "node-2", "name": "Transform Node", "type": "transform"},
		{"id": "node-3", "name": "Comment Node", "type": "comment"},
		{"id": "node-4", "name": "RSS Node", "type": "rss_parser"},
	}

	updateReq := map[string]any{
		"name":  "Updated Workflow",
		"nodes": validNodes,
	}

	w := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), updateReq)
	assert.Equal(t, http.StatusOK, w.Code, "All valid nodes should pass. Response: %s", w.Body.String())

	invalidNodes := []map[string]any{
		{"id": "node-1", "name": "HTTP Node", "type": "http"},
		{"id": "node-2", "name": "Invalid Node", "type": "invalid_type"},
	}

	updateReq = map[string]any{
		"name":  "Updated Workflow",
		"nodes": invalidNodes,
	}

	w = testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), updateReq)
	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "invalid type 'invalid_type'")
}

// TestValidateNodes_RequiredFields tests required field validation
func TestValidateNodes_RequiredFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		node        map[string]any
		expectedErr string
	}{
		{
			name:        "missing_id",
			node:        map[string]any{"name": "Test", "type": "http"},
			expectedErr: "id",
		},
		{
			name:        "missing_name",
			node:        map[string]any{"id": "node-1", "type": "http"},
			expectedErr: "name",
		},
		{
			name:        "missing_type",
			node:        map[string]any{"id": "node-1", "name": "Test"},
			expectedErr: "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, workflowID, cleanup := setupValidationTest(t)
			defer cleanup()

			updateReq := map[string]any{
				"name":  "Updated Workflow",
				"nodes": []map[string]any{tt.node},
			}

			w := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), updateReq)
			assert.Equal(t, http.StatusBadRequest, w.Code, "Should fail validation for %s", tt.name)

			var errorResp map[string]any
			testutil.ParseResponse(t, w, &errorResp)

			message, ok := errorResp["message"]
			if !ok {
				message = errorResp["error"]
			}
			messageStr := fmt.Sprintf("%v", message)
			assert.True(t, strings.Contains(strings.ToLower(messageStr), strings.ToLower(tt.expectedErr)),
				"Error message should contain '%s', got: %s", tt.expectedErr, messageStr)
		})
	}
}

// TestValidateNodes_DuplicateIDs tests duplicate node ID validation
func TestValidateNodes_DuplicateIDs(t *testing.T) {
	t.Parallel()
	router, workflowID, cleanup := setupValidationTest(t)
	defer cleanup()

	updateReq := map[string]any{
		"name": "Updated Workflow",
		"nodes": []map[string]any{
			{"id": "node-1", "name": "First", "type": "http"},
			{"id": "node-1", "name": "Duplicate", "type": "transform"},
		},
	}

	w := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), updateReq)
	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "duplicate node id: node-1")
}

// TestValidateNodes_FieldLengths tests field length validation
func TestValidateNodes_FieldLengths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		node        map[string]any
		expectedErr string
	}{
		{
			name: "id_too_long",
			node: map[string]any{
				"id":   strings.Repeat("a", 101),
				"name": "Test",
				"type": "http",
			},
			expectedErr: "node id too long",
		},
		{
			name: "name_too_long",
			node: map[string]any{
				"id":   "node-1",
				"name": strings.Repeat("a", 256),
				"type": "http",
			},
			expectedErr: "name too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, workflowID, cleanup := setupValidationTest(t)
			defer cleanup()

			updateReq := map[string]any{
				"name":  "Updated Workflow",
				"nodes": []map[string]any{tt.node},
			}

			w := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), updateReq)
			testutil.AssertErrorResponse(t, w, http.StatusBadRequest, tt.expectedErr)
		})
	}
}
