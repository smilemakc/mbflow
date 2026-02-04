package storage

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// Migrator wraps bun's migrate.Migrator
type Migrator struct {
	migrator *migrate.Migrator
	db       *bun.DB
}

// MigratorWithAccess extends Migrator with direct access to underlying bun migrator methods.
// This is used by the SDK to provide more detailed migration results.
type MigratorWithAccess struct {
	*Migrator
}

// NewMigratorWithAccess creates a new MigratorWithAccess instance.
func NewMigratorWithAccess(db *bun.DB, migrationsFS fs.FS) (*MigratorWithAccess, error) {
	m, err := NewMigrator(db, migrationsFS)
	if err != nil {
		return nil, err
	}
	return &MigratorWithAccess{Migrator: m}, nil
}

// Migrate runs pending migrations and returns the migration group.
func (m *MigratorWithAccess) Migrate(ctx context.Context) (*migrate.MigrationGroup, error) {
	group, err := m.migrator.Migrate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}
	return group, nil
}

// Rollback rolls back the last migration group and returns it.
func (m *MigratorWithAccess) Rollback(ctx context.Context) (*migrate.MigrationGroup, error) {
	group, err := m.migrator.Rollback(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to rollback: %w", err)
	}
	return group, nil
}

// MigrationsWithStatus returns all migrations with their current status.
func (m *MigratorWithAccess) MigrationsWithStatus(ctx context.Context) (migrate.MigrationSlice, error) {
	ms, err := m.migrator.MigrationsWithStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}
	return ms, nil
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *bun.DB, migrationsFS fs.FS) (*Migrator, error) {
	migrations := migrate.NewMigrations()

	if err := migrations.Discover(migrationsFS); err != nil {
		return nil, fmt.Errorf("failed to discover migrations: %w", err)
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
