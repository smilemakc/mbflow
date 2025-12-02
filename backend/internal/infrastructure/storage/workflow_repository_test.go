package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func setupTestDB(t *testing.T) *bun.DB {
	dsn := "postgres://mbflow:mbflow@localhost:5566/mbflow?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Clean up any existing test data
	ctx := context.Background()
	_, _ = db.NewDelete().Model((*models.NodeExecutionModel)(nil)).Where("1=1").Exec(ctx)
	_, _ = db.NewDelete().Model((*models.ExecutionModel)(nil)).Where("1=1").Exec(ctx)
	_, _ = db.NewDelete().Model((*models.EdgeModel)(nil)).Where("1=1").Exec(ctx)
	_, _ = db.NewDelete().Model((*models.NodeModel)(nil)).Where("1=1").Exec(ctx)
	_, _ = db.NewDelete().Model((*models.WorkflowModel)(nil)).Where("name LIKE 'test_%'").Exec(ctx)

	return db
}

func TestWorkflowRepository_SyncNodesWithExecutionHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

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
