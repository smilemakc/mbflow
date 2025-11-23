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

// LogExecutionStarted logs when a workflow execution starts.
func (l *ConsoleLogger) LogExecutionStarted(workflowID, executionID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("[%s] Execution started: workflow=%s execution=%s", l.prefix, workflowID, executionID)
}

// LogExecutionCompleted logs when a workflow execution completes successfully.
func (l *ConsoleLogger) LogExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("[%s] Execution completed: workflow=%s execution=%s duration=%s",
		l.prefix, workflowID, executionID, duration)
}

// LogExecutionFailed logs when a workflow execution fails.
func (l *ConsoleLogger) LogExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("[%s] Execution failed: workflow=%s execution=%s duration=%s error=%v",
		l.prefix, workflowID, executionID, duration, err)
}

// LogNodeStarted logs when a node starts executing.
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeStartedFromConfig should be used instead.
func (l *ConsoleLogger) LogNodeStarted(executionID string, node *domain.Node, attemptNumber int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		l.logger.Printf("[%s] Node started: execution=%s node=<nil> attempt=%d",
			l.prefix, executionID, attemptNumber)
		return
	}

	if attemptNumber > 1 {
		l.logger.Printf("[%s] Node started (retry %d): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
			l.prefix, attemptNumber, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config())
	} else {
		l.logger.Printf("[%s] Node started: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
			l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config())
	}
}

// LogNodeStartedFromConfig logs when a node starts executing from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ConsoleLogger) LogNodeStartedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if attemptNumber > 1 {
		l.logger.Printf("[%s] Node started (retry %d): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
			l.prefix, attemptNumber, executionID, nodeID, workflowID, nodeType, name, config)
	} else {
		l.logger.Printf("[%s] Node started: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
			l.prefix, executionID, nodeID, workflowID, nodeType, name, config)
	}
}

// LogNodeCompleted logs when a node completes successfully.
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeCompletedFromConfig should be used instead.
func (l *ConsoleLogger) LogNodeCompleted(executionID string, node *domain.Node, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		l.logger.Printf("[%s] Node completed: execution=%s node=<nil> duration=%s",
			l.prefix, executionID, duration)
		return
	}

	l.logger.Printf("[%s] Node completed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s",
		l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), duration)
}

// LogNodeCompletedFromConfig logs when a node completes successfully from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ConsoleLogger) LogNodeCompletedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger.Printf("[%s] Node completed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s",
		l.prefix, executionID, nodeID, workflowID, nodeType, name, config, duration)
}

// LogNodeFailed logs when a node fails.
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeFailedFromConfig should be used instead.
func (l *ConsoleLogger) LogNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		if willRetry {
			l.logger.Printf("[%s] Node failed (will retry): execution=%s node=<nil> duration=%s error=%v",
				l.prefix, executionID, duration, err)
		} else {
			l.logger.Printf("[%s] Node failed: execution=%s node=<nil> duration=%s error=%v",
				l.prefix, executionID, duration, err)
		}
		return
	}

	if willRetry {
		l.logger.Printf("[%s] Node failed (will retry): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
			l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), duration, err)
	} else {
		l.logger.Printf("[%s] Node failed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
			l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), duration, err)
	}
}

// LogNodeFailedFromConfig logs when a node fails from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ConsoleLogger) LogNodeFailedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, err error, duration time.Duration, willRetry bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if willRetry {
		l.logger.Printf("[%s] Node failed (will retry): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
			l.prefix, executionID, nodeID, workflowID, nodeType, name, config, duration, err)
	} else {
		l.logger.Printf("[%s] Node failed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
			l.prefix, executionID, nodeID, workflowID, nodeType, name, config, duration, err)
	}
}

// LogNodeRetrying logs when a node is being retried.
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeRetryingFromConfig should be used instead.
func (l *ConsoleLogger) LogNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		l.logger.Printf("[%s] Node retrying: execution=%s node=<nil> attempt=%d delay=%s",
			l.prefix, executionID, attemptNumber, delay)
		return
	}

	l.logger.Printf("[%s] Node retrying: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v attempt=%d delay=%s",
		l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), attemptNumber, delay)
}

// LogNodeRetryingFromConfig logs when a node is being retried from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ConsoleLogger) LogNodeRetryingFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int, delay time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger.Printf("[%s] Node retrying: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v attempt=%d delay=%s",
		l.prefix, executionID, nodeID, workflowID, nodeType, name, config, attemptNumber, delay)
}

// LogNodeSkipped logs when a node is skipped.
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeSkippedFromConfig should be used instead.
func (l *ConsoleLogger) LogNodeSkipped(executionID string, node *domain.Node, reason string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		l.logger.Printf("[%s] Node skipped: execution=%s node=<nil> reason=%s",
			l.prefix, executionID, reason)
		return
	}

	l.logger.Printf("[%s] Node skipped: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v reason=%s",
		l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), reason)
}

// LogNodeSkippedFromConfig logs when a node is skipped from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ConsoleLogger) LogNodeSkippedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, reason string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger.Printf("[%s] Node skipped: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v reason=%s",
		l.prefix, executionID, nodeID, workflowID, nodeType, name, config, reason)
}

// LogNode logs all fields of a node.
// It accepts either a domain.Node or its configuration.
// If node is provided, all fields are extracted from it.
// If node is nil, LogNodeFromConfig should be used instead.
func (l *ConsoleLogger) LogNode(executionID string, node *domain.Node) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		l.logger.Printf("[%s] Node info: execution=%s node=<nil>", l.prefix, executionID)
		return
	}

	l.logger.Printf("[%s] Node info: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
		l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config())
}

// LogNodeFromConfig logs all fields of a node from its configuration and metadata.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ConsoleLogger) LogNodeFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger.Printf("[%s] Node info: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
		l.prefix, executionID, nodeID, workflowID, nodeType, name, config)
}

// LogVariableSet logs when a variable is set (verbose mode only).
func (l *ConsoleLogger) LogVariableSet(executionID, key string, value interface{}) {
	if !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("[%s] Variable set: execution=%s key=%s value=%v",
		l.prefix, executionID, key, value)
}

// LogError logs a general error.
func (l *ConsoleLogger) LogError(executionID string, message string, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("[%s] Error: execution=%s message=%s error=%v",
		l.prefix, executionID, message, err)
}

// LogInfo logs an informational message.
func (l *ConsoleLogger) LogInfo(executionID string, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("[%s] Info: execution=%s message=%s", l.prefix, executionID, message)
}

// LogDebug logs a debug message (verbose mode only).
func (l *ConsoleLogger) LogDebug(executionID string, message string) {
	if !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("[%s] Debug: execution=%s message=%s", l.prefix, executionID, message)
}

// LogTransition logs a state transition.
func (l *ConsoleLogger) LogTransition(executionID, nodeID, fromState, toState string) {
	if !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("[%s] State transition: execution=%s node=%s from=%s to=%s",
		l.prefix, executionID, nodeID, fromState, toState)
}

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

// logf is a helper method for formatted logging (internal use).
func (l *ConsoleLogger) logf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf(format, args...)
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

// Helper function to format config for logging
func formatConfig(config map[string]any) string {
	if config == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", config)
}
