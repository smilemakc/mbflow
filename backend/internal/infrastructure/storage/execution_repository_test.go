package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func setupExecutionRepoTest(t *testing.T) (*ExecutionRepository, *bun.DB, func()) {
	ctx := context.Background()

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "mbflow_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := postgres.Host(ctx)
	require.NoError(t, err)

	port, err := postgres.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Connect to database
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/mbflow_test?sslmode=disable", host, port.Port())

	// Wait a bit for the database to be fully ready
	time.Sleep(500 * time.Millisecond)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())

	// Run migrations
	migrator, err := NewMigrator(db, migrations.FS)
	require.NoError(t, err)
	err = migrator.Init(ctx)
	require.NoError(t, err)
	err = migrator.Up(ctx)
	require.NoError(t, err)

	repo := NewExecutionRepository(db)

	cleanup := func() {
		db.Close()
		postgres.Terminate(ctx)
	}

	return repo, db, cleanup
}

func createTestWorkflow(t *testing.T, workflowRepo *WorkflowRepository) *models.WorkflowModel {
	workflow := &models.WorkflowModel{
		Name:        "Test Workflow",
		Description: "Test workflow for execution tests",
		Status:      "active",
		Version:     1,
		Variables:   models.JSONBMap{},
		Metadata:    models.JSONBMap{},
		Nodes: []*models.NodeModel{
			{
				NodeID:   "node1",
				Name:     "Node 1",
				Type:     "transform",
				Config:   models.JSONBMap{"type": "passthrough"},
				Position: models.JSONBMap{"x": 0, "y": 0},
			},
			{
				NodeID:   "node2",
				Name:     "Node 2",
				Type:     "transform",
				Config:   models.JSONBMap{"type": "passthrough"},
				Position: models.JSONBMap{"x": 100, "y": 0},
			},
		},
		Edges: []*models.EdgeModel{
			{
				EdgeID:     "edge1",
				FromNodeID: "node1",
				ToNodeID:   "node2",
				Condition:  models.JSONBMap{},
			},
		},
	}

	err := workflowRepo.Create(context.Background(), workflow)
	require.NoError(t, err)
	return workflow
}

// ========== CREATE TESTS ==========

func TestExecutionRepo_Create_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	now := time.Now()
	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "pending",
		StartedAt:  &now,
		Variables:  models.JSONBMap{"key": "value"},
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	// Verify creation
	found, err := repo.FindByID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Equal(t, execution.ID, found.ID)
	assert.Equal(t, execution.WorkflowID, found.WorkflowID)
	assert.Equal(t, "pending", found.Status)
}

func TestExecutionRepo_Create_GeneratesID(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.Nil, // No ID provided
		WorkflowID: workflow.ID,
		Status:     "pending",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, execution.ID)
}

// ========== UPDATE TESTS ==========

func TestExecutionRepo_Update_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create execution
	now := time.Now()
	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
		StartedAt:  &now,
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	// Update status
	completedAt := time.Now()
	execution.Status = "completed"
	execution.CompletedAt = &completedAt
	execution.OutputData = models.JSONBMap{"result": "success"}

	err = repo.Update(context.Background(), execution)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", found.Status)
	assert.NotNil(t, found.CompletedAt)
	assert.Equal(t, "success", found.OutputData["result"])
}

func TestExecutionRepo_Update_WithNodeExecutions(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create execution
	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	// Create node execution
	nodeExec := &models.NodeExecutionModel{
		ExecutionID: execution.ID,
		NodeID:      workflow.Nodes[0].ID,
		Status:      "completed",
		OutputData:  models.JSONBMap{"output": "test"},
		Wave:        0,
	}

	err = repo.CreateNodeExecution(context.Background(), nodeExec)
	require.NoError(t, err)

	// Update execution status
	completedAt := time.Now()
	execution.Status = "completed"
	execution.CompletedAt = &completedAt

	// Populate NodeExecutions before Update (Update method replaces all node executions)
	execution.NodeExecutions = []*models.NodeExecutionModel{nodeExec}

	err = repo.Update(context.Background(), execution)
	require.NoError(t, err)

	// Verify node execution still exists
	nodeExecs, err := repo.FindNodeExecutionsByExecutionID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Len(t, nodeExecs, 1)
}

// ========== DELETE TESTS ==========

func TestExecutionRepo_Delete_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create execution
	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "pending",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	// Delete execution
	err = repo.Delete(context.Background(), execution.ID)
	require.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByID(context.Background(), execution.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestExecutionRepo_Delete_CascadesNodeExecutions(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create execution with node executions
	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	// Create node execution
	nodeExec := &models.NodeExecutionModel{
		ExecutionID: execution.ID,
		NodeID:      workflow.Nodes[0].ID,
		Status:      "completed",
		Wave:        0,
	}

	err = repo.CreateNodeExecution(context.Background(), nodeExec)
	require.NoError(t, err)

	// Delete execution
	err = repo.Delete(context.Background(), execution.ID)
	require.NoError(t, err)

	// Verify node executions are also deleted
	nodeExecs, err := repo.FindNodeExecutionsByExecutionID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Len(t, nodeExecs, 0)
}

// ========== FIND BY ID TESTS ==========

func TestExecutionRepo_FindByID_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "pending",
		Variables:  models.JSONBMap{"key": "value"},
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Equal(t, execution.ID, found.ID)
	assert.Equal(t, execution.WorkflowID, found.WorkflowID)
	assert.Equal(t, "value", found.Variables["key"])
}

func TestExecutionRepo_FindByID_NotFound(t *testing.T) {
	repo, _, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	nonExistentID := uuid.New()
	found, err := repo.FindByID(context.Background(), nonExistentID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestExecutionRepo_FindByID_WithRelations(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	// Create node execution
	nodeExec := &models.NodeExecutionModel{
		ExecutionID: execution.ID,
		NodeID:      workflow.Nodes[0].ID,
		Status:      "completed",
		Wave:        0,
	}

	err = repo.CreateNodeExecution(context.Background(), nodeExec)
	require.NoError(t, err)

	// Find with relations
	found, err := repo.FindByID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Equal(t, execution.ID, found.ID)

	// Load node executions
	nodeExecs, err := repo.FindNodeExecutionsByExecutionID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Len(t, nodeExecs, 1)
}

// ========== FIND BY WORKFLOW ID TESTS ==========

func TestExecutionRepo_FindByWorkflowID_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create multiple executions
	for i := 0; i < 3; i++ {
		execution := &models.ExecutionModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "pending",
		}
		err := repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	executions, err := repo.FindByWorkflowID(context.Background(), workflow.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, executions, 3)
}

func TestExecutionRepo_FindByWorkflowID_Pagination(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create 5 executions
	for i := 0; i < 5; i++ {
		execution := &models.ExecutionModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "pending",
		}
		err := repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	// Get first page (2 items)
	page1, err := repo.FindByWorkflowID(context.Background(), workflow.ID, 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Get second page (2 items)
	page2, err := repo.FindByWorkflowID(context.Background(), workflow.ID, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Verify different executions
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

// ========== FIND BY STATUS TESTS ==========

func TestExecutionRepo_FindByStatus_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create executions with different statuses
	statuses := []string{"pending", "running", "completed", "pending"}
	for _, status := range statuses {
		execution := &models.ExecutionModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     status,
		}
		err := repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	// Find pending executions
	pending, err := repo.FindByStatus(context.Background(), "pending", 10, 0)
	require.NoError(t, err)
	assert.Len(t, pending, 2)

	// Find running executions
	running, err := repo.FindByStatus(context.Background(), "running", 10, 0)
	require.NoError(t, err)
	assert.Len(t, running, 1)
}

// ========== FIND ALL TESTS ==========

func TestExecutionRepo_FindAll_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create executions
	for i := 0; i < 3; i++ {
		execution := &models.ExecutionModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "pending",
		}
		err := repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	executions, err := repo.FindAll(context.Background(), 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(executions), 3)
}

// ========== FIND RUNNING TESTS ==========

func TestExecutionRepo_FindRunning_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create executions with different statuses
	statuses := []string{"pending", "running", "completed", "running"}
	for _, status := range statuses {
		execution := &models.ExecutionModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     status,
		}
		err := repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	// Find running executions
	running, err := repo.FindRunning(context.Background())
	require.NoError(t, err)
	assert.Len(t, running, 2)
	for _, exec := range running {
		assert.Equal(t, "running", exec.Status)
	}
}

// ========== COUNT TESTS ==========

func TestExecutionRepo_Count_Total(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create executions
	for i := 0; i < 5; i++ {
		execution := &models.ExecutionModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "pending",
		}
		err := repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	count, err := repo.Count(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 5)
}

func TestExecutionRepo_CountByWorkflowID_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create executions
	for i := 0; i < 3; i++ {
		execution := &models.ExecutionModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "pending",
		}
		err := repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	count, err := repo.CountByWorkflowID(context.Background(), workflow.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestExecutionRepo_CountByStatus_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create executions with different statuses
	statuses := []string{"pending", "running", "completed", "pending"}
	for _, status := range statuses {
		execution := &models.ExecutionModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     status,
		}
		err := repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	pendingCount, err := repo.CountByStatus(context.Background(), "pending")
	require.NoError(t, err)
	assert.Equal(t, 2, pendingCount)

	runningCount, err := repo.CountByStatus(context.Background(), "running")
	require.NoError(t, err)
	assert.Equal(t, 1, runningCount)
}

// ========== NODE EXECUTION TESTS ==========

func TestExecutionRepo_CreateNodeExecution_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	nodeExec := &models.NodeExecutionModel{
		ExecutionID: execution.ID,
		NodeID:      workflow.Nodes[0].ID,
		Status:      "pending",
		Wave:        0,
	}

	err = repo.CreateNodeExecution(context.Background(), nodeExec)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, nodeExec.ID)
}

func TestExecutionRepo_UpdateNodeExecution_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	nodeExec := &models.NodeExecutionModel{
		ExecutionID: execution.ID,
		NodeID:      workflow.Nodes[0].ID,
		Status:      "running",
		Wave:        0,
	}

	err = repo.CreateNodeExecution(context.Background(), nodeExec)
	require.NoError(t, err)

	// Update node execution
	nodeExec.Status = "completed"
	nodeExec.OutputData = models.JSONBMap{"result": "success"}

	err = repo.UpdateNodeExecution(context.Background(), nodeExec)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindNodeExecutionByID(context.Background(), nodeExec.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", found.Status)
	assert.Equal(t, "success", found.OutputData["result"])
}

func TestExecutionRepo_DeleteNodeExecution_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	nodeExec := &models.NodeExecutionModel{
		ExecutionID: execution.ID,
		NodeID:      workflow.Nodes[0].ID,
		Status:      "pending",
		Wave:        0,
	}

	err = repo.CreateNodeExecution(context.Background(), nodeExec)
	require.NoError(t, err)

	// Delete node execution
	err = repo.DeleteNodeExecution(context.Background(), nodeExec.ID)
	require.NoError(t, err)

	// Verify deletion
	found, err := repo.FindNodeExecutionByID(context.Background(), nodeExec.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestExecutionRepo_FindNodeExecutionByID_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	nodeExec := &models.NodeExecutionModel{
		ExecutionID: execution.ID,
		NodeID:      workflow.Nodes[0].ID,
		Status:      "completed",
		OutputData:  models.JSONBMap{"output": "test"},
		Wave:        0,
	}

	err = repo.CreateNodeExecution(context.Background(), nodeExec)
	require.NoError(t, err)

	found, err := repo.FindNodeExecutionByID(context.Background(), nodeExec.ID)
	require.NoError(t, err)
	assert.Equal(t, nodeExec.ID, found.ID)
	assert.Equal(t, "test", found.OutputData["output"])
}

func TestExecutionRepo_FindNodeExecutionsByExecutionID_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	// Create multiple node executions
	for i, node := range workflow.Nodes {
		nodeExec := &models.NodeExecutionModel{
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Status:      "completed",
			Wave:        i,
		}
		err = repo.CreateNodeExecution(context.Background(), nodeExec)
		require.NoError(t, err)
	}

	nodeExecs, err := repo.FindNodeExecutionsByExecutionID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Len(t, nodeExecs, 2)
}

func TestExecutionRepo_FindNodeExecutionsByExecutionID_Empty(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "pending",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	nodeExecs, err := repo.FindNodeExecutionsByExecutionID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Len(t, nodeExecs, 0)
}

// ========== NODE EXECUTION QUERY TESTS ==========

func TestExecutionRepo_FindNodeExecutionsByWave_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	// Create node executions with different waves
	for i, node := range workflow.Nodes {
		nodeExec := &models.NodeExecutionModel{
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Status:      "completed",
			Wave:        i,
		}
		err = repo.CreateNodeExecution(context.Background(), nodeExec)
		require.NoError(t, err)
	}

	// Find wave 0
	wave0, err := repo.FindNodeExecutionsByWave(context.Background(), execution.ID, 0)
	require.NoError(t, err)
	assert.Len(t, wave0, 1)
	assert.Equal(t, 0, wave0[0].Wave)

	// Find wave 1
	wave1, err := repo.FindNodeExecutionsByWave(context.Background(), execution.ID, 1)
	require.NoError(t, err)
	assert.Len(t, wave1, 1)
	assert.Equal(t, 1, wave1[0].Wave)
}

func TestExecutionRepo_FindNodeExecutionsByStatus_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err := repo.Create(context.Background(), execution)
	require.NoError(t, err)

	// Create node executions with different statuses
	statuses := []string{"pending", "running"}
	for i, status := range statuses {
		nodeExec := &models.NodeExecutionModel{
			ExecutionID: execution.ID,
			NodeID:      workflow.Nodes[i].ID,
			Status:      status,
			Wave:        0,
		}
		err = repo.CreateNodeExecution(context.Background(), nodeExec)
		require.NoError(t, err)
	}

	// Find pending node executions
	pending, err := repo.FindNodeExecutionsByStatus(context.Background(), execution.ID, "pending")
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "pending", pending[0].Status)
}

// ========== STATISTICS TESTS ==========

func TestExecutionRepo_GetStatistics_Success(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow := createTestWorkflow(t, workflowRepo)

	// Create executions with different statuses
	statuses := []string{"pending", "running", "completed", "failed"}
	now := time.Now()
	startedAt := now.Add(-1 * time.Hour) // Started 1 hour ago
	completedAt := now

	for _, status := range statuses {
		execution := &models.ExecutionModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     status,
		}

		// Set StartedAt for running/completed/failed executions
		if status != "pending" {
			execution.StartedAt = &startedAt
		}

		// Set CompletedAt for completed/failed executions
		if status == "completed" || status == "failed" {
			execution.CompletedAt = &completedAt
		}

		err := repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	stats, err := repo.GetStatistics(context.Background(), &workflow.ID, from, to)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Greater(t, stats.TotalExecutions, 0)
}

func TestExecutionRepo_GetStatistics_AllWorkflows(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	defer cleanup()

	workflowRepo := NewWorkflowRepository(db)
	workflow1 := createTestWorkflow(t, workflowRepo)

	// Create another workflow
	workflow2 := &models.WorkflowModel{
		Name:      "Test Workflow 2",
		Status:    "active",
		Version:   1,
		Variables: models.JSONBMap{},
		Metadata:  models.JSONBMap{},
	}
	err := workflowRepo.Create(context.Background(), workflow2)
	require.NoError(t, err)

	// Create executions for both workflows
	now := time.Now()
	startedAt := now.Add(-1 * time.Hour)
	completedAt := now

	for _, wf := range []*models.WorkflowModel{workflow1, workflow2} {
		execution := &models.ExecutionModel{
			ID:          uuid.New(),
			WorkflowID:  wf.ID,
			Status:      "completed",
			StartedAt:   &startedAt,
			CompletedAt: &completedAt,
		}
		err = repo.Create(context.Background(), execution)
		require.NoError(t, err)
	}

	// Get stats for all workflows (pass nil)
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	stats, err := repo.GetStatistics(context.Background(), nil, from, to)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalExecutions, 2)
}
