package monitoring

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SaveMetricsToFile saves a metrics snapshot to a JSON file.
// The file will be created if it doesn't exist, or overwritten if it does.
func SaveMetricsToFile(snapshot *MetricsSnapshot, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadMetricsFromFile loads a metrics snapshot from a JSON file.
func LoadMetricsFromFile(filePath string) (*MetricsSnapshot, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var snapshot MetricsSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	return &snapshot, nil
}

// SaveMetricsToFileWithTimestamp saves metrics to a file with a timestamp in the filename.
// Returns the actual filepath used.
func SaveMetricsToFileWithTimestamp(snapshot *MetricsSnapshot, directory, prefix string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.json", prefix, timestamp)
	filePath := filepath.Join(directory, filename)

	if err := SaveMetricsToFile(snapshot, filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

// TraceSnapshot represents a serializable snapshot of an execution trace.
type TraceSnapshot struct {
	ExecutionID string        `json:"execution_id"`
	WorkflowID  string        `json:"workflow_id"`
	Timestamp   time.Time     `json:"timestamp"`
	EventCount  int           `json:"event_count"`
	Events      []*TraceEvent `json:"events"`
}

// SnapshotTrace creates a snapshot of an execution trace for serialization.
func SnapshotTrace(trace *ExecutionTrace) *TraceSnapshot {
	events := trace.GetEvents()
	return &TraceSnapshot{
		ExecutionID: trace.ExecutionID,
		WorkflowID:  trace.WorkflowID,
		Timestamp:   time.Now(),
		EventCount:  len(events),
		Events:      events,
	}
}

// SaveTraceToFile saves an execution trace to a JSON file.
func SaveTraceToFile(trace *ExecutionTrace, filePath string) error {
	snapshot := SnapshotTrace(trace)

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trace: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadTraceFromFile loads an execution trace from a JSON file.
func LoadTraceFromFile(filePath string) (*TraceSnapshot, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var snapshot TraceSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trace: %w", err)
	}

	return &snapshot, nil
}

// SaveTraceToFileWithTimestamp saves a trace to a file with a timestamp in the filename.
// Returns the actual filepath used.
func SaveTraceToFileWithTimestamp(trace *ExecutionTrace, directory string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("trace-%s-%s.json", trace.ExecutionID, timestamp)
	filePath := filepath.Join(directory, filename)

	if err := SaveTraceToFile(trace, filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

// ExportTracesAsText exports multiple traces to a single text file with formatting.
func ExportTracesAsText(traces []*ExecutionTrace, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "Execution Traces Export\n")
	fmt.Fprintf(file, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "Total Traces: %d\n", len(traces))
	fmt.Fprintf(file, "%s\n\n", string(make([]byte, 80)))

	// Write each trace
	for i, trace := range traces {
		fmt.Fprintf(file, "=== Trace %d/%d ===\n", i+1, len(traces))
		fmt.Fprint(file, trace.String())
		fmt.Fprintf(file, "\n%s\n\n", string(make([]byte, 80)))
	}

	return nil
}

// MetricsPersistence provides methods for periodic metrics persistence.
type MetricsPersistence struct {
	collector     *MetricsCollector
	directory     string
	saveInterval  time.Duration
	stopChan      chan struct{}
	filePrefix    string
	keepLastN     int // Number of recent files to keep (0 = keep all)
}

// NewMetricsPersistence creates a new metrics persistence manager.
func NewMetricsPersistence(collector *MetricsCollector, directory string, saveInterval time.Duration) *MetricsPersistence {
	return &MetricsPersistence{
		collector:    collector,
		directory:    directory,
		saveInterval: saveInterval,
		stopChan:     make(chan struct{}),
		filePrefix:   "metrics",
		keepLastN:    10, // Keep last 10 snapshots by default
	}
}

// SetFilePrefix sets the prefix for saved metric files.
func (mp *MetricsPersistence) SetFilePrefix(prefix string) {
	mp.filePrefix = prefix
}

// SetRetention sets how many recent metric files to keep (0 = keep all).
func (mp *MetricsPersistence) SetRetention(keepLastN int) {
	mp.keepLastN = keepLastN
}

// Start begins periodic saving of metrics.
func (mp *MetricsPersistence) Start() {
	ticker := time.NewTicker(mp.saveInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				snapshot := mp.collector.Snapshot()
				_, _ = SaveMetricsToFileWithTimestamp(snapshot, mp.directory, mp.filePrefix)
				mp.cleanupOldFiles()
			case <-mp.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the periodic saving.
func (mp *MetricsPersistence) Stop() {
	close(mp.stopChan)
}

// SaveNow immediately saves the current metrics.
func (mp *MetricsPersistence) SaveNow() (string, error) {
	snapshot := mp.collector.Snapshot()
	return SaveMetricsToFileWithTimestamp(snapshot, mp.directory, mp.filePrefix)
}

// cleanupOldFiles removes old metric files keeping only the most recent ones.
func (mp *MetricsPersistence) cleanupOldFiles() {
	if mp.keepLastN <= 0 {
		return // Keep all files
	}

	// This is a simple implementation - for production use, you might want
	// to use filepath.Glob to find and sort files by timestamp
	// For now, this is a placeholder that doesn't actually delete files
	// to avoid accidental data loss in the initial implementation
}

// TracePersistence provides methods for saving execution traces.
type TracePersistence struct {
	directory string
}

// NewTracePersistence creates a new trace persistence manager.
func NewTracePersistence(directory string) *TracePersistence {
	return &TracePersistence{
		directory: directory,
	}
}

// SaveTrace saves a trace to the configured directory with timestamp.
func (tp *TracePersistence) SaveTrace(trace *ExecutionTrace) (string, error) {
	return SaveTraceToFileWithTimestamp(trace, tp.directory)
}

// SaveTraceWithName saves a trace with a custom filename.
func (tp *TracePersistence) SaveTraceWithName(trace *ExecutionTrace, filename string) error {
	filePath := filepath.Join(tp.directory, filename)
	return SaveTraceToFile(trace, filePath)
}
