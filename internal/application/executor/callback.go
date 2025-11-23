package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NodeCallbackProcessor defines the interface for processing callbacks after successful node execution.
// Callbacks are executed asynchronously and their results do not affect the workflow execution.
type NodeCallbackProcessor interface {
	// Process executes the callback with the given node execution data.
	// It should not block the workflow execution and should handle errors internally.
	Process(ctx context.Context, data *NodeCallbackData) error
}

// NodeCallbackData contains the data passed to the callback processor.
type NodeCallbackData struct {
	ExecutionID string                 `json:"execution_id"`
	WorkflowID  string                 `json:"workflow_id"`
	NodeID      string                 `json:"node_id"`
	NodeType    string                 `json:"node_type"`
	Output      interface{}            `json:"output"`
	Duration    time.Duration          `json:"duration_ms"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
}

// HTTPCallbackProcessor sends a POST request to an HTTP endpoint with the callback data.
type HTTPCallbackProcessor struct {
	url        string
	method     string
	headers    map[string]string
	timeout    time.Duration
	httpClient *http.Client
}

// HTTPCallbackConfig configures the HTTP callback processor.
type HTTPCallbackConfig struct {
	URL              string            `json:"url"`
	Method           string            `json:"method,omitempty"` // Default: POST
	Headers          map[string]string `json:"headers,omitempty"`
	TimeoutSeconds   int               `json:"timeout_seconds,omitempty"`   // Default: 30
	IncludeVariables bool              `json:"include_variables,omitempty"` // Default: false
}

// NewHTTPCallbackProcessor creates a new HTTP callback processor.
func NewHTTPCallbackProcessor(config HTTPCallbackConfig) (*HTTPCallbackProcessor, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("callback URL is required")
	}

	method := config.Method
	if method == "" {
		method = "POST"
	}

	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &HTTPCallbackProcessor{
		url:     config.URL,
		method:  method,
		headers: config.Headers,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Process sends the callback data to the configured HTTP endpoint.
func (p *HTTPCallbackProcessor) Process(ctx context.Context, data *NodeCallbackData) error {
	// Prepare JSON payload
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal callback data: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, p.method, p.url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range p.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for logging
	_, _ = io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback returned non-success status code: %d", resp.StatusCode)
	}

	return nil
}

// parseCallbackConfig parses the callback configuration from node config.
func parseCallbackConfig(config map[string]any) (*HTTPCallbackConfig, error) {
	callbackConfigRaw, ok := config["on_success_callback"]
	if !ok {
		return nil, nil
	}
	// Shortcut for inline config
	if cfg, ok := callbackConfigRaw.(HTTPCallbackConfig); ok {
		return &cfg, nil
	}
	// Validate config format
	callbackConfigMap, ok := callbackConfigRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("on_success_callback must be an object")
	}

	// Marshal to JSON and unmarshal to config struct
	data, err := json.Marshal(callbackConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal callback config: %w", err)
	}

	var callbackConfig HTTPCallbackConfig
	if err := json.Unmarshal(data, &callbackConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal callback config: %w", err)
	}

	return &callbackConfig, nil
}
