package websocket

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBroadcaster is a mock implementation of the Broadcaster interface
type mockBroadcaster struct {
	mu       sync.Mutex
	events   []*WSEvent
	userIDs  []string
	wfIDs    []string
	execIDs  []string
	received chan *WSEvent
}

func newMockBroadcaster() *mockBroadcaster {
	return &mockBroadcaster{
		events:   make([]*WSEvent, 0),
		userIDs:  make([]string, 0),
		wfIDs:    make([]string, 0),
		execIDs:  make([]string, 0),
		received: make(chan *WSEvent, 100),
	}
}

func (m *mockBroadcaster) Broadcast(userID, workflowID, executionID string, event *WSEvent) {
	m.mu.Lock()
	m.events = append(m.events, event)
	m.userIDs = append(m.userIDs, userID)
	m.wfIDs = append(m.wfIDs, workflowID)
	m.execIDs = append(m.execIDs, executionID)
	m.mu.Unlock()

	select {
	case m.received <- event:
	default:
	}
}

func (m *mockBroadcaster) lastEvent() *WSEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) == 0 {
		return nil
	}
	return m.events[len(m.events)-1]
}

func (m *mockBroadcaster) eventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

// mockNode is a mock implementation of the domain.Node interface
type mockNode struct {
	id       uuid.UUID
	name     string
	nodeType domain.NodeType
}

func newMockNode(id uuid.UUID, name string, nodeType domain.NodeType) *mockNode {
	return &mockNode{
		id:       id,
		name:     name,
		nodeType: nodeType,
	}
}

func (m *mockNode) ID() uuid.UUID                                  { return m.id }
func (m *mockNode) Name() string                                   { return m.name }
func (m *mockNode) Type() domain.NodeType                          { return m.nodeType }
func (m *mockNode) Config() map[string]any                         { return nil }
func (m *mockNode) IOSchema() *domain.NodeIOSchema                 { return nil }
func (m *mockNode) InputBindingConfig() *domain.InputBindingConfig { return nil }

func TestSocketObserver_ImplementsInterface(t *testing.T) {
	// Verify SocketObserver implements monitoring.ExecutionObserver
	var _ monitoring.ExecutionObserver = (*SocketObserver)(nil)
}

func TestNewSocketObserver(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	assert.NotNil(t, observer)
	assert.Equal(t, broadcaster, observer.hub)
}

func TestSocketObserver_OnExecutionStarted(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	observer.OnExecutionStarted("wf-123", "exec-456")

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventExecutionStarted, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
}

func TestSocketObserver_OnExecutionCompleted(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	duration := 5 * time.Second
	observer.OnExecutionCompleted("wf-123", "exec-456", duration)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventExecutionCompleted, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.Equal(t, int64(5000), event.DurationMs)
}

func TestSocketObserver_OnExecutionFailed(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	duration := 3 * time.Second
	err := errors.New("workflow failed")
	observer.OnExecutionFailed("wf-123", "exec-456", err, duration)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventExecutionFailed, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.Equal(t, int64(3000), event.DurationMs)
	assert.Equal(t, "workflow failed", event.Error)
}

func TestSocketObserver_OnExecutionFailed_NilError(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	observer.OnExecutionFailed("wf-123", "exec-456", nil, time.Second)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventExecutionFailed, event.Type)
	assert.Empty(t, event.Error)
}

func TestSocketObserver_OnNodeStarted(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "process_data", domain.NodeTypeTransform)

	observer.OnNodeStarted("wf-123", "exec-456", node, 1)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventNodeStarted, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.Equal(t, nodeID.String(), event.NodeID)
	assert.Equal(t, "process_data", event.NodeName)
	assert.Equal(t, string(domain.NodeTypeTransform), event.NodeType)
	assert.Equal(t, 1, event.AttemptNumber)
}

func TestSocketObserver_OnNodeCompleted(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "process_data", domain.NodeTypeTransform)
	output := map[string]string{"result": "success"}
	duration := 150 * time.Millisecond

	observer.OnNodeCompleted("wf-123", "exec-456", node, output, duration)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventNodeCompleted, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.Equal(t, nodeID.String(), event.NodeID)
	assert.Equal(t, "process_data", event.NodeName)
	assert.Equal(t, int64(150), event.DurationMs)
	assert.Equal(t, output, event.Output)
}

func TestSocketObserver_OnNodeFailed(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "process_data", domain.NodeTypeTransform)
	err := errors.New("node execution failed")
	duration := 100 * time.Millisecond

	observer.OnNodeFailed("wf-123", "exec-456", node, err, duration, true)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventNodeFailed, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.Equal(t, nodeID.String(), event.NodeID)
	assert.Equal(t, "process_data", event.NodeName)
	assert.Equal(t, int64(100), event.DurationMs)
	assert.Equal(t, "node execution failed", event.Error)
	assert.True(t, event.WillRetry)
}

func TestSocketObserver_OnNodeFailed_NoRetry(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "process_data", domain.NodeTypeTransform)

	observer.OnNodeFailed("wf-123", "exec-456", node, errors.New("failed"), time.Second, false)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.False(t, event.WillRetry)
}

func TestSocketObserver_OnNodeFailed_NilError(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "process_data", domain.NodeTypeTransform)

	observer.OnNodeFailed("wf-123", "exec-456", node, nil, time.Second, false)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Empty(t, event.Error)
}

func TestSocketObserver_OnNodeRetrying(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "process_data", domain.NodeTypeTransform)
	delay := 500 * time.Millisecond

	observer.OnNodeRetrying("wf-123", "exec-456", node, 2, delay)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventNodeRetrying, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.Equal(t, nodeID.String(), event.NodeID)
	assert.Equal(t, "process_data", event.NodeName)
	assert.Equal(t, 2, event.AttemptNumber)
	assert.Equal(t, int64(500), event.DelayMs)
}

func TestSocketObserver_OnVariableSet(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	value := map[string]int{"count": 42}
	observer.OnVariableSet("wf-123", "exec-456", "myVar", value)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventVariableSet, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.Equal(t, "myVar", event.Key)
	assert.Equal(t, value, event.Value)
}

func TestSocketObserver_OnNodeCallbackStarted(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "callback_node", domain.NodeTypeFunctionCall)

	observer.OnNodeCallbackStarted("wf-123", "exec-456", node)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventCallbackStarted, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.Equal(t, nodeID.String(), event.NodeID)
	assert.Equal(t, "callback_node", event.NodeName)
	assert.Equal(t, string(domain.NodeTypeFunctionCall), event.NodeType)
}

func TestSocketObserver_OnNodeCallbackCompleted(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "callback_node", domain.NodeTypeFunctionCall)
	duration := 200 * time.Millisecond

	observer.OnNodeCallbackCompleted("wf-123", "exec-456", node, nil, duration)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventCallbackCompleted, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.Equal(t, nodeID.String(), event.NodeID)
	assert.Equal(t, "callback_node", event.NodeName)
	assert.Equal(t, int64(200), event.DurationMs)
	assert.Empty(t, event.Error)
}

func TestSocketObserver_OnNodeCallbackCompleted_WithError(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "callback_node", domain.NodeTypeFunctionCall)
	err := errors.New("callback error")

	observer.OnNodeCallbackCompleted("wf-123", "exec-456", node, err, time.Second)

	event := broadcaster.lastEvent()
	require.NotNil(t, event)

	assert.Equal(t, EventCallbackCompleted, event.Type)
	assert.Equal(t, "callback error", event.Error)
}

func TestPopulateNodeFields_NilNode(t *testing.T) {
	event := NewWSEvent(EventNodeStarted, "wf-123", "exec-456")

	populateNodeFields(event, nil)

	// Fields should remain empty
	assert.Empty(t, event.NodeID)
	assert.Empty(t, event.NodeName)
	assert.Empty(t, event.NodeType)
}

func TestPopulateNodeFields_ValidNode(t *testing.T) {
	event := NewWSEvent(EventNodeStarted, "wf-123", "exec-456")
	nodeID := uuid.New()
	node := newMockNode(nodeID, "my_node", domain.NodeTypeStart)

	populateNodeFields(event, node)

	assert.Equal(t, nodeID.String(), event.NodeID)
	assert.Equal(t, "my_node", event.NodeName)
	assert.Equal(t, string(domain.NodeTypeStart), event.NodeType)
}

func TestSocketObserver_BroadcastParameters(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	// Check that broadcast is called with correct parameters
	observer.OnExecutionStarted("wf-123", "exec-456")

	broadcaster.mu.Lock()
	defer broadcaster.mu.Unlock()

	require.Len(t, broadcaster.userIDs, 1)
	require.Len(t, broadcaster.wfIDs, 1)
	require.Len(t, broadcaster.execIDs, 1)

	// userID should be empty (not filtered by user)
	assert.Empty(t, broadcaster.userIDs[0])
	assert.Equal(t, "wf-123", broadcaster.wfIDs[0])
	assert.Equal(t, "exec-456", broadcaster.execIDs[0])
}

func TestSocketObserver_MultipleEvents(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeID := uuid.New()
	node := newMockNode(nodeID, "test_node", domain.NodeTypeTransform)

	// Simulate a typical execution flow
	observer.OnExecutionStarted("wf-1", "exec-1")
	observer.OnNodeStarted("wf-1", "exec-1", node, 1)
	observer.OnNodeCompleted("wf-1", "exec-1", node, "output", 100*time.Millisecond)
	observer.OnExecutionCompleted("wf-1", "exec-1", 200*time.Millisecond)

	assert.Equal(t, 4, broadcaster.eventCount())

	broadcaster.mu.Lock()
	events := broadcaster.events
	broadcaster.mu.Unlock()

	assert.Equal(t, EventExecutionStarted, events[0].Type)
	assert.Equal(t, EventNodeStarted, events[1].Type)
	assert.Equal(t, EventNodeCompleted, events[2].Type)
	assert.Equal(t, EventExecutionCompleted, events[3].Type)
}

func TestSocketObserver_AllNodeTypes(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	nodeTypes := []domain.NodeType{
		domain.NodeTypeStart,
		domain.NodeTypeEnd,
		domain.NodeTypeTransform,
		domain.NodeTypeHTTP,
		domain.NodeTypeLLM,
		domain.NodeTypeCode,
		domain.NodeTypeParallel,
		domain.NodeTypeConditionalRoute,
		domain.NodeTypeFunctionCall,
	}

	for _, nt := range nodeTypes {
		t.Run(string(nt), func(t *testing.T) {
			nodeID := uuid.New()
			node := newMockNode(nodeID, "node-"+string(nt), nt)

			observer.OnNodeStarted("wf", "exec", node, 1)

			event := broadcaster.lastEvent()
			require.NotNil(t, event)
			assert.Equal(t, string(nt), event.NodeType)
		})
	}
}

func TestSocketObserver_ConcurrentBroadcasts(t *testing.T) {
	broadcaster := newMockBroadcaster()
	observer := NewSocketObserver(broadcaster)

	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				observer.OnExecutionStarted("wf", "exec")
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, numGoroutines*eventsPerGoroutine, broadcaster.eventCount())
}
