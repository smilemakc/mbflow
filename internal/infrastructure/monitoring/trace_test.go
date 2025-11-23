package monitoring

import (
	"fmt"
	"testing"
	"time"
)

func TestExecutionTrace_Basic(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	if trace.ExecutionID != "exec-123" {
		t.Errorf("Expected execution ID 'exec-123', got '%s'", trace.ExecutionID)
	}

	if trace.WorkflowID != "workflow-1" {
		t.Errorf("Expected workflow ID 'workflow-1', got '%s'", trace.WorkflowID)
	}

	if len(trace.Events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(trace.Events))
	}
}

func TestExecutionTrace_AddEvent(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	trace.AddEvent("execution_started", "", "", "Started", nil, nil)
	trace.AddEvent("node_started", "node-1", "http", "Node started", nil, nil)

	events := trace.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].EventType != "execution_started" {
		t.Errorf("Expected first event type 'execution_started', got '%s'", events[0].EventType)
	}

	if events[1].NodeID != "node-1" {
		t.Errorf("Expected second event node ID 'node-1', got '%s'", events[1].NodeID)
	}
}

func TestExecutionTrace_GetDuration(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	// Initially no duration
	duration := trace.GetDuration()
	if duration != 0 {
		t.Errorf("Expected 0 duration for empty trace, got %v", duration)
	}

	// Add events with time gap
	trace.AddEvent("event1", "", "", "First", nil, nil)
	time.Sleep(50 * time.Millisecond)
	trace.AddEvent("event2", "", "", "Second", nil, nil)

	duration = trace.GetDuration()
	if duration < 40*time.Millisecond {
		t.Errorf("Expected duration >= 40ms, got %v", duration)
	}
	if duration > 100*time.Millisecond {
		t.Errorf("Expected duration <= 100ms, got %v", duration)
	}
}

func TestExecutionTrace_GetEventsByType(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	trace.AddEvent("execution_started", "", "", "Started", nil, nil)
	trace.AddEvent("node_started", "node-1", "http", "Node 1", nil, nil)
	trace.AddEvent("node_started", "node-2", "transform", "Node 2", nil, nil)
	trace.AddEvent("execution_completed", "", "", "Completed", nil, nil)

	nodeEvents := trace.GetEventsByType("node_started")
	if len(nodeEvents) != 2 {
		t.Errorf("Expected 2 node_started events, got %d", len(nodeEvents))
	}

	execEvents := trace.GetEventsByType("execution_started")
	if len(execEvents) != 1 {
		t.Errorf("Expected 1 execution_started event, got %d", len(execEvents))
	}

	nonexistent := trace.GetEventsByType("nonexistent")
	if len(nonexistent) != 0 {
		t.Errorf("Expected 0 nonexistent events, got %d", len(nonexistent))
	}
}

func TestExecutionTrace_GetEventsByNodeID(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	trace.AddEvent("node_started", "node-1", "http", "Node 1 start", nil, nil)
	trace.AddEvent("node_completed", "node-1", "http", "Node 1 complete", nil, nil)
	trace.AddEvent("node_started", "node-2", "transform", "Node 2 start", nil, nil)

	node1Events := trace.GetEventsByNodeID("node-1")
	if len(node1Events) != 2 {
		t.Errorf("Expected 2 events for node-1, got %d", len(node1Events))
	}

	node2Events := trace.GetEventsByNodeID("node-2")
	if len(node2Events) != 1 {
		t.Errorf("Expected 1 event for node-2, got %d", len(node2Events))
	}
}

func TestExecutionTrace_GetErrorEvents(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	trace.AddEvent("execution_started", "", "", "Started", nil, nil)
	trace.AddEvent("node_started", "node-1", "http", "Node started", nil, nil)
	trace.AddEvent("node_failed", "node-1", "http", "Node failed",
		nil, fmt.Errorf("connection timeout"))
	trace.AddEvent("node_failed", "node-2", "transform", "Transform failed",
		nil, fmt.Errorf("invalid expression"))

	errorEvents := trace.GetErrorEvents()
	if len(errorEvents) != 2 {
		t.Errorf("Expected 2 error events, got %d", len(errorEvents))
	}

	for _, event := range errorEvents {
		if event.Error == nil {
			t.Error("Error event should have non-nil error")
		}
	}
}

func TestExecutionTrace_HasErrors(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	// Initially no errors
	if trace.HasErrors() {
		t.Error("New trace should not have errors")
	}

	// Add normal events
	trace.AddEvent("execution_started", "", "", "Started", nil, nil)
	trace.AddEvent("node_completed", "node-1", "http", "Completed", nil, nil)

	if trace.HasErrors() {
		t.Error("Trace with no error events should return false")
	}

	// Add error event
	trace.AddEvent("node_failed", "node-2", "transform", "Failed",
		nil, fmt.Errorf("error"))

	if !trace.HasErrors() {
		t.Error("Trace with error events should return true")
	}
}

func TestExecutionTrace_GetSummary(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	trace.AddEvent("execution_started", "", "", "Started", nil, nil)
	trace.AddEvent("node_started", "node-1", "http", "Node 1", nil, nil)
	trace.AddEvent("node_completed", "node-1", "http", "Node 1 done", nil, nil)
	trace.AddEvent("node_started", "node-2", "transform", "Node 2", nil, nil)
	trace.AddEvent("node_failed", "node-2", "transform", "Node 2 failed",
		nil, fmt.Errorf("error"))
	time.Sleep(10 * time.Millisecond)
	trace.AddEvent("execution_failed", "", "", "Failed", nil, fmt.Errorf("workflow failed"))

	summary := trace.GetSummary()

	if summary.ExecutionID != "exec-123" {
		t.Errorf("Expected execution ID 'exec-123', got '%s'", summary.ExecutionID)
	}

	if summary.WorkflowID != "workflow-1" {
		t.Errorf("Expected workflow ID 'workflow-1', got '%s'", summary.WorkflowID)
	}

	if summary.TotalEvents != 6 {
		t.Errorf("Expected 6 total events, got %d", summary.TotalEvents)
	}

	if summary.ErrorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", summary.ErrorCount)
	}

	if len(summary.NodeIDs) != 2 {
		t.Errorf("Expected 2 unique node IDs, got %d", len(summary.NodeIDs))
	}

	// Check event type counts
	if summary.EventTypes["execution_started"] != 1 {
		t.Errorf("Expected 1 execution_started event, got %d",
			summary.EventTypes["execution_started"])
	}

	if summary.EventTypes["node_started"] != 2 {
		t.Errorf("Expected 2 node_started events, got %d",
			summary.EventTypes["node_started"])
	}

	if summary.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", summary.Duration)
	}
}

func TestExecutionTrace_String(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	trace.AddEvent("execution_started", "", "", "Started", nil, nil)
	trace.AddEvent("node_started", "node-1", "http", "HTTP node", nil, nil)
	trace.AddEvent("node_failed", "node-1", "http", "Failed",
		nil, fmt.Errorf("timeout"))

	output := trace.String()

	// Check that output contains expected elements
	if len(output) == 0 {
		t.Error("String output should not be empty")
	}

	// Should contain execution ID
	if !contains(output, "exec-123") {
		t.Error("Output should contain execution ID")
	}

	// Should contain workflow ID
	if !contains(output, "workflow-1") {
		t.Error("Output should contain workflow ID")
	}

	// Should contain event count
	if !contains(output, "3") {
		t.Error("Output should contain event count")
	}
}

func TestExecutionTrace_ConcurrentAccess(t *testing.T) {
	trace := NewExecutionTrace("exec-123", "workflow-1")

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			trace.AddEvent("test_event", fmt.Sprintf("node-%d", id), "test",
				fmt.Sprintf("Event %d", id), nil, nil)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = trace.GetEvents()
			_ = trace.GetSummary()
			_ = trace.String()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	events := trace.GetEvents()
	if len(events) != 10 {
		t.Errorf("Expected 10 events after concurrent writes, got %d", len(events))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
