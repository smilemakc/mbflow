// Package builtin provides built-in executor implementations.
package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
)

// HTTPExecutor executes HTTP requests.
type HTTPExecutor struct {
	*executor.BaseExecutor
	client *http.Client
}

// NewHTTPExecutor creates a new HTTP executor.
func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		BaseExecutor: executor.NewBaseExecutor("http"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute executes an HTTP request.
func (e *HTTPExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	// Get required fields
	method, err := e.GetString(config, "method")
	if err != nil {
		return nil, err
	}

	url, err := e.GetString(config, "url")
	if err != nil {
		return nil, err
	}

	// Build request
	var body io.Reader
	if config["body"] != nil {
		bodyData, err := json.Marshal(config["body"])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(bodyData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if headers, err := e.GetMap(config, "headers"); err == nil {
		for key, value := range headers {
			if strVal, ok := value.(string); ok {
				req.Header.Set(key, strVal)
			}
		}
	}

	// Set default content type
	if req.Header.Get("Content-Type") == "" && body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			// If not JSON, return as string
			result = string(respBody)
		}
	}

	return map[string]interface{}{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    result,
	}, nil
}

// Validate validates the HTTP executor configuration.
func (e *HTTPExecutor) Validate(config map[string]interface{}) error {
	// Validate required fields
	if err := e.ValidateRequired(config, "method", "url"); err != nil {
		return err
	}

	// Validate method
	method, err := e.GetString(config, "method")
	if err != nil {
		return err
	}

	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"PATCH":   true,
		"HEAD":    true,
		"OPTIONS": true,
	}

	if !validMethods[method] {
		return fmt.Errorf("invalid HTTP method: %s", method)
	}

	// Validate URL
	url, err := e.GetString(config, "url")
	if err != nil {
		return err
	}

	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	return nil
}
