// Package sdk provides the official Go SDK for MBFlow workflow orchestration engine.
//
// The SDK offers a clean, type-safe interface for interacting with MBFlow workflows,
// executions, and triggers. It supports both embedded (in-process) and remote (HTTP API)
// deployment modes.
//
// Example usage:
//
//	client := sdk.NewClient(
//		sdk.WithHTTPEndpoint("http://localhost:8181"),
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
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/executor/builtin"
	"github.com/smilemakc/mbflow/pkg/models"
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
	executorManager  executor.Manager
	executionManager *engine.ExecutionManager
	workflowRepo     repository.WorkflowRepository
	executionRepo    repository.ExecutionRepository
	eventRepo        repository.EventRepository

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
	DatabaseURL string
	RedisURL    string

	// Executor configuration
	ExecutorManager executor.Manager

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
		// Close database connections, etc.
		// TODO: Implement cleanup
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

		// Create repositories
		c.workflowRepo = storage.NewWorkflowRepository(db)
		c.executionRepo = storage.NewExecutionRepository(db)
		// Note: eventRepo is deferred for MVP

		// Create execution manager
		c.executionManager = engine.NewExecutionManager(
			c.executorManager,
			c.workflowRepo,
			c.executionRepo,
			nil, // eventRepo - will be nil for MVP
		)
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

	// TODO: Perform health check against remote API

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

	// TODO: Implement health check
	return &HealthStatus{
		Status: "ok",
		Mode:   c.config.Mode,
	}, nil
}

// HealthStatus represents the health status of the MBFlow system.
type HealthStatus struct {
	Status  string     `json:"status"`
	Mode    ClientMode `json:"mode"`
	Version string     `json:"version,omitempty"`
}
