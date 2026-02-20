package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

// Config holds database configuration
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Debug           bool
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	return &Config{
		MaxOpenConns:    20,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
		Debug:           false,
	}
}

// NewDB creates a new Bun database connection
func NewDB(cfg *Config) (*bun.DB, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Parse DSN
	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(cfg.DSN),
		pgdriver.WithTimeout(30*time.Second),
		pgdriver.WithDialTimeout(10*time.Second),
		pgdriver.WithReadTimeout(10*time.Second),
		pgdriver.WithWriteTimeout(10*time.Second),
	)

	// Create SQL DB
	sqldb := sql.OpenDB(connector)

	// Configure connection pool
	sqldb.SetMaxOpenConns(cfg.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.MaxIdleConns)
	sqldb.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqldb.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Create Bun DB
	db := bun.NewDB(sqldb, pgdialect.New())

	// Add query hook for debugging if enabled
	if cfg.Debug {
		db.WithQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	// Register models for Bun
	registerModels(db)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("database connection established",
		slog.Int("max_open_conns", cfg.MaxOpenConns),
		slog.Int("max_idle_conns", cfg.MaxIdleConns),
	)

	return db, nil
}

// registerModels registers all Bun models
func registerModels(db *bun.DB) {
	db.RegisterModel(
		(*models.WorkflowModel)(nil),
		(*models.NodeModel)(nil),
		(*models.EdgeModel)(nil),
		(*models.ExecutionModel)(nil),
		(*models.NodeExecutionModel)(nil),
		(*models.EventModel)(nil),
		(*models.TriggerModel)(nil),
		// Auth models (UserRoleModel must be registered first for m2m relations)
		(*models.UserRoleModel)(nil),
		(*models.UserModel)(nil),
		(*models.SessionModel)(nil),
		(*models.RoleModel)(nil),
		(*models.AuditLogModel)(nil),
	)
}

// Close closes the database connection
func Close(db *bun.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

// Ping pings the database to verify connection
func Ping(ctx context.Context, db *bun.DB) error {
	return db.PingContext(ctx)
}

// Stats returns database connection statistics
func Stats(db *bun.DB) sql.DBStats {
	return db.DB.Stats()
}

// WithTransaction executes a function within a database transaction
func WithTransaction(ctx context.Context, db *bun.DB, fn func(tx bun.Tx) error) error {
	return db.RunInTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	}, func(ctx context.Context, tx bun.Tx) error {
		return fn(tx)
	})
}
