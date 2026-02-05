package rest

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupNodeHandlersTest(t *testing.T) (*NodeHandlers, *gin.Engine, *storage.WorkflowRepository, func()) {
	testDB := testutil.SetupTestDB(t)
	workflowRepo := storage.NewWorkflowRepository(testDB.DB)
	log := logger.New(config.LoggingConfig{Level: "error", Format: "text"})
	handlers := NewNodeHandlers(workflowRepo, log)

	router := gin.New()
	api := router.Group("/api/v1/workflows/:workflow_id")
	{
		api.POST("/nodes", handlers.HandleAddNode)
		api.GET("/nodes", handlers.HandleListNodes)
		api.GET("/nodes/:nodeId", handlers.HandleGetNode)
		api.PUT("/nodes/:nodeId", handlers.HandleUpdateNode)
		api.DELETE("/nodes/:nodeId", handlers.HandleDeleteNode)
	}

	return handlers, router, workflowRepo, func() { testDB.Cleanup(t) }
}

// ========== ADD NODE TESTS ==========

func TestHandlers_AddNode_Success(t *testing.T) {
	_, router, workflowRepo, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	// Create workflow
	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	req := map[string]interface{}{
		"id":   "new_node",
		"name": "New Node",
		"type": "transform",
		"config": map[string]interface{}{
			"mode": "passthrough",
		},
		"position": map[string]interface{}{
			"x": 100,
			"y": 200,
		},
	}

	w := testutil.MakeRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/workflows/%s/nodes", workflowModel.ID), req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, w, &result)
	assert.Equal(t, "new_node", result["id"])
	assert.Equal(t, "New Node", result["name"])
}

func TestHandlers_AddNode_WorkflowNotFound(t *testing.T) {
	_, router, _, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	req := map[string]interface{}{
		"id":   "new_node",
		"name": "New Node",
		"type": "transform",
	}

	randomID := uuid.New()
	w := testutil.MakeRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/workflows/%s/nodes", randomID), req)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

func TestHandlers_AddNode_DuplicateID(t *testing.T) {
	_, router, workflowRepo, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Try to add node with existing ID
	req := map[string]interface{}{
		"id":   "n1", // This ID already exists in simple workflow
		"name": "Duplicate",
		"type": "transform",
	}

	w := testutil.MakeRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/workflows/%s/nodes", workflowModel.ID), req)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "node with this ID already exists")
}

// ========== LIST NODES TESTS ==========

func TestHandlers_ListNodes_Success(t *testing.T) {
	_, router, workflowRepo, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow() // Has 3 nodes
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	w := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/nodes", workflowModel.ID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var nodes []interface{}
	testutil.ParseListResponse(t, w, &nodes)
	assert.Len(t, nodes, 3)
}

func TestHandlers_ListNodes_WorkflowNotFound(t *testing.T) {
	_, router, _, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	randomID := uuid.New()
	w := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/nodes", randomID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== GET NODE TESTS ==========

func TestHandlers_GetNode_Success(t *testing.T) {
	_, router, workflowRepo, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	w := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/nodes/n1", workflowModel.ID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, w, &result)
	assert.Equal(t, "n1", result["id"])
}

func TestHandlers_GetNode_NotFound(t *testing.T) {
	_, router, workflowRepo, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	w := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/nodes/nonexistent", workflowModel.ID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== UPDATE NODE TESTS ==========

func TestHandlers_UpdateNode_Success(t *testing.T) {
	_, router, workflowRepo, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	req := map[string]interface{}{
		"name": "Updated Node Name",
		"config": map[string]interface{}{
			"mode": "template",
		},
	}

	w := testutil.MakeRequest(t, router, "PUT",
		fmt.Sprintf("/api/v1/workflows/%s/nodes/n1", workflowModel.ID), req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	testutil.ParseResponse(t, w, &result)
	assert.Equal(t, "Updated Node Name", result["name"])
}

func TestHandlers_UpdateNode_NotFound(t *testing.T) {
	_, router, workflowRepo, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	req := map[string]interface{}{
		"name": "Updated",
	}

	w := testutil.MakeRequest(t, router, "PUT",
		fmt.Sprintf("/api/v1/workflows/%s/nodes/nonexistent", workflowModel.ID), req)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== DELETE NODE TESTS ==========

func TestHandlers_DeleteNode_Success(t *testing.T) {
	_, router, workflowRepo, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	w := testutil.MakeRequest(t, router, "DELETE",
		fmt.Sprintf("/api/v1/workflows/%s/nodes/n1", workflowModel.ID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify node is deleted
	getW := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/nodes/n1", workflowModel.ID), nil)
	assert.Equal(t, http.StatusNotFound, getW.Code)
}

func TestHandlers_DeleteNode_NotFound(t *testing.T) {
	_, router, workflowRepo, cleanup := setupNodeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	w := testutil.MakeRequest(t, router, "DELETE",
		fmt.Sprintf("/api/v1/workflows/%s/nodes/nonexistent", workflowModel.ID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}
