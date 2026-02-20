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

	"github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/executor/builtin"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// Client is the main entry point for the MBFlow SDK.
// It provides access to workflow, execution, and trigger management APIs.
//
// The SDK supports two modes:
//   - Remote mode: Connects to a remote MBFlow API server via HTTP
//   - Standalone mode: Executes workflows in-memory without persistence
//
// For embedded mode with database persistence, use pkg/server.Server directly.
type Client struct {
	config *ClientConfig
	mu     sync.RWMutex

	// API clients
	workflows  *WorkflowAPI
	executions *ExecutionAPI
	triggers   *TriggerAPI

	// HTTP client for remote mode
	httpClient *http.Client

	// Standalone mode components
	executorManager    executor.Manager
	standaloneExecutor engine.StandaloneExecutor
	observerManager    engine.ObserverManager

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

	// Standalone mode settings (kept for backward compatibility, but ignored)
	DatabaseURL    string
	RedisURL       string
	WebhookBaseURL string
	AutoMigrate    bool

	// Executor configuration
	ExecutorManager executor.Manager

	// Observer configuration (for real-time event notifications)
	ObserverManager engine.ObserverManager

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
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
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

// RegisterExecutor registers a custom executor with the standalone engine.
// Only available in standalone/embedded mode.
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

// initializeEmbedded sets up the standalone workflow engine.
// Note: For embedded mode with database persistence, use pkg/server.Server directly.
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

	// Create standalone executor for in-memory workflow execution
	c.standaloneExecutor = engine.NewStandaloneExecutor(c.executorManager)

	// Set observer manager if provided
	if c.config.ObserverManager != nil {
		c.observerManager = c.config.ObserverManager
	}

	return nil
}

// getStandaloneExecutor returns the standalone executor
func (c *Client) getStandaloneExecutor() engine.StandaloneExecutor {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.standaloneExecutor
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
		// Standalone mode - always healthy if executor is available
		if c.standaloneExecutor != nil {
			status.Status = "ok"
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
