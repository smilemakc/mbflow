package sdk

import (
	"context"
	"fmt"
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
// Note: Migrations are not available in SDK standalone mode.
// Use pkg/server.Server for database operations.
func (m *MigrationAPI) Up(ctx context.Context) (*MigrationResult, error) {
	if err := m.client.checkClosed(); err != nil {
		return nil, err
	}
	_, err := m.getMigrator()
	return nil, err
}

// Down rolls back the last migration group.
// Note: Migrations are not available in SDK standalone mode.
// Use pkg/server.Server for database operations.
func (m *MigrationAPI) Down(ctx context.Context) (*MigrationResult, error) {
	if err := m.client.checkClosed(); err != nil {
		return nil, err
	}
	_, err := m.getMigrator()
	return nil, err
}

// Status returns the current status of all migrations.
// Note: Migrations are not available in SDK standalone mode.
// Use pkg/server.Server for database operations.
func (m *MigrationAPI) Status(ctx context.Context) ([]MigrationStatus, error) {
	if err := m.client.checkClosed(); err != nil {
		return nil, err
	}
	_, err := m.getMigrator()
	return nil, err
}

// Reset rolls back all applied migrations.
// Note: Migrations are not available in SDK standalone mode.
// Use pkg/server.Server for database operations.
func (m *MigrationAPI) Reset(ctx context.Context) error {
	if err := m.client.checkClosed(); err != nil {
		return err
	}
	_, err := m.getMigrator()
	return err
}

var errMigrationsNotAvailable = fmt.Errorf("migrations not available in SDK standalone mode; use pkg/server.Server for database operations")

// getMigrator returns an error since migrations are not supported in SDK standalone mode.
// For migration support, use pkg/server.Server directly.
func (m *MigrationAPI) getMigrator() (interface{}, error) {
	return nil, errMigrationsNotAvailable
}
