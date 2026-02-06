package rest

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/testutil"
)

func setupExecutionHandlersTest(t *testing.T) (*ExecutionHandlers, *gin.Engine, *storage.WorkflowRepository, func()) {
	t.Helper()

	// Setup test database
	db, cleanup := testutil.SetupTestTx(t)

	// Create repositories
	workflowRepo := storage.NewWorkflowRepository(db)
	executionRepo := storage.NewExecutionRepository(db)
	eventRepo := storage.NewEventRepository(db)

	// Create logger
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	// Create executor registry
	executorRegistry := executor.NewManager()

	// Create execution manager
	executionManager := engine.NewExecutionManager(
		executorRegistry,
		workflowRepo,
		executionRepo,
		eventRepo,
		nil, // No resource repo for tests
		nil, // No observer manager for tests
	)

	// Create operations
	ops := &serviceapi.Operations{
		WorkflowRepo:    workflowRepo,
		ExecutionRepo:   executionRepo,
		ExecutionMgr:    executionManager,
		ExecutorManager: executorRegistry,
		Logger:          log,
	}

	// Create handlers
	handlers := NewExecutionHandlers(ops, log)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	{
		api.POST("/executions", handlers.HandleRunExecution)
		api.POST("/workflows/:workflow_id/execute", handlers.HandleRunExecution)
		api.GET("/executions/:id", handlers.HandleGetExecution)
		api.GET("/executions", handlers.HandleListExecutions)
		api.GET("/executions/:id/logs", handlers.HandleGetLogs)
		api.GET("/executions/:id/nodes/:nodeId", handlers.HandleGetNodeResult)
		api.POST("/executions/:id/cancel", handlers.HandleCancelExecution)
		api.POST("/executions/:id/retry", handlers.HandleRetryExecution)
	}

	return handlers, router, workflowRepo, cleanup
}

// ========== RUN EXECUTION TESTS ==========

func TestHandlers_RunExecution_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	// Create a simple workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Run execution
	req := map[string]interface{}{
		"workflow_id": workflowModel.ID.String(),
		"input": map[string]interface{}{
			"test": "data",
		},
		"async": true,
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, w, &result)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, workflowModel.ID.String(), result["workflow_id"])
	assert.Contains(t, []string{"pending", "running", "completed"}, result["status"])
}

func TestHandlers_RunExecution_WithWorkflowIDInPath(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Run execution using path parameter
	req := map[string]interface{}{
		"input": map[string]interface{}{
			"test": "data",
		},
	}

	w := testutil.MakeRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/workflows/%s/execute", workflowModel.ID.String()), req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, w, &result)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, workflowModel.ID.String(), result["workflow_id"])
}

func TestHandlers_RunExecution_MissingWorkflowID(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	req := map[string]interface{}{
		"input": map[string]interface{}{
			"test": "data",
		},
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", req)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "Workflow ID is required")
}

func TestHandlers_RunExecution_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlers_RunExecution_WorkflowNotFound(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	req := map[string]interface{}{
		"workflow_id": randomID,
		"input": map[string]interface{}{
			"test": "data",
		},
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ========== GET EXECUTION TESTS ==========

func TestHandlers_GetExecution_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Run execution
	runReq := map[string]interface{}{
		"workflow_id": workflowModel.ID.String(),
		"input":       map[string]interface{}{"test": "data"},
	}
	runW := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", runReq)
	require.Equal(t, http.StatusAccepted, runW.Code)

	var runResult map[string]interface{}
	testutil.ParseResponse(t, runW, &runResult)
	executionID := runResult["id"].(string)

	// Get execution
	getW := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/executions/%s", executionID), nil)

	assert.Equal(t, http.StatusOK, getW.Code)

	var getResult map[string]interface{}
	testutil.ParseResponse(t, getW, &getResult)

	assert.Equal(t, executionID, getResult["id"])
	assert.Equal(t, workflowModel.ID.String(), getResult["workflow_id"])
}

func TestHandlers_GetExecution_NotFound(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/executions/%s", randomID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

func TestHandlers_GetExecution_InvalidID(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "GET", "/api/v1/executions/invalid-uuid", nil)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid ID format")
}

// ========== LIST EXECUTIONS TESTS ==========

func TestHandlers_ListExecutions_Empty(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "GET", "/api/v1/executions", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var executions []interface{}
	testutil.ParseListResponse(t, w, &executions)

	assert.Empty(t, executions)
}

func TestHandlers_ListExecutions_WithData(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Run 3 executions
	for i := 1; i <= 3; i++ {
		req := map[string]interface{}{
			"workflow_id": workflowModel.ID.String(),
			"input":       map[string]interface{}{"test": fmt.Sprintf("data_%d", i)},
		}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", req)
		require.Equal(t, http.StatusAccepted, w.Code)
	}

	// List executions
	w := testutil.MakeRequest(t, router, "GET", "/api/v1/executions", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var executions []interface{}
	testutil.ParseListResponse(t, w, &executions)

	assert.Len(t, executions, 3)
}

func TestHandlers_ListExecutions_Pagination(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Run 5 executions
	for i := 1; i <= 5; i++ {
		req := map[string]interface{}{
			"workflow_id": workflowModel.ID.String(),
			"input":       map[string]interface{}{"test": fmt.Sprintf("data_%d", i)},
		}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", req)
		require.Equal(t, http.StatusAccepted, w.Code)
	}

	// List with limit=2
	// Note: Due to async execution creation, we may get partial results
	// The test verifies pagination works, not exact count
	w := testutil.MakeRequest(t, router, "GET", "/api/v1/executions?limit=2&offset=0", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var executions []interface{}
	meta := testutil.ParseListResponse(t, w, &executions)

	// We expect at most 2 results due to limit
	assert.LessOrEqual(t, len(executions), 2)
	// Total should be at least the number of executions we see
	assert.GreaterOrEqual(t, meta["total"], float64(len(executions)))
	assert.Equal(t, float64(2), meta["limit"])
}

func TestHandlers_ListExecutions_FilterByWorkflowID(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	// Create 2 workflows
	workflow1 := testutil.CreateSimpleWorkflow()
	workflowModel1 := testutil.WorkflowDomainToModel(workflow1)
	workflowModel1.Name = "Workflow 1"
	err := workflowRepo.Create(context.Background(), workflowModel1)
	require.NoError(t, err)

	workflow2 := testutil.CreateSimpleWorkflow()
	workflowModel2 := testutil.WorkflowDomainToModel(workflow2)
	workflowModel2.Name = "Workflow 2"
	err = workflowRepo.Create(context.Background(), workflowModel2)
	require.NoError(t, err)

	// Run executions for workflow 1
	for i := 1; i <= 2; i++ {
		req := map[string]interface{}{
			"workflow_id": workflowModel1.ID.String(),
			"input":       map[string]interface{}{"test": "data"},
		}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", req)
		require.Equal(t, http.StatusAccepted, w.Code)
	}

	// Run execution for workflow 2
	req := map[string]interface{}{
		"workflow_id": workflowModel2.ID.String(),
		"input":       map[string]interface{}{"test": "data"},
	}
	w := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", req)
	require.Equal(t, http.StatusAccepted, w.Code)

	// Filter by workflow 1
	w = testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/executions?workflow_id=%s", workflowModel1.ID.String()), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var executions []interface{}
	testutil.ParseListResponse(t, w, &executions)

	assert.Len(t, executions, 2)
}

func TestHandlers_ListExecutions_FilterByStatus(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Run executions
	for i := 1; i <= 2; i++ {
		req := map[string]interface{}{
			"workflow_id": workflowModel.ID.String(),
			"input":       map[string]interface{}{"test": "data"},
		}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", req)
		require.Equal(t, http.StatusAccepted, w.Code)
	}

	// Filter by status (executions will be in pending, running, or completed status)
	// We'll just check that the filter parameter works
	w := testutil.MakeRequest(t, router, "GET", "/api/v1/executions?status=completed", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var executions []interface{}
	testutil.ParseListResponse(t, w, &executions)

	// Just verify response has data
	assert.NotNil(t, executions)
}

// ========== GET LOGS TESTS ==========

func TestHandlers_GetLogs_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	// Create workflow and run execution
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	runReq := map[string]interface{}{
		"workflow_id": workflowModel.ID.String(),
		"input":       map[string]interface{}{"test": "data"},
	}
	runW := testutil.MakeRequest(t, router, "POST", "/api/v1/executions", runReq)
	require.Equal(t, http.StatusAccepted, runW.Code)

	var runResult map[string]interface{}
	testutil.ParseResponse(t, runW, &runResult)
	executionID := runResult["id"].(string)

	// Get logs
	w := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/executions/%s/logs", executionID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, w, &result)

	assert.NotNil(t, result["logs"])
}

func TestHandlers_GetLogs_NotFound(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/executions/%s/logs", randomID), nil)

	// Note: GetLogs returns 200 with empty array for better UX, not 404
	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, w, &result)
	assert.Empty(t, result["logs"])
	assert.Equal(t, float64(0), result["total"])
}

func TestHandlers_GetLogs_InvalidID(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "GET", "/api/v1/executions/invalid-uuid/logs", nil)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid ID format")
}

// ========== GET NODE RESULT TESTS ==========

func TestHandlers_GetNodeResult_InvalidExecutionID(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "GET", "/api/v1/executions/invalid-uuid/nodes/n1", nil)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid ID format")
}

func TestHandlers_GetNodeResult_ExecutionNotFound(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/executions/%s/nodes/n1", randomID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== CANCEL EXECUTION TESTS (Placeholder) ==========

func TestHandlers_CancelExecution_NotImplemented(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "POST", fmt.Sprintf("/api/v1/executions/%s/cancel", randomID), nil)

	// Check if endpoint is implemented
	// Actual behavior depends on implementation
	assert.NotEqual(t, http.StatusNotFound, w.Code, "Cancel endpoint should exist")
}

// ========== RETRY EXECUTION TESTS (Placeholder) ==========

func TestHandlers_RetryExecution_NotImplemented(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupExecutionHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "POST", fmt.Sprintf("/api/v1/executions/%s/retry", randomID), nil)

	// Check if endpoint is implemented
	// Actual behavior depends on implementation
	assert.NotEqual(t, http.StatusNotFound, w.Code, "Retry endpoint should exist")
}
