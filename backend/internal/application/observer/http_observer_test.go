package observer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPCallbackObserver(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		obs := NewHTTPCallbackObserver("http://example.com/webhook")

		assert.NotNil(t, obs)
		assert.Equal(t, "http_callback", obs.Name())
		assert.Equal(t, "http://example.com/webhook", obs.url)
		assert.Equal(t, "POST", obs.method)
		assert.Nil(t, obs.Filter())
		assert.Equal(t, 3, obs.maxRetries)
		assert.Equal(t, 1*time.Second, obs.retryDelay)
		assert.Equal(t, 2.0, obs.retryBackoff)
		assert.NotNil(t, obs.client)
		assert.Equal(t, 10*time.Second, obs.client.Timeout)
	})

	t.Run("with custom HTTP method", func(t *testing.T) {
		obs := NewHTTPCallbackObserver(
			"http://example.com/webhook",
			WithHTTPMethod("PUT"),
		)

		assert.Equal(t, "PUT", obs.method)
	})

	t.Run("with custom headers", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom":      "value",
		}

		obs := NewHTTPCallbackObserver(
			"http://example.com/webhook",
			WithHTTPHeaders(headers),
		)

		assert.Equal(t, "Bearer token123", obs.headers["Authorization"])
		assert.Equal(t, "value", obs.headers["X-Custom"])
	})

	t.Run("with event filter", func(t *testing.T) {
		filter := NewEventTypeFilter(EventTypeExecutionStarted)
		obs := NewHTTPCallbackObserver(
			"http://example.com/webhook",
			WithHTTPFilter(filter),
		)

		assert.NotNil(t, obs.Filter())
	})

	t.Run("with custom timeout", func(t *testing.T) {
		obs := NewHTTPCallbackObserver(
			"http://example.com/webhook",
			WithHTTPTimeout(5*time.Second),
		)

		assert.Equal(t, 5*time.Second, obs.client.Timeout)
	})

	t.Run("with custom retry configuration", func(t *testing.T) {
		obs := NewHTTPCallbackObserver(
			"http://example.com/webhook",
			WithHTTPRetry(5, 2*time.Second, 1.5),
		)

		assert.Equal(t, 5, obs.maxRetries)
		assert.Equal(t, 2*time.Second, obs.retryDelay)
		assert.Equal(t, 1.5, obs.retryBackoff)
	})

	t.Run("with all options", func(t *testing.T) {
		filter := NewEventTypeFilter(EventTypeNodeCompleted)
		headers := map[string]string{"X-API-Key": "secret"}

		obs := NewHTTPCallbackObserver(
			"http://example.com/webhook",
			WithHTTPMethod("POST"),
			WithHTTPHeaders(headers),
			WithHTTPFilter(filter),
			WithHTTPTimeout(15*time.Second),
			WithHTTPRetry(3, 1*time.Second, 2.0),
		)

		assert.Equal(t, "POST", obs.method)
		assert.NotNil(t, obs.headers)
		assert.NotNil(t, obs.Filter())
		assert.Equal(t, 15*time.Second, obs.client.Timeout)
		assert.Equal(t, 3, obs.maxRetries)
	})
}

func TestHTTPCallbackObserver_Name(t *testing.T) {
	obs := NewHTTPCallbackObserver("http://example.com/webhook")
	assert.Equal(t, "http_callback", obs.Name())
}

func TestHTTPCallbackObserver_Filter(t *testing.T) {
	t.Run("no filter by default", func(t *testing.T) {
		obs := NewHTTPCallbackObserver("http://example.com/webhook")
		assert.Nil(t, obs.Filter())
	})

	t.Run("with filter", func(t *testing.T) {
		filter := NewEventTypeFilter(EventTypeExecutionStarted)
		obs := NewHTTPCallbackObserver(
			"http://example.com/webhook",
			WithHTTPFilter(filter),
		)
		assert.NotNil(t, obs.Filter())
	})
}

func TestHTTPCallbackObserver_OnEvent(t *testing.T) {
	t.Run("successful POST request", func(t *testing.T) {
		var receivedPayload map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedPayload)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		obs := NewHTTPCallbackObserver(server.URL)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "running",
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)

		assert.Equal(t, "execution.started", receivedPayload["event_type"])
		assert.Equal(t, "exec-123", receivedPayload["execution_id"])
		assert.Equal(t, "wf-456", receivedPayload["workflow_id"])
		assert.Equal(t, "running", receivedPayload["status"])
	})

	t.Run("custom HTTP method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		obs := NewHTTPCallbackObserver(server.URL, WithHTTPMethod("PUT"))

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("custom headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		headers := map[string]string{
			"Authorization":   "Bearer token123",
			"X-Custom-Header": "custom-value",
		}

		obs := NewHTTPCallbackObserver(server.URL, WithHTTPHeaders(headers))

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("event with all fields", func(t *testing.T) {
		var receivedPayload map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedPayload)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		obs := NewHTTPCallbackObserver(server.URL)

		nodeID := "node-123"
		nodeName := "HTTP Request"
		nodeType := "http"
		waveIndex := 2
		nodeCount := 5
		durationMs := int64(1500)

		event := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			NodeID:      &nodeID,
			NodeName:    &nodeName,
			NodeType:    &nodeType,
			WaveIndex:   &waveIndex,
			NodeCount:   &nodeCount,
			Status:      "completed",
			DurationMs:  &durationMs,
			Input: map[string]any{
				"url": "https://api.example.com",
			},
			Output: map[string]any{
				"status": 200,
			},
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)

		assert.Equal(t, "node.completed", receivedPayload["event_type"])
		assert.Equal(t, "node-123", receivedPayload["node_id"])
		assert.Equal(t, "HTTP Request", receivedPayload["node_name"])
		assert.Equal(t, "http", receivedPayload["node_type"])
		assert.Equal(t, float64(2), receivedPayload["wave_index"])
		assert.Equal(t, float64(5), receivedPayload["node_count"])
		assert.Equal(t, float64(1500), receivedPayload["duration_ms"])
	})

	t.Run("event with error", func(t *testing.T) {
		var receivedPayload map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedPayload)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		obs := NewHTTPCallbackObserver(server.URL)

		testErr := errors.New("execution failed")
		event := Event{
			Type:        EventTypeExecutionFailed,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "failed",
			Error:       testErr,
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)

		assert.Equal(t, "execution failed", receivedPayload["error"])
	})
}

func TestHTTPCallbackObserver_Retry(t *testing.T) {
	t.Run("retry on server error", func(t *testing.T) {
		attemptCount := int32(0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 3 {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		obs := NewHTTPCallbackObserver(
			server.URL,
			WithHTTPRetry(3, 10*time.Millisecond, 1.5),
		)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
		assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount))
	})

	t.Run("fails after max retries", func(t *testing.T) {
		attemptCount := int32(0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attemptCount, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		obs := NewHTTPCallbackObserver(
			server.URL,
			WithHTTPRetry(2, 10*time.Millisecond, 1.5),
		)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		err := obs.OnEvent(context.Background(), event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "http callback failed after 3 attempts")
		assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount)) // Initial + 2 retries
	})

	t.Run("exponential backoff timing", func(t *testing.T) {
		attemptTimes := make([]time.Time, 0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptTimes = append(attemptTimes, time.Now())
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		obs := NewHTTPCallbackObserver(
			server.URL,
			WithHTTPRetry(3, 100*time.Millisecond, 2.0),
		)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		obs.OnEvent(context.Background(), event)

		require.Len(t, attemptTimes, 4) // Initial + 3 retries

		// Check delay between attempts (allow some tolerance)
		delay1 := attemptTimes[1].Sub(attemptTimes[0])
		delay2 := attemptTimes[2].Sub(attemptTimes[1])
		delay3 := attemptTimes[3].Sub(attemptTimes[2])

		// First retry: ~100ms
		assert.Greater(t, delay1, 90*time.Millisecond)
		assert.Less(t, delay1, 150*time.Millisecond)

		// Second retry: ~200ms (100ms * 2.0)
		assert.Greater(t, delay2, 180*time.Millisecond)
		assert.Less(t, delay2, 250*time.Millisecond)

		// Third retry: ~400ms (200ms * 2.0)
		assert.Greater(t, delay3, 350*time.Millisecond)
		assert.Less(t, delay3, 500*time.Millisecond)
	})

	t.Run("retry on network error", func(t *testing.T) {
		// Use invalid URL to simulate network error
		obs := NewHTTPCallbackObserver(
			"http://invalid-domain-that-does-not-exist-12345.com",
			WithHTTPRetry(2, 10*time.Millisecond, 1.5),
			WithHTTPTimeout(100*time.Millisecond),
		)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		err := obs.OnEvent(context.Background(), event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "http callback failed after 3 attempts")
	})

	t.Run("no retry on success", func(t *testing.T) {
		attemptCount := int32(0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attemptCount, 1)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		obs := NewHTTPCallbackObserver(
			server.URL,
			WithHTTPRetry(3, 10*time.Millisecond, 2.0),
		)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), atomic.LoadInt32(&attemptCount))
	})
}

func TestHTTPCallbackObserver_StatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		expectErr  bool
	}{
		{"200 OK", http.StatusOK, false},
		{"201 Created", http.StatusCreated, false},
		{"202 Accepted", http.StatusAccepted, false},
		{"204 No Content", http.StatusNoContent, false},
		{"400 Bad Request", http.StatusBadRequest, true},
		{"401 Unauthorized", http.StatusUnauthorized, true},
		{"404 Not Found", http.StatusNotFound, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			obs := NewHTTPCallbackObserver(
				server.URL,
				WithHTTPRetry(0, 0, 1.0), // No retries for this test
			)

			event := Event{
				Type:        EventTypeExecutionStarted,
				ExecutionID: "exec-123",
				WorkflowID:  "wf-456",
				Timestamp:   time.Now(),
			}

			err := obs.OnEvent(context.Background(), event)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPCallbackObserver_buildPayload(t *testing.T) {
	obs := NewHTTPCallbackObserver("http://example.com")

	t.Run("minimal event", func(t *testing.T) {
		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "running",
		}

		payload := obs.buildPayload(event)

		assert.Equal(t, "execution.started", payload["event_type"])
		assert.Equal(t, "exec-123", payload["execution_id"])
		assert.Equal(t, "wf-456", payload["workflow_id"])
		assert.Equal(t, "running", payload["status"])
		assert.NotEmpty(t, payload["timestamp"])
	})

	t.Run("event with all optional fields", func(t *testing.T) {
		nodeID := "node-123"
		nodeName := "Transform"
		nodeType := "transform"
		waveIndex := 1
		nodeCount := 3
		durationMs := int64(500)

		event := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			NodeID:      &nodeID,
			NodeName:    &nodeName,
			NodeType:    &nodeType,
			WaveIndex:   &waveIndex,
			NodeCount:   &nodeCount,
			Status:      "completed",
			DurationMs:  &durationMs,
			Error:       errors.New("test error"),
			Input: map[string]any{
				"data": "input",
			},
			Output: map[string]any{
				"result": "output",
			},
		}

		payload := obs.buildPayload(event)

		assert.Equal(t, "node-123", payload["node_id"])
		assert.Equal(t, "Transform", payload["node_name"])
		assert.Equal(t, "transform", payload["node_type"])
		assert.Equal(t, 1, payload["wave_index"])
		assert.Equal(t, 3, payload["node_count"])
		assert.Equal(t, int64(500), payload["duration_ms"])
		assert.Equal(t, "test error", payload["error"])
		assert.NotNil(t, payload["input"])
		assert.NotNil(t, payload["output"])
	})
}

func TestHTTPCallbackObserver_WithHTTPName(t *testing.T) {
	// Arrange
	customName := "my-custom-webhook-observer"

	// Act
	obs := NewHTTPCallbackObserver(
		"http://example.com/webhook",
		WithHTTPName(customName),
	)

	// Assert
	assert.Equal(t, customName, obs.Name(), "WithHTTPName should override the default observer name")
}

func TestHTTPCallbackObserver_ContextCancellation(t *testing.T) {
	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	obs := NewHTTPCallbackObserver(server.URL)

	event := Event{
		Type:        EventTypeExecutionStarted,
		ExecutionID: "exec-123",
		WorkflowID:  "wf-456",
		Timestamp:   time.Now(),
	}

	// Create context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := obs.OnEvent(ctx, event)
	assert.Error(t, err)
}
