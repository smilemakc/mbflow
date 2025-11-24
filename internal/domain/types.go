package domain

import (
	"fmt"
)

// EdgeType defines the type of connection between nodes
type EdgeType string

const (
	// EdgeTypeDirect represents a simple directed edge from one node to another
	EdgeTypeDirect EdgeType = "direct"

	// EdgeTypeConditional represents an edge that is traversed only if a condition is met
	EdgeTypeConditional EdgeType = "conditional"

	// EdgeTypeFork represents an edge that splits execution into parallel branches
	EdgeTypeFork EdgeType = "fork"

	// EdgeTypeJoin represents an edge that waits for multiple parallel branches
	EdgeTypeJoin EdgeType = "join"
)

// IsValid checks if the EdgeType is valid
func (et EdgeType) IsValid() bool {
	switch et {
	case EdgeTypeDirect, EdgeTypeConditional, EdgeTypeFork, EdgeTypeJoin:
		return true
	default:
		return false
	}
}

// String returns string representation of EdgeType
func (et EdgeType) String() string {
	return string(et)
}

// NodeType defines the type of operation a node performs
type NodeType string

const (
	NodeTypeStart                NodeType = "start"
	NodeTypeEnd                  NodeType = "end"
	NodeTypeTransform            NodeType = "transform"
	NodeTypeHTTP                 NodeType = "http"
	NodeTypeLLM                  NodeType = "llm"
	NodeTypeCode                 NodeType = "code"
	NodeTypeParallel             NodeType = "parallel"
	NodeTypeConditionalRoute     NodeType = "conditional-router"
	NodeTypeDataMerger           NodeType = "data-merger"
	NodeTypeDataAggregator       NodeType = "data-aggregator"
	NodeTypeScriptExecutor       NodeType = "script-executor"
	NodeTypeJSONParser           NodeType = "json-parser"
	NodeTypeOpenAICompletion     NodeType = "openai-completion"
	NodeTypeOpenAIResponses      NodeType = "openai-responses"
	NodeTypeHTTPRequest          NodeType = "http-request"
	NodeTypeTelegramMessage      NodeType = "telegram-message"
	NodeTypeFunctionCall         NodeType = "function-call"
	NodeTypeFunctionExecution    NodeType = "function-execution"
	NodeTypeOpenAIFunctionResult NodeType = "openai-function-result"
)

// IsValid checks if the NodeType is valid
func (nt NodeType) IsValid() bool {
	switch nt {
	case NodeTypeStart, NodeTypeEnd, NodeTypeTransform, NodeTypeHTTP, NodeTypeLLM,
		NodeTypeCode, NodeTypeParallel, NodeTypeConditionalRoute, NodeTypeDataMerger,
		NodeTypeDataAggregator, NodeTypeScriptExecutor, NodeTypeJSONParser,
		NodeTypeOpenAICompletion, NodeTypeOpenAIResponses, NodeTypeHTTPRequest,
		NodeTypeTelegramMessage, NodeTypeFunctionCall, NodeTypeFunctionExecution,
		NodeTypeOpenAIFunctionResult:
		return true
	default:
		return false
	}
}

// String returns string representation of NodeType
func (nt NodeType) String() string {
	return string(nt)
}

// ExecutionPhase defines the phase of workflow execution
type ExecutionPhase string

const (
	// ExecutionPhasePlanning represents the planning phase before execution
	ExecutionPhasePlanning ExecutionPhase = "planning"

	// ExecutionPhaseExecuting represents the active execution phase
	ExecutionPhaseExecuting ExecutionPhase = "executing"

	// ExecutionPhasePaused represents a paused execution
	ExecutionPhasePaused ExecutionPhase = "paused"

	// ExecutionPhaseCompleted represents successfully completed execution
	ExecutionPhaseCompleted ExecutionPhase = "completed"

	// ExecutionPhaseFailed represents failed execution
	ExecutionPhaseFailed ExecutionPhase = "failed"

	// ExecutionPhaseCancelled represents cancelled execution
	ExecutionPhaseCancelled ExecutionPhase = "cancelled"
)

// IsValid checks if the ExecutionPhase is valid
func (ep ExecutionPhase) IsValid() bool {
	switch ep {
	case ExecutionPhasePlanning, ExecutionPhaseExecuting, ExecutionPhasePaused,
		ExecutionPhaseCompleted, ExecutionPhaseFailed:
		return true
	default:
		return false
	}
}

// String returns string representation of ExecutionPhase
func (ep ExecutionPhase) String() string {
	return string(ep)
}

// IsTerminal returns true if this phase is terminal (no further transitions)
func (ep ExecutionPhase) IsTerminal() bool {
	return ep == ExecutionPhaseCompleted || ep == ExecutionPhaseFailed
}

// NodeStatus defines the status of a node during execution
type NodeStatus string

const (
	// NodeStatusPending means the node is waiting to be executed
	NodeStatusPending NodeStatus = "pending"

	// NodeStatusRunning means the node is currently executing
	NodeStatusRunning NodeStatus = "running"

	// NodeStatusCompleted means the node completed successfully
	NodeStatusCompleted NodeStatus = "completed"

	// NodeStatusFailed means the node execution failed
	NodeStatusFailed NodeStatus = "failed"

	// NodeStatusSkipped means the node was skipped (e.g., conditional edge evaluated to false)
	NodeStatusSkipped NodeStatus = "skipped"
)

// IsValid checks if the NodeStatus is valid
func (ns NodeStatus) IsValid() bool {
	switch ns {
	case NodeStatusPending, NodeStatusRunning, NodeStatusCompleted,
		NodeStatusFailed, NodeStatusSkipped:
		return true
	default:
		return false
	}
}

// String returns string representation of NodeStatus
func (ns NodeStatus) String() string {
	return string(ns)
}

// IsTerminal returns true if this status is terminal (no further transitions)
func (ns NodeStatus) IsTerminal() bool {
	return ns == NodeStatusCompleted || ns == NodeStatusFailed || ns == NodeStatusSkipped
}

// JoinStrategy defines how a join node waits for incoming branches
type JoinStrategy string

const (
	// JoinStrategyWaitAll waits for all incoming branches to complete
	JoinStrategyWaitAll JoinStrategy = "wait_all"

	// JoinStrategyWaitAny waits for any one incoming branch to complete
	JoinStrategyWaitAny JoinStrategy = "wait_any"

	// JoinStrategyWaitFirst waits for the first incoming branch to complete
	JoinStrategyWaitFirst JoinStrategy = "wait_first"

	// JoinStrategyWaitN waits for N incoming branches to complete
	JoinStrategyWaitN JoinStrategy = "wait_n"
)

// IsValid checks if the JoinStrategy is valid
func (js JoinStrategy) IsValid() bool {
	switch js {
	case JoinStrategyWaitAll, JoinStrategyWaitAny, JoinStrategyWaitFirst, JoinStrategyWaitN:
		return true
	default:
		return false
	}
}

// String returns string representation of JoinStrategy
func (js JoinStrategy) String() string {
	return string(js)
}

// ErrorStrategy defines how errors are handled during workflow execution
type ErrorStrategy string

const (
	// ErrorStrategyFailFast stops execution on the first error
	ErrorStrategyFailFast ErrorStrategy = "fail_fast"

	// ErrorStrategyContinueOnError continues execution and collects errors
	ErrorStrategyContinueOnError ErrorStrategy = "continue_on_error"

	// ErrorStrategyRequireN requires at least N successful executions
	ErrorStrategyRequireN ErrorStrategy = "require_n"

	// ErrorStrategyBestEffort continues and uses partial results
	ErrorStrategyBestEffort ErrorStrategy = "best_effort"
)

// IsValid checks if the ErrorStrategy is valid
func (es ErrorStrategy) IsValid() bool {
	switch es {
	case ErrorStrategyFailFast, ErrorStrategyContinueOnError,
		ErrorStrategyRequireN, ErrorStrategyBestEffort:
		return true
	default:
		return false
	}
}

// String returns string representation of ErrorStrategy
func (es ErrorStrategy) String() string {
	return string(es)
}

// TriggerType defines the type of trigger for workflow execution
type TriggerType string

const (
	// TriggerTypeManual represents manual trigger (started by user)
	TriggerTypeManual TriggerType = "manual"

	// TriggerTypeAuto represents automatic trigger (starts immediately)
	TriggerTypeAuto TriggerType = "auto"

	// TriggerTypeHTTP represents HTTP webhook trigger
	TriggerTypeHTTP TriggerType = "http"

	// TriggerTypeSchedule represents scheduled trigger (cron-like)
	TriggerTypeSchedule TriggerType = "schedule"

	// TriggerTypeEvent represents event-based trigger
	TriggerTypeEvent TriggerType = "event"
)

// IsValid checks if the TriggerType is valid
func (tt TriggerType) IsValid() bool {
	switch tt {
	case TriggerTypeManual, TriggerTypeAuto, TriggerTypeHTTP,
		TriggerTypeSchedule, TriggerTypeEvent:
		return true
	default:
		return false
	}
}

// String returns string representation of TriggerType
func (tt TriggerType) String() string {
	return string(tt)
}

// VariableType defines the type of a variable
type VariableType string

const (
	VariableTypeString  VariableType = "string"
	VariableTypeInt     VariableType = "int"
	VariableTypeFloat   VariableType = "float"
	VariableTypeBool    VariableType = "bool"
	VariableTypeObject  VariableType = "object"
	VariableTypeArray   VariableType = "array"
	VariableTypeAny     VariableType = "any"
	VariableTypeUnknown VariableType = "unknown"
)

// IsValid checks if the VariableType is valid
func (vt VariableType) IsValid() bool {
	switch vt {
	case VariableTypeString, VariableTypeInt, VariableTypeFloat, VariableTypeBool,
		VariableTypeObject, VariableTypeArray, VariableTypeAny, VariableTypeUnknown:
		return true
	default:
		return false
	}
}

// String returns string representation of VariableType
func (vt VariableType) String() string {
	return string(vt)
}

// InferType infers the VariableType from a Go value
func InferType(v interface{}) VariableType {
	if v == nil {
		return VariableTypeUnknown
	}

	switch v.(type) {
	case string:
		return VariableTypeString
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return VariableTypeInt
	case float32, float64:
		return VariableTypeFloat
	case bool:
		return VariableTypeBool
	case map[string]interface{}:
		return VariableTypeObject
	case []interface{}:
		return VariableTypeArray
	default:
		return VariableTypeAny
	}
}

// DomainError represents a domain-specific error
type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// Common domain error codes
const (
	ErrCodeInvalidInput      = "INVALID_INPUT"
	ErrCodeValidationFailed  = "VALIDATION_FAILED"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeAlreadyExists     = "ALREADY_EXISTS"
	ErrCodeInvariantViolated = "INVARIANT_VIOLATED"
	ErrCodeInvalidState      = "INVALID_STATE"
	ErrCodeCyclicDependency  = "CYCLIC_DEPENDENCY"
	ErrCodeInvalidType       = "INVALID_TYPE"
)

// NewDomainError creates a new domain error
func NewDomainError(code, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
