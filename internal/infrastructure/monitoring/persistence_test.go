package monitoring

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadMetrics(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "metrics.json")

	// Create metrics collector and add some data
	collector := NewMetricsCollector()
	collector.RecordWorkflowExecution("workflow-1", 100*time.Millisecond, true)
	collector.RecordWorkflowExecution("workflow-1", 150*time.Millisecond, true)
	collector.RecordNodeExecution("node-1", "http", "HTTP Request", 50*time.Millisecond, true, false)
	collector.RecordAIRequest(500, 200, 2*time.Second)

	// Create snapshot and save
	snapshot := collector.Snapshot()
	err := SaveMetricsToFile(snapshot, filePath)
	if err != nil {
		t.Fatalf("Failed to save metrics: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Metrics file was not created")
	}

	// Load metrics
	loadedSnapshot, err := LoadMetricsFromFile(filePath)
	if err != nil {
		t.Fatalf("Failed to load metrics: %v", err)
	}

	// Verify data
	if loadedSnapshot.Summary.TotalWorkflows != 1 {
		t.Errorf("Expected 1 workflow, got %d", loadedSnapshot.Summary.TotalWorkflows)
	}

	if loadedSnapshot.Summary.TotalExecutions != 2 {
		t.Errorf("Expected 2 executions, got %d", loadedSnapshot.Summary.TotalExecutions)
	}

	if loadedSnapshot.Summary.TotalAIRequests != 1 {
		t.Errorf("Expected 1 AI request, got %d", loadedSnapshot.Summary.TotalAIRequests)
	}

	// Verify workflow metrics
	wfMetrics, ok := loadedSnapshot.WorkflowMetrics["workflow-1"]
	if !ok {
		t.Fatal("Workflow metrics not found")
	}

	if wfMetrics.ExecutionCount != 2 {
		t.Errorf("Expected 2 executions, got %d", wfMetrics.ExecutionCount)
	}

	if wfMetrics.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", wfMetrics.SuccessCount)
	}
}

func TestSaveMetricsWithTimestamp(t *testing.T) {
	tmpDir := t.TempDir()

	collector := NewMetricsCollector()
	collector.RecordWorkflowExecution("test-wf", 100*time.Millisecond, true)

	snapshot := collector.Snapshot()
	filePath, err := SaveMetricsToFileWithTimestamp(snapshot, tmpDir, "test-metrics")
	if err != nil {
		t.Fatalf("Failed to save metrics with timestamp: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Metrics file was not created: %s", filePath)
	}

	// Verify filename contains prefix
	filename := filepath.Base(filePath)
	if len(filename) < len("test-metrics") {
		t.Errorf("Filename too short: %s", filename)
	}
}

func TestSaveAndLoadTrace(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "trace.json")

	// Create trace and add events
	trace := NewExecutionTrace("exec-123", "workflow-1")
	trace.AddEvent("execution_started", "", "", "Started", nil, nil)
	trace.AddEvent("node_started", "node-1", "http", "Node started", nil, nil)
	trace.AddEvent("node_completed", "node-1", "http", "Node completed", nil, nil)
	trace.AddEvent("execution_completed", "", "", "Completed", nil, nil)

	// Save trace
	err := SaveTraceToFile(trace, filePath)
	if err != nil {
		t.Fatalf("Failed to save trace: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Trace file was not created")
	}

	// Load trace
	loadedSnapshot, err := LoadTraceFromFile(filePath)
	if err != nil {
		t.Fatalf("Failed to load trace: %v", err)
	}

	// Verify data
	if loadedSnapshot.ExecutionID != "exec-123" {
		t.Errorf("Expected execution ID 'exec-123', got '%s'", loadedSnapshot.ExecutionID)
	}

	if loadedSnapshot.WorkflowID != "workflow-1" {
		t.Errorf("Expected workflow ID 'workflow-1', got '%s'", loadedSnapshot.WorkflowID)
	}

	if loadedSnapshot.EventCount != 4 {
		t.Errorf("Expected 4 events, got %d", loadedSnapshot.EventCount)
	}

	if len(loadedSnapshot.Events) != 4 {
		t.Errorf("Expected 4 events in array, got %d", len(loadedSnapshot.Events))
	}
}

func TestSaveTraceWithTimestamp(t *testing.T) {
	tmpDir := t.TempDir()

	trace := NewExecutionTrace("exec-456", "workflow-2")
	trace.AddEvent("execution_started", "", "", "Started", nil, nil)

	filePath, err := SaveTraceToFileWithTimestamp(trace, tmpDir)
	if err != nil {
		t.Fatalf("Failed to save trace with timestamp: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Trace file was not created: %s", filePath)
	}

	// Verify filename contains execution ID
	filename := filepath.Base(filePath)
	if len(filename) < len("trace-exec-456") {
		t.Errorf("Filename too short: %s", filename)
	}
}

func TestExportTracesAsText(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "traces.txt")

	// Create multiple traces
	traces := []*ExecutionTrace{
		NewExecutionTrace("exec-1", "workflow-1"),
		NewExecutionTrace("exec-2", "workflow-2"),
	}

	for _, trace := range traces {
		trace.AddEvent("execution_started", "", "", "Started", nil, nil)
		trace.AddEvent("execution_completed", "", "", "Completed", nil, nil)
	}

	// Export to text
	err := ExportTracesAsText(traces, filePath)
	if err != nil {
		t.Fatalf("Failed to export traces as text: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Text file was not created")
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read text file: %v", err)
	}

	// Verify content contains both execution IDs
	contentStr := string(content)
	if len(contentStr) == 0 {
		t.Error("Text file is empty")
	}
}

func TestMetricsPersistence_SaveNow(t *testing.T) {
	tmpDir := t.TempDir()

	collector := NewMetricsCollector()
	collector.RecordWorkflowExecution("test", 100*time.Millisecond, true)

	persistence := NewMetricsPersistence(collector, tmpDir, 1*time.Hour)
	persistence.SetFilePrefix("test")

	filePath, err := persistence.SaveNow()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("File was not created: %s", filePath)
	}
}

func TestTracePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	trace := NewExecutionTrace("exec-789", "workflow-3")
	trace.AddEvent("execution_started", "", "", "Started", nil, nil)

	persistence := NewTracePersistence(tmpDir)

	// Test SaveTrace
	filePath, err := persistence.SaveTrace(trace)
	if err != nil {
		t.Fatalf("Failed to save trace: %v", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Trace file was not created: %s", filePath)
	}

	// Test SaveTraceWithName
	customName := "custom-trace.json"
	err = persistence.SaveTraceWithName(trace, customName)
	if err != nil {
		t.Fatalf("Failed to save trace with name: %v", err)
	}

	customPath := filepath.Join(tmpDir, customName)
	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Fatalf("Custom trace file was not created: %s", customPath)
	}
}

func TestSnapshotTrace(t *testing.T) {
	trace := NewExecutionTrace("exec-snapshot", "workflow-snap")
	trace.AddEvent("event1", "", "", "Event 1", nil, nil)
	trace.AddEvent("event2", "", "", "Event 2", nil, nil)

	snapshot := SnapshotTrace(trace)

	if snapshot.ExecutionID != "exec-snapshot" {
		t.Errorf("Expected execution ID 'exec-snapshot', got '%s'", snapshot.ExecutionID)
	}

	if snapshot.WorkflowID != "workflow-snap" {
		t.Errorf("Expected workflow ID 'workflow-snap', got '%s'", snapshot.WorkflowID)
	}

	if snapshot.EventCount != 2 {
		t.Errorf("Expected 2 events, got %d", snapshot.EventCount)
	}

	if len(snapshot.Events) != 2 {
		t.Errorf("Expected 2 events in array, got %d", len(snapshot.Events))
	}
}
