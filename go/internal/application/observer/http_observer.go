package observer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPCallbackObserver sends HTTP callbacks for workflow events
type HTTPCallbackObserver struct {
	name         string
	url          string
	method       string
	headers      map[string]string
	filter       EventFilter
	client       *http.Client
	maxRetries   int
	retryDelay   time.Duration
	retryBackoff float64
}

// HTTPObserverOption configures HTTPCallbackObserver
type HTTPObserverOption func(*HTTPCallbackObserver)

// WithHTTPMethod sets the HTTP method
func WithHTTPMethod(method string) HTTPObserverOption {
	return func(o *HTTPCallbackObserver) {
		o.method = method
	}
}

// WithHTTPHeaders sets custom HTTP headers
func WithHTTPHeaders(headers map[string]string) HTTPObserverOption {
	return func(o *HTTPCallbackObserver) {
		o.headers = headers
	}
}

// WithHTTPName sets a custom observer name (required for per-execution observers to ensure uniqueness)
func WithHTTPName(name string) HTTPObserverOption {
	return func(o *HTTPCallbackObserver) {
		o.name = name
	}
}

// WithHTTPFilter sets event filter
func WithHTTPFilter(filter EventFilter) HTTPObserverOption {
	return func(o *HTTPCallbackObserver) {
		o.filter = filter
	}
}

// WithHTTPTimeout sets request timeout
func WithHTTPTimeout(timeout time.Duration) HTTPObserverOption {
	return func(o *HTTPCallbackObserver) {
		o.client.Timeout = timeout
	}
}

// WithHTTPRetry configures retry behavior
func WithHTTPRetry(maxRetries int, delay time.Duration, backoff float64) HTTPObserverOption {
	return func(o *HTTPCallbackObserver) {
		o.maxRetries = maxRetries
		o.retryDelay = delay
		o.retryBackoff = backoff
	}
}

// NewHTTPCallbackObserver creates a new HTTP callback observer
func NewHTTPCallbackObserver(url string, opts ...HTTPObserverOption) *HTTPCallbackObserver {
	obs := &HTTPCallbackObserver{
		name:         "http_callback",
		url:          url,
		method:       "POST",
		headers:      make(map[string]string),
		filter:       nil, // nil = all events
		client:       &http.Client{Timeout: 10 * time.Second},
		maxRetries:   3,
		retryDelay:   1 * time.Second,
		retryBackoff: 2.0,
	}

	for _, opt := range opts {
		opt(obs)
	}

	return obs
}

// Name returns the observer's name
func (o *HTTPCallbackObserver) Name() string {
	return o.name
}

// Filter returns the event filter
func (o *HTTPCallbackObserver) Filter() EventFilter {
	return o.filter
}

// OnEvent handles event by sending HTTP callback
func (o *HTTPCallbackObserver) OnEvent(ctx context.Context, event Event) error {
	payload := o.buildPayload(event)
	return o.sendWithRetry(ctx, payload)
}

// buildPayload constructs the HTTP request payload
func (o *HTTPCallbackObserver) buildPayload(event Event) map[string]any {
	payload := map[string]any{
		"event_type":   string(event.Type),
		"execution_id": event.ExecutionID,
		"workflow_id":  event.WorkflowID,
		"timestamp":    event.Timestamp.Format(time.RFC3339),
		"status":       event.Status,
	}

	// Add optional fields
	if event.NodeID != nil {
		payload["node_id"] = *event.NodeID
		payload["node_name"] = *event.NodeName
		payload["node_type"] = *event.NodeType
	}

	if event.WaveIndex != nil {
		payload["wave_index"] = *event.WaveIndex
	}

	if event.NodeCount != nil {
		payload["node_count"] = *event.NodeCount
	}

	if event.DurationMs != nil {
		payload["duration_ms"] = *event.DurationMs
	}

	if event.Error != nil {
		payload["error"] = event.Error.Error()
	}

	if event.Input != nil {
		payload["input"] = event.Input
	}

	if event.Output != nil {
		payload["output"] = event.Output
	}

	return payload
}

// sendWithRetry sends HTTP request with exponential backoff retry
func (o *HTTPCallbackObserver) sendWithRetry(ctx context.Context, payload map[string]any) error {
	var lastErr error
	delay := o.retryDelay

	for attempt := 0; attempt <= o.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * o.retryBackoff)
		}

		if err := o.send(ctx, payload); err != nil {
			lastErr = err
			continue
		}

		return nil // Success
	}

	return fmt.Errorf("http callback failed after %d attempts: %w", o.maxRetries+1, lastErr)
}

// send sends a single HTTP request
func (o *HTTPCallbackObserver) send(ctx context.Context, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, o.method, o.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http callback returned status %d", resp.StatusCode)
	}

	return nil
}
