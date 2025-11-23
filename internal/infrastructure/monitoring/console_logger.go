package monitoring

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"mbflow/internal/domain"
)

// ConsoleLogger provides structured logging for workflow execution to console or any writer.
// It logs node transitions, errors, and execution events with context.
type ConsoleLogger struct {
	// prefix is prepended to all log messages
	prefix string
	// verbose enables verbose logging
	verbose bool
	// writer is the destination for log output
	writer io.Writer
	// logger is the underlying logger
	logger *log.Logger
	// mu protects concurrent writes
	mu sync.Mutex
}

// ConsoleLoggerConfig configures the console logger.
type ConsoleLoggerConfig struct {
	// Prefix is prepended to all log messages
	Prefix string
	// Verbose enables verbose logging
	Verbose bool
	// Writer is the destination for log output (defaults to os.Stdout)
	Writer io.Writer
}

// NewConsoleLogger creates a new ConsoleLogger with the given configuration.
func NewConsoleLogger(config ConsoleLoggerConfig) *ConsoleLogger {
	writer := config.Writer
	if writer == nil {
		writer = os.Stdout
	}

	return &ConsoleLogger{
		prefix:  config.Prefix,
		verbose: config.Verbose,
		writer:  writer,
		logger:  log.New(writer, "", log.LstdFlags),
	}
}

// NewDefaultConsoleLogger creates a new ConsoleLogger with default settings (stdout, non-verbose).
func NewDefaultConsoleLogger(prefix string) *ConsoleLogger {
	return NewConsoleLogger(ConsoleLoggerConfig{
		Prefix:  prefix,
		Verbose: false,
		Writer:  os.Stdout,
	})
}

// Log logs a single event. This is the main logging method.
func (l *ConsoleLogger) Log(event *LogEvent) {
	if event == nil {
		return
	}

	// Skip debug events if not in verbose mode
	if event.Level == LevelDebug && !l.verbose {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Format message based on event type
	message := l.formatEvent(event)
	l.logger.Print(message)
}

// formatEvent formats a log event into a human-readable string.
func (l *ConsoleLogger) formatEvent(event *LogEvent) string {
	// Build message based on event type
	switch event.Type {
	case EventExecutionStarted:
		return fmt.Sprintf("[%s] Execution started: workflow=%s execution=%s",
			l.prefix, event.WorkflowID, event.ExecutionID)

	case EventExecutionCompleted:
		return fmt.Sprintf("[%s] Execution completed: workflow=%s execution=%s duration=%s",
			l.prefix, event.WorkflowID, event.ExecutionID, event.Duration)

	case EventExecutionFailed:
		return fmt.Sprintf("[%s] Execution failed: workflow=%s execution=%s duration=%s error=%v",
			l.prefix, event.WorkflowID, event.ExecutionID, event.Duration, event.ErrorMessage)

	case EventNodeStarted:
		if event.AttemptNumber > 1 {
			return fmt.Sprintf("[%s] Node started (retry %d): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
				l.prefix, event.AttemptNumber, event.ExecutionID, event.NodeID, event.WorkflowID, event.NodeType, event.NodeName, event.Config)
		}
		return fmt.Sprintf("[%s] Node started: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
			l.prefix, event.ExecutionID, event.NodeID, event.WorkflowID, event.NodeType, event.NodeName, event.Config)

	case EventNodeCompleted:
		return fmt.Sprintf("[%s] Node completed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s",
			l.prefix, event.ExecutionID, event.NodeID, event.WorkflowID, event.NodeType, event.NodeName, event.Config, event.Duration)

	case EventNodeFailed:
		if event.WillRetry {
			return fmt.Sprintf("[%s] Node failed (will retry): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
				l.prefix, event.ExecutionID, event.NodeID, event.WorkflowID, event.NodeType, event.NodeName, event.Config, event.Duration, event.ErrorMessage)
		}
		return fmt.Sprintf("[%s] Node failed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
			l.prefix, event.ExecutionID, event.NodeID, event.WorkflowID, event.NodeType, event.NodeName, event.Config, event.Duration, event.ErrorMessage)

	case EventNodeRetrying:
		return fmt.Sprintf("[%s] Node retrying: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v attempt=%d delay=%s",
			l.prefix, event.ExecutionID, event.NodeID, event.WorkflowID, event.NodeType, event.NodeName, event.Config, event.AttemptNumber, event.RetryDelay)

	case EventNodeSkipped:
		return fmt.Sprintf("[%s] Node skipped: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v reason=%s",
			l.prefix, event.ExecutionID, event.NodeID, event.WorkflowID, event.NodeType, event.NodeName, event.Config, event.Reason)

	case EventVariableSet:
		return fmt.Sprintf("[%s] Variable set: execution=%s key=%s value=%v",
			l.prefix, event.ExecutionID, event.VariableKey, event.VariableValue)

	case EventStateTransition:
		return fmt.Sprintf("[%s] State transition: execution=%s node=%s from=%s to=%s",
			l.prefix, event.ExecutionID, event.NodeID, event.FromState, event.ToState)

	case EventInfo:
		return fmt.Sprintf("[%s] Info: execution=%s message=%s", l.prefix, event.ExecutionID, event.Message)

	case EventDebug:
		return fmt.Sprintf("[%s] Debug: execution=%s message=%s", l.prefix, event.ExecutionID, event.Message)

	case EventError:
		return fmt.Sprintf("[%s] Error: execution=%s message=%s error=%v",
			l.prefix, event.ExecutionID, event.Message, event.ErrorMessage)

	default:
		// Generic formatting for unknown event types
		return fmt.Sprintf("[%s] %s: execution=%s message=%s",
			l.prefix, event.Type, event.ExecutionID, event.Message)
	}
}

func (l *ConsoleLogger) LogNode(executionID string, node *domain.Node) {
	// Convert to info event
	if node == nil {
		l.Log(NewInfoEvent(executionID, "Node info: node=<nil>"))
		return
	}

	l.Log(&LogEvent{
		Timestamp:   time.Now(),
		Type:        EventInfo,
		Level:       LevelInfo,
		Message:     "Node info",
		ExecutionID: executionID,
		WorkflowID:  node.WorkflowID(),
		NodeID:      node.ID(),
		NodeType:    node.Type(),
		NodeName:    node.Name(),
		Config:      node.Config(),
	})
}

func (l *ConsoleLogger) LogNodeFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any) {
	l.Log(&LogEvent{
		Timestamp:   time.Now(),
		Type:        EventInfo,
		Level:       LevelInfo,
		Message:     "Node info",
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		NodeName:    name,
		Config:      config,
	})
}

func (l *ConsoleLogger) LogVariableSet(executionID, key string, value interface{}) {
	if !l.verbose {
		return
	}
	l.Log(NewVariableSetEvent(executionID, key, value))
}

func (l *ConsoleLogger) LogError(executionID string, message string, err error) {
	l.Log(NewErrorEvent(executionID, message, err))
}

func (l *ConsoleLogger) LogInfo(executionID string, message string) {
	l.Log(NewInfoEvent(executionID, message))
}

func (l *ConsoleLogger) LogDebug(executionID string, message string) {
	if !l.verbose {
		return
	}
	l.Log(NewDebugEvent(executionID, message))
}

func (l *ConsoleLogger) LogTransition(executionID, nodeID, fromState, toState string) {
	if !l.verbose {
		return
	}
	l.Log(NewStateTransitionEvent(executionID, nodeID, fromState, toState))
}

// Additional utility methods

// SetWriter changes the output writer for the logger.
// This is useful for redirecting logs to a file or other destination.
func (l *ConsoleLogger) SetWriter(writer io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer = writer
	l.logger = log.New(writer, "", log.LstdFlags)
}

// GetWriter returns the current writer.
func (l *ConsoleLogger) GetWriter() io.Writer {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.writer
}

// SetVerbose enables or disables verbose logging.
func (l *ConsoleLogger) SetVerbose(verbose bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.verbose = verbose
}

// IsVerbose returns whether verbose logging is enabled.
func (l *ConsoleLogger) IsVerbose() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.verbose
}

// Flush ensures all buffered logs are written (if the writer supports flushing).
func (l *ConsoleLogger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if writer implements Flush method (e.g., bufio.Writer)
	type flusher interface {
		Flush() error
	}

	if f, ok := l.writer.(flusher); ok {
		return f.Flush()
	}

	return nil
}

// NewExecutionLogger creates a new ConsoleLogger for backward compatibility.
// Deprecated: Use NewConsoleLogger instead.
func NewExecutionLogger(prefix string, verbose bool) ExecutionLogger {
	return NewConsoleLogger(ConsoleLoggerConfig{
		Prefix:  prefix,
		Verbose: verbose,
		Writer:  os.Stdout,
	})
}
