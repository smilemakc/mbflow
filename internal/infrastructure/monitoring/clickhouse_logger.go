package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/internal/domain"
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

// Log logs a single event. This is the main logging method.
func (l *ClickHouseLogger) Log(event *LogEvent) {
	if event == nil {
		return
	}

	// Skip debug events if not in verbose mode
	if event.Level == LevelDebug && !l.verbose {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return
	}

	// Ensure timestamp is set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

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
		metadataJSON := "{}"
		if event.Config != nil || event.Metadata != nil || event.VariableValue != nil {
			metadata := make(map[string]interface{})

			if event.Config != nil {
				metadata["config"] = event.Config
			}
			if event.Metadata != nil {
				for k, v := range event.Metadata {
					metadata[k] = v
				}
			}
			if event.VariableKey != "" {
				metadata["variable_key"] = event.VariableKey
				metadata["variable_value"] = event.VariableValue
			}
			if event.FromState != "" {
				metadata["from_state"] = event.FromState
				metadata["to_state"] = event.ToState
			}
			if event.Reason != "" {
				metadata["reason"] = event.Reason
			}

			metadataBytes, err := json.Marshal(metadata)
			if err == nil {
				metadataJSON = string(metadataBytes)
			}
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
			string(event.Type),
			string(event.Level),
			event.Message,
			event.Duration.Milliseconds(),
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

func (l *ClickHouseLogger) LogNode(workflowID, executionID string, node domain.Node) {
	if node == nil {
		l.Log(NewInfoEvent(workflowID, executionID, "Node info: node=<nil>"))
		return
	}

	l.Log(&LogEvent{
		Timestamp:   time.Now(),
		Type:        EventInfo,
		Level:       LevelDebug,
		Message:     "Node info",
		ExecutionID: executionID,
		NodeID:      node.ID().String(),
		NodeType:    string(node.Type()),
		NodeName:    node.Name(),
		Config:      node.Config(),
	})
}

func (l *ClickHouseLogger) LogNodeFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any) {
	l.Log(&LogEvent{
		Timestamp:   time.Now(),
		Type:        EventInfo,
		Level:       LevelDebug,
		Message:     "Node info",
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		NodeName:    name,
		Config:      config,
	})
}

func (l *ClickHouseLogger) LogVariableSet(workflowID, executionID, key string, value interface{}) {
	if !l.verbose {
		return
	}
	l.Log(NewVariableSetEvent(workflowID, executionID, key, value))
}

func (l *ClickHouseLogger) LogError(workflowID, executionID string, message string, err error) {
	l.Log(NewErrorEvent(workflowID, executionID, message, err))
}

func (l *ClickHouseLogger) LogInfo(workflowID, executionID string, message string) {
	l.Log(NewInfoEvent(workflowID, executionID, message))
}

func (l *ClickHouseLogger) LogDebug(workflowID, executionID string, message string) {
	if !l.verbose {
		return
	}
	l.Log(NewDebugEvent(workflowID, executionID, message))
}

func (l *ClickHouseLogger) LogTransition(workflowID, executionID, nodeID, fromState, toState string) {
	if !l.verbose {
		return
	}
	l.Log(NewStateTransitionEvent(workflowID, executionID, nodeID, fromState, toState))
}
