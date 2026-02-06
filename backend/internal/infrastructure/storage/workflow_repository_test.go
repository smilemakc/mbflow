package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func setupWorkflowRepoTest(t *testing.T) (*WorkflowRepository, bun.IDB, func()) {
	t.Helper()
	db, cleanup := testutil.SetupTestTx(t)
	return NewWorkflowRepository(db), db, cleanup
}

func setupTestDBWithContainer(t *testing.T) (bun.IDB, func()) {
	t.Helper()
	return testutil.SetupTestTx(t)
}

func TestWorkflowRepository_SyncNodesWithExecutionHistory(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDBWithContainer(t)
	defer cleanup()

	repo := NewWorkflowRepository(db)
	ctx := context.Background()

	// 1. Create a workflow with 2 nodes
	workflowID := uuid.New()
	workflow := &models.WorkflowModel{
		ID:          workflowID,
		Name:        "test_workflow_sync",
		Description: "Test workflow for node sync",
		Status:      "draft",
		Version:     1,
		Variables:   models.JSONBMap{},
		Metadata:    models.JSONBMap{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := db.NewInsert().Model(workflow).Exec(ctx)
	require.NoError(t, err)

	// Create 2 nodes
	node1 := &models.NodeModel{
		ID:         uuid.New(),
		NodeID:     "node_1",
		WorkflowID: workflowID,
		Name:       "Node 1",
		Type:       "http",
		Config:     models.JSONBMap{"url": "https://api.example.com/1"},
		Position:   models.JSONBMap{"x": 100, "y": 100},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	node2 := &models.NodeModel{
		ID:         uuid.New(),
		NodeID:     "node_2",
		WorkflowID: workflowID,
		Name:       "Node 2",
		Type:       "transform",
		Config:     models.JSONBMap{"type": "passthrough"},
		Position:   models.JSONBMap{"x": 300, "y": 100},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err = db.NewInsert().Model(node1).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(node2).Exec(ctx)
	require.NoError(t, err)

	// 2. Create an execution with node executions
	now := time.Now()
	startedAt := now
	completedAt := now.Add(5 * time.Second)

	execution := &models.ExecutionModel{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		Status:      "completed",
		StartedAt:   &startedAt,
		CompletedAt: &completedAt,
		InputData:   models.JSONBMap{},
		OutputData:  models.JSONBMap{},
		Variables:   models.JSONBMap{},
		StrictMode:  false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = db.NewInsert().Model(execution).Exec(ctx)
	require.NoError(t, err)

	// Create node executions for both nodes
	nodeExec1StartedAt := now
	nodeExec1CompletedAt := now.Add(2 * time.Second)
	nodeExec2StartedAt := now.Add(2 * time.Second)
	nodeExec2CompletedAt := now.Add(4 * time.Second)

	nodeExec1 := &models.NodeExecutionModel{
		ID:          uuid.New(),
		ExecutionID: execution.ID,
		NodeID:      node1.ID,
		Status:      "completed",
		StartedAt:   &nodeExec1StartedAt,
		CompletedAt: &nodeExec1CompletedAt,
		InputData:   models.JSONBMap{},
		OutputData:  models.JSONBMap{"result": "success"},
		RetryCount:  0,
		Wave:        0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	nodeExec2 := &models.NodeExecutionModel{
		ID:          uuid.New(),
		ExecutionID: execution.ID,
		NodeID:      node2.ID,
		Status:      "completed",
		StartedAt:   &nodeExec2StartedAt,
		CompletedAt: &nodeExec2CompletedAt,
		InputData:   models.JSONBMap{},
		OutputData:  models.JSONBMap{"result": "transformed"},
		RetryCount:  0,
		Wave:        1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = db.NewInsert().Model(nodeExec1).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(nodeExec2).Exec(ctx)
	require.NoError(t, err)

	// 3. Verify we have 2 nodes and 2 node executions
	var nodeCount int
	nodeCount, err = db.NewSelect().Model((*models.NodeModel)(nil)).Where("workflow_id = ?", workflowID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, nodeCount, "Should have 2 nodes initially")

	var nodeExecCount int
	nodeExecCount, err = db.NewSelect().Model((*models.NodeExecutionModel)(nil)).Where("execution_id = ?", execution.ID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, nodeExecCount, "Should have 2 node executions initially")

	// 4. Now sync nodes, removing node_1 (which has execution history)
	// This should CASCADE delete the node execution for node_1
	newNodes := []*models.NodeModel{
		{
			NodeID:     "node_2", // Keep node 2
			WorkflowID: workflowID,
			Name:       "Node 2 Updated",
			Type:       "transform",
			Config:     models.JSONBMap{"type": "passthrough"},
			Position:   models.JSONBMap{"x": 300, "y": 150},
		},
		{
			NodeID:     "node_3", // Add new node 3
			WorkflowID: workflowID,
			Name:       "Node 3",
			Type:       "llm",
			Config:     models.JSONBMap{"provider": "openai"},
			Position:   models.JSONBMap{"x": 500, "y": 100},
		},
	}

	// Start a transaction for syncing
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	err = repo.syncNodes(ctx, tx, workflowID, newNodes)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to sync nodes: %v", err)
	}

	err = tx.Commit()
	require.NoError(t, err, "Transaction should commit successfully")

	// 5. Verify the results
	// Should now have 2 nodes (node_2 updated, node_3 new)
	nodeCount, err = db.NewSelect().Model((*models.NodeModel)(nil)).Where("workflow_id = ?", workflowID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, nodeCount, "Should have 2 nodes after sync")

	// Verify node_1 is deleted
	var node1AfterSync models.NodeModel
	err = db.NewSelect().Model(&node1AfterSync).Where("workflow_id = ? AND node_id = ?", workflowID, "node_1").Scan(ctx)
	assert.Error(t, err, "node_1 should be deleted")
	assert.Equal(t, sql.ErrNoRows, err)

	// Verify node_2 still exists and was updated
	var node2AfterSync models.NodeModel
	err = db.NewSelect().Model(&node2AfterSync).Where("workflow_id = ? AND node_id = ?", workflowID, "node_2").Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Node 2 Updated", node2AfterSync.Name)

	// Verify node_3 was created
	var node3AfterSync models.NodeModel
	err = db.NewSelect().Model(&node3AfterSync).Where("workflow_id = ? AND node_id = ?", workflowID, "node_3").Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Node 3", node3AfterSync.Name)

	// 6. Verify CASCADE delete worked - node execution for node_1 should be deleted
	nodeExecCount, err = db.NewSelect().Model((*models.NodeExecutionModel)(nil)).Where("execution_id = ?", execution.ID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, nodeExecCount, "Should have only 1 node execution after cascade delete")

	// Verify only node_2's execution remains
	var remainingNodeExec models.NodeExecutionModel
	err = db.NewSelect().Model(&remainingNodeExec).Where("execution_id = ? AND node_id = ?", execution.ID, node2.ID).Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, "completed", remainingNodeExec.Status)

	t.Log("✓ Test passed: Node sync with CASCADE delete works correctly")
}

// Test Workflow CRUD Operations

func TestWorkflowRepo_Create_BasicWorkflow(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:          uuid.New(),
		Name:        "Test Workflow Create",
		Description: "Basic workflow creation test",
		Status:      "draft",
		Version:     1,
		Variables:   models.JSONBMap{"key": "value"},
		Metadata:    models.JSONBMap{},
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)
	assert.False(t, workflow.CreatedAt.IsZero())
	assert.False(t, workflow.UpdatedAt.IsZero())
}

func TestWorkflowRepo_Create_WithNodesAndEdges(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:          uuid.New(),
		Name:        "Workflow With Nodes",
		Description: "Workflow with nodes and edges",
		Status:      "draft",
		Version:     1,
		Variables:   models.JSONBMap{},
		Nodes: []*models.NodeModel{
			{
				NodeID:   "node1",
				Name:     "HTTP Node",
				Type:     "http",
				Config:   models.JSONBMap{"url": "https://api.example.com"},
				Position: models.JSONBMap{"x": 100, "y": 100},
			},
			{
				NodeID:   "node2",
				Name:     "Transform Node",
				Type:     "transform",
				Config:   models.JSONBMap{"type": "passthrough"},
				Position: models.JSONBMap{"x": 300, "y": 100},
			},
		},
		Edges: []*models.EdgeModel{
			{
				EdgeID:     "edge1",
				FromNodeID: "node1",
				ToNodeID:   "node2",
			},
		},
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	// Verify nodes were created
	nodes, err := repo.FindNodesByWorkflowID(context.Background(), workflow.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)

	// Verify edges were created
	edges, err := repo.FindEdgesByWorkflowID(context.Background(), workflow.ID)
	require.NoError(t, err)
	assert.Len(t, edges, 1)
}

func TestWorkflowRepo_FindByID_Success(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Find By ID Test",
		Status:  "draft",
		Version: 1,
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), workflow.ID)
	require.NoError(t, err)
	assert.Equal(t, workflow.ID, found.ID)
	assert.Equal(t, workflow.Name, found.Name)
}

func TestWorkflowRepo_FindByID_NotFound(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	_, err := repo.FindByID(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestWorkflowRepo_FindByIDWithRelations_LoadsNodesAndEdges(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "With Relations Test",
		Status:  "active",
		Version: 1,
		Nodes: []*models.NodeModel{
			{NodeID: "n1", Name: "Node 1", Type: "http", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
			{NodeID: "n2", Name: "Node 2", Type: "http", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
		},
		Edges: []*models.EdgeModel{
			{EdgeID: "e1", FromNodeID: "n1", ToNodeID: "n2"},
		},
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	found, err := repo.FindByIDWithRelations(context.Background(), workflow.ID)
	require.NoError(t, err)
	assert.Len(t, found.Nodes, 2)
	assert.Len(t, found.Edges, 1)
}

func TestWorkflowRepo_FindByName_Success(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Unique Workflow Name",
		Status:  "draft",
		Version: 1,
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	found, err := repo.FindByName(context.Background(), "Unique Workflow Name", 1)
	require.NoError(t, err)
	assert.Equal(t, workflow.ID, found.ID)
}

func TestWorkflowRepo_Update_Metadata(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:          uuid.New(),
		Name:        "Original Name",
		Description: "Original Description",
		Status:      "draft",
		Version:     1,
		Variables:   models.JSONBMap{},
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	// Update workflow
	workflow.Name = "Updated Name"
	workflow.Description = "Updated Description"
	workflow.Status = "active"

	err = repo.Update(context.Background(), workflow)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(context.Background(), workflow.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "Updated Description", updated.Description)
	assert.Equal(t, "active", updated.Status)
}

func TestWorkflowRepo_Delete_Success(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Workflow To Delete",
		Status:  "draft",
		Version: 1,
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	// Delete
	err = repo.Delete(context.Background(), workflow.ID)
	require.NoError(t, err)

	// Verify deleted
	_, err = repo.FindByID(context.Background(), workflow.ID)
	assert.Error(t, err)
}

func TestWorkflowRepo_FindAll_Pagination(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	// Create 5 workflows
	for i := 0; i < 5; i++ {
		workflow := &models.WorkflowModel{
			ID:      uuid.New(),
			Name:    fmt.Sprintf("Workflow %d", i),
			Status:  "draft",
			Version: 1,
		}
		err := repo.Create(context.Background(), workflow)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get first page
	page1, err := repo.FindAll(context.Background(), 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Get second page
	page2, err := repo.FindAll(context.Background(), 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Verify different workflows
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestWorkflowRepo_FindByStatus_FilterActive(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	// Create workflows with different statuses
	draft := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Draft Workflow",
		Status:  "draft",
		Version: 1,
	}
	active := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Active Workflow",
		Status:  "active",
		Version: 1,
	}

	err := repo.Create(context.Background(), draft)
	require.NoError(t, err)

	err = repo.Create(context.Background(), active)
	require.NoError(t, err)

	// Find only active workflows
	activeWorkflows, err := repo.FindByStatus(context.Background(), "active", 10, 0)
	require.NoError(t, err)
	assert.Len(t, activeWorkflows, 1)
	assert.Equal(t, "active", activeWorkflows[0].Status)
}

func TestWorkflowRepo_Count_Total(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	// Create 3 workflows
	for i := 0; i < 3; i++ {
		workflow := &models.WorkflowModel{
			ID:      uuid.New(),
			Name:    fmt.Sprintf("Count Test %d", i),
			Status:  "draft",
			Version: 1,
		}
		err := repo.Create(context.Background(), workflow)
		require.NoError(t, err)
	}

	count, err := repo.Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestWorkflowRepo_CountByStatus_FilterDraft(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	// Create 2 draft and 1 active workflow
	for i := 0; i < 2; i++ {
		workflow := &models.WorkflowModel{
			ID:      uuid.New(),
			Name:    fmt.Sprintf("Draft %d", i),
			Status:  "draft",
			Version: 1,
		}
		err := repo.Create(context.Background(), workflow)
		require.NoError(t, err)
	}

	active := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Active",
		Status:  "active",
		Version: 1,
	}
	err := repo.Create(context.Background(), active)
	require.NoError(t, err)

	draftCount, err := repo.CountByStatus(context.Background(), "draft")
	require.NoError(t, err)
	assert.Equal(t, 2, draftCount)
}

// Test Node Operations

func TestWorkflowRepo_CreateNode_AddToWorkflow(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Node Test Workflow",
		Status:  "draft",
		Version: 1,
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	node := &models.NodeModel{
		NodeID:     "new_node",
		WorkflowID: workflow.ID,
		Name:       "New Node",
		Type:       "http",
		Config:     models.JSONBMap{"url": "https://example.com"},
		Position:   models.JSONBMap{"x": 100, "y": 100},
	}

	err = repo.CreateNode(context.Background(), node)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, node.ID)
}

func TestWorkflowRepo_FindNodesByWorkflowID_ReturnsAll(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Multi Node Workflow",
		Status:  "draft",
		Version: 1,
		Nodes: []*models.NodeModel{
			{NodeID: "n1", Name: "Node 1", Type: "http", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
			{NodeID: "n2", Name: "Node 2", Type: "transform", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
			{NodeID: "n3", Name: "Node 3", Type: "llm", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
		},
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	nodes, err := repo.FindNodesByWorkflowID(context.Background(), workflow.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 3)
}

// Test DAG Validation

func TestWorkflowRepo_ValidateDAG_ValidDAG(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Valid DAG",
		Status:  "draft",
		Version: 1,
		Nodes: []*models.NodeModel{
			{NodeID: "n1", Name: "Node 1", Type: "http", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
			{NodeID: "n2", Name: "Node 2", Type: "http", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
			{NodeID: "n3", Name: "Node 3", Type: "http", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
		},
		Edges: []*models.EdgeModel{
			{EdgeID: "e1", FromNodeID: "n1", ToNodeID: "n2"},
			{EdgeID: "e2", FromNodeID: "n2", ToNodeID: "n3"},
		},
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	err = repo.ValidateDAG(context.Background(), workflow.ID)
	assert.NoError(t, err, "Valid DAG should not return error")
}

func TestWorkflowRepo_ValidateDAG_DetectsCycle(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupWorkflowRepoTest(t)
	defer cleanup()

	workflow := &models.WorkflowModel{
		ID:      uuid.New(),
		Name:    "Cyclic DAG",
		Status:  "draft",
		Version: 1,
		Nodes: []*models.NodeModel{
			{NodeID: "n1", Name: "Node 1", Type: "http", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
			{NodeID: "n2", Name: "Node 2", Type: "http", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
			{NodeID: "n3", Name: "Node 3", Type: "http", Config: models.JSONBMap{}, Position: models.JSONBMap{}},
		},
		Edges: []*models.EdgeModel{
			{EdgeID: "e1", FromNodeID: "n1", ToNodeID: "n2"},
			{EdgeID: "e2", FromNodeID: "n2", ToNodeID: "n3"},
			{EdgeID: "e3", FromNodeID: "n3", ToNodeID: "n1"}, // Creates cycle
		},
	}

	err := repo.Create(context.Background(), workflow)
	require.NoError(t, err)

	err = repo.ValidateDAG(context.Background(), workflow.ID)
	assert.Error(t, err, "Cycle should be detected")
	assert.Contains(t, err.Error(), "cycle detected")
}
