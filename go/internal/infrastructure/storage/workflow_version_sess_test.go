package storage

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func TestSess_WorkflowRepository_UpdateIncrementsVersion(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewWorkflowRepository(db)

	wf := rcWorkflow("versioned", nil)
	require.NoError(t, repo.Create(ctx, wf))

	got, err := repo.FindByIDWithRelations(ctx, wf.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version)
	assert.Equal(t, 1, models.WorkflowModelToDomain(got).Version)

	got.Description = "first edit"
	// A stale version supplied by the caller must not win.
	got.Version = 1
	require.NoError(t, repo.Update(ctx, got))
	assert.Equal(t, 2, got.Version, "Update reports the new version on the model")

	got.Status = "active"
	require.NoError(t, repo.Update(ctx, got))
	assert.Equal(t, 3, got.Version)

	reloaded, err := repo.FindByIDWithRelations(ctx, wf.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, reloaded.Version)
	assert.Equal(t, "first edit", reloaded.Description)
	assert.Len(t, reloaded.Nodes, 3)
	assert.Equal(t, 3, models.WorkflowModelToDomain(reloaded).Version)

	// Updating a missing workflow is a no-op, not a version bump elsewhere.
	missing := rcWorkflow("missing", nil)
	missing.ID = uuid.New()
	missing.Nodes, missing.Edges = nil, nil
	require.NoError(t, repo.Update(ctx, missing))
	again, err := repo.FindByID(ctx, wf.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, again.Version)
}
