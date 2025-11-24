package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/internal/domain"
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

// HTTPCallbackObserverConfig holds configuration for HTTPCallbackObserver.
type HTTPCallbackObserverConfig struct {
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
func NewHTTPCallbackObserver(config HTTPCallbackObserverConfig) (*HTTPCallbackObserver, error) {
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
func (o *HTTPCallbackObserver) sendEvent(payload any) error {
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
	_ = o.sendEvent(NewExecutionStartedEvent(workflowID, executionID))
}

// OnExecutionCompleted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	_ = o.sendEvent(NewExecutionCompletedEvent(workflowID, executionID, duration))
}

// OnExecutionFailed implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	_ = o.sendEvent(NewExecutionFailedEvent(workflowID, executionID, err, duration))
}

// OnNodeStarted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeStarted(workflowID, executionID string, node domain.Node, attemptNumber int) {
	_ = o.sendEvent(NewNodeStartedEvent(workflowID, executionID, node, attemptNumber))
}

// OnNodeCompleted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeCompleted(workflowID, executionID string, node domain.Node, output interface{}, duration time.Duration) {
	_ = o.sendEvent(NewNodeCompletedEvent(workflowID, executionID, node, output, duration))
}

// OnNodeFailed implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeFailed(workflowID, executionID string, node domain.Node, err error, duration time.Duration, willRetry bool) {
	_ = o.sendEvent(NewNodeFailedEvent(workflowID, executionID, node, err, duration, willRetry))
}

// OnNodeRetrying implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeRetrying(workflowID, executionID string, node domain.Node, attemptNumber int, delay time.Duration) {
	_ = o.sendEvent(NewNodeRetryingEvent(workflowID, executionID, node, attemptNumber, delay))
}

// OnVariableSet implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnVariableSet(workflowID, executionID, key string, value interface{}) {
	_ = o.sendEvent(NewVariableSetEvent(workflowID, executionID, key, value))
}

// OnNodeCallbackStarted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeCallbackStarted(workflowID, executionID string, node domain.Node) {
	_ = o.sendEvent(NewNodeCallbackStartedEvent(workflowID, executionID, node))
}

// OnNodeCallbackCompleted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeCallbackCompleted(workflowID, executionID string, node domain.Node, err error, duration time.Duration) {
	_ = o.sendEvent(NewNodeCallbackCompletedEvent(workflowID, executionID, node, err, duration))
}
