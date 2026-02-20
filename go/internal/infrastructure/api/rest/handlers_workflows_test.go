package rest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func setupWorkflowHandlersTest(t *testing.T) (*WorkflowHandlers, *gin.Engine, func()) {
	t.Helper()

	// Setup test database
	db, cleanup := testutil.SetupTestTx(t)

	// Create repository
	workflowRepo := storage.NewWorkflowRepository(db)

	// Create logger with minimal config
	log := logger.New(config.LoggingConfig{
		Level:  "error", // Minimal logging for tests
		Format: "text",
	})

	// Create executor manager and register executors
	executorManager := executor.NewManager()
	if err := builtin.RegisterBuiltins(executorManager); err != nil {
		t.Fatalf("Failed to register builtins: %v", err)
	}
	if err := builtin.RegisterAdapters(executorManager); err != nil {
		t.Fatalf("Failed to register adapters: %v", err)
	}

	// Create operations struct
	ops := &serviceapi.Operations{
		WorkflowRepo:    workflowRepo,
		ExecutorManager: executorManager,
		Logger:          log,
	}

	// Create handlers
	handlers := NewWorkflowHandlers(ops, log)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	{
		api.POST("/workflows", handlers.HandleCreateWorkflow)
		api.GET("/workflows/:workflow_id", handlers.HandleGetWorkflow)
		api.GET("/workflows", handlers.HandleListWorkflows)
		api.PUT("/workflows/:workflow_id", handlers.HandleUpdateWorkflow)
		api.DELETE("/workflows/:workflow_id", handlers.HandleDeleteWorkflow)
		api.POST("/workflows/:workflow_id/publish", handlers.HandlePublishWorkflow)
		api.POST("/workflows/:workflow_id/unpublish", handlers.HandleUnpublishWorkflow)
		api.GET("/workflows/:workflow_id/diagram", handlers.HandleGetWorkflowDiagram)
	}

	return handlers, router, cleanup
}

// ========== CREATE WORKFLOW TESTS ==========

func TestHandlers_CreateWorkflow_Success(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	req := map[string]any{
		"name":        "Test Workflow",
		"description": "Test Description",
		"variables": map[string]any{
			"api_key": "test-key",
		},
		"metadata": map[string]any{
			"author": "test-user",
		},
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]any
	testutil.ParseResponse(t, w, &result)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "Test Workflow", result["name"])
	assert.Equal(t, "Test Description", result["description"])
	assert.Equal(t, "draft", result["status"])
}

func TestHandlers_CreateWorkflow_MissingName(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	req := map[string]any{
		"description": "Test Description",
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", req)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "name is required")
}

func TestHandlers_CreateWorkflow_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/workflows", nil)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlers_CreateWorkflow_WithMinimalData(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	req := map[string]any{
		"name": "Minimal Workflow",
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]any
	testutil.ParseResponse(t, w, &result)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "Minimal Workflow", result["name"])
}

// ========== GET WORKFLOW TESTS ==========

func TestHandlers_GetWorkflow_Success(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create workflow first
	createReq := map[string]any{
		"name":        "Test Workflow",
		"description": "Test Description",
	}

	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]any
	testutil.ParseResponse(t, createW, &created)
	workflowID := created["id"].(string)

	// Get workflow
	getW := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/workflows/%s", workflowID), nil)

	assert.Equal(t, http.StatusOK, getW.Code)

	var result map[string]any
	testutil.ParseResponse(t, getW, &result)

	assert.Equal(t, workflowID, result["id"])
	assert.Equal(t, "Test Workflow", result["name"])
}

func TestHandlers_GetWorkflow_NotFound(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/workflows/%s", randomID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

func TestHandlers_GetWorkflow_InvalidID(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "GET", "/api/v1/workflows/invalid-uuid", nil)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid ID format")
}

// ========== LIST WORKFLOWS TESTS ==========

func TestHandlers_ListWorkflows_Empty(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "GET", "/api/v1/workflows", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var workflows []any
	meta := testutil.ParseListResponse(t, w, &workflows)

	assert.Empty(t, workflows)
	assert.Equal(t, float64(0), meta["total"])
}

func TestHandlers_ListWorkflows_WithData(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create 3 workflows
	for i := 1; i <= 3; i++ {
		req := map[string]any{
			"name": fmt.Sprintf("Workflow %d", i),
		}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// List workflows
	w := testutil.MakeRequest(t, router, "GET", "/api/v1/workflows", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var workflows []any
	meta := testutil.ParseListResponse(t, w, &workflows)

	assert.Len(t, workflows, 3)
	assert.Equal(t, float64(3), meta["total"])
}

func TestHandlers_ListWorkflows_Pagination(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create 5 workflows
	for i := 1; i <= 5; i++ {
		req := map[string]any{
			"name": fmt.Sprintf("Workflow %d", i),
		}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// List with limit=2, offset=0
	w := testutil.MakeRequest(t, router, "GET", "/api/v1/workflows?limit=2&offset=0", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var workflows []any
	meta := testutil.ParseListResponse(t, w, &workflows)

	assert.Len(t, workflows, 2)

	assert.Equal(t, float64(5), meta["total"])
	assert.Equal(t, float64(2), meta["limit"])
	assert.Equal(t, float64(0), meta["offset"])
}

func TestHandlers_ListWorkflows_FilterByStatus(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create 2 workflows (both will be draft by default)
	for i := 1; i <= 2; i++ {
		req := map[string]any{"name": fmt.Sprintf("Workflow %d", i)}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// Filter by status=draft
	w := testutil.MakeRequest(t, router, "GET", "/api/v1/workflows?status=draft", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var workflows []any
	testutil.ParseListResponse(t, w, &workflows)

	assert.Len(t, workflows, 2)
}

// ========== UPDATE WORKFLOW TESTS ==========

func TestHandlers_UpdateWorkflow_Success(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create workflow
	createReq := map[string]any{
		"name": "Original Name",
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]any
	testutil.ParseResponse(t, createW, &created)
	workflowID := created["id"].(string)

	// Update workflow
	updateReq := map[string]any{
		"name":        "Updated Name",
		"description": "Updated Description",
	}
	updateW := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), updateReq)

	assert.Equal(t, http.StatusOK, updateW.Code)

	var result map[string]any
	testutil.ParseResponse(t, updateW, &result)

	assert.Equal(t, "Updated Name", result["name"])
	assert.Equal(t, "Updated Description", result["description"])
}

func TestHandlers_UpdateWorkflow_NotFound(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	updateReq := map[string]any{
		"name": "Updated Name",
	}

	w := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/workflows/%s", randomID), updateReq)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

func TestHandlers_UpdateWorkflow_InvalidID(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	updateReq := map[string]any{
		"name": "Updated Name",
	}

	w := testutil.MakeRequest(t, router, "PUT", "/api/v1/workflows/invalid-uuid", updateReq)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid ID format")
}

// ========== DELETE WORKFLOW TESTS ==========

func TestHandlers_DeleteWorkflow_Success(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create workflow
	createReq := map[string]any{
		"name": "To Delete",
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]any
	testutil.ParseResponse(t, createW, &created)
	workflowID := created["id"].(string)

	// Delete workflow
	deleteW := testutil.MakeRequest(t, router, "DELETE", fmt.Sprintf("/api/v1/workflows/%s", workflowID), nil)

	assert.Equal(t, http.StatusOK, deleteW.Code)

	var deleteResult map[string]any
	testutil.ParseResponse(t, deleteW, &deleteResult)
	assert.Equal(t, "workflow deleted successfully", deleteResult["message"])

	// Verify deletion - should return 404
	getW := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/workflows/%s", workflowID), nil)
	assert.Equal(t, http.StatusNotFound, getW.Code)
}

func TestHandlers_DeleteWorkflow_NotFound(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "DELETE", fmt.Sprintf("/api/v1/workflows/%s", randomID), nil)

	// Note: Current implementation returns 200 OK even when workflow doesn't exist
	// This is because the repository Delete method doesn't check if rows were affected
	// TODO: Consider returning 404 when workflow is not found
	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]any
	testutil.ParseResponse(t, w, &result)
	assert.Equal(t, "workflow deleted successfully", result["message"])
}

func TestHandlers_DeleteWorkflow_InvalidID(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "DELETE", "/api/v1/workflows/invalid-uuid", nil)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid ID format")
}

// ========== PUBLISH/UNPUBLISH TESTS ==========

func TestHandlers_PublishWorkflow_Success(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create workflow
	createReq := map[string]any{
		"name": "To Publish",
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]any
	testutil.ParseResponse(t, createW, &created)
	workflowID := created["id"].(string)

	// Publish workflow
	publishW := testutil.MakeRequest(t, router, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)

	assert.Equal(t, http.StatusOK, publishW.Code)

	var result map[string]any
	testutil.ParseResponse(t, publishW, &result)

	assert.Equal(t, "active", result["status"])
}

func TestHandlers_UnpublishWorkflow_Success(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create and publish workflow
	createReq := map[string]any{
		"name": "To Unpublish",
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]any
	testutil.ParseResponse(t, createW, &created)
	workflowID := created["id"].(string)

	// Publish first
	publishW := testutil.MakeRequest(t, router, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)
	require.Equal(t, http.StatusOK, publishW.Code)

	// Unpublish
	unpublishW := testutil.MakeRequest(t, router, "POST", fmt.Sprintf("/api/v1/workflows/%s/unpublish", workflowID), nil)

	assert.Equal(t, http.StatusOK, unpublishW.Code)

	var result map[string]any
	testutil.ParseResponse(t, unpublishW, &result)

	assert.Equal(t, "draft", result["status"])
}

// ========== DIAGRAM TESTS ==========

func TestHandlers_GetWorkflowDiagram_Success(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create workflow
	createReq := map[string]any{
		"name": "Diagram Test",
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]any
	testutil.ParseResponse(t, createW, &created)
	workflowID := created["id"].(string)

	// Get diagram with default format (mermaid)
	w := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/workflows/%s/diagram", workflowID), nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))

	// Check that diagram is returned as plain text
	diagram := w.Body.String()
	assert.NotEmpty(t, diagram)
	// Mermaid diagrams typically start with "flowchart" or "graph"
	assert.True(t, len(diagram) > 0, "Diagram should not be empty")
}

func TestHandlers_GetWorkflowDiagram_ASCIIFormat(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create workflow
	createReq := map[string]any{
		"name": "ASCII Diagram Test",
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]any
	testutil.ParseResponse(t, createW, &created)
	workflowID := created["id"].(string)

	// Get diagram with ASCII format
	w := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/workflows/%s/diagram?format=ascii", workflowID), nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))

	// Check that diagram is returned as plain text
	diagram := w.Body.String()
	assert.NotEmpty(t, diagram)
	assert.True(t, len(diagram) > 0, "ASCII diagram should not be empty")
}

func TestHandlers_GetWorkflowDiagram_NotFound(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/workflows/%s/diagram", randomID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== USER FILTER TESTS ==========

func setupWorkflowHandlersTestWithAuth(t *testing.T, userID string, isAdmin bool) (*WorkflowHandlers, *gin.Engine, func()) {
	t.Helper()

	// Setup test database
	db, cleanup := testutil.SetupTestTx(t)

	// Create repository
	workflowRepo := storage.NewWorkflowRepository(db)

	// Create logger with minimal config
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	// Create executor manager and register executors
	executorManager := executor.NewManager()
	if err := builtin.RegisterBuiltins(executorManager); err != nil {
		t.Fatalf("Failed to register builtins: %v", err)
	}
	if err := builtin.RegisterAdapters(executorManager); err != nil {
		t.Fatalf("Failed to register adapters: %v", err)
	}

	// Create operations struct
	ops := &serviceapi.Operations{
		WorkflowRepo:    workflowRepo,
		ExecutorManager: executorManager,
		Logger:          log,
	}

	// Create handlers
	handlers := NewWorkflowHandlers(ops, log)

	// Setup router with auth middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add mock auth middleware
	router.Use(testutil.MockAuthMiddleware(userID, isAdmin))

	api := router.Group("/api/v1")
	{
		api.POST("/workflows", handlers.HandleCreateWorkflow)
		api.GET("/workflows/:workflow_id", handlers.HandleGetWorkflow)
		api.GET("/workflows", handlers.HandleListWorkflows)
		api.PUT("/workflows/:workflow_id", handlers.HandleUpdateWorkflow)
		api.DELETE("/workflows/:workflow_id", handlers.HandleDeleteWorkflow)
		api.POST("/workflows/:workflow_id/publish", handlers.HandlePublishWorkflow)
		api.POST("/workflows/:workflow_id/unpublish", handlers.HandleUnpublishWorkflow)
		api.GET("/workflows/:workflow_id/diagram", handlers.HandleGetWorkflowDiagram)
	}

	return handlers, router, cleanup
}

func TestHandlers_CreateWorkflow_WithoutAuthentication(t *testing.T) {
	t.Parallel()
	// No auth middleware - created_by should be empty
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	req := map[string]any{
		"name": "Anonymous Workflow",
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]any
	testutil.ParseResponse(t, w, &result)

	assert.NotEmpty(t, result["id"])
	// created_by should be empty for unauthenticated requests
	createdBy, exists := result["created_by"]
	assert.True(t, !exists || createdBy == "" || createdBy == nil,
		"created_by should be empty for unauthenticated requests, got: %v", createdBy)
}

func TestHandlers_ListWorkflows_FilterByUserID_InvalidFormat(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "GET", "/api/v1/workflows?user_id=invalid-uuid", nil)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid user_id format")
}

func TestHandlers_ListWorkflows_FilterByUserID_Admin(t *testing.T) {
	t.Parallel()
	adminUserID := uuid.New().String()
	otherUserID := uuid.New()

	// Setup as admin
	_, adminRouter, cleanup := setupWorkflowHandlersTestWithAuth(t, adminUserID, true)
	defer cleanup()

	// Admin can filter by any user_id without error
	w := testutil.MakeRequest(t, adminRouter, "GET",
		fmt.Sprintf("/api/v1/workflows?user_id=%s", otherUserID.String()), nil)

	// Admin should be able to filter by any user_id
	assert.Equal(t, http.StatusOK, w.Code)

	var workflows []any
	testutil.ParseListResponse(t, w, &workflows)

	// Should return empty list (no workflows for that user)
	assert.Empty(t, workflows)
}

func TestHandlers_ListWorkflows_FilterByUserID_Forbidden(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	otherUserID := uuid.New().String()

	// Setup as regular user
	_, userRouter, cleanup := setupWorkflowHandlersTestWithAuth(t, userID, false)
	defer cleanup()

	// Try to filter by another user's ID
	w := testutil.MakeRequest(t, userRouter, "GET",
		fmt.Sprintf("/api/v1/workflows?user_id=%s", otherUserID), nil)

	// Non-admin should get forbidden
	testutil.AssertErrorResponse(t, w, http.StatusForbidden, "You can only view your own workflows")
}

func TestHandlers_ListWorkflows_FilterByOwnUserID_Empty(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()

	// Setup as regular user
	_, userRouter, cleanup := setupWorkflowHandlersTestWithAuth(t, userID, false)
	defer cleanup()

	// Filter by own user_id - should return empty list (no workflows created)
	w := testutil.MakeRequest(t, userRouter, "GET",
		fmt.Sprintf("/api/v1/workflows?user_id=%s", userID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var workflows []any
	testutil.ParseListResponse(t, w, &workflows)

	assert.Empty(t, workflows)
}

func TestHandlers_ListWorkflows_UnauthenticatedSeeAll(t *testing.T) {
	t.Parallel()
	// No auth - unauthenticated users see all workflows (backward compatibility)
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	// Create some workflows without auth (created_by = NULL)
	for i := 1; i <= 3; i++ {
		req := map[string]any{
			"name": fmt.Sprintf("Workflow %d", i),
		}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// List without auth - should see all workflows
	w := testutil.MakeRequest(t, router, "GET", "/api/v1/workflows", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var workflows []any
	testutil.ParseListResponse(t, w, &workflows)

	assert.Len(t, workflows, 3)
}
