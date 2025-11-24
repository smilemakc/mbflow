package mbflow

import (
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// ========== Type Exports ==========

// Core domain types
type (
	NodeType       = domain.NodeType
	EdgeType       = domain.EdgeType
	TriggerType    = domain.TriggerType
	EventType      = domain.EventType
	JoinStrategy   = domain.JoinStrategy
	ExecutionPhase = domain.ExecutionPhase
	NodeStatus     = domain.NodeStatus
)

// Node type constants
const (
	NodeTypeStart                = domain.NodeTypeStart
	NodeTypeEnd                  = domain.NodeTypeEnd
	NodeTypeTransform            = domain.NodeTypeTransform
	NodeTypeHTTP                 = domain.NodeTypeHTTP
	NodeTypeLLM                  = domain.NodeTypeLLM
	NodeTypeCode                 = domain.NodeTypeCode
	NodeTypeParallel             = domain.NodeTypeParallel
	NodeTypeConditionalRoute     = domain.NodeTypeConditionalRoute
	NodeTypeDataMerger           = domain.NodeTypeDataMerger
	NodeTypeDataAggregator       = domain.NodeTypeDataAggregator
	NodeTypeScriptExecutor       = domain.NodeTypeScriptExecutor
	NodeTypeJSONParser           = domain.NodeTypeJSONParser
	NodeTypeOpenAICompletion     = domain.NodeTypeOpenAICompletion
	NodeTypeOpenAIResponses      = domain.NodeTypeOpenAIResponses
	NodeTypeHTTPRequest          = domain.NodeTypeHTTPRequest
	NodeTypeTelegramMessage      = domain.NodeTypeTelegramMessage
	NodeTypeFunctionCall         = domain.NodeTypeFunctionCall
	NodeTypeFunctionExecution    = domain.NodeTypeFunctionExecution
	NodeTypeOpenAIFunctionResult = domain.NodeTypeOpenAIFunctionResult
)

// Edge type constants
const (
	EdgeTypeDirect      = domain.EdgeTypeDirect
	EdgeTypeConditional = domain.EdgeTypeConditional
	EdgeTypeFork        = domain.EdgeTypeFork
	EdgeTypeJoin        = domain.EdgeTypeJoin
)

// Trigger type constants
const (
	TriggerTypeManual   = domain.TriggerTypeManual
	TriggerTypeAuto     = domain.TriggerTypeAuto
	TriggerTypeHTTP     = domain.TriggerTypeHTTP
	TriggerTypeSchedule = domain.TriggerTypeSchedule
	TriggerTypeEvent    = domain.TriggerTypeEvent
)

// Error strategy constants
const (
	ErrorStrategyFailFast        = domain.ErrorStrategyFailFast
	ErrorStrategyContinueOnError = domain.ErrorStrategyContinueOnError
	ErrorStrategyBestEffort      = domain.ErrorStrategyBestEffort
	ErrorStrategyRequireN        = domain.ErrorStrategyRequireN
)

// Join strategy constants
const (
	JoinStrategyWaitAll   = domain.JoinStrategyWaitAll
	JoinStrategyWaitAny   = domain.JoinStrategyWaitAny
	JoinStrategyWaitFirst = domain.JoinStrategyWaitFirst
	JoinStrategyWaitN     = domain.JoinStrategyWaitN
)

// Execution phase constants
const (
	ExecutionPhasePlanning  = domain.ExecutionPhasePlanning
	ExecutionPhaseExecuting = domain.ExecutionPhaseExecuting
	ExecutionPhasePaused    = domain.ExecutionPhasePaused
	ExecutionPhaseCompleted = domain.ExecutionPhaseCompleted
	ExecutionPhaseFailed    = domain.ExecutionPhaseFailed
	ExecutionPhaseCancelled = domain.ExecutionPhaseCancelled
)

// Event type constants
const (
	EventTypeExecutionStarted   = domain.EventTypeExecutionStarted
	EventTypeExecutionCompleted = domain.EventTypeExecutionCompleted
	EventTypeExecutionFailed    = domain.EventTypeExecutionFailed
	EventTypeExecutionPaused    = domain.EventTypeExecutionPaused
	EventTypeExecutionResumed   = domain.EventTypeExecutionResumed
	EventTypeExecutionCancelled = domain.EventTypeExecutionCancelled
	EventTypeNodeStarted        = domain.EventTypeNodeStarted
	EventTypeNodeCompleted      = domain.EventTypeNodeCompleted
	EventTypeNodeFailed         = domain.EventTypeNodeFailed
	EventTypeNodeSkipped        = domain.EventTypeNodeSkipped
	EventTypeNodeRetrying       = domain.EventTypeNodeRetrying
	EventTypeVariableSet        = domain.EventTypeVariableSet
	EventTypeVariableUpdated    = domain.EventTypeVariableUpdated
	EventTypeVariableDeleted    = domain.EventTypeVariableDeleted
)

type Workflow = domain.Workflow

type Execution = domain.Execution

type Trigger = domain.Trigger

type Node = domain.Node

type Edge = domain.Edge

// ========== Domain Interfaces ==========

// Event represents a domain event in the event sourcing system
type Event = domain.Event

// VariableSet represents a set of variables with optional schema validation
type VariableSet = domain.VariableSet

// ========== Repository Interfaces ==========

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository = domain.WorkflowRepository

// ExecutionRepository defines the interface for execution persistence (event sourcing based)
type ExecutionRepository = domain.ExecutionRepository

// EventStore defines the interface for event sourcing persistence
type EventStore = domain.EventStore

// Storage combines all repository interfaces
type Storage = domain.Storage

// ExecutorState represents the state of a workflow execution.
type ExecutorState interface {
	// GetExecutionID returns the execution ID
	GetExecutionID() uuid.UUID

	// GetWorkflowID returns the workflow ID
	GetWorkflowID() uuid.UUID

	// GetStatus returns the current status as string
	GetStatusString() string

	// GetVariable retrieves a variable
	GetVariable(key string) (interface{}, bool)

	// GetAllVariables returns all variables
	GetAllVariables() map[string]interface{}

	// GetExecutionDuration returns the execution duration
	GetExecutionDuration() time.Duration
}

// ExecutorMetrics provides execution metrics.
type ExecutorMetrics interface {
	// GetWorkflowMetrics returns metrics for a workflow
	GetWorkflowMetrics(workflowID uuid.UUID) *WorkflowMetrics

	// GetNodeMetrics returns aggregated metrics for a node type
	GetNodeMetrics(nodeType string) *NodeMetrics

	// GetNodeMetricsByID returns metrics for a specific node ID
	GetNodeMetricsByID(nodeID uuid.UUID) *NodeMetrics

	// GetAIMetrics returns AI API usage metrics
	GetAIMetrics() *AIMetrics

	// GetSummary returns a summary of all metrics
	GetSummary() *MetricsSummary
}

// WorkflowMetrics represents metrics for a workflow.
type WorkflowMetrics = monitoring.WorkflowMetrics

// NodeMetrics represents metrics for a node type.
type NodeMetrics = monitoring.NodeMetrics

// AIMetrics represents AI API usage metrics.
type AIMetrics = monitoring.AIMetrics

// MetricsSummary represents a summary of all collected metrics.
type MetricsSummary = monitoring.MetricsSummary
