package storage

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// Migrator wraps bun's migrate.Migrator
type Migrator struct {
	migrator *migrate.Migrator
	db       *bun.DB
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *bun.DB, migrationsDir string) (*Migrator, error) {
	migrations := migrate.NewMigrations()

	// Load migrations from filesystem directory
	if migrationsDir != "" {
		if err := migrations.Discover(os.DirFS(migrationsDir)); err != nil {
			return nil, fmt.Errorf("failed to discover migrations: %w", err)
		}
	}

	migrator := migrate.NewMigrator(db, migrations)

	return &Migrator{
		migrator: migrator,
		db:       db,
	}, nil
}

// Init initializes the migration tables
func (m *Migrator) Init(ctx context.Context) error {
	slog.Info("initializing migration tables")
	return m.migrator.Init(ctx)
}

// Up runs all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	slog.Info("running migrations up")

	group, err := m.migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("failed to migrate: %w", err)
	}

	if group.IsZero() {
		slog.Info("no new migrations to run")
		return nil
	}

	slog.Info("migrations applied successfully",
		slog.Int64("id", group.ID),
		slog.String("migrations", fmt.Sprintf("%v", group.Migrations.Applied())),
	)

	return nil
}

// Down rolls back the last migration group
func (m *Migrator) Down(ctx context.Context) error {
	slog.Info("rolling back last migration")

	group, err := m.migrator.Rollback(ctx)
	if err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}

	if group.IsZero() {
		slog.Info("no migrations to rollback")
		return nil
	}

	slog.Info("migration rolled back successfully",
		slog.Int64("id", group.ID),
		slog.String("migrations", fmt.Sprintf("%v", group.Migrations.Unapplied())),
	)

	return nil
}

// Status returns the current migration status
func (m *Migrator) Status(ctx context.Context) error {
	ms, err := m.migrator.MigrationsWithStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	slog.Info("migration status", slog.Int("total", len(ms)))

	for _, migration := range ms {
		status := "pending"
		if migration.GroupID > 0 {
			status = "applied"
		}
		slog.Info("migration",
			slog.String("name", migration.Name),
			slog.String("status", status),
		)
	}

	return nil
}

// Reset rolls back all migrations
func (m *Migrator) Reset(ctx context.Context) error {
	slog.Warn("resetting all migrations (this will drop all tables)")

	for {
		group, err := m.migrator.Rollback(ctx)
		if err != nil {
			return fmt.Errorf("failed to rollback: %w", err)
		}
		if group.IsZero() {
			break
		}
		slog.Info("rolled back migration group", slog.Int64("id", group.ID))
	}

	slog.Info("all migrations rolled back")
	return nil
}

// CreateMigrationTable creates the migration tracking table
func (m *Migrator) CreateMigrationTable(ctx context.Context) error {
	return m.migrator.Init(ctx)
}
