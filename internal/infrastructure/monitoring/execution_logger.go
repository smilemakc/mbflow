package monitoring

import (
	"time"

	"mbflow/internal/domain"
)

// ExecutionLogger defines the interface for logging workflow execution events.
// Implementations can log to console, files, databases (ClickHouse), or other destinations.
type ExecutionLogger interface {
	// LogExecutionStarted logs when a workflow execution starts.
	LogExecutionStarted(workflowID, executionID string)

	// LogExecutionCompleted logs when a workflow execution completes successfully.
	LogExecutionCompleted(workflowID, executionID string, duration time.Duration)

	// LogExecutionFailed logs when a workflow execution fails.
	LogExecutionFailed(workflowID, executionID string, err error, duration time.Duration)

	// LogNodeStarted logs when a node starts executing.
	// It accepts either a domain.Node or its configuration.
	// If node is nil, LogNodeStartedFromConfig should be used instead.
	LogNodeStarted(executionID string, node *domain.Node, attemptNumber int)

	// LogNodeStartedFromConfig logs when a node starts executing from its configuration.
	// This method is used when you have the node configuration but not the full domain.Node object.
	LogNodeStartedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int)

	// LogNodeCompleted logs when a node completes successfully.
	// It accepts either a domain.Node or its configuration.
	// If node is nil, LogNodeCompletedFromConfig should be used instead.
	LogNodeCompleted(executionID string, node *domain.Node, duration time.Duration)

	// LogNodeCompletedFromConfig logs when a node completes successfully from its configuration.
	// This method is used when you have the node configuration but not the full domain.Node object.
	LogNodeCompletedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, duration time.Duration)

	// LogNodeFailed logs when a node fails.
	// It accepts either a domain.Node or its configuration.
	// If node is nil, LogNodeFailedFromConfig should be used instead.
	LogNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool)

	// LogNodeFailedFromConfig logs when a node fails from its configuration.
	// This method is used when you have the node configuration but not the full domain.Node object.
	LogNodeFailedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, err error, duration time.Duration, willRetry bool)

	// LogNodeRetrying logs when a node is being retried.
	// It accepts either a domain.Node or its configuration.
	// If node is nil, LogNodeRetryingFromConfig should be used instead.
	LogNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration)

	// LogNodeRetryingFromConfig logs when a node is being retried from its configuration.
	// This method is used when you have the node configuration but not the full domain.Node object.
	LogNodeRetryingFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int, delay time.Duration)

	// LogNodeSkipped logs when a node is skipped.
	// It accepts either a domain.Node or its configuration.
	// If node is nil, LogNodeSkippedFromConfig should be used instead.
	LogNodeSkipped(executionID string, node *domain.Node, reason string)

	// LogNodeSkippedFromConfig logs when a node is skipped from its configuration.
	// This method is used when you have the node configuration but not the full domain.Node object.
	LogNodeSkippedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, reason string)

	// LogNode logs all fields of a node.
	// It accepts either a domain.Node or its configuration.
	// If node is provided, all fields are extracted from it.
	// If node is nil, LogNodeFromConfig should be used instead.
	LogNode(executionID string, node *domain.Node)

	// LogNodeFromConfig logs all fields of a node from its configuration and metadata.
	// This method is used when you have the node configuration but not the full domain.Node object.
	LogNodeFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any)

	// LogVariableSet logs when a variable is set (verbose mode only).
	LogVariableSet(executionID, key string, value interface{})

	// LogError logs a general error.
	LogError(executionID string, message string, err error)

	// LogInfo logs an informational message.
	LogInfo(executionID string, message string)

	// LogDebug logs a debug message (verbose mode only).
	LogDebug(executionID string, message string)

	// LogTransition logs a state transition.
	LogTransition(executionID, nodeID, fromState, toState string)
}
