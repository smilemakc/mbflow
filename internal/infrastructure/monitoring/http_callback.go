package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"mbflow/internal/domain"
)

// HTTPCallbackObserver sends execution events to an HTTP callback URL.
// It implements the ExecutionObserver interface and sends POST requests
// with JSON payloads for each execution event.
type HTTPCallbackObserver struct {
	// callbackURL is the HTTP endpoint to send events to
	callbackURL string
	// client is the HTTP client used for making requests
	client *http.Client
	// headers are additional headers to include in requests
	headers map[string]string
	// timeout is the request timeout duration
	timeout time.Duration
	// mu protects concurrent access to the observer
	mu sync.RWMutex
	// enabled indicates whether the observer is active
	enabled bool
}

// HTTPCallbackConfig holds configuration for HTTPCallbackObserver.
type HTTPCallbackConfig struct {
	// CallbackURL is the HTTP endpoint to send events to (required)
	CallbackURL string
	// Timeout is the request timeout (default: 5 seconds)
	Timeout time.Duration
	// Headers are additional headers to include in requests
	Headers map[string]string
	// Client is an optional HTTP client (if nil, a default client is created)
	Client *http.Client
}

// NewHTTPCallbackObserver creates a new HTTPCallbackObserver with the given configuration.
func NewHTTPCallbackObserver(config HTTPCallbackConfig) (*HTTPCallbackObserver, error) {
	if config.CallbackURL == "" {
		return nil, fmt.Errorf("callback URL is required")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	client := config.Client
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
		}
	}

	headers := make(map[string]string)
	if config.Headers != nil {
		for k, v := range config.Headers {
			headers[k] = v
		}
	}
	// Set default Content-Type if not provided
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}

	return &HTTPCallbackObserver{
		callbackURL: config.CallbackURL,
		client:      client,
		headers:     headers,
		timeout:     timeout,
		enabled:     true,
	}, nil
}

// SetEnabled enables or disables the observer.
func (o *HTTPCallbackObserver) SetEnabled(enabled bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.enabled = enabled
}

// IsEnabled returns whether the observer is enabled.
func (o *HTTPCallbackObserver) IsEnabled() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.enabled
}

// sendEvent sends an HTTP POST request with the given event payload.
func (o *HTTPCallbackObserver) sendEvent(eventType string, payload map[string]interface{}) error {
	o.mu.RLock()
	enabled := o.enabled
	url := o.callbackURL
	client := o.client
	headers := make(map[string]string)
	for k, v := range o.headers {
		headers[k] = v
	}
	o.mu.RUnlock()

	if !enabled {
		return nil
	}

	// Add event type to payload
	payload["event_type"] = eventType
	payload["timestamp"] = time.Now().Format(time.RFC3339)

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Create request with context and timeout
	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback returned non-success status: %d", resp.StatusCode)
	}

	return nil
}

// OnExecutionStarted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnExecutionStarted(workflowID, executionID string) {
	payload := map[string]interface{}{
		"workflow_id":  workflowID,
		"execution_id": executionID,
	}
	_ = o.sendEvent("execution_started", payload)
}

// OnExecutionCompleted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	payload := map[string]interface{}{
		"workflow_id":  workflowID,
		"execution_id": executionID,
		"duration_ms":  duration.Milliseconds(),
		"success":      true,
	}
	_ = o.sendEvent("execution_completed", payload)
}

// OnExecutionFailed implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	payload := map[string]interface{}{
		"workflow_id":  workflowID,
		"execution_id": executionID,
		"duration_ms":  duration.Milliseconds(),
		"success":      false,
		"error":        err.Error(),
	}
	_ = o.sendEvent("execution_failed", payload)
}

// OnNodeStarted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeStarted(executionID string, node *domain.Node, attemptNumber int) {
	payload := map[string]interface{}{
		"execution_id":   executionID,
		"attempt_number": attemptNumber,
	}
	if node != nil {
		payload["node_id"] = node.ID()
		payload["workflow_id"] = node.WorkflowID()
		payload["node_type"] = node.Type()
		payload["name"] = node.Name()
		payload["config"] = node.Config()
	}
	_ = o.sendEvent("node_started", payload)
}

// OnNodeCompleted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeCompleted(executionID string, node *domain.Node, output interface{}, duration time.Duration) {
	payload := map[string]interface{}{
		"execution_id": executionID,
		"duration_ms":  duration.Milliseconds(),
		"success":      true,
	}
	if node != nil {
		payload["node_id"] = node.ID()
		payload["workflow_id"] = node.WorkflowID()
		payload["node_type"] = node.Type()
		payload["name"] = node.Name()
		payload["config"] = node.Config()
	}
	// Include output if it can be serialized
	if output != nil {
		payload["output"] = output
	}
	_ = o.sendEvent("node_completed", payload)
}

// OnNodeFailed implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool) {
	payload := map[string]interface{}{
		"execution_id": executionID,
		"duration_ms":  duration.Milliseconds(),
		"success":      false,
		"will_retry":   willRetry,
		"error":        err.Error(),
	}
	if node != nil {
		payload["node_id"] = node.ID()
		payload["workflow_id"] = node.WorkflowID()
		payload["node_type"] = node.Type()
		payload["name"] = node.Name()
		payload["config"] = node.Config()
	}
	_ = o.sendEvent("node_failed", payload)
}

// OnNodeRetrying implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration) {
	payload := map[string]interface{}{
		"execution_id":   executionID,
		"attempt_number": attemptNumber,
		"delay_ms":       delay.Milliseconds(),
	}
	if node != nil {
		payload["node_id"] = node.ID()
		payload["workflow_id"] = node.WorkflowID()
		payload["node_type"] = node.Type()
		payload["name"] = node.Name()
		payload["config"] = node.Config()
	}
	_ = o.sendEvent("node_retrying", payload)
}

// OnVariableSet implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnVariableSet(executionID, key string, value interface{}) {
	payload := map[string]interface{}{
		"execution_id":   executionID,
		"variable_key":   key,
		"variable_value": value,
	}
	_ = o.sendEvent("variable_set", payload)
}

// OnNodeCallbackStarted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeCallbackStarted(executionID string, node *domain.Node) {
	payload := map[string]interface{}{
		"execution_id": executionID,
	}
	if node != nil {
		payload["node_id"] = node.ID()
		payload["workflow_id"] = node.WorkflowID()
		payload["node_type"] = node.Type()
		payload["name"] = node.Name()
	}
	_ = o.sendEvent("node_callback_started", payload)
}

// OnNodeCallbackCompleted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeCallbackCompleted(executionID string, node *domain.Node, err error, duration time.Duration) {
	payload := map[string]interface{}{
		"execution_id": executionID,
		"duration_ms":  duration.Milliseconds(),
		"success":      err == nil,
	}
	if node != nil {
		payload["node_id"] = node.ID()
		payload["workflow_id"] = node.WorkflowID()
		payload["node_type"] = node.Type()
		payload["name"] = node.Name()
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	_ = o.sendEvent("node_callback_completed", payload)
}
