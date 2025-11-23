package monitoring

import (
	"time"

	"mbflow/internal/domain"
)

// ExecutionLogger defines the interface for logging workflow execution events.
// Implementations can log to console, files, databases (ClickHouse), or other destinations.
type ExecutionLogger interface {
	// Log logs a single event. This is the main method for all logging.
	Log(event *LogEvent)
}

// Legacy interface methods for backward compatibility
// These methods are deprecated and will be removed in a future version.
// Use the Log method with appropriate event helper functions instead.
type LegacyExecutionLogger interface {
	ExecutionLogger

	// LogExecutionStarted logs when a workflow execution starts.
	// Deprecated: Use Log(NewExecutionStartedEvent(...)) instead.
	LogExecutionStarted(workflowID, executionID string)

	// LogExecutionCompleted logs when a workflow execution completes successfully.
	// Deprecated: Use Log(NewExecutionCompletedEvent(...)) instead.
	LogExecutionCompleted(workflowID, executionID string, duration time.Duration)

	// LogExecutionFailed logs when a workflow execution fails.
	// Deprecated: Use Log(NewExecutionFailedEvent(...)) instead.
	LogExecutionFailed(workflowID, executionID string, err error, duration time.Duration)

	// LogNodeStarted logs when a node starts executing.
	// Deprecated: Use Log(NewNodeStartedEvent(...)) instead.
	LogNodeStarted(executionID string, node *domain.Node, attemptNumber int)

	// LogNodeStartedFromConfig logs when a node starts executing from its configuration.
	// Deprecated: Use Log(NewNodeStartedEventFromConfig(...)) instead.
	LogNodeStartedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int)

	// LogNodeCompleted logs when a node completes successfully.
	// Deprecated: Use Log(NewNodeCompletedEvent(...)) instead.
	LogNodeCompleted(executionID string, node *domain.Node, duration time.Duration)

	// LogNodeCompletedFromConfig logs when a node completes successfully from its configuration.
	// Deprecated: Use Log(NewNodeCompletedEventFromConfig(...)) instead.
	LogNodeCompletedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, duration time.Duration)

	// LogNodeFailed logs when a node fails.
	// Deprecated: Use Log(NewNodeFailedEvent(...)) instead.
	LogNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool)

	// LogNodeFailedFromConfig logs when a node fails from its configuration.
	// Deprecated: Use Log(NewNodeFailedEventFromConfig(...)) instead.
	LogNodeFailedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, err error, duration time.Duration, willRetry bool)

	// LogNodeRetrying logs when a node is being retried.
	// Deprecated: Use Log(NewNodeRetryingEvent(...)) instead.
	LogNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration)

	// LogNodeRetryingFromConfig logs when a node is being retried from its configuration.
	// Deprecated: Use Log(NewNodeRetryingEventFromConfig(...)) instead.
	LogNodeRetryingFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int, delay time.Duration)

	// LogNodeSkipped logs when a node is skipped.
	// Deprecated: Use Log(NewNodeSkippedEvent(...)) instead.
	LogNodeSkipped(executionID string, node *domain.Node, reason string)

	// LogNodeSkippedFromConfig logs when a node is skipped from its configuration.
	// Deprecated: Use Log(NewNodeSkippedEventFromConfig(...)) instead.
	LogNodeSkippedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, reason string)

	// LogNode logs all fields of a node.
	// Deprecated: No direct replacement, use appropriate event type.
	LogNode(executionID string, node *domain.Node)

	// LogNodeFromConfig logs all fields of a node from its configuration and metadata.
	// Deprecated: No direct replacement, use appropriate event type.
	LogNodeFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any)

	// LogVariableSet logs when a variable is set (verbose mode only).
	// Deprecated: Use Log(NewVariableSetEvent(...)) instead.
	LogVariableSet(executionID, key string, value interface{})

	// LogError logs a general error.
	// Deprecated: Use Log(NewErrorEvent(...)) instead.
	LogError(executionID string, message string, err error)

	// LogInfo logs an informational message.
	// Deprecated: Use Log(NewInfoEvent(...)) instead.
	LogInfo(executionID string, message string)

	// LogDebug logs a debug message (verbose mode only).
	// Deprecated: Use Log(NewDebugEvent(...)) instead.
	LogDebug(executionID string, message string)

	// LogTransition logs a state transition.
	// Deprecated: Use Log(NewStateTransitionEvent(...)) instead.
	LogTransition(executionID, nodeID, fromState, toState string)
}
