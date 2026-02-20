package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"

	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/migrations"
)

const (
	embeddedUser     = "mbflow_test"
	embeddedPassword = "mbflow_test"
	templateDatabase = "mbflow_template"
)

var (
	adminDB      *bun.DB
	sharedEPG    *embeddedpostgres.EmbeddedPostgres
	embeddedPort uint32
)

// freePort asks the OS for an available TCP port.
func freePort() (uint32, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return uint32(port), nil
}

func dsnForDB(dbName string) string {
	return fmt.Sprintf(
		"postgres://%s:%s@localhost:%d/%s?sslmode=disable",
		embeddedUser, embeddedPassword, embeddedPort, dbName,
	)
}

// RunWithEmbeddedDB is a TestMain helper that starts embedded PostgreSQL
// on a random free port, runs all tests, then stops it.
// Each package gets its own postgres instance, so packages can run in parallel.
//
//	func TestMain(m *testing.M) {
//	    os.Exit(testutil.RunWithEmbeddedDB(m))
//	}
func RunWithEmbeddedDB(m *testing.M) int {
	if err := startEmbeddedDB(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start embedded postgres: %v\n", err)
		return 1
	}
	defer stopSharedDB()

	return m.Run()
}

func startEmbeddedDB() error {
	port, err := freePort()
	if err != nil {
		return fmt.Errorf("free port: %w", err)
	}
	embeddedPort = port

	// Each instance gets its own data directory to avoid conflicts
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("epg-%d", port))
	os.RemoveAll(dataDir)

	sharedEPG = embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(port).
			Username(embeddedUser).
			Password(embeddedPassword).
			Database(embeddedUser).
			RuntimePath(dataDir),
	)

	if err := sharedEPG.Start(); err != nil {
		return fmt.Errorf("start on port %d: %w", port, err)
	}

	// Connect to default DB to create the template
	adminDB = openDB(embeddedUser)

	// Create template database with migrations
	ctx := context.Background()
	if _, err := adminDB.ExecContext(ctx, "DROP DATABASE IF EXISTS "+templateDatabase); err != nil {
		sharedEPG.Stop()
		return fmt.Errorf("drop old template: %w", err)
	}
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+templateDatabase); err != nil {
		sharedEPG.Stop()
		return fmt.Errorf("create template db: %w", err)
	}

	// Connect to template DB, run migrations, then close
	tmplDB := openDB(templateDatabase)
	if err := runMigrations(tmplDB); err != nil {
		tmplDB.Close()
		sharedEPG.Stop()
		return fmt.Errorf("migrations: %w", err)
	}
	tmplDB.Close()

	return nil
}

func openDB(dbName string) *bun.DB {
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsnForDB(dbName)))
	sqldb := sql.OpenDB(connector)
	db := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())
	db.RegisterModel((*storagemodels.UserRoleModel)(nil))
	return db
}

func stopSharedDB() {
	if adminDB != nil {
		adminDB.Close()
		adminDB = nil
	}
	if sharedEPG != nil {
		_ = sharedEPG.Stop()
		sharedEPG = nil
	}
	// Clean up data directory
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("epg-%d", embeddedPort))
	os.RemoveAll(dataDir)
}

func runMigrations(db *bun.DB) error {
	discoveredMigrations := migrate.NewMigrations()
	if err := discoveredMigrations.Discover(migrations.FS); err != nil {
		return fmt.Errorf("discover migrations: %w", err)
	}

	migrator := migrate.NewMigrator(db, discoveredMigrations,
		migrate.WithTableName("mbflow_bun_migrations"),
		migrate.WithLocksTableName("mbflow_bun_migration_locks"),
	)

	ctx := context.Background()
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// SetupTestTx creates an isolated database from template for each test.
// Safe for parallel execution — each test gets its own database.
// Requires RunWithEmbeddedDB in TestMain.
func SetupTestTx(t *testing.T) (bun.IDB, func()) {
	t.Helper()

	if adminDB == nil {
		t.Fatal("embedded postgres not started — add TestMain with testutil.RunWithEmbeddedDB(m)")
	}

	// Unique DB name per test (postgres identifiers max 63 chars)
	short := strings.ReplaceAll(uuid.New().String()[:8], "-", "")
	dbName := "mbflow_t_" + short

	ctx := context.Background()

	// Create DB from template — fast copy, no migrations needed
	_, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", dbName, templateDatabase))
	if err != nil {
		t.Fatalf("create test db %s: %v", dbName, err)
	}

	db := openDB(dbName)

	cleanup := func() {
		db.Close()
		_, _ = adminDB.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+dbName)
	}
	t.Cleanup(cleanup)

	return db, cleanup
}
