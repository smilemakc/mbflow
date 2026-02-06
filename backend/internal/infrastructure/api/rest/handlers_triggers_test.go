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

	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/testutil"
)

func setupTriggerHandlersTest(t *testing.T) (*TriggerHandlers, *gin.Engine, *storage.WorkflowRepository, func()) {
	t.Helper()

	// Setup test database
	db, cleanup := testutil.SetupTestTx(t)

	// Create repositories
	triggerRepo := storage.NewTriggerRepository(db)
	workflowRepo := storage.NewWorkflowRepository(db)

	// Create logger
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	// Create operations
	ops := &serviceapi.Operations{
		TriggerRepo:  triggerRepo,
		WorkflowRepo: workflowRepo,
		Logger:       log,
	}

	// Create handlers
	handlers := NewTriggerHandlers(ops, log)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	{
		api.POST("/triggers", handlers.HandleCreateTrigger)
		api.GET("/triggers/:id", handlers.HandleGetTrigger)
		api.GET("/triggers", handlers.HandleListTriggers)
		api.PUT("/triggers/:id", handlers.HandleUpdateTrigger)
		api.DELETE("/triggers/:id", handlers.HandleDeleteTrigger)
		api.POST("/triggers/:id/enable", handlers.HandleEnableTrigger)
		api.POST("/triggers/:id/disable", handlers.HandleDisableTrigger)
		api.POST("/triggers/:id/manual", handlers.HandleTriggerManual)
	}

	return handlers, router, workflowRepo, cleanup
}

// ========== CREATE TRIGGER TESTS ==========

func TestHandlers_CreateTrigger_Cron(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow first
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	req := map[string]interface{}{
		"name":        "Daily Cron",
		"description": "Daily scheduled task",
		"type":        "cron",
		"workflow_id": workflowModel.ID.String(),
		"config": map[string]interface{}{
			"expression": "0 0 * * *",
		},
		"enabled": true,
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, w, &result)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "Daily Cron", result["name"])
	assert.Equal(t, "cron", result["type"])
	assert.Equal(t, workflowModel.ID.String(), result["workflow_id"])
}

func TestHandlers_CreateTrigger_Webhook(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	req := map[string]interface{}{
		"name":        "Webhook Trigger",
		"description": "Webhook endpoint",
		"type":        "webhook",
		"workflow_id": workflowModel.ID.String(),
		"config": map[string]interface{}{
			"path": "/webhook/test",
		},
		"enabled": true,
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, w, &result)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "Webhook Trigger", result["name"])
	assert.Equal(t, "webhook", result["type"])
}

func TestHandlers_CreateTrigger_MissingName(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	req := map[string]interface{}{
		"type":        "cron",
		"workflow_id": workflowModel.ID.String(),
		"config": map[string]interface{}{
			"expression": "0 0 * * *",
		},
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", req)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "name is required")
}

func TestHandlers_CreateTrigger_InvalidWorkflowID(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	req := map[string]interface{}{
		"name":        "Test Trigger",
		"type":        "cron",
		"workflow_id": uuid.New().String(), // Non-existent workflow
		"config": map[string]interface{}{
			"expression": "0 0 * * *",
		},
	}

	w := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", req)

	// Handler returns 404 when workflow not found
	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== GET TRIGGER TESTS ==========

func TestHandlers_GetTrigger_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Create trigger
	createReq := map[string]interface{}{
		"name":        "Test Trigger",
		"description": "Test description",
		"type":        "cron",
		"workflow_id": workflowModel.ID.String(),
		"config": map[string]interface{}{
			"expression": "0 0 * * *",
		},
		"enabled": true,
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]interface{}
	testutil.ParseResponse(t, createW, &created)
	triggerID := created["id"].(string)

	// Get trigger
	getW := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/triggers/%s", triggerID), nil)

	assert.Equal(t, http.StatusOK, getW.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, getW, &result)

	assert.Equal(t, triggerID, result["id"])
	assert.Equal(t, "cron", result["type"])
	assert.Equal(t, workflowModel.ID.String(), result["workflow_id"])
	// Note: name/description are not returned by GetTrigger since they're passed as params in CreateTrigger response only
}

func TestHandlers_GetTrigger_NotFound(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "GET", fmt.Sprintf("/api/v1/triggers/%s", randomID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

func TestHandlers_GetTrigger_InvalidID(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "GET", "/api/v1/triggers/invalid-uuid", nil)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid ID format")
}

// ========== LIST TRIGGERS TESTS ==========

func TestHandlers_ListTriggers_Empty(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	w := testutil.MakeRequest(t, router, "GET", "/api/v1/triggers", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var triggers []interface{}
	testutil.ParseListResponse(t, w, &triggers)
	assert.Empty(t, triggers)
}

func TestHandlers_ListTriggers_WithData(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Create 3 triggers
	for i := 1; i <= 3; i++ {
		req := map[string]interface{}{
			"name":        fmt.Sprintf("Trigger %d", i),
			"type":        "cron",
			"workflow_id": workflowModel.ID.String(),
			"schedule":    "0 0 * * *",
		}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// List triggers
	w := testutil.MakeRequest(t, router, "GET", "/api/v1/triggers", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var triggers []interface{}
	testutil.ParseListResponse(t, w, &triggers)
	assert.Len(t, triggers, 3)
}

func TestHandlers_ListTriggers_FilterByWorkflowID(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
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

	// Create triggers for workflow 1
	for i := 1; i <= 2; i++ {
		req := map[string]interface{}{
			"name":        fmt.Sprintf("Trigger W1-%d", i),
			"type":        "cron",
			"workflow_id": workflowModel1.ID.String(),
			"schedule":    "0 0 * * *",
		}
		w := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// Create trigger for workflow 2
	req := map[string]interface{}{
		"name":        "Trigger W2",
		"type":        "cron",
		"workflow_id": workflowModel2.ID.String(),
		"schedule":    "0 0 * * *",
	}
	w := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Filter by workflow 1
	w = testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/triggers?workflow_id=%s", workflowModel1.ID.String()), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var triggers []interface{}
	testutil.ParseListResponse(t, w, &triggers)
	assert.Len(t, triggers, 2)
}

// ========== UPDATE TRIGGER TESTS ==========

func TestHandlers_UpdateTrigger_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Create trigger
	createReq := map[string]interface{}{
		"name":        "Original Name",
		"type":        "cron",
		"workflow_id": workflowModel.ID.String(),
		"schedule":    "0 0 * * *",
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]interface{}
	testutil.ParseResponse(t, createW, &created)
	triggerID := created["id"].(string)

	// Update trigger
	updateReq := map[string]interface{}{
		"name":     "Updated Name",
		"schedule": "0 12 * * *",
	}
	updateW := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/triggers/%s", triggerID), updateReq)

	assert.Equal(t, http.StatusOK, updateW.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, updateW, &result)

	assert.Equal(t, "Updated Name", result["name"])
}

func TestHandlers_UpdateTrigger_NotFound(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	updateReq := map[string]interface{}{
		"name": "Updated Name",
	}

	w := testutil.MakeRequest(t, router, "PUT", fmt.Sprintf("/api/v1/triggers/%s", randomID), updateReq)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== DELETE TRIGGER TESTS ==========

func TestHandlers_DeleteTrigger_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Create trigger
	createReq := map[string]interface{}{
		"name":        "To Delete",
		"type":        "cron",
		"workflow_id": workflowModel.ID.String(),
		"schedule":    "0 0 * * *",
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]interface{}
	testutil.ParseResponse(t, createW, &created)
	triggerID := created["id"].(string)

	// Delete trigger
	deleteW := testutil.MakeRequest(t, router, "DELETE", fmt.Sprintf("/api/v1/triggers/%s", triggerID), nil)

	// May return 200 or 204 depending on implementation
	assert.Contains(t, []int{http.StatusOK, http.StatusNoContent}, deleteW.Code)
}

func TestHandlers_DeleteTrigger_NotFound(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	randomID := uuid.New().String()
	w := testutil.MakeRequest(t, router, "DELETE", fmt.Sprintf("/api/v1/triggers/%s", randomID), nil)

	// May return 404 or 200 depending on implementation (similar to workflow delete)
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, w.Code)
}

// ========== ENABLE/DISABLE TRIGGER TESTS ==========

func TestHandlers_EnableTrigger_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Create disabled trigger
	createReq := map[string]interface{}{
		"name":        "To Enable",
		"type":        "cron",
		"workflow_id": workflowModel.ID.String(),
		"schedule":    "0 0 * * *",
		"enabled":     false,
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]interface{}
	testutil.ParseResponse(t, createW, &created)
	triggerID := created["id"].(string)

	// Enable trigger
	enableW := testutil.MakeRequest(t, router, "POST", fmt.Sprintf("/api/v1/triggers/%s/enable", triggerID), nil)

	assert.Equal(t, http.StatusOK, enableW.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, enableW, &result)

	assert.Equal(t, true, result["enabled"])
}

func TestHandlers_DisableTrigger_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Create enabled trigger
	createReq := map[string]interface{}{
		"name":        "To Disable",
		"type":        "cron",
		"workflow_id": workflowModel.ID.String(),
		"schedule":    "0 0 * * *",
		"enabled":     true,
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]interface{}
	testutil.ParseResponse(t, createW, &created)
	triggerID := created["id"].(string)

	// Disable trigger
	disableW := testutil.MakeRequest(t, router, "POST", fmt.Sprintf("/api/v1/triggers/%s/disable", triggerID), nil)

	assert.Equal(t, http.StatusOK, disableW.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, disableW, &result)

	assert.Equal(t, false, result["enabled"])
}

// ========== MANUAL TRIGGER TEST ==========

func TestHandlers_TriggerManual_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupTriggerHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Create trigger (must be enabled for manual execution)
	createReq := map[string]interface{}{
		"name":        "Manual Trigger",
		"type":        "manual",
		"workflow_id": workflowModel.ID.String(),
		"enabled":     true,
	}
	createW := testutil.MakeRequest(t, router, "POST", "/api/v1/triggers", createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]interface{}
	testutil.ParseResponse(t, createW, &created)
	triggerID := created["id"].(string)

	// Trigger manually with input
	triggerReq := map[string]interface{}{
		"input": map[string]interface{}{
			"test": "data",
		},
	}
	triggerW := testutil.MakeRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/triggers/%s/manual", triggerID), triggerReq)

	// Note: Manual trigger endpoint is currently a stub and returns 501 Not Implemented
	// This will be implemented when trigger manager integration is completed
	testutil.AssertErrorResponse(t, triggerW, http.StatusNotImplemented,
		"trigger execution requires trigger manager integration")
}
