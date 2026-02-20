package rest

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEdgeHandlersTest(t *testing.T) (*EdgeHandlers, *gin.Engine, *storage.WorkflowRepository, func()) {
	db, cleanup := testutil.SetupTestTx(t)
	workflowRepo := storage.NewWorkflowRepository(db)
	log := logger.New(config.LoggingConfig{Level: "error", Format: "text"})
	handlers := NewEdgeHandlers(workflowRepo, log)

	router := gin.New()
	api := router.Group("/api/v1/workflows/:workflow_id")
	{
		api.POST("/edges", handlers.HandleAddEdge)
		api.GET("/edges", handlers.HandleListEdges)
		api.GET("/edges/:edgeId", handlers.HandleGetEdge)
		api.PUT("/edges/:edgeId", handlers.HandleUpdateEdge)
		api.DELETE("/edges/:edgeId", handlers.HandleDeleteEdge)
	}

	return handlers, router, workflowRepo, cleanup
}

// ========== ADD EDGE TESTS ==========

func TestHandlers_AddEdge_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	req := map[string]any{
		"id":   "new_edge",
		"from": "n1",
		"to":   "n3",
	}

	w := testutil.MakeRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/workflows/%s/edges", workflowModel.ID), req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]any
	testutil.ParseResponse(t, w, &result)
	assert.Equal(t, "new_edge", result["id"])
	assert.Equal(t, "n1", result["from"])
	assert.Equal(t, "n3", result["to"])
}

func TestHandlers_AddEdge_WorkflowNotFound(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	req := map[string]any{
		"id":   "new_edge",
		"from": "n1",
		"to":   "n2",
	}

	randomID := uuid.New()
	w := testutil.MakeRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/workflows/%s/edges", randomID), req)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

func TestHandlers_AddEdge_CreatesCycle(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow() // n1 -> n2 -> n3
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Try to add edge n3 -> n1 which creates a cycle
	req := map[string]any{
		"id":   "cycle_edge",
		"from": "n3",
		"to":   "n1",
	}

	w := testutil.MakeRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/workflows/%s/edges", workflowModel.ID), req)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "creates a cycle in the workflow")
}

func TestHandlers_AddEdge_InvalidNodes(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	req := map[string]any{
		"id":   "invalid_edge",
		"from": "nonexistent",
		"to":   "n2",
	}

	w := testutil.MakeRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/workflows/%s/edges", workflowModel.ID), req)

	testutil.AssertErrorResponse(t, w, http.StatusBadRequest, "node does not exist")
}

// ========== LIST EDGES TESTS ==========

func TestHandlers_ListEdges_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow() // Has 2 edges
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	w := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/edges", workflowModel.ID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var edges []any
	testutil.ParseListResponse(t, w, &edges)
	assert.Len(t, edges, 2)
}

func TestHandlers_ListEdges_WorkflowNotFound(t *testing.T) {
	t.Parallel()
	_, router, _, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	randomID := uuid.New()
	w := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/edges", randomID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== GET EDGE TESTS ==========

func TestHandlers_GetEdge_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	// Get first edge (n1 -> n2)
	edgeID := workflowModel.Edges[0].EdgeID

	w := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/edges/%s", workflowModel.ID, edgeID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]any
	testutil.ParseResponse(t, w, &result)
	assert.Equal(t, edgeID, result["id"])
}

func TestHandlers_GetEdge_NotFound(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	w := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/edges/nonexistent", workflowModel.ID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== UPDATE EDGE TESTS ==========

func TestHandlers_UpdateEdge_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	edgeID := workflowModel.Edges[0].EdgeID

	req := map[string]any{
		"condition": "input.value > 10",
	}

	w := testutil.MakeRequest(t, router, "PUT",
		fmt.Sprintf("/api/v1/workflows/%s/edges/%s", workflowModel.ID, edgeID), req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]any
	testutil.ParseResponse(t, w, &result)
	assert.NotNil(t, result["condition"])
}

func TestHandlers_UpdateEdge_NotFound(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	req := map[string]any{
		"condition": "input.value > 5",
	}

	w := testutil.MakeRequest(t, router, "PUT",
		fmt.Sprintf("/api/v1/workflows/%s/edges/nonexistent", workflowModel.ID), req)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}

// ========== DELETE EDGE TESTS ==========

func TestHandlers_DeleteEdge_Success(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	edgeID := workflowModel.Edges[0].EdgeID

	w := testutil.MakeRequest(t, router, "DELETE",
		fmt.Sprintf("/api/v1/workflows/%s/edges/%s", workflowModel.ID, edgeID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify edge is deleted
	getW := testutil.MakeRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/workflows/%s/edges/%s", workflowModel.ID, edgeID), nil)
	assert.Equal(t, http.StatusNotFound, getW.Code)
}

func TestHandlers_DeleteEdge_NotFound(t *testing.T) {
	t.Parallel()
	_, router, workflowRepo, cleanup := setupEdgeHandlersTest(t)
	defer cleanup()

	workflow := testutil.CreateSimpleWorkflow()
	workflowModel := testutil.WorkflowDomainToModel(workflow)
	err := workflowRepo.Create(context.Background(), workflowModel)
	require.NoError(t, err)

	w := testutil.MakeRequest(t, router, "DELETE",
		fmt.Sprintf("/api/v1/workflows/%s/edges/nonexistent", workflowModel.ID), nil)

	testutil.AssertErrorResponse(t, w, http.StatusNotFound, "")
}
