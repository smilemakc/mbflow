package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"mbflow/internal/domain"
)

// ClickHouseLogger provides structured logging for workflow execution to ClickHouse.
// It batches log events and writes them asynchronously to ClickHouse for efficient storage and querying.
type ClickHouseLogger struct {
	// db is the ClickHouse database connection
	db *sql.DB
	// tableName is the name of the table to write logs to
	tableName string
	// batchSize is the number of events to batch before writing
	batchSize int
	// flushInterval is how often to flush batched events
	flushInterval time.Duration
	// verbose enables verbose logging
	verbose bool
	// buffer stores events before they are written
	buffer []*LogEvent
	// mu protects concurrent access to buffer
	mu sync.Mutex
	// ctx is the context for background operations
	ctx context.Context
	// cancel cancels background operations
	cancel context.CancelFunc
	// wg waits for background goroutines
	wg sync.WaitGroup
	// closed indicates if the logger is closed
	closed bool
}

// LogEvent represents a single log event to be written to ClickHouse.
type LogEvent struct {
	Timestamp     time.Time              `json:"timestamp"`
	ExecutionID   string                 `json:"execution_id"`
	WorkflowID    string                 `json:"workflow_id"`
	NodeID        string                 `json:"node_id,omitempty"`
	NodeType      string                 `json:"node_type,omitempty"`
	NodeName      string                 `json:"node_name,omitempty"`
	EventType     string                 `json:"event_type"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	Duration      int64                  `json:"duration_ms,omitempty"` // Duration in milliseconds
	AttemptNumber int                    `json:"attempt_number,omitempty"`
	WillRetry     bool                   `json:"will_retry,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ClickHouseLoggerConfig configures the ClickHouse logger.
type ClickHouseLoggerConfig struct {
	// DB is the ClickHouse database connection
	DB *sql.DB
	// TableName is the name of the table to write logs to (defaults to "workflow_execution_logs")
	TableName string
	// BatchSize is the number of events to batch before writing (defaults to 100)
	BatchSize int
	// FlushInterval is how often to flush batched events (defaults to 5 seconds)
	FlushInterval time.Duration
	// Verbose enables verbose logging
	Verbose bool
	// CreateTable automatically creates the table if it doesn't exist
	CreateTable bool
}

// NewClickHouseLogger creates a new ClickHouseLogger with the given configuration.
func NewClickHouseLogger(config ClickHouseLoggerConfig) (*ClickHouseLogger, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	tableName := config.TableName
	if tableName == "" {
		tableName = "workflow_execution_logs"
	}

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	flushInterval := config.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger := &ClickHouseLogger{
		db:            config.DB,
		tableName:     tableName,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		verbose:       config.Verbose,
		buffer:        make([]*LogEvent, 0, batchSize),
		ctx:           ctx,
		cancel:        cancel,
		closed:        false,
	}

	// Create table if requested
	if config.CreateTable {
		if err := logger.createTable(); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Start background flusher
	logger.wg.Add(1)
	go logger.backgroundFlusher()

	return logger, nil
}

// createTable creates the log table in ClickHouse if it doesn't exist.
func (l *ClickHouseLogger) createTable() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			timestamp DateTime64(3),
			execution_id String,
			workflow_id String,
			node_id String,
			node_type String,
			node_name String,
			event_type String,
			level String,
			message String,
			duration_ms Int64,
			attempt_number Int32,
			will_retry UInt8,
			error_message String,
			metadata String
		) ENGINE = MergeTree()
		ORDER BY (workflow_id, execution_id, timestamp)
		PARTITION BY toYYYYMM(timestamp)
	`, l.tableName)

	_, err := l.db.ExecContext(l.ctx, query)
	return err
}

// backgroundFlusher periodically flushes buffered events.
func (l *ClickHouseLogger) backgroundFlusher() {
	defer l.wg.Done()

	ticker := time.NewTicker(l.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			// Final flush before shutdown
			l.flush()
			return
		case <-ticker.C:
			l.flush()
		}
	}
}

// addEvent adds an event to the buffer and flushes if batch size is reached.
func (l *ClickHouseLogger) addEvent(event *LogEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return
	}

	event.Timestamp = time.Now()
	l.buffer = append(l.buffer, event)

	// Flush if batch size is reached
	if len(l.buffer) >= l.batchSize {
		go l.flush()
	}
}

// flush writes all buffered events to ClickHouse.
func (l *ClickHouseLogger) flush() {
	l.mu.Lock()
	if len(l.buffer) == 0 {
		l.mu.Unlock()
		return
	}

	// Swap buffer
	events := l.buffer
	l.buffer = make([]*LogEvent, 0, l.batchSize)
	l.mu.Unlock()

	// Write to ClickHouse
	if err := l.writeEvents(events); err != nil {
		// In production, you might want to handle this error differently
		// (e.g., write to a file, send to error monitoring service)
		fmt.Printf("ClickHouseLogger: failed to write events: %v\n", err)
	}
}

// writeEvents writes a batch of events to ClickHouse.
func (l *ClickHouseLogger) writeEvents(events []*LogEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := l.db.BeginTx(l.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(l.ctx, fmt.Sprintf(`
		INSERT INTO %s (
			timestamp, execution_id, workflow_id, node_id, node_type, node_name,
			event_type, level, message, duration_ms, attempt_number, will_retry,
			error_message, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, l.tableName))
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		// Serialize metadata to JSON
		var metadataJSON string
		if event.Metadata != nil {
			metadataBytes, err := json.Marshal(event.Metadata)
			if err != nil {
				metadataJSON = "{}"
			} else {
				metadataJSON = string(metadataBytes)
			}
		} else {
			metadataJSON = "{}"
		}

		willRetryInt := 0
		if event.WillRetry {
			willRetryInt = 1
		}

		_, err := stmt.ExecContext(l.ctx,
			event.Timestamp,
			event.ExecutionID,
			event.WorkflowID,
			event.NodeID,
			event.NodeType,
			event.NodeName,
			event.EventType,
			event.Level,
			event.Message,
			event.Duration,
			event.AttemptNumber,
			willRetryInt,
			event.ErrorMessage,
			metadataJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close closes the logger and flushes any remaining events.
func (l *ClickHouseLogger) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()

	// Cancel background operations
	l.cancel()

	// Wait for background goroutines
	l.wg.Wait()

	return nil
}

// Implementation of ExecutionLogger interface

func (l *ClickHouseLogger) LogExecutionStarted(workflowID, executionID string) {
	l.addEvent(&LogEvent{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		EventType:   "execution_started",
		Level:       "info",
		Message:     "Workflow execution started",
	})
}

func (l *ClickHouseLogger) LogExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	l.addEvent(&LogEvent{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		EventType:   "execution_completed",
		Level:       "info",
		Message:     "Workflow execution completed",
		Duration:    duration.Milliseconds(),
	})
}

func (l *ClickHouseLogger) LogExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	l.addEvent(&LogEvent{
		ExecutionID:  executionID,
		WorkflowID:   workflowID,
		EventType:    "execution_failed",
		Level:        "error",
		Message:      "Workflow execution failed",
		Duration:     duration.Milliseconds(),
		ErrorMessage: errorMsg,
	})
}

func (l *ClickHouseLogger) LogNodeStarted(executionID string, node *domain.Node, attemptNumber int) {
	if node == nil {
		l.LogNodeStartedFromConfig(executionID, "", "", "", "", nil, attemptNumber)
		return
	}

	l.LogNodeStartedFromConfig(
		executionID,
		node.ID(),
		node.WorkflowID(),
		node.Type(),
		node.Name(),
		node.Config(),
		attemptNumber,
	)
}

func (l *ClickHouseLogger) LogNodeStartedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int) {
	l.addEvent(&LogEvent{
		ExecutionID:   executionID,
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		NodeType:      nodeType,
		NodeName:      name,
		EventType:     "node_started",
		Level:         "info",
		Message:       fmt.Sprintf("Node started: %s", name),
		AttemptNumber: attemptNumber,
		Metadata:      config,
	})
}

func (l *ClickHouseLogger) LogNodeCompleted(executionID string, node *domain.Node, duration time.Duration) {
	if node == nil {
		l.LogNodeCompletedFromConfig(executionID, "", "", "", "", nil, duration)
		return
	}

	l.LogNodeCompletedFromConfig(
		executionID,
		node.ID(),
		node.WorkflowID(),
		node.Type(),
		node.Name(),
		node.Config(),
		duration,
	)
}

func (l *ClickHouseLogger) LogNodeCompletedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, duration time.Duration) {
	l.addEvent(&LogEvent{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		NodeName:    name,
		EventType:   "node_completed",
		Level:       "info",
		Message:     fmt.Sprintf("Node completed: %s", name),
		Duration:    duration.Milliseconds(),
		Metadata:    config,
	})
}

func (l *ClickHouseLogger) LogNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool) {
	if node == nil {
		l.LogNodeFailedFromConfig(executionID, "", "", "", "", nil, err, duration, willRetry)
		return
	}

	l.LogNodeFailedFromConfig(
		executionID,
		node.ID(),
		node.WorkflowID(),
		node.Type(),
		node.Name(),
		node.Config(),
		err,
		duration,
		willRetry,
	)
}

func (l *ClickHouseLogger) LogNodeFailedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, err error, duration time.Duration, willRetry bool) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	l.addEvent(&LogEvent{
		ExecutionID:  executionID,
		WorkflowID:   workflowID,
		NodeID:       nodeID,
		NodeType:     nodeType,
		NodeName:     name,
		EventType:    "node_failed",
		Level:        "error",
		Message:      fmt.Sprintf("Node failed: %s", name),
		Duration:     duration.Milliseconds(),
		ErrorMessage: errorMsg,
		WillRetry:    willRetry,
		Metadata:     config,
	})
}

func (l *ClickHouseLogger) LogNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration) {
	if node == nil {
		l.LogNodeRetryingFromConfig(executionID, "", "", "", "", nil, attemptNumber, delay)
		return
	}

	l.LogNodeRetryingFromConfig(
		executionID,
		node.ID(),
		node.WorkflowID(),
		node.Type(),
		node.Name(),
		node.Config(),
		attemptNumber,
		delay,
	)
}

func (l *ClickHouseLogger) LogNodeRetryingFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int, delay time.Duration) {
	l.addEvent(&LogEvent{
		ExecutionID:   executionID,
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		NodeType:      nodeType,
		NodeName:      name,
		EventType:     "node_retrying",
		Level:         "warning",
		Message:       fmt.Sprintf("Node retrying: %s", name),
		AttemptNumber: attemptNumber,
		Duration:      delay.Milliseconds(),
		Metadata:      config,
	})
}

func (l *ClickHouseLogger) LogNodeSkipped(executionID string, node *domain.Node, reason string) {
	if node == nil {
		l.LogNodeSkippedFromConfig(executionID, "", "", "", "", nil, reason)
		return
	}

	l.LogNodeSkippedFromConfig(
		executionID,
		node.ID(),
		node.WorkflowID(),
		node.Type(),
		node.Name(),
		node.Config(),
		reason,
	)
}

func (l *ClickHouseLogger) LogNodeSkippedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, reason string) {
	l.addEvent(&LogEvent{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		NodeName:    name,
		EventType:   "node_skipped",
		Level:       "info",
		Message:     fmt.Sprintf("Node skipped: %s (reason: %s)", name, reason),
		Metadata: map[string]interface{}{
			"reason": reason,
			"config": config,
		},
	})
}

func (l *ClickHouseLogger) LogNode(executionID string, node *domain.Node) {
	if node == nil {
		l.LogNodeFromConfig(executionID, "", "", "", "", nil)
		return
	}

	l.LogNodeFromConfig(
		executionID,
		node.ID(),
		node.WorkflowID(),
		node.Type(),
		node.Name(),
		node.Config(),
	)
}

func (l *ClickHouseLogger) LogNodeFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any) {
	l.addEvent(&LogEvent{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		NodeName:    name,
		EventType:   "node_info",
		Level:       "debug",
		Message:     fmt.Sprintf("Node info: %s", name),
		Metadata:    config,
	})
}

func (l *ClickHouseLogger) LogVariableSet(executionID, key string, value interface{}) {
	if !l.verbose {
		return
	}

	l.addEvent(&LogEvent{
		ExecutionID: executionID,
		EventType:   "variable_set",
		Level:       "debug",
		Message:     fmt.Sprintf("Variable set: %s", key),
		Metadata: map[string]interface{}{
			"key":   key,
			"value": value,
		},
	})
}

func (l *ClickHouseLogger) LogError(executionID string, message string, err error) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	l.addEvent(&LogEvent{
		ExecutionID:  executionID,
		EventType:    "error",
		Level:        "error",
		Message:      message,
		ErrorMessage: errorMsg,
	})
}

func (l *ClickHouseLogger) LogInfo(executionID string, message string) {
	l.addEvent(&LogEvent{
		ExecutionID: executionID,
		EventType:   "info",
		Level:       "info",
		Message:     message,
	})
}

func (l *ClickHouseLogger) LogDebug(executionID string, message string) {
	if !l.verbose {
		return
	}

	l.addEvent(&LogEvent{
		ExecutionID: executionID,
		EventType:   "debug",
		Level:       "debug",
		Message:     message,
	})
}

func (l *ClickHouseLogger) LogTransition(executionID, nodeID, fromState, toState string) {
	if !l.verbose {
		return
	}

	l.addEvent(&LogEvent{
		ExecutionID: executionID,
		NodeID:      nodeID,
		EventType:   "state_transition",
		Level:       "debug",
		Message:     fmt.Sprintf("State transition: %s -> %s", fromState, toState),
		Metadata: map[string]interface{}{
			"from_state": fromState,
			"to_state":   toState,
		},
	})
}
