package monitoring

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/smilemakc/mbflow/internal/domain"
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
	logger zerolog.Logger
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

	level := zerolog.InfoLevel
	if config.Verbose {
		level = zerolog.DebugLevel
	}
	logger := zerolog.New(writer).With().Timestamp().Logger().Level(level).Output(zerolog.ConsoleWriter{Out: writer})
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("prefix", config.Prefix)
	})
	return &ConsoleLogger{
		prefix:  config.Prefix,
		verbose: config.Verbose,
		writer:  writer,
		logger:  logger,
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
	l.formatEvent(event)
}

// formatEvent formats a log event into a human-readable string.
func (l *ConsoleLogger) formatEvent(event *LogEvent) {
	// Build message based on event type
	var zeroEvent *zerolog.Event
	switch event.Level {
	case LevelDebug:
		zeroEvent = l.logger.Debug()
	case LevelInfo:
		zeroEvent = l.logger.Info()
	case LevelError:
		zeroEvent = l.logger.Error()
	case LevelWarning:
		zeroEvent = l.logger.Warn()
	default:
		zeroEvent = l.logger.Info()
	}

	// if event.NodeID != "" {
	// 	zeroEvent.Str("node", event.NodeID)
	// }
	if event.NodeName != "" {
		zeroEvent.Str("node-name", event.NodeName)
	} else {
		zeroEvent.Str("execution", event.ExecutionID).Str("workflow", event.WorkflowID)
	}
	// if event.NodeType != "" {
	// 	zeroEvent.Str("node-type", event.NodeType)
	// }
	if event.FromState != "" {
		zeroEvent.Str("from-state", event.FromState)
	}
	if event.ToState != "" {
		zeroEvent.Str("to-state", event.ToState)
	}
	if event.Reason != "" {
		zeroEvent.Str("reason", event.Reason)
	}
	if event.VariableKey != "" {
		zeroEvent.Str("variable-key", event.VariableKey)
	}
	if event.ErrorMessage != "" {
		zeroEvent.Err(fmt.Errorf(event.ErrorMessage))
	}
	if event.AttemptNumber > 0 {
		zeroEvent.Int("attempt", event.AttemptNumber)
	}
	if event.Duration > 0 {
		zeroEvent.Dur("duration", event.Duration)
	}
	// if event.Config != nil {
	// 	zeroEvent.Int("config-keys-count", len(event.Config))
	// }
	if event.RetryDelay > 0 {
		zeroEvent.Dur("retry-delay", event.RetryDelay)
	}
	if event.Message != "" {
		zeroEvent.Msg(event.Message)
	} else {
		zeroEvent.Send()
	}
}

func (l *ConsoleLogger) LogNode(workflowID, executionID string, node domain.Node) {
	// Convert to info event
	if node == nil {
		l.Log(NewInfoEvent(workflowID, executionID, "Node info: node=<nil>"))
		return
	}

	l.Log(&LogEvent{
		Timestamp:   time.Now(),
		Type:        EventInfo,
		Level:       LevelInfo,
		Message:     "Node info",
		ExecutionID: executionID,
		NodeID:      node.ID().String(),
		NodeType:    string(node.Type()),
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

func (l *ConsoleLogger) LogVariableSet(workflowID, executionID, key string, value interface{}) {
	if !l.verbose {
		return
	}
	l.Log(NewVariableSetEvent(workflowID, executionID, key, value))
}

func (l *ConsoleLogger) LogError(workflowID, executionID string, message string, err error) {
	l.Log(NewErrorEvent(workflowID, executionID, message, err))
}

func (l *ConsoleLogger) LogInfo(workflowID, executionID string, message string) {
	l.Log(NewInfoEvent(workflowID, executionID, message))
}

func (l *ConsoleLogger) LogDebug(workflowID, executionID string, message string) {
	if !l.verbose {
		return
	}
	l.Log(NewDebugEvent(workflowID, executionID, message))
}

func (l *ConsoleLogger) LogTransition(workflowID, executionID, nodeID, fromState, toState string) {
	if !l.verbose {
		return
	}
	l.Log(NewStateTransitionEvent(workflowID, executionID, nodeID, fromState, toState))
}

// Additional utility methods

// SetWriter changes the output writer for the logger.
// This is useful for redirecting logs to a file or other destination.
func (l *ConsoleLogger) SetWriter(writer io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer = writer
	level := zerolog.InfoLevel
	if l.verbose {
		level = zerolog.DebugLevel
	}
	logger := zerolog.New(writer).With().Timestamp().Logger().Level(level)
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("prefix", l.prefix)
	})
	l.logger = logger

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
