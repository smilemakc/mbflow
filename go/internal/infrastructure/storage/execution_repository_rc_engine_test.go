package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// insertLeaseExecution inserts an execution and back-dates its lease.
func insertLeaseExecution(t *testing.T, repo *ExecutionRepository, db bun.IDB, status string, age time.Duration) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	exec := &models.ExecutionModel{ID: uuid.New(), Status: status, WorkflowSource: "inline", WorkflowSnapshot: models.JSONBMap{"name": "lease"}}
	require.NoError(t, repo.Create(ctx, exec))
	_, err := db.NewUpdate().Model((*models.ExecutionModel)(nil)).
		Set("updated_at = current_timestamp - ? * interval '1 second'", int(age.Seconds())).
		Where("id = ?", exec.ID).Exec(ctx)
	require.NoError(t, err)
	return exec.ID
}

func TestExecutionRepo_Update_RenewsUpdatedAt(t *testing.T) {
	repo, db, _ := setupExecutionRepoTest(t)
	ctx := context.Background()

	id := insertLeaseExecution(t, repo, db, "pending", time.Hour)
	before := time.Now().Add(-time.Minute)

	require.NoError(t, repo.Update(ctx, &models.ExecutionModel{ID: id, Status: "running", WorkflowSource: "inline", WorkflowSnapshot: models.JSONBMap{"name": "lease"}}))

	got, err := repo.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "running", got.Status)
	assert.True(t, got.UpdatedAt.After(before), "updated_at must be renewed by Update, got %s", got.UpdatedAt)
}

func TestExecutionRepo_TouchExecutions(t *testing.T) {
	repo, db, _ := setupExecutionRepoTest(t)
	ctx := context.Background()

	running := insertLeaseExecution(t, repo, db, "running", time.Hour)
	pending := insertLeaseExecution(t, repo, db, "pending", time.Hour)
	completed := insertLeaseExecution(t, repo, db, "completed", time.Hour)

	n, err := repo.TouchExecutions(ctx, nil)
	require.NoError(t, err)
	assert.Zero(t, n)

	n, err = repo.TouchExecutions(ctx, []uuid.UUID{running, pending, completed, uuid.New()})
	require.NoError(t, err)
	assert.Equal(t, 2, n, "only pending/running executions hold a lease")

	cutoff := time.Now().Add(-time.Minute)
	for _, id := range []uuid.UUID{running, pending} {
		got, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.True(t, got.UpdatedAt.After(cutoff))
	}
	got, err := repo.FindByID(ctx, completed)
	require.NoError(t, err)
	assert.True(t, got.UpdatedAt.Before(cutoff), "terminal executions are not touched")
}

func TestExecutionRepo_FailStaleExecutions(t *testing.T) {
	repo, db, _ := setupExecutionRepoTest(t)
	ctx := context.Background()

	staleRunning := insertLeaseExecution(t, repo, db, "running", time.Hour)
	stalePending := insertLeaseExecution(t, repo, db, "pending", time.Hour)
	freshRunning := insertLeaseExecution(t, repo, db, "running", 0)
	staleCompleted := insertLeaseExecution(t, repo, db, "completed", time.Hour)

	ids, err := repo.FailStaleExecutions(ctx, 10*time.Minute, "interrupted")
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{staleRunning, stalePending}, ids)

	for _, id := range ids {
		got, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "failed", got.Status)
		assert.Equal(t, "interrupted", got.Error)
		assert.NotNil(t, got.CompletedAt)
	}
	for id, status := range map[uuid.UUID]string{freshRunning: "running", staleCompleted: "completed"} {
		got, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, status, got.Status)
	}

	// A second sweep finds nothing: failed executions no longer hold a lease.
	ids, err = repo.FailStaleExecutions(ctx, 10*time.Minute, "interrupted")
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestExecutionRepo_FailStaleExecutions_InvalidThreshold(t *testing.T) {
	repo, _, _ := setupExecutionRepoTest(t)
	_, err := repo.FailStaleExecutions(context.Background(), 0, "x")
	require.Error(t, err)
}

func TestExecutionRepo_LeaseMethods_DBErrors(t *testing.T) {
	repo, db, cleanup := setupExecutionRepoTest(t)
	cleanup()
	_ = db

	_, err := repo.TouchExecutions(context.Background(), []uuid.UUID{uuid.New()})
	require.Error(t, err)
	_, err = repo.FailStaleExecutions(context.Background(), time.Minute, "x")
	require.Error(t, err)
}
