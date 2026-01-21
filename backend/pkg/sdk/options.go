package sdk

import (
	"fmt"
	"net/http"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
)

// ClientOption is a function that configures a Client.
type ClientOption func(*ClientConfig) error

// WithHTTPEndpoint configures the client to connect to a remote MBFlow API server.
// This sets the client to remote mode.
func WithHTTPEndpoint(baseURL string) ClientOption {
	return func(c *ClientConfig) error {
		if baseURL == "" {
			return fmt.Errorf("base URL cannot be empty")
		}
		c.Mode = ModeRemote
		c.BaseURL = baseURL
		return nil
	}
}

// WithAPIKey sets the API key for authenticating with the remote server.
func WithAPIKey(apiKey string) ClientOption {
	return func(c *ClientConfig) error {
		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}
		c.APIKey = apiKey
		return nil
	}
}

// WithHTTPClient sets a custom HTTP client for remote mode.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *ClientConfig) error {
		if httpClient == nil {
			return fmt.Errorf("HTTP client cannot be nil")
		}
		c.HTTPClient = httpClient
		return nil
	}
}

// WithTimeout sets the timeout for API requests.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *ClientConfig) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
		c.Timeout = timeout
		if c.HTTPClient != nil {
			c.HTTPClient.Timeout = timeout
		}
		return nil
	}
}

// WithStandaloneMode configures the client for in-memory workflow execution without persistence.
// In this mode, only ExecuteWorkflowStandalone() is available.
// No database or Redis connection is required.
// Perfect for testing, examples, and simple automation scripts.
func WithStandaloneMode() ClientOption {
	return func(c *ClientConfig) error {
		c.Mode = ModeEmbedded
		c.DatabaseURL = ""
		c.RedisURL = ""
		return nil
	}
}

// WithEmbeddedMode configures the client to run the workflow engine in-process with persistence.
// This requires database and optionally Redis URLs.
// Provides full functionality including workflow CRUD operations and execution history.
func WithEmbeddedMode(databaseURL, redisURL string) ClientOption {
	return func(c *ClientConfig) error {
		if databaseURL == "" {
			return fmt.Errorf("database URL is required for embedded mode with persistence (use WithStandaloneMode for in-memory execution)")
		}
		c.Mode = ModeEmbedded
		c.DatabaseURL = databaseURL
		c.RedisURL = redisURL
		return nil
	}
}

// WithDatabase sets the database URL for embedded mode.
func WithDatabase(databaseURL string) ClientOption {
	return func(c *ClientConfig) error {
		if databaseURL == "" {
			return fmt.Errorf("database URL cannot be empty")
		}
		c.DatabaseURL = databaseURL
		return nil
	}
}

// WithRedis sets the Redis URL for embedded mode.
func WithRedis(redisURL string) ClientOption {
	return func(c *ClientConfig) error {
		c.RedisURL = redisURL
		return nil
	}
}

// WithExecutorManager sets a custom executor manager.
// This is useful for registering custom executors before client initialization.
func WithExecutorManager(manager executor.Manager) ClientOption {
	return func(c *ClientConfig) error {
		if manager == nil {
			return fmt.Errorf("executor manager cannot be nil")
		}
		c.ExecutorManager = manager
		return nil
	}
}

// WithLogger sets a custom logger for the client.
func WithLogger(logger Logger) ClientOption {
	return func(c *ClientConfig) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}
		c.Logger = logger
		return nil
	}
}

// WithWebhookBaseURL sets the base URL for webhook endpoints in embedded mode.
// This is used when generating webhook URLs for triggers.
// Example: "http://localhost:8585" or "https://api.example.com"
func WithWebhookBaseURL(baseURL string) ClientOption {
	return func(c *ClientConfig) error {
		c.WebhookBaseURL = baseURL
		return nil
	}
}
