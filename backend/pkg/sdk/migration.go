package sdk

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/models"
)

// MigrationAPI provides methods for database migrations in embedded mode.
// It allows manual control over database schema migrations including
// running migrations up, rolling back, checking status, and resetting.
type MigrationAPI struct {
	client *Client
}

// MigrationResult contains the result of a migration operation.
type MigrationResult struct {
	// Applied contains the names of migrations that were applied (for Up)
	// or rolled back (for Down).
	Applied []string
	// GroupID is the migration group ID assigned to this operation.
	GroupID int64
}

// MigrationStatus represents the status of a single migration.
type MigrationStatus struct {
	// Name is the migration file name.
	Name string
	// Applied indicates whether this migration has been applied.
	Applied bool
	// GroupID is the group ID if the migration was applied, 0 otherwise.
	GroupID int64
}

// newMigrationAPI creates a new MigrationAPI instance.
func newMigrationAPI(client *Client) *MigrationAPI {
	return &MigrationAPI{client: client}
}

// Up runs all pending migrations.
// Returns information about the applied migrations or an error.
func (m *MigrationAPI) Up(ctx context.Context) (*MigrationResult, error) {
	if err := m.client.checkClosed(); err != nil {
		return nil, err
	}

	if m.client.config.Mode != ModeEmbedded {
		return nil, fmt.Errorf("migrations only available in embedded mode")
	}

	migrator, err := m.getMigrator()
	if err != nil {
		return nil, err
	}

	// Initialize migration tables if not exists
	if err := migrator.Init(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize migrations: %w", err)
	}

	// Run migrations using the underlying bun migrator
	group, err := migrator.Migrate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	result := &MigrationResult{
		Applied: make([]string, 0),
		GroupID: group.ID,
	}

	if !group.IsZero() {
		for _, migration := range group.Migrations {
			result.Applied = append(result.Applied, migration.Name)
		}
	}

	return result, nil
}

// Down rolls back the last migration group.
// Returns information about the rolled back migrations or an error.
func (m *MigrationAPI) Down(ctx context.Context) (*MigrationResult, error) {
	if err := m.client.checkClosed(); err != nil {
		return nil, err
	}

	if m.client.config.Mode != ModeEmbedded {
		return nil, fmt.Errorf("migrations only available in embedded mode")
	}

	migrator, err := m.getMigrator()
	if err != nil {
		return nil, err
	}

	group, err := migrator.Rollback(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to rollback migrations: %w", err)
	}

	result := &MigrationResult{
		Applied: make([]string, 0),
		GroupID: group.ID,
	}

	if !group.IsZero() {
		for _, migration := range group.Migrations {
			result.Applied = append(result.Applied, migration.Name)
		}
	}

	return result, nil
}

// Status returns the current status of all migrations.
func (m *MigrationAPI) Status(ctx context.Context) ([]MigrationStatus, error) {
	if err := m.client.checkClosed(); err != nil {
		return nil, err
	}

	if m.client.config.Mode != ModeEmbedded {
		return nil, fmt.Errorf("migrations only available in embedded mode")
	}

	migrator, err := m.getMigrator()
	if err != nil {
		return nil, err
	}

	// Initialize migration tables if not exists (needed to get status)
	if err := migrator.Init(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize migrations: %w", err)
	}

	ms, err := migrator.MigrationsWithStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	statuses := make([]MigrationStatus, 0, len(ms))
	for _, migration := range ms {
		statuses = append(statuses, MigrationStatus{
			Name:    migration.Name,
			Applied: migration.GroupID > 0,
			GroupID: migration.GroupID,
		})
	}

	return statuses, nil
}

// Reset rolls back all applied migrations.
// This is a destructive operation that will drop all tables managed by migrations.
func (m *MigrationAPI) Reset(ctx context.Context) error {
	if err := m.client.checkClosed(); err != nil {
		return err
	}

	if m.client.config.Mode != ModeEmbedded {
		return fmt.Errorf("migrations only available in embedded mode")
	}

	migrator, err := m.getMigrator()
	if err != nil {
		return err
	}

	// Roll back all migrations one group at a time
	for {
		group, err := migrator.Rollback(ctx)
		if err != nil {
			return fmt.Errorf("failed to rollback migrations: %w", err)
		}
		if group.IsZero() {
			break
		}
	}

	return nil
}

// getMigrator returns the migrator instance.
func (m *MigrationAPI) getMigrator() (*storage.MigratorWithAccess, error) {
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	if m.client.db == nil {
		return nil, fmt.Errorf("database connection not initialized (use WithEmbeddedMode)")
	}

	if m.client.migrator != nil {
		return m.client.migrator, nil
	}

	return nil, models.ErrMigratorNotInitialized
}
