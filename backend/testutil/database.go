//go:build integration

package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/migrations"
)

// TestDB encapsulates a test database with cleanup
type TestDB struct {
	DB       *bun.DB
	Pool     *dockertest.Pool
	Resource *dockertest.Resource
}

// SetupTestDB creates a PostgreSQL database using dockertest
// It will start a PostgreSQL 16 container and run migrations
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	testDB := &TestDB{}

	// Setup Dockertest pool
	var pool *dockertest.Pool
	var err error

	// Determine Docker endpoint
	dockerEndpoint := os.Getenv("DOCKER_HOST")
	if dockerEndpoint == "" {
		// Try macOS Docker Desktop socket
		macOSSocket := os.Getenv("HOME") + "/.docker/run/docker.sock"
		if _, statErr := os.Stat(macOSSocket); statErr == nil {
			dockerEndpoint = "unix://" + macOSSocket
		}
	}

	pool, err = dockertest.NewPool(dockerEndpoint)
	require.NoError(t, err, "Failed to connect to Docker. Is Docker running? Tried endpoint: %s", dockerEndpoint)

	// Verify Docker is accessible
	err = pool.Client.Ping()
	require.NoError(t, err, "Failed to ping Docker daemon")
	testDB.Pool = pool

	// Start PostgreSQL 16 container
	testDB.Resource, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16-alpine",
		Env: []string{
			"POSTGRES_USER=mbflow_test",
			"POSTGRES_PASSWORD=mbflow_test",
			"POSTGRES_DB=mbflow_test",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Set expiry for the container (10 minutes)
	testDB.Resource.Expire(600)

	// Wait for PostgreSQL to be ready
	var db *bun.DB
	err = pool.Retry(func() error {
		dsn := fmt.Sprintf("postgres://mbflow_test:mbflow_test@localhost:%s/mbflow_test?sslmode=disable",
			testDB.Resource.GetPort("5432/tcp"))

		connector := pgdriver.NewConnector(
			pgdriver.WithDSN(dsn),
			pgdriver.WithTimeout(5*time.Second),
		)
		sqldb := sql.OpenDB(connector)
		db = bun.NewDB(sqldb, pgdialect.New())

		return db.Ping()
	})
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	testDB.DB = db

	// Register m2m junction models required by bun for relation queries
	db.RegisterModel((*storagemodels.UserRoleModel)(nil))

	// Run migrations
	migrator := testDB.createMigrator(t)
	err = migrator.Init(context.Background())
	require.NoError(t, err, "Failed to initialize migrator")

	err = migrator.Up(context.Background())
	require.NoError(t, err, "Failed to run migrations")

	// Cleanup on test completion
	t.Cleanup(func() {
		testDB.Cleanup(t)
	})

	return testDB
}

// createMigrator creates a migrator using embedded migrations.
func (td *TestDB) createMigrator(t *testing.T) *storage.Migrator {
	t.Helper()

	migrator, err := storage.NewMigrator(td.DB, migrations.FS)
	require.NoError(t, err, "Failed to create migrator")

	return migrator
}

// Cleanup tears down the test database
func (td *TestDB) Cleanup(t *testing.T) {
	t.Helper()

	if td.DB != nil {
		td.DB.Close()
	}

	if td.Pool != nil && td.Resource != nil {
		if err := td.Pool.Purge(td.Resource); err != nil {
			t.Logf("Failed to purge PostgreSQL container: %v", err)
		}
	}
}

// GetDSN returns the database connection string
func (td *TestDB) GetDSN() string {
	return fmt.Sprintf("postgres://mbflow_test:mbflow_test@localhost:%s/mbflow_test?sslmode=disable",
		td.Resource.GetPort("5432/tcp"))
}

// Reset clears all data from the database (useful between tests)
func (td *TestDB) Reset(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	// Truncate all tables except migrations
	tables := []string{
		"mbflow_node_executions",
		"mbflow_executions",
		"mbflow_edges",
		"mbflow_nodes",
		"mbflow_workflows",
		"mbflow_triggers",
		"mbflow_events",
		"mbflow_files",
		"mbflow_workflow_resources",
		"mbflow_resource_credentials",
		"mbflow_credential_audit_log",
		"mbflow_resource_rental_key",
		"mbflow_rental_key_usage",
	}

	for _, table := range tables {
		_, err := td.DB.NewTruncateTable().Table(table).Cascade().Exec(ctx)
		if err != nil {
			// Ignore errors for tables that might not exist
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}
}
