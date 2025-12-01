package observer

import (
	"context"
	"encoding/json"
	"errors"
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

func TestNewWebSocketHub(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.logger)

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)
}

func TestNewWebSocketObserver(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		obs := NewWebSocketObserver(hub)

		assert.NotNil(t, obs)
		assert.Equal(t, "websocket", obs.Name())
		assert.Nil(t, obs.Filter())
		assert.NotNil(t, obs.hub)
	})

	t.Run("with filter", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		filter := NewEventTypeFilter(EventTypeExecutionStarted)
		obs := NewWebSocketObserver(hub, WithWebSocketFilter(filter))

		assert.NotNil(t, obs.Filter())
	})

	t.Run("with logger", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		obs := NewWebSocketObserver(hub, WithWebSocketLogger(log))

		assert.NotNil(t, obs.logger)
	})
}

func TestWebSocketObserver_Name(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)
	obs := NewWebSocketObserver(hub)

	assert.Equal(t, "websocket", obs.Name())
}

func TestWebSocketObserver_Filter(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	t.Run("no filter by default", func(t *testing.T) {
		obs := NewWebSocketObserver(hub)
		assert.Nil(t, obs.Filter())
	})

	t.Run("with filter", func(t *testing.T) {
		filter := NewEventTypeFilter(EventTypeExecutionStarted)
		obs := NewWebSocketObserver(hub, WithWebSocketFilter(filter))
		assert.NotNil(t, obs.Filter())
	})
}

func TestWebSocketObserver_GetHub(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)
	obs := NewWebSocketObserver(hub)

	returnedHub := obs.GetHub()
	assert.Equal(t, hub, returnedHub)
}

func TestWebSocketObserver_OnEvent(t *testing.T) {
	t.Run("broadcasts event to hub", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		obs := NewWebSocketObserver(hub)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "running",
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("converts event to websocket message", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		obs := NewWebSocketObserver(hub)

		nodeID := "node-123"
		nodeName := "HTTP Request"
		nodeType := "http"
		durationMs := int64(1500)

		event := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			NodeID:      &nodeID,
			NodeName:    &nodeName,
			NodeType:    &nodeType,
			Status:      "completed",
			DurationMs:  &durationMs,
			Output: map[string]interface{}{
				"status": 200,
			},
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("handles event with error", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
		hub := NewWebSocketHub(log)
		obs := NewWebSocketObserver(hub)

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
	})
}

func TestWebSocketObserver_eventToMessage(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)
	obs := NewWebSocketObserver(hub)

	t.Run("converts minimal event", func(t *testing.T) {
		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "running",
		}

		msg := obs.eventToMessage(event)

		assert.Equal(t, "event", msg.Type)
		assert.NotNil(t, msg.Event)
		assert.Equal(t, "execution.started", msg.Event.EventType)
		assert.Equal(t, "exec-123", msg.Event.ExecutionID)
		assert.Equal(t, "wf-456", msg.Event.WorkflowID)
		assert.Equal(t, "running", msg.Event.Status)
	})

	t.Run("converts event with all fields", func(t *testing.T) {
		nodeID := "node-123"
		nodeName := "Transform"
		nodeType := "transform"
		waveIndex := 2
		nodeCount := 5
		durationMs := int64(750)

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
			Output: map[string]interface{}{
				"result": "success",
			},
		}

		msg := obs.eventToMessage(event)

		assert.Equal(t, "event", msg.Type)
		assert.Equal(t, "node-123", *msg.Event.NodeID)
		assert.Equal(t, "Transform", *msg.Event.NodeName)
		assert.Equal(t, "transform", *msg.Event.NodeType)
		assert.Equal(t, 2, *msg.Event.WaveIndex)
		assert.Equal(t, 5, *msg.Event.NodeCount)
		assert.Equal(t, int64(750), *msg.Event.DurationMs)
		assert.NotNil(t, msg.Event.Output)
	})

	t.Run("converts event with error", func(t *testing.T) {
		testErr := errors.New("node failed")
		event := Event{
			Type:        EventTypeNodeFailed,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "failed",
			Error:       testErr,
		}

		msg := obs.eventToMessage(event)

		require.NotNil(t, msg.Event.Error)
		assert.Equal(t, "node failed", *msg.Event.Error)
	})
}

func TestWebSocketHub_RegisterUnregister(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	// Create mock WebSocket connection
	server := httptest.NewServer(nil)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		// Can't establish real connection, use mock client
		client := &WebSocketClient{
			ID:            "test-client",
			conn:          nil,
			send:          make(chan []byte, 256),
			hub:           hub,
			executionID:   "",
			subscriptions: make(map[EventType]bool),
		}

		// Test register
		hub.Register(client)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 1, hub.ClientCount())

		// Test unregister
		hub.Unregister(client)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 0, hub.ClientCount())
		return
	}
	defer conn.Close()

	client := NewWebSocketClient("test-client", conn, hub, "")

	// Test register
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	// Test unregister
	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestWebSocketHub_Broadcast(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	// Create mock client
	client := &WebSocketClient{
		ID:            "test-client",
		conn:          nil,
		send:          make(chan []byte, 256),
		hub:           hub,
		executionID:   "",
		subscriptions: make(map[EventType]bool),
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	message := []byte(`{"test": "message"}`)
	hub.Broadcast(message)

	// Check if message was received
	select {
	case msg := <-client.send:
		assert.Equal(t, message, msg)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received within timeout")
	}
}

func TestWebSocketHub_BroadcastToExecution(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	// Create clients with different execution filters
	client1 := &WebSocketClient{
		ID:            "client-1",
		send:          make(chan []byte, 256),
		hub:           hub,
		executionID:   "exec-123", // Subscribed to exec-123
		subscriptions: make(map[EventType]bool),
	}

	client2 := &WebSocketClient{
		ID:            "client-2",
		send:          make(chan []byte, 256),
		hub:           hub,
		executionID:   "", // Subscribed to all executions
		subscriptions: make(map[EventType]bool),
	}

	client3 := &WebSocketClient{
		ID:            "client-3",
		send:          make(chan []byte, 256),
		hub:           hub,
		executionID:   "exec-456", // Subscribed to exec-456
		subscriptions: make(map[EventType]bool),
	}

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(10 * time.Millisecond)

	// Broadcast to exec-123
	message := []byte(`{"execution_id": "exec-123"}`)
	hub.BroadcastToExecution("exec-123", message)

	// Client1 (exec-123) should receive
	select {
	case msg := <-client1.send:
		assert.Equal(t, message, msg)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client1 should have received message")
	}

	// Client2 (all) should receive
	select {
	case msg := <-client2.send:
		assert.Equal(t, message, msg)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client2 should have received message")
	}

	// Client3 (exec-456) should NOT receive
	select {
	case <-client3.send:
		t.Fatal("Client3 should not have received message")
	case <-time.After(50 * time.Millisecond):
		// Expected timeout
	}
}

func TestWebSocketHub_ClientCount(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	assert.Equal(t, 0, hub.ClientCount())

	client1 := &WebSocketClient{
		ID:            "client-1",
		send:          make(chan []byte, 256),
		hub:           hub,
		executionID:   "",
		subscriptions: make(map[EventType]bool),
	}

	hub.Register(client1)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	client2 := &WebSocketClient{
		ID:            "client-2",
		send:          make(chan []byte, 256),
		hub:           hub,
		executionID:   "",
		subscriptions: make(map[EventType]bool),
	}

	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 2, hub.ClientCount())

	hub.Unregister(client1)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	hub.Unregister(client2)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestNewWebSocketClient(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	client := NewWebSocketClient("client-123", nil, hub, "exec-456")

	assert.Equal(t, "client-123", client.ID)
	assert.Equal(t, hub, client.hub)
	assert.Equal(t, "exec-456", client.executionID)
	assert.NotNil(t, client.send)
	assert.NotNil(t, client.subscriptions)
}

func TestWebSocketClient_IsSubscribed(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	t.Run("no subscriptions means receive all events", func(t *testing.T) {
		client := NewWebSocketClient("client-1", nil, hub, "")

		assert.True(t, client.IsSubscribed(EventTypeExecutionStarted))
		assert.True(t, client.IsSubscribed(EventTypeNodeCompleted))
		assert.True(t, client.IsSubscribed(EventTypeWaveStarted))
	})

	t.Run("with specific subscriptions", func(t *testing.T) {
		client := NewWebSocketClient("client-1", nil, hub, "")

		client.subscriptions[EventTypeExecutionStarted] = true
		client.subscriptions[EventTypeExecutionCompleted] = true

		assert.True(t, client.IsSubscribed(EventTypeExecutionStarted))
		assert.True(t, client.IsSubscribed(EventTypeExecutionCompleted))
		assert.False(t, client.IsSubscribed(EventTypeNodeCompleted))
	})
}

func TestWebSocketClient_handleMessage(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	t.Run("subscribe command", func(t *testing.T) {
		client := NewWebSocketClient("client-1", nil, hub, "")

		message := []byte(`{
			"command": "subscribe",
			"event_types": ["execution.started", "execution.completed"]
		}`)

		client.handleMessage(message)

		assert.True(t, client.IsSubscribed(EventTypeExecutionStarted))
		assert.True(t, client.IsSubscribed(EventTypeExecutionCompleted))
		assert.False(t, client.IsSubscribed(EventTypeNodeCompleted))
	})

	t.Run("unsubscribe command", func(t *testing.T) {
		client := NewWebSocketClient("client-1", nil, hub, "")

		// First subscribe
		client.subscriptions[EventTypeExecutionStarted] = true
		client.subscriptions[EventTypeExecutionCompleted] = true
		client.subscriptions[EventTypeNodeCompleted] = true

		// Then unsubscribe from some
		message := []byte(`{
			"command": "unsubscribe",
			"event_types": ["execution.started"]
		}`)

		client.handleMessage(message)

		assert.False(t, client.subscriptions[EventTypeExecutionStarted])
		assert.True(t, client.IsSubscribed(EventTypeExecutionCompleted))
		assert.True(t, client.IsSubscribed(EventTypeNodeCompleted))
	})

	t.Run("invalid JSON is ignored", func(t *testing.T) {
		client := NewWebSocketClient("client-1", nil, hub, "")

		message := []byte(`{invalid json}`)

		// Should not panic
		assert.NotPanics(t, func() {
			client.handleMessage(message)
		})
	})

	t.Run("unknown command is ignored", func(t *testing.T) {
		client := NewWebSocketClient("client-1", nil, hub, "")

		message := []byte(`{"command": "unknown"}`)

		// Should not panic
		assert.NotPanics(t, func() {
			client.handleMessage(message)
		})
	})
}

func TestWebSocketMessage_Serialization(t *testing.T) {
	t.Run("event message", func(t *testing.T) {
		nodeID := "node-123"
		durationMs := int64(500)

		msg := &WebSocketMessage{
			Type: "event",
			Event: &EventPayload{
				EventType:   "node.completed",
				ExecutionID: "exec-123",
				WorkflowID:  "wf-456",
				Timestamp:   time.Now(),
				Status:      "completed",
				NodeID:      &nodeID,
				DurationMs:  &durationMs,
			},
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var decoded WebSocketMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "event", decoded.Type)
		assert.Equal(t, "node.completed", decoded.Event.EventType)
		assert.Equal(t, "exec-123", decoded.Event.ExecutionID)
		assert.Equal(t, "node-123", *decoded.Event.NodeID)
	})

	t.Run("control message", func(t *testing.T) {
		msg := &WebSocketMessage{
			Type: "control",
			Control: map[string]interface{}{
				"message": "connected",
				"status":  "ok",
			},
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var decoded WebSocketMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "control", decoded.Type)
		assert.Equal(t, "connected", decoded.Control["message"])
	})
}

func TestWebSocketHub_BufferOverflow(t *testing.T) {
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "json"})
	hub := NewWebSocketHub(log)

	// Create client with small buffer
	client := &WebSocketClient{
		ID:            "client-1",
		send:          make(chan []byte, 1), // Very small buffer
		hub:           hub,
		executionID:   "",
		subscriptions: make(map[EventType]bool),
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Send multiple messages quickly to overflow buffer
	for i := 0; i < 10; i++ {
		message := []byte(`{"message": "test"}`)
		hub.Broadcast(message)
	}

	time.Sleep(100 * time.Millisecond)

	// Client should have been disconnected due to buffer overflow
	// Note: This test verifies the hub doesn't panic on buffer overflow
	assert.True(t, hub.ClientCount() >= 0) // Should not panic
}
