// Package sdk provides the official Go SDK for MBFlow workflow orchestration engine.
//
// The SDK offers a clean, type-safe interface for interacting with MBFlow workflows,
// executions, and triggers. It supports both embedded (in-process) and remote (HTTP API)
// deployment modes.
//
// Example usage:
//
//	client := sdk.NewClient(
//		sdk.WithHTTPEndpoint("http://localhost:8585"),
//		sdk.WithAPIKey("your-api-key"),
//	)
//	defer client.Close()
//
//	workflow, err := client.Workflows().Create(ctx, &models.Workflow{
//		Name: "My Workflow",
//		// ... configure nodes and edges
//	})
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/executor/builtin"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/uptrace/bun"
)

// Client is the main entry point for the MBFlow SDK.
// It provides access to workflow, execution, and trigger management APIs.
type Client struct {
	config *ClientConfig
	mu     sync.RWMutex

	// API clients
	workflows  *WorkflowAPI
	executions *ExecutionAPI
	triggers   *TriggerAPI

	// HTTP client for remote mode
	httpClient *http.Client

	// Embedded mode components
	db               *bun.DB
	executorManager  executor.Manager
	executionManager *engine.ExecutionManager
	workflowRepo     repository.WorkflowRepository
	executionRepo    repository.ExecutionRepository
	eventRepo        repository.EventRepository
	resourceRepo     repository.ResourceRepository
	triggerRepo      repository.TriggerRepository
	observerManager  *observer.ObserverManager

	// Lifecycle
	closed bool
}

// ClientConfig holds the configuration for the MBFlow client.
type ClientConfig struct {
	// Mode determines how the client operates
	Mode ClientMode

	// Remote mode settings
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Timeout    time.Duration

	// Embedded mode settings
	DatabaseURL    string
	RedisURL       string
	WebhookBaseURL string // Base URL for webhook endpoints (e.g., "http://localhost:8585")
	MigrationsDir  string // Directory with migration files (enables auto-migrate if set)

	// Executor configuration
	ExecutorManager executor.Manager

	// Observer configuration (for real-time event notifications)
	ObserverManager *observer.ObserverManager

	// Logging
	Logger Logger
}

// ClientMode defines how the client operates.
type ClientMode int

const (
	// ModeEmbedded runs the workflow engine in-process
	ModeEmbedded ClientMode = iota

	// ModeRemote connects to a remote MBFlow API server
	ModeRemote
)

// Logger is the interface for structured logging.
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// NewClient creates a new MBFlow SDK client with the given options.
// By default, it operates in embedded mode.
func NewClient(opts ...ClientOption) (*Client, error) {
	config := &ClientConfig{
		Mode:    ModeEmbedded,
		Timeout: 30 * time.Second,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	client := &Client{
		config:     config,
		httpClient: config.HTTPClient,
	}

	// Initialize mode-specific components
	if err := client.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	// Initialize API clients
	client.workflows = newWorkflowAPI(client)
	client.executions = newExecutionAPI(client)
	client.triggers = newTriggerAPI(client)

	return client, nil
}

// NewStandaloneClient creates a new MBFlow SDK client in standalone mode.
// In standalone mode, workflows are executed in-memory without any database persistence.
// Only ExecuteWorkflowStandalone() is available - no workflow CRUD operations.
// Perfect for examples, testing, and simple automation scripts.
//
// Example:
//
//	client, err := sdk.NewStandaloneClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, nil)
func NewStandaloneClient(opts ...ClientOption) (*Client, error) {
	// Prepend WithStandaloneMode to the options
	allOpts := append([]ClientOption{WithStandaloneMode()}, opts...)
	return NewClient(allOpts...)
}

// Workflows returns the Workflow API for CRUD operations and DAG management.
func (c *Client) Workflows() *WorkflowAPI {
	return c.workflows
}

// Executions returns the Execution API for running and monitoring workflows.
func (c *Client) Executions() *ExecutionAPI {
	return c.executions
}

// Triggers returns the Trigger API for managing workflow triggers.
func (c *Client) Triggers() *TriggerAPI {
	return c.triggers
}

// RegisterExecutor registers a custom executor with the embedded engine.
// Only available in embedded mode.
func (c *Client) RegisterExecutor(nodeType string, exec executor.Executor) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return models.ErrClientClosed
	}

	if c.config.Mode != ModeEmbedded {
		return fmt.Errorf("executor registration only available in embedded mode")
	}

	if c.executorManager == nil {
		return fmt.Errorf("executor manager not initialized")
	}

	return c.executorManager.Register(nodeType, exec)
}

// Close releases all resources held by the client.
// It should be called when the client is no longer needed.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Close mode-specific resources
	if c.config.Mode == ModeEmbedded {
		// Close database connection
		if c.db != nil {
			if err := c.db.Close(); err != nil {
				return fmt.Errorf("failed to close database connection: %w", err)
			}
		}
	}

	return nil
}

// checkClosed returns an error if the client has been closed
func (c *Client) checkClosed() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return models.ErrClientClosed
	}

	return nil
}

// initialize sets up mode-specific components.
func (c *Client) initialize() error {
	switch c.config.Mode {
	case ModeEmbedded:
		return c.initializeEmbedded()
	case ModeRemote:
		return c.initializeRemote()
	default:
		return fmt.Errorf("unknown client mode: %d", c.config.Mode)
	}
}

// initializeEmbedded sets up the embedded workflow engine.
func (c *Client) initializeEmbedded() error {
	// Initialize executor manager
	if c.config.ExecutorManager != nil {
		c.executorManager = c.config.ExecutorManager
	} else {
		// Create default executor manager
		c.executorManager = executor.NewManager()
	}

	// Register built-in executors
	if err := builtin.RegisterBuiltins(c.executorManager); err != nil {
		return fmt.Errorf("failed to register built-in executors: %w", err)
	}

	// Initialize database connection if DatabaseURL provided
	if c.config.DatabaseURL != "" {
		dbConfig := &storage.Config{
			DSN:             c.config.DatabaseURL,
			MaxOpenConns:    20,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
			Debug:           false,
		}

		db, err := storage.NewDB(dbConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		c.db = db

		// Run migrations if configured
		if c.config.MigrationsDir != "" {
			if err := c.runMigrations(db); err != nil {
				return fmt.Errorf("failed to run migrations: %w", err)
			}
		}

		// Create repositories
		c.workflowRepo = storage.NewWorkflowRepository(db)
		c.executionRepo = storage.NewExecutionRepository(db)
		c.triggerRepo = storage.NewTriggerRepository(db)
		c.eventRepo = storage.NewEventRepository(db)
		c.resourceRepo = storage.NewResourceRepository(db)

		// Initialize observer manager if provided or create default
		if c.config.ObserverManager != nil {
			c.observerManager = c.config.ObserverManager
		} else {
			c.observerManager = observer.NewObserverManager()
		}

		// Create execution manager with all components
		c.executionManager = engine.NewExecutionManager(
			c.executorManager,
			c.workflowRepo,
			c.executionRepo,
			c.eventRepo,
			c.resourceRepo,
			c.observerManager,
		)
	}

	return nil
}

// runMigrations runs database migrations using the configured migrations directory.
func (c *Client) runMigrations(db *bun.DB) error {
	ctx := context.Background()

	migrator, err := storage.NewMigrator(db, c.config.MigrationsDir)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Initialize migration tables (creates bun_migrations table if not exists)
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}

	// Run pending migrations
	if err := migrator.Up(ctx); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// getExecutionManager returns the execution manager (internal method for ExecutionAPI)
func (c *Client) getExecutionManager() *engine.ExecutionManager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.executionManager
}

// initializeRemote validates the remote connection.
func (c *Client) initializeRemote() error {
	if c.config.BaseURL == "" {
		return fmt.Errorf("base URL is required for remote mode")
	}

	// Perform health check against remote API
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to remote API: %w", err)
	}
	defer resp.Body.Close()

	// Accept any 2xx status code as healthy
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("remote API health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// validateConfig validates the client configuration.
func validateConfig(config *ClientConfig) error {
	if config.Mode == ModeRemote {
		if config.BaseURL == "" {
			return fmt.Errorf("base URL is required for remote mode")
		}
	}

	// For embedded mode, DatabaseURL is optional
	// If not provided, only ExecuteWorkflowStandalone() will be available

	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}

// Health checks the health of the MBFlow system.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, models.ErrClientClosed
	}

	status := &HealthStatus{
		Status:  "ok",
		Mode:    c.config.Mode,
		Version: "1.0.0",
	}

	switch c.config.Mode {
	case ModeEmbedded:
		// Check database connectivity
		if c.db != nil {
			if err := c.db.PingContext(ctx); err != nil {
				status.Status = "degraded"
				status.Database = &ComponentHealth{
					Status: "unhealthy",
					Error:  err.Error(),
				}
			} else {
				status.Database = &ComponentHealth{
					Status: "healthy",
				}
			}
		}

	case ModeRemote:
		// Check remote API health
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL+"/health", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create health check request: %w", err)
		}

		if c.config.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			status.Status = "unhealthy"
			status.Remote = &ComponentHealth{
				Status: "unhealthy",
				Error:  err.Error(),
			}
			return status, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			status.Status = "unhealthy"
			status.Remote = &ComponentHealth{
				Status: "unhealthy",
				Error:  fmt.Sprintf("health check returned status %d", resp.StatusCode),
			}
		} else {
			// Parse remote health response
			var remoteHealth HealthStatus
			if err := json.NewDecoder(resp.Body).Decode(&remoteHealth); err == nil {
				status.Remote = &ComponentHealth{
					Status:  "healthy",
					Version: remoteHealth.Version,
				}
			} else {
				status.Remote = &ComponentHealth{
					Status: "healthy",
				}
			}
		}
	}

	return status, nil
}

// HealthStatus represents the health status of the MBFlow system.
type HealthStatus struct {
	Status   string           `json:"status"`
	Mode     ClientMode       `json:"mode"`
	Version  string           `json:"version,omitempty"`
	Database *ComponentHealth `json:"database,omitempty"`
	Remote   *ComponentHealth `json:"remote,omitempty"`
}

// ComponentHealth represents the health of a specific component.
type ComponentHealth struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Version string `json:"version,omitempty"`
}
