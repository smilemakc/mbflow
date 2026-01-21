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
	isErrorStatus := resp.StatusCode >= 400
	if isErrorStatus {
		// Check if we should ignore status errors or use custom success codes
		ignoreStatusErrors := e.GetBoolDefault(config, "ignore_status_errors", false)
		successStatusCodes := e.getIntSlice(config, "success_status_codes")

		if len(successStatusCodes) > 0 {
			// Use explicit success status codes if provided
			isAllowed := false
			for _, code := range successStatusCodes {
				if resp.StatusCode == code {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			}
		} else if !ignoreStatusErrors {
			// Default behavior: error on 4xx/5xx
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
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
		"is_error":     isErrorStatus,
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

// getIntSlice retrieves a slice of integers from config.
func (e *HTTPExecutor) getIntSlice(config map[string]interface{}, key string) []int {
	val, ok := config[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []int:
		return v
	case []interface{}:
		result := make([]int, 0, len(v))
		for _, item := range v {
			switch n := item.(type) {
			case float64:
				result = append(result, int(n))
			case int:
				result = append(result, n)
			}
		}
		return result
	default:
		return nil
	}
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

	// Validate ignore_status_errors if provided
	if val, ok := config["ignore_status_errors"]; ok {
		if _, isBool := val.(bool); !isBool {
			return fmt.Errorf("ignore_status_errors must be a boolean")
		}
	}

	// Validate success_status_codes if provided
	if val, ok := config["success_status_codes"]; ok {
		codes := e.getIntSlice(config, "success_status_codes")
		if codes == nil {
			return fmt.Errorf("success_status_codes must be an array of integers, got: %T", val)
		}
		for _, code := range codes {
			if code < 100 || code > 599 {
				return fmt.Errorf("invalid HTTP status code in success_status_codes: %d", code)
			}
		}
	}

	return nil
}
