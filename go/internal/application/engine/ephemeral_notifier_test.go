package engine

import (
	"context"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/internal/application/observer"
	pkgengine "github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestEphemeralNotifier() (*EphemeralNotifier, *observer.MockObserver) {
	mgr := observer.NewObserverManager()
	mock := observer.NewMockObserver("test")
	_ = mgr.Register(mock)
	redactor := NewEventRedactor()
	notifier := NewEphemeralNotifier(mgr, redactor)
	return notifier, mock
}

func makeExecutionEvent(eventType string) pkgengine.ExecutionEvent {
	return pkgengine.ExecutionEvent{
		Type:        eventType,
		ExecutionID: "exec-1",
		WorkflowID:  "wf-1",
		Timestamp:   time.Now(),
		Status:      "running",
	}
}

func TestEphemeralNotifierSequence(t *testing.T) {
	notifier, mock := newTestEphemeralNotifier()
	ctx := context.Background()

	const callCount = 5
	for i := 0; i < callCount; i++ {
		notifier.Notify(ctx, makeExecutionEvent(pkgengine.EventTypeNodeCompleted))
	}

	// Wait briefly for async goroutines in ObserverManager to complete.
	time.Sleep(50 * time.Millisecond)

	events := mock.GetEvents()
	require.Len(t, events, callCount)

	for i, evt := range events {
		require.NotNil(t, evt.Metadata, "event %d has nil Metadata", i)
	}

	seqs := make([]int64, callCount)
	for i, evt := range events {
		seq, ok := evt.Metadata["sequence"].(int64)
		require.True(t, ok, "event %d sequence is not int64", i)
		seqs[i] = seq
	}

	seen := make(map[int64]bool)
	for _, s := range seqs {
		assert.False(t, seen[s], "duplicate sequence %d", s)
		seen[s] = true
	}

	for _, s := range seqs {
		assert.GreaterOrEqual(t, s, int64(1))
		assert.LessOrEqual(t, s, int64(callCount))
	}
}

func TestEphemeralNotifierRedaction(t *testing.T) {
	notifier, mock := newTestEphemeralNotifier()
	ctx := context.Background()

	event := makeExecutionEvent(pkgengine.EventTypeExecutionStarted)
	event.Variables = map[string]any{
		"api_key": "sk-1234567890abc",
		"count":   42,
	}

	notifier.Notify(ctx, event)
	time.Sleep(50 * time.Millisecond)

	events := mock.GetEvents()
	require.Len(t, events, 1)

	vars := events[0].Variables
	require.NotNil(t, vars)

	assert.Equal(t, "sk-***abc", vars["api_key"], "string variable must be redacted")
	assert.Equal(t, 42, vars["count"], "non-string variable must be preserved")
}

func TestEphemeralNotifierDelegation(t *testing.T) {
	notifier, mock := newTestEphemeralNotifier()
	ctx := context.Background()

	nodeID := "node-42"
	event := pkgengine.ExecutionEvent{
		Type:        pkgengine.EventTypeNodeCompleted,
		ExecutionID: "exec-delegation",
		WorkflowID:  "wf-delegation",
		NodeID:      nodeID,
		Status:      "completed",
		Timestamp:   time.Now(),
	}

	notifier.Notify(ctx, event)
	time.Sleep(50 * time.Millisecond)

	events := mock.GetEvents()
	require.Len(t, events, 1)

	received := events[0]
	assert.Equal(t, observer.EventType(pkgengine.EventTypeNodeCompleted), received.Type)
	assert.Equal(t, "exec-delegation", received.ExecutionID)
	assert.Equal(t, "wf-delegation", received.WorkflowID)
	assert.Equal(t, "completed", received.Status)
	require.NotNil(t, received.NodeID)
	assert.Equal(t, nodeID, *received.NodeID)
}
