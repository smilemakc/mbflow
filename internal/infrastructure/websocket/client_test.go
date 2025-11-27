package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	client := NewClient("client-1", "user-1", hub, nil)

	assert.Equal(t, "client-1", client.id)
	assert.Equal(t, "user-1", client.userID)
	assert.Equal(t, hub, client.hub)
	assert.NotNil(t, client.send)
	assert.NotNil(t, client.subs)
}

func TestClient_ShouldReceive_NoSubscriptions(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	client := NewClient("client-1", "user-1", hub, nil)

	// No subscriptions - should not receive anything
	assert.False(t, client.shouldReceive("wf-123", "exec-456"))
	assert.False(t, client.shouldReceive("wf-123", ""))
	assert.False(t, client.shouldReceive("", "exec-456"))
	assert.False(t, client.shouldReceive("", ""))
}

func TestClient_ShouldReceive_WorkflowSubscription(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	client := NewClient("client-1", "user-1", hub, nil)

	// Subscribe to workflow
	client.subs.mu.Lock()
	client.subs.workflows["wf-123"] = true
	client.subs.mu.Unlock()

	// Should receive events for subscribed workflow
	assert.True(t, client.shouldReceive("wf-123", "exec-456"))
	assert.True(t, client.shouldReceive("wf-123", ""))

	// Should not receive events for other workflows
	assert.False(t, client.shouldReceive("wf-other", "exec-456"))
}

func TestClient_ShouldReceive_ExecutionSubscription(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	client := NewClient("client-1", "user-1", hub, nil)

	// Subscribe to execution
	client.subs.mu.Lock()
	client.subs.executions["exec-456"] = true
	client.subs.mu.Unlock()

	// Should receive events for subscribed execution
	assert.True(t, client.shouldReceive("wf-123", "exec-456"))
	assert.True(t, client.shouldReceive("", "exec-456"))

	// Should not receive events for other executions
	assert.False(t, client.shouldReceive("wf-123", "exec-other"))
}

func TestClient_ShouldReceive_BothSubscriptions(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	client := NewClient("client-1", "user-1", hub, nil)

	// Subscribe to both workflow and execution
	client.subs.mu.Lock()
	client.subs.workflows["wf-123"] = true
	client.subs.executions["exec-456"] = true
	client.subs.mu.Unlock()

	// Should receive events matching either subscription
	assert.True(t, client.shouldReceive("wf-123", "exec-other"))
	assert.True(t, client.shouldReceive("wf-other", "exec-456"))
	assert.True(t, client.shouldReceive("wf-123", "exec-456"))

	// Should not receive events matching neither
	assert.False(t, client.shouldReceive("wf-other", "exec-other"))
}

// Integration test with real WebSocket connection
func TestClient_IntegrationWithWebSocket(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	// Connect as WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Give time for connection to establish
	time.Sleep(50 * time.Millisecond)

	// Verify client is registered
	assert.Equal(t, 1, hub.ClientCount())
}

func TestClient_HandleSubscribeCommand(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Create test server that handles commands
	var receivedResponse *WSResponse
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send subscribe command
	cmd := WSCommand{
		Action:     CmdSubscribe,
		WorkflowID: "wf-123",
	}
	err = ws.WriteJSON(cmd)
	require.NoError(t, err)

	// Read response
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&receivedResponse)
	require.NoError(t, err)

	assert.Equal(t, CmdSubscribe, receivedResponse.Type)
	assert.True(t, receivedResponse.Success)
	assert.Contains(t, receivedResponse.Message, "wf-123")
}

func TestClient_HandleUnsubscribeCommand(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		// Pre-subscribe to workflow
		hub.Subscribe(client, "wf-123", "")

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send unsubscribe command
	cmd := WSCommand{
		Action:     CmdUnsubscribe,
		WorkflowID: "wf-123",
	}
	err = ws.WriteJSON(cmd)
	require.NoError(t, err)

	// Read response
	var response WSResponse
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.Equal(t, CmdUnsubscribe, response.Type)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "wf-123")
}

func TestClient_HandleInvalidCommand(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send invalid JSON
	err = ws.WriteMessage(websocket.TextMessage, []byte("not valid json"))
	require.NoError(t, err)

	// Read error response
	var response WSResponse
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "invalid command format")
}

func TestClient_HandleUnknownCommand(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send unknown command
	cmd := WSCommand{
		Action: "unknown_action",
	}
	err = ws.WriteJSON(cmd)
	require.NoError(t, err)

	// Read error response
	var response WSResponse
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "unknown command")
}

func TestClient_HandleSubscribeWithoutID(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send subscribe without workflow_id or execution_id
	cmd := WSCommand{
		Action: CmdSubscribe,
	}
	err = ws.WriteJSON(cmd)
	require.NoError(t, err)

	// Read error response
	var response WSResponse
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "required")
}

func TestClient_HandleCancelNotImplemented(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send cancel command
	cmd := WSCommand{
		Action:      CmdCancel,
		ExecutionID: "exec-123",
	}
	err = ws.WriteJSON(cmd)
	require.NoError(t, err)

	// Read error response (not implemented)
	var response WSResponse
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "not implemented")
}

func TestClient_HandleCancelWithoutExecutionID(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send cancel without execution_id
	cmd := WSCommand{
		Action: CmdCancel,
	}
	err = ws.WriteJSON(cmd)
	require.NoError(t, err)

	// Read error response
	var response WSResponse
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "execution_id required")
}

func TestClient_ReceiveBroadcastEvent(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	var serverClient *Client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		serverClient = NewClient("test-client", "test-user", hub, conn)
		hub.register <- serverClient

		go serverClient.writePump()
		go serverClient.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Subscribe to workflow
	subCmd := WSCommand{
		Action:     CmdSubscribe,
		WorkflowID: "wf-123",
	}
	err = ws.WriteJSON(subCmd)
	require.NoError(t, err)

	// Read subscribe response
	var subResp WSResponse
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&subResp)
	require.NoError(t, err)
	assert.True(t, subResp.Success)

	// Broadcast event from server
	event := NewWSEvent(EventExecutionStarted, "wf-123", "exec-1")
	hub.Broadcast("", "wf-123", "exec-1", event)

	// Read the broadcast event
	var receivedEvent WSEvent
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&receivedEvent)
	require.NoError(t, err)

	assert.Equal(t, EventExecutionStarted, receivedEvent.Type)
	assert.Equal(t, "wf-123", receivedEvent.WorkflowID)
	assert.Equal(t, "exec-1", receivedEvent.ExecutionID)
}

func TestClient_ConnectionClose(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	// Close connection
	ws.Close()

	// Wait for unregister
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestClient_SubscribeToExecution(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}

		client := NewClient("test-client", "test-user", hub, conn)
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Subscribe to execution
	cmd := WSCommand{
		Action:      CmdSubscribe,
		ExecutionID: "exec-456",
	}
	err = ws.WriteJSON(cmd)
	require.NoError(t, err)

	var response WSResponse
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "exec-456")
}

func TestSubscriptions_ThreadSafety(t *testing.T) {
	subs := NewSubscriptions()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			subs.mu.Lock()
			subs.workflows["wf-"+string(rune('0'+idx))] = true
			subs.mu.Unlock()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	subs.mu.RLock()
	count := len(subs.workflows)
	subs.mu.RUnlock()

	assert.Equal(t, 10, count)
}

func TestClient_WriteJSON(t *testing.T) {
	// Test with mock connection is complex, tested via integration tests above
	// This is a placeholder for documentation purposes
	t.Skip("WriteJSON tested through integration tests")
}

func TestClient_Constants(t *testing.T) {
	// Verify constants are reasonable
	assert.Equal(t, 10*time.Second, writeWait)
	assert.Equal(t, 60*time.Second, pongWait)
	assert.Less(t, pingPeriod, pongWait, "ping period must be less than pong wait")
	assert.Equal(t, 512, maxMessageSize)
	assert.Equal(t, 64, sendBufferSize)
}

func TestClient_HandleCommand_JSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonCmd  string
		wantErr  bool
		wantType string
	}{
		{
			name:     "valid subscribe workflow",
			jsonCmd:  `{"action":"subscribe","workflow_id":"wf-123"}`,
			wantErr:  false,
			wantType: CmdSubscribe,
		},
		{
			name:     "valid subscribe execution",
			jsonCmd:  `{"action":"subscribe","execution_id":"exec-456"}`,
			wantErr:  false,
			wantType: CmdSubscribe,
		},
		{
			name:     "valid unsubscribe",
			jsonCmd:  `{"action":"unsubscribe","workflow_id":"wf-123"}`,
			wantErr:  false,
			wantType: CmdUnsubscribe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd WSCommand
			err := json.Unmarshal([]byte(tt.jsonCmd), &cmd)
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, cmd.Action)
		})
	}
}
