package websocket

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewHub(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.byUserID)
	assert.NotNil(t, hub.byWorkflowID)
	assert.NotNil(t, hub.byExecutionID)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_RegisterClient(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	// Start hub in background
	go hub.Run()

	// Create a mock client (without actual websocket connection)
	client := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	// Register client
	hub.register <- client

	// Wait for registration
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())

	// Check user index
	hub.mu.RLock()
	_, ok := hub.byUserID["user-1"][client]
	hub.mu.RUnlock()
	assert.True(t, ok)
}

func TestHub_UnregisterClient(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	go hub.Run()

	client := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())

	// Check that user index is cleaned up
	hub.mu.RLock()
	_, ok := hub.byUserID["user-1"]
	hub.mu.RUnlock()
	assert.False(t, ok)
}

func TestHub_Subscribe(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	client := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	// Subscribe to workflow
	hub.Subscribe(client, "wf-123", "")

	hub.mu.RLock()
	_, wfOk := hub.byWorkflowID["wf-123"][client]
	hub.mu.RUnlock()
	assert.True(t, wfOk)

	client.subs.mu.RLock()
	_, subsOk := client.subs.workflows["wf-123"]
	client.subs.mu.RUnlock()
	assert.True(t, subsOk)

	// Subscribe to execution
	hub.Subscribe(client, "", "exec-456")

	hub.mu.RLock()
	_, execOk := hub.byExecutionID["exec-456"][client]
	hub.mu.RUnlock()
	assert.True(t, execOk)

	client.subs.mu.RLock()
	_, execSubsOk := client.subs.executions["exec-456"]
	client.subs.mu.RUnlock()
	assert.True(t, execSubsOk)
}

func TestHub_Unsubscribe(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	client := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	// Subscribe first
	hub.Subscribe(client, "wf-123", "exec-456")

	// Verify subscribed
	hub.mu.RLock()
	_, wfOk := hub.byWorkflowID["wf-123"][client]
	_, execOk := hub.byExecutionID["exec-456"][client]
	hub.mu.RUnlock()
	assert.True(t, wfOk)
	assert.True(t, execOk)

	// Unsubscribe from workflow
	hub.Unsubscribe(client, "wf-123", "")

	hub.mu.RLock()
	_, wfOkAfter := hub.byWorkflowID["wf-123"]
	hub.mu.RUnlock()
	assert.False(t, wfOkAfter)

	// Unsubscribe from execution
	hub.Unsubscribe(client, "", "exec-456")

	hub.mu.RLock()
	_, execOkAfter := hub.byExecutionID["exec-456"]
	hub.mu.RUnlock()
	assert.False(t, execOkAfter)
}

func TestHub_BroadcastToWorkflowSubscribers(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client1 := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	client2 := &Client{
		hub:    hub,
		id:     "client-2",
		userID: "user-2",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	// Register both clients
	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	// Subscribe client1 to workflow, client2 to different workflow
	hub.Subscribe(client1, "wf-123", "")
	hub.Subscribe(client2, "wf-456", "")

	// Broadcast to wf-123
	event := NewWSEvent(EventExecutionStarted, "wf-123", "exec-1")
	hub.Broadcast("", "wf-123", "exec-1", event)

	// Only client1 should receive the event
	select {
	case received := <-client1.send:
		assert.Equal(t, EventExecutionStarted, received.Type)
		assert.Equal(t, "wf-123", received.WorkflowID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("client1 did not receive event")
	}

	// client2 should NOT receive the event
	select {
	case <-client2.send:
		t.Fatal("client2 should not receive event for different workflow")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event received
	}
}

func TestHub_BroadcastToExecutionSubscribers(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Subscribe(client, "", "exec-123")

	event := NewWSEvent(EventNodeCompleted, "wf-1", "exec-123")
	hub.Broadcast("", "wf-1", "exec-123", event)

	select {
	case received := <-client.send:
		assert.Equal(t, EventNodeCompleted, received.Type)
		assert.Equal(t, "exec-123", received.ExecutionID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("client did not receive event")
	}
}

func TestHub_BroadcastByUserID(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client1 := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	client2 := &Client{
		hub:    hub,
		id:     "client-2",
		userID: "user-2",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	// Both subscribe to the same workflow
	hub.Subscribe(client1, "wf-123", "")
	hub.Subscribe(client2, "wf-123", "")

	// Broadcast to user-1 only
	event := NewWSEvent(EventExecutionStarted, "wf-123", "exec-1")
	hub.Broadcast("user-1", "wf-123", "exec-1", event)

	// client1 should receive (matches user_id and workflow subscription)
	select {
	case received := <-client1.send:
		assert.Equal(t, EventExecutionStarted, received.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("client1 did not receive event")
	}

	// client2 should NOT receive (different user_id)
	select {
	case <-client2.send:
		t.Fatal("client2 should not receive event for different user")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

func TestHub_ClientCount(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount())

	// Register multiple clients
	for i := 0; i < 3; i++ {
		client := &Client{
			hub:    hub,
			id:     "client-" + string(rune('0'+i)),
			userID: "user-" + string(rune('0'+i)),
			subs:   NewSubscriptions(),
			send:   make(chan *WSEvent, sendBufferSize),
		}
		hub.register <- client
	}

	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 3, hub.ClientCount())
}

func TestHub_UnregisterCleansUpSubscriptions(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Subscribe to workflow and execution
	hub.Subscribe(client, "wf-123", "exec-456")

	// Verify subscriptions
	hub.mu.RLock()
	_, wfOk := hub.byWorkflowID["wf-123"][client]
	_, execOk := hub.byExecutionID["exec-456"][client]
	hub.mu.RUnlock()
	assert.True(t, wfOk)
	assert.True(t, execOk)

	// Unregister
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	// Verify cleanup
	hub.mu.RLock()
	_, wfExists := hub.byWorkflowID["wf-123"]
	_, execExists := hub.byExecutionID["exec-456"]
	hub.mu.RUnlock()
	assert.False(t, wfExists)
	assert.False(t, execExists)
}

func TestHub_BroadcasterInterface(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	// Verify Hub implements Broadcaster interface
	var _ Broadcaster = hub
}

func TestHub_MultipleSubscriptionsToSameResource(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client1 := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	client2 := &Client{
		hub:    hub,
		id:     "client-2",
		userID: "user-2",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	// Both clients subscribe to the same workflow
	hub.Subscribe(client1, "wf-123", "")
	hub.Subscribe(client2, "wf-123", "")

	// Broadcast without user filter - both should receive
	event := NewWSEvent(EventExecutionStarted, "wf-123", "exec-1")
	hub.Broadcast("", "wf-123", "exec-1", event)

	receivedCount := 0
	timeout := time.After(100 * time.Millisecond)

	for receivedCount < 2 {
		select {
		case <-client1.send:
			receivedCount++
		case <-client2.send:
			receivedCount++
		case <-timeout:
			break
		}
		if receivedCount >= 2 {
			break
		}
	}

	assert.Equal(t, 2, receivedCount, "both clients should receive the broadcast")
}

func TestHub_UnsubscribePreservesOtherSubscribers(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	client1 := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	client2 := &Client{
		hub:    hub,
		id:     "client-2",
		userID: "user-2",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	// Both subscribe to same workflow
	hub.Subscribe(client1, "wf-123", "")
	hub.Subscribe(client2, "wf-123", "")

	// Unsubscribe client1
	hub.Unsubscribe(client1, "wf-123", "")

	// client2 should still be subscribed
	hub.mu.RLock()
	_, client2Ok := hub.byWorkflowID["wf-123"][client2]
	hub.mu.RUnlock()

	assert.True(t, client2Ok, "client2 should still be subscribed")

	// Verify client1 is not subscribed
	client1.subs.mu.RLock()
	_, client1SubsOk := client1.subs.workflows["wf-123"]
	client1.subs.mu.RUnlock()
	assert.False(t, client1SubsOk)
}

func TestNewSubscriptions(t *testing.T) {
	subs := NewSubscriptions()

	assert.NotNil(t, subs)
	assert.NotNil(t, subs.workflows)
	assert.NotNil(t, subs.executions)
	assert.Len(t, subs.workflows, 0)
	assert.Len(t, subs.executions, 0)
}

func TestHub_UnregisterUnknownClient(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Try to unregister a client that was never registered
	unknownClient := &Client{
		hub:    hub,
		id:     "unknown",
		userID: "user-1",
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	// Should not panic
	hub.unregister <- unknownClient
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_RegisterClientWithEmptyUserID(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client := &Client{
		hub:    hub,
		id:     "client-1",
		userID: "", // Empty user ID
		subs:   NewSubscriptions(),
		send:   make(chan *WSEvent, sendBufferSize),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())

	// Should not be indexed by user ID
	hub.mu.RLock()
	_, exists := hub.byUserID[""]
	hub.mu.RUnlock()
	assert.False(t, exists)
}

func TestBroadcastMsg_Structure(t *testing.T) {
	event := NewWSEvent(EventNodeStarted, "wf-1", "exec-1")
	msg := &broadcastMsg{
		userID:      "user-1",
		workflowID:  "wf-1",
		executionID: "exec-1",
		event:       event,
	}

	require.NotNil(t, msg)
	assert.Equal(t, "user-1", msg.userID)
	assert.Equal(t, "wf-1", msg.workflowID)
	assert.Equal(t, "exec-1", msg.executionID)
	assert.Equal(t, event, msg.event)
}
