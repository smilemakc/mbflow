package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/migrations"
)

func main() {
	// Load .env file if exists
	_ = godotenv.Load()

	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		os.Exit(1)
	}
}

// run parses args (without the program name), connects to the database and
// executes the migration command. Progress and errors are logged to out.
func run(args []string, getenv func(string) string, out io.Writer) error {
	logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(out)
	command := fs.String("command", "up", "Migration command: init, up, down, status, reset")
	databaseURL := fs.String("database-url", "", "PostgreSQL database URL (overrides MBFLOW_DATABASE_URL / DATABASE_URL env vars)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dbURL := *databaseURL
	if dbURL == "" {
		dbURL = getenv("MBFLOW_DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = getenv("DATABASE_URL")
	}
	if dbURL == "" {
		err := fmt.Errorf("MBFLOW_DATABASE_URL (or DATABASE_URL) is required")
		logger.Error(err.Error())
		return err
	}

	cfg := &storage.Config{
		DSN:             dbURL,
		MaxOpenConns:    5, // Lower for migrations
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
		Debug:           getenv("DEBUG") == "true",
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		return err
	}
	defer func() {
		if cerr := storage.Close(db); cerr != nil {
			logger.Error("failed to close database", slog.String("error", cerr.Error()))
		}
	}()

	migrator, err := storage.NewMigrator(db, migrations.FS)
	if err != nil {
		logger.Error("failed to create migrator", slog.String("error", err.Error()))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := executeCommand(ctx, migrator, *command); err != nil {
		logger.Error("migration command failed",
			slog.String("command", *command),
			slog.String("error", err.Error()),
		)
		return err
	}

	logger.Info("migration command completed successfully", slog.String("command", *command))
	return nil
}

// migrationRunner is the part of *storage.Migrator the commands use.
type migrationRunner interface {
	Init(ctx context.Context) error
	Up(ctx context.Context) error
	Down(ctx context.Context) error
	Status(ctx context.Context) error
	Reset(ctx context.Context) error
}

func executeCommand(ctx context.Context, migrator migrationRunner, cmd string) error {
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
