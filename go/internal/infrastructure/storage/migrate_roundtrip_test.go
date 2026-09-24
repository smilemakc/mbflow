package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/migrations"
	"github.com/smilemakc/mbflow/go/testutil"
)

// TestMigrations_RoundTrip rolls back every migration and re-applies them,
// verifying that all down migrations work against the current schema.
func TestMigrations_RoundTrip(t *testing.T) {
	t.Parallel()

	db, err := NewDB(&Config{DSN: testutil.SetupTestDSN(t), MaxOpenConns: 2, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = Close(db) })

	migrator, err := NewMigrator(db, migrations.FS)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, migrator.Init(ctx))

	// The template database applies all migrations as a single group.
	require.NoError(t, migrator.Down(ctx))

	var exists bool
	require.NoError(t, db.NewRaw("SELECT to_regclass('mbflow_workflows') IS NOT NULL").Scan(ctx, &exists))
	require.False(t, exists, "schema should be gone after rolling back all migrations")

	require.NoError(t, migrator.Up(ctx))
	require.NoError(t, db.NewRaw(
		"SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mbflow_files' AND column_name = 'created_by')",
	).Scan(ctx, &exists))
	require.True(t, exists, "latest migration should be applied again")
}
