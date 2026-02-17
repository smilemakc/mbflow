package observer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebSocketHandler(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)
	handler := NewWebSocketHandler(hub, log)

	assert.NotNil(t, handler)
	assert.Equal(t, hub, handler.hub)
	assert.Equal(t, log, handler.logger)
}

func TestWebSocketHandler_ServeHTTP(t *testing.T) {
	t.Run("successful WebSocket upgrade", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		server := httptest.NewServer(handler)
		defer server.Close()

		// Convert http:// to ws://
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		// Connect to WebSocket
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

		// Should receive welcome message
		var welcomeMsg map[string]any
		err = conn.ReadJSON(&welcomeMsg)
		require.NoError(t, err)

		assert.Equal(t, "control", welcomeMsg["type"])
		assert.Equal(t, "Connected to MBFlow WebSocket", welcomeMsg["message"])
		assert.NotEmpty(t, welcomeMsg["client_id"])
		assert.NotEmpty(t, welcomeMsg["timestamp"])

		// Verify client was registered
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 1, hub.ClientCount())
	})

	t.Run("with execution_id query parameter", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?execution_id=exec-123"

		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

		// Read welcome message
		var welcomeMsg map[string]any
		err = conn.ReadJSON(&welcomeMsg)
		require.NoError(t, err)

		assert.Equal(t, "exec-123", welcomeMsg["execution_id"])
	})

	t.Run("without execution_id parameter", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		// Read welcome message
		var welcomeMsg map[string]any
		err = conn.ReadJSON(&welcomeMsg)
		require.NoError(t, err)

		// execution_id should be empty string
		assert.Equal(t, "", welcomeMsg["execution_id"])
	})

	t.Run("multiple clients connection", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		// Connect 3 clients
		conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn1.Close() }()

		conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn2.Close() }()

		conn3, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn3.Close() }()

		// Read welcome messages
		var welcomeMsg map[string]any
		_ = conn1.ReadJSON(&welcomeMsg)
		_ = conn2.ReadJSON(&welcomeMsg)
		_ = conn3.ReadJSON(&welcomeMsg)

		time.Sleep(50 * time.Millisecond)

		// All 3 clients should be registered
		assert.Equal(t, 3, hub.ClientCount())
	})

	t.Run("client disconnection", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)

		// Read welcome message
		var welcomeMsg map[string]any
		_ = conn.ReadJSON(&welcomeMsg)

		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 1, hub.ClientCount())

		// Close connection
		_ = conn.Close()

		time.Sleep(50 * time.Millisecond)

		// Client should be unregistered
		assert.Equal(t, 0, hub.ClientCount())
	})

	t.Run("receives broadcasted messages", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		// Read welcome message
		var welcomeMsg map[string]any
		_ = conn.ReadJSON(&welcomeMsg)

		time.Sleep(10 * time.Millisecond)

		// Broadcast a message
		testMessage := []byte(`{"type": "test", "data": "hello"}`)
		hub.Broadcast(testMessage)

		// Client should receive the message
		_, message, err := conn.ReadMessage()
		require.NoError(t, err)

		var receivedMsg map[string]any
		json.Unmarshal(message, &receivedMsg)

		assert.Equal(t, "test", receivedMsg["type"])
		assert.Equal(t, "hello", receivedMsg["data"])
	})

	t.Run("execution-specific filtering", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL1 := "ws" + strings.TrimPrefix(server.URL, "http") + "?execution_id=exec-123"
		wsURL2 := "ws" + strings.TrimPrefix(server.URL, "http") + "?execution_id=exec-456"

		conn1, _, err := websocket.DefaultDialer.Dial(wsURL1, nil)
		require.NoError(t, err)
		defer func() { _ = conn1.Close() }()

		conn2, _, err := websocket.DefaultDialer.Dial(wsURL2, nil)
		require.NoError(t, err)
		defer func() { _ = conn2.Close() }()

		// Read welcome messages
		var welcomeMsg map[string]any
		_ = conn1.ReadJSON(&welcomeMsg)
		_ = conn2.ReadJSON(&welcomeMsg)

		time.Sleep(10 * time.Millisecond)

		// Broadcast to exec-123
		message := []byte(`{"execution_id": "exec-123"}`)
		hub.BroadcastToExecution("exec-123", message)

		// conn1 should receive
		_ = conn1.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, receivedMsg, err := conn1.ReadMessage()
		require.NoError(t, err)
		assert.Contains(t, string(receivedMsg), "exec-123")

		// conn2 should NOT receive (should timeout)
		_ = conn2.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, _, err = conn2.ReadMessage()
		assert.Error(t, err) // Should timeout
	})
}

func TestWebSocketHandler_HandleHealthCheck(t *testing.T) {
	t.Run("returns healthy status", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler.HandleHealthCheck(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var status map[string]any
		err := json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		assert.Equal(t, "healthy", status["status"])
		assert.Equal(t, float64(0), status["connected_clients"])
		assert.NotEmpty(t, status["timestamp"])
	})

	t.Run("returns correct client count", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		// Register mock clients
		client1 := &WebSocketClient{
			ID:            "client-1",
			send:          make(chan []byte, 256),
			hub:           hub,
			executionID:   "",
			subscriptions: make(map[EventType]bool),
		}
		client2 := &WebSocketClient{
			ID:            "client-2",
			send:          make(chan []byte, 256),
			hub:           hub,
			executionID:   "",
			subscriptions: make(map[EventType]bool),
		}

		hub.Register(client1)
		hub.Register(client2)
		time.Sleep(10 * time.Millisecond)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler.HandleHealthCheck(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		var status map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&status)

		assert.Equal(t, float64(2), status["connected_clients"])
	})
}

func TestWebSocketUpgrader(t *testing.T) {
	t.Run("allows all origins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set("Origin", "http://example.com")

		allowed := upgrader.CheckOrigin(req)
		assert.True(t, allowed, "Should allow all origins in development")
	})

	t.Run("buffer sizes configured", func(t *testing.T) {
		assert.Equal(t, 1024, upgrader.ReadBufferSize)
		assert.Equal(t, 1024, upgrader.WriteBufferSize)
	})
}

func TestWebSocketHandler_Integration(t *testing.T) {
	t.Run("end-to-end event broadcasting", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)
		obs := NewWebSocketObserver(hub)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		// Connect client
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		// Read welcome message
		var welcomeMsg map[string]any
		_ = conn.ReadJSON(&welcomeMsg)

		time.Sleep(10 * time.Millisecond)

		// Send event through observer
		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "running",
		}

		err = obs.OnEvent(nil, event)
		require.NoError(t, err)

		// Client should receive the event
		var eventMsg WebSocketMessage
		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		err = conn.ReadJSON(&eventMsg)
		require.NoError(t, err)

		assert.Equal(t, "event", eventMsg.Type)
		assert.Equal(t, "execution.started", eventMsg.Event.EventType)
		assert.Equal(t, "exec-123", eventMsg.Event.ExecutionID)
		assert.Equal(t, "wf-456", eventMsg.Event.WorkflowID)
	})

	t.Run("multiple events in sequence", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)
		obs := NewWebSocketObserver(hub)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		// Read welcome message
		var welcomeMsg map[string]any
		_ = conn.ReadJSON(&welcomeMsg)

		time.Sleep(10 * time.Millisecond)

		// Send multiple events
		events := []Event{
			{
				Type:        EventTypeExecutionStarted,
				ExecutionID: "exec-123",
				WorkflowID:  "wf-456",
				Timestamp:   time.Now(),
				Status:      "running",
			},
			{
				Type:        EventTypeWaveStarted,
				ExecutionID: "exec-123",
				WorkflowID:  "wf-456",
				Timestamp:   time.Now(),
				Status:      "running",
			},
			{
				Type:        EventTypeExecutionCompleted,
				ExecutionID: "exec-123",
				WorkflowID:  "wf-456",
				Timestamp:   time.Now(),
				Status:      "completed",
			},
		}

		for _, event := range events {
			_ = obs.OnEvent(nil, event)
			time.Sleep(10 * time.Millisecond) // Give time for event to be broadcast
		}

		// Read all events
		receivedTypes := make([]string, 0)
		for i := 0; i < len(events); i++ {
			var eventMsg WebSocketMessage
			err = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			require.NoError(t, err)
			err = conn.ReadJSON(&eventMsg)
			require.NoError(t, err)
			receivedTypes = append(receivedTypes, eventMsg.Event.EventType)
		}

		assert.Contains(t, receivedTypes, "execution.started")
		assert.Contains(t, receivedTypes, "wave.started")
		assert.Contains(t, receivedTypes, "execution.completed")
	})
}

func TestWebSocketClient_MessageHandling(t *testing.T) {
	t.Run("client can send subscribe command", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		// Read welcome message
		var welcomeMsg map[string]any
		_ = conn.ReadJSON(&welcomeMsg)

		time.Sleep(10 * time.Millisecond)

		// Send subscribe command
		subscribeMsg := map[string]any{
			"command": "subscribe",
			"event_types": []string{
				"execution.started",
				"execution.completed",
			},
		}

		err = conn.WriteJSON(subscribeMsg)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		// No error means subscription was processed
		// (Can't verify internal state without exposing client, but handler accepted it)
	})

	t.Run("client can send unsubscribe command", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		handler := NewWebSocketHandler(hub, log)

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		// Read welcome message
		var welcomeMsg map[string]any
		_ = conn.ReadJSON(&welcomeMsg)

		time.Sleep(10 * time.Millisecond)

		// Send unsubscribe command
		unsubscribeMsg := map[string]any{
			"command": "unsubscribe",
			"event_types": []string{
				"node.started",
			},
		}

		err = conn.WriteJSON(unsubscribeMsg)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		// No error means unsubscription was processed
	})
}
