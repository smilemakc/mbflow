package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/migrations"
)

var (
	command     string
	databaseURL string
)

func init() {
	flag.StringVar(&command, "command", "up", "Migration command: init, up, down, status, reset")
	flag.StringVar(&databaseURL, "database-url", "", "PostgreSQL database URL (overrides DATABASE_URL env var)")
}

func main() {
	flag.Parse()

	// Load .env file if exists
	_ = godotenv.Load()

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Get database URL
	dbURL := databaseURL
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	// Create database connection
	cfg := &storage.Config{
		DSN:             dbURL,
		MaxOpenConns:    5, // Lower for migrations
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
		Debug:           os.Getenv("DEBUG") == "true",
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer storage.Close(db)

	// Create migrator
	migrator, err := storage.NewMigrator(db, migrations.FS)
	if err != nil {
		slog.Error("failed to create migrator", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Execute command
	if err := executeCommand(ctx, migrator, command); err != nil {
		slog.Error("migration command failed",
			slog.String("command", command),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	slog.Info("migration command completed successfully", slog.String("command", command))
}

func executeCommand(ctx context.Context, migrator *storage.Migrator, cmd string) error {
	switch cmd {
	case "init":
		return migrator.Init(ctx)
	case "up":
		// Initialize if needed, then run migrations
		if err := migrator.Init(ctx); err != nil {
			return fmt.Errorf("init failed: %w", err)
		}
		return migrator.Up(ctx)
	case "down":
		return migrator.Down(ctx)
	case "status":
		return migrator.Status(ctx)
	case "reset":
		return migrator.Reset(ctx)
	default:
		return fmt.Errorf("unknown command: %s (available: init, up, down, status, reset)", cmd)
	}
}
