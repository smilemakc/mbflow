package monitoring

import (
	"time"

	"github.com/smilemakc/mbflow/internal/domain"
)

// LogObserver is an implementation of ExecutionObserver that logs events using ExecutionLogger.
// It bridges the observer pattern with the logging system by converting observer events
// to log events and passing them to the logger.
type LogObserver struct {
	logger ExecutionLogger
}

// NewLogObserver creates a new LogObserver with the given ExecutionLogger.
func NewLogObserver(logger ExecutionLogger) *LogObserver {
	return &LogObserver{
		logger: logger,
	}
}

// OnExecutionStarted is called when a workflow execution starts.
func (lo *LogObserver) OnExecutionStarted(workflowID, executionID string) {
	if lo.logger == nil {
		return
	}
	lo.logger.Log(NewExecutionStartedEvent(workflowID, executionID))
}

// OnExecutionCompleted is called when a workflow execution completes successfully.
func (lo *LogObserver) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	if lo.logger == nil {
		return
	}
	lo.logger.Log(NewExecutionCompletedEvent(workflowID, executionID, duration))
}

// OnExecutionFailed is called when a workflow execution fails.
func (lo *LogObserver) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	if lo.logger == nil {
		return
	}
	lo.logger.Log(NewExecutionFailedEvent(workflowID, executionID, err, duration))
}

// OnNodeStarted is called when a node starts executing.
func (lo *LogObserver) OnNodeStarted(executionID string, node *domain.Node, attemptNumber int) {
	if lo.logger == nil {
		return
	}
	lo.logger.Log(NewNodeStartedEvent(executionID, node, attemptNumber))
}

// OnNodeCompleted is called when a node completes successfully.
func (lo *LogObserver) OnNodeCompleted(executionID string, node *domain.Node, output interface{}, duration time.Duration) {
	if lo.logger == nil {
		return
	}

	event := NewNodeCompletedEvent(executionID, node, output, duration)
	lo.logger.Log(event)
}

// OnNodeFailed is called when a node fails.
func (lo *LogObserver) OnNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool) {
	if lo.logger == nil {
		return
	}
	lo.logger.Log(NewNodeFailedEvent(executionID, node, err, duration, willRetry))
}

// OnNodeRetrying is called when a node is being retried.
func (lo *LogObserver) OnNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration) {
	if lo.logger == nil {
		return
	}
	lo.logger.Log(NewNodeRetryingEvent(executionID, node, attemptNumber, delay))
}

// OnVariableSet is called when a variable is set in the execution context.
func (lo *LogObserver) OnVariableSet(executionID, key string, value interface{}) {
	if lo.logger == nil {
		return
	}
	lo.logger.Log(NewVariableSetEvent(executionID, key, value))
}

// OnNodeCallbackStarted is called when a node callback starts processing.
func (lo *LogObserver) OnNodeCallbackStarted(executionID string, node *domain.Node) {
	if lo.logger == nil {
		return
	}
	lo.logger.Log(NewNodeCallbackStartedEvent(executionID, node))
}

// OnNodeCallbackCompleted is called when a node callback completes.
func (lo *LogObserver) OnNodeCallbackCompleted(executionID string, node *domain.Node, err error, duration time.Duration) {
	if lo.logger == nil {
		return
	}
	lo.logger.Log(NewNodeCallbackCompletedEvent(executionID, node, err, duration))
}
