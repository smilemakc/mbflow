package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

func TestExecutionRepo_CancelExecution(t *testing.T) {
	repo, db, _ := setupExecutionRepoTest(t)
	ctx := context.Background()

	running := insertLeaseExecution(t, repo, db, "running", time.Hour)
	pending := insertLeaseExecution(t, repo, db, "pending", time.Hour)
	completed := insertLeaseExecution(t, repo, db, "completed", time.Hour)

	for _, id := range []uuid.UUID{running, pending} {
		changed, err := repo.CancelExecution(ctx, id, "execution cancelled")
		require.NoError(t, err)
		assert.True(t, changed)

		got, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "cancelled", got.Status)
		assert.Equal(t, "execution cancelled", got.Error)
		assert.NotNil(t, got.CompletedAt)
		assert.True(t, got.UpdatedAt.After(time.Now().Add(-time.Minute)), "the lease is renewed")
	}

	for _, id := range []uuid.UUID{running, completed, uuid.New()} {
		changed, err := repo.CancelExecution(ctx, id, "execution cancelled")
		require.NoError(t, err)
		assert.False(t, changed, "only pending/running executions can be cancelled")
	}
	got, err := repo.FindByID(ctx, completed)
	require.NoError(t, err)
	assert.Equal(t, "completed", got.Status)
}

func TestExecutionRepo_CancelledExecutions(t *testing.T) {
	repo, db, _ := setupExecutionRepoTest(t)
	ctx := context.Background()

	ids, err := repo.CancelledExecutions(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, ids)

	running := insertLeaseExecution(t, repo, db, "running", 0)
	cancelled := insertLeaseExecution(t, repo, db, "running", 0)
	_, err = repo.CancelExecution(ctx, cancelled, "execution cancelled")
	require.NoError(t, err)

	ids, err = repo.CancelledExecutions(ctx, []uuid.UUID{running, cancelled, uuid.New()})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{cancelled}, ids)
}

func TestExecutionRepo_Update_CancellationIsSticky(t *testing.T) {
	repo, db, _ := setupExecutionRepoTest(t)
	ctx := context.Background()

	id := insertLeaseExecution(t, repo, db, "running", 0)
	_, err := repo.CancelExecution(ctx, id, "execution cancelled")
	require.NoError(t, err)

	nodeExec := func() []*models.NodeExecutionModel {
		key := "n1"
		return []*models.NodeExecutionModel{{ID: uuid.New(), ExecutionID: id, NodeKey: &key, Status: "completed"}}
	}

	// The owner tries to store it as completed: refused, nothing changes.
	err = repo.Update(ctx, &models.ExecutionModel{ID: id, Status: "completed", WorkflowSource: "inline", NodeExecutions: nodeExec()})
	require.ErrorIs(t, err, repository.ErrExecutionAlreadyCancelled)
	got, err := repo.FindByIDWithRelations(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", got.Status)
	assert.Empty(t, got.NodeExecutions)

	// Storing it as cancelled records its node executions.
	require.NoError(t, repo.Update(ctx, &models.ExecutionModel{ID: id, Status: "cancelled", Error: "execution cancelled", WorkflowSource: "inline", NodeExecutions: nodeExec()}))
	got, err = repo.FindByIDWithRelations(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", got.Status)
	assert.Len(t, got.NodeExecutions, 1)

	// Updating an unknown execution keeps its previous (no-op) behavior.
	require.NoError(t, repo.Update(ctx, &models.ExecutionModel{ID: uuid.New(), Status: "completed", WorkflowSource: "inline"}))
}

func TestExecutionRepo_FindByID_NotFoundIsTyped(t *testing.T) {
	repo, _, _ := setupExecutionRepoTest(t)
	_, err := repo.FindByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, pkgmodels.ErrExecutionNotFound)
	_, err = repo.FindByIDWithRelations(context.Background(), uuid.New())
	require.ErrorIs(t, err, pkgmodels.ErrExecutionNotFound)
}

func TestExecutionRepo_CancelMethods_DBErrors(t *testing.T) {
	repo, _, cleanup := setupExecutionRepoTest(t)
	cleanup()

	_, err := repo.CancelExecution(context.Background(), uuid.New(), "x")
	require.Error(t, err)
	_, err = repo.CancelledExecutions(context.Background(), []uuid.UUID{uuid.New()})
	require.Error(t, err)
}
