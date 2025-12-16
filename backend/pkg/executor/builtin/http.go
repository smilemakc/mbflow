// Package builtin provides built-in executor implementations.
package builtin

import (
	"bytes"
	"context"
	"encoding/base64"
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

	// Build request body
	var body io.Reader
	if config["body"] != nil {
		var bodyData []byte
		var err error

		switch v := config["body"].(type) {
		case string:
			// If body is already a string, use it directly (avoid double serialization)
			bodyData = []byte(v)
		case []byte:
			// If body is bytes, use directly
			bodyData = v
		default:
			// For maps, slices, etc. - serialize to JSON
			bodyData, err = json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
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

	// Get content type
	contentType := resp.Header.Get("Content-Type")

	// Check if binary response is requested or content type indicates binary
	responseType := e.GetStringDefault(config, "response_type", "auto")
	isBinary := responseType == "binary" || isBinaryContentType(contentType)

	result := map[string]interface{}{
		"status":       resp.StatusCode,
		"headers":      resp.Header,
		"content_type": contentType,
	}

	if isBinary {
		// Return base64 encoded body for binary content
		result["body"] = nil
		result["body_base64"] = base64.StdEncoding.EncodeToString(respBody)
		result["size"] = len(respBody)
	} else {
		// Parse response as JSON or string
		var parsedBody interface{}
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &parsedBody); err != nil {
				// If not JSON, return as string
				parsedBody = string(respBody)
			}
		}
		result["body"] = parsedBody
	}

	return result, nil
}

// isBinaryContentType checks if content type indicates binary data
func isBinaryContentType(contentType string) bool {
	binaryPrefixes := []string{
		"image/",
		"audio/",
		"video/",
		"application/octet-stream",
		"application/pdf",
		"application/zip",
		"application/gzip",
	}
	for _, prefix := range binaryPrefixes {
		if len(contentType) >= len(prefix) && contentType[:len(prefix)] == prefix {
			return true
		}
	}
	return false
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
