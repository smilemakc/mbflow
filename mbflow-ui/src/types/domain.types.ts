/**
 * Domain types - mirrors Go domain types from internal/domain/types.go
 */

// ============================================================================
// Edge Types
// ============================================================================

export type EdgeType = 'direct' | 'conditional' | 'fork' | 'join'

export const EdgeTypes = {
    DIRECT: 'direct' as EdgeType,
    CONDITIONAL: 'conditional' as EdgeType,
    FORK: 'fork' as EdgeType,
    JOIN: 'join' as EdgeType,
}

// ============================================================================
// Node Types
// ============================================================================

export type NodeType =
    | 'transform'
    | 'http'
    | 'llm'
    | 'code'
    | 'parallel'
    | 'conditional-router'
    | 'data-merger'
    | 'data-aggregator'
    | 'script-executor'
    | 'json-parser'
    | 'openai-completion'
    | 'openai-responses'
    | 'http-request'
    | 'telegram-message'
    | 'function-call'
    | 'function-execution'
    | 'openai-function-result'

export const NodeTypes = {
    TRANSFORM: 'transform' as NodeType,
    HTTP: 'http' as NodeType,
    LLM: 'llm' as NodeType,
    CODE: 'code' as NodeType,
    PARALLEL: 'parallel' as NodeType,
    CONDITIONAL_ROUTER: 'conditional-router' as NodeType,
    DATA_MERGER: 'data-merger' as NodeType,
    DATA_AGGREGATOR: 'data-aggregator' as NodeType,
    SCRIPT_EXECUTOR: 'script-executor' as NodeType,
    JSON_PARSER: 'json-parser' as NodeType,
    OPENAI_COMPLETION: 'openai-completion' as NodeType,
    OPENAI_RESPONSES: 'openai-responses' as NodeType,
    HTTP_REQUEST: 'http-request' as NodeType,
    TELEGRAM_MESSAGE: 'telegram-message' as NodeType,
    FUNCTION_CALL: 'function-call' as NodeType,
    FUNCTION_EXECUTION: 'function-execution' as NodeType,
    OPENAI_FUNCTION_RESULT: 'openai-function-result' as NodeType,
}

// ============================================================================
// Execution Phase
// ============================================================================

export type ExecutionPhase =
    | 'planning'
    | 'executing'
    | 'paused'
    | 'completed'
    | 'failed'
    | 'cancelled'

export const ExecutionPhases = {
    PLANNING: 'planning' as ExecutionPhase,
    EXECUTING: 'executing' as ExecutionPhase,
    PAUSED: 'paused' as ExecutionPhase,
    COMPLETED: 'completed' as ExecutionPhase,
    FAILED: 'failed' as ExecutionPhase,
    CANCELLED: 'cancelled' as ExecutionPhase,
}

export function isTerminalPhase(phase: ExecutionPhase): boolean {
    return phase === 'completed' || phase === 'failed' || phase === 'cancelled'
}

// ============================================================================
// Node Status
// ============================================================================

export type NodeStatus = 'pending' | 'running' | 'completed' | 'failed' | 'skipped'

export const NodeStatuses = {
    PENDING: 'pending' as NodeStatus,
    RUNNING: 'running' as NodeStatus,
    COMPLETED: 'completed' as NodeStatus,
    FAILED: 'failed' as NodeStatus,
    SKIPPED: 'skipped' as NodeStatus,
}

export function isTerminalStatus(status: NodeStatus): boolean {
    return status === 'completed' || status === 'failed' || status === 'skipped'
}

// ============================================================================
// Join Strategy
// ============================================================================

export type JoinStrategy = 'wait_all' | 'wait_any' | 'wait_first' | 'wait_n'

export const JoinStrategies = {
    WAIT_ALL: 'wait_all' as JoinStrategy,
    WAIT_ANY: 'wait_any' as JoinStrategy,
    WAIT_FIRST: 'wait_first' as JoinStrategy,
    WAIT_N: 'wait_n' as JoinStrategy,
}

// ============================================================================
// Error Strategy
// ============================================================================

export type ErrorStrategy =
    | 'fail_fast'
    | 'continue_on_error'
    | 'require_n'
    | 'best_effort'

export const ErrorStrategies = {
    FAIL_FAST: 'fail_fast' as ErrorStrategy,
    CONTINUE_ON_ERROR: 'continue_on_error' as ErrorStrategy,
    REQUIRE_N: 'require_n' as ErrorStrategy,
    BEST_EFFORT: 'best_effort' as ErrorStrategy,
}

// ============================================================================
// Trigger Type
// ============================================================================

export type TriggerType = 'manual' | 'auto' | 'http' | 'schedule' | 'event'

export const TriggerTypes = {
    MANUAL: 'manual' as TriggerType,
    AUTO: 'auto' as TriggerType,
    HTTP: 'http' as TriggerType,
    SCHEDULE: 'schedule' as TriggerType,
    EVENT: 'event' as TriggerType,
}

// ============================================================================
// Variable Type
// ============================================================================

export type VariableType =
    | 'string'
    | 'int'
    | 'float'
    | 'bool'
    | 'object'
    | 'array'
    | 'any'
    | 'unknown'

export const VariableTypes = {
    STRING: 'string' as VariableType,
    INT: 'int' as VariableType,
    FLOAT: 'float' as VariableType,
    BOOL: 'bool' as VariableType,
    OBJECT: 'object' as VariableType,
    ARRAY: 'array' as VariableType,
    ANY: 'any' as VariableType,
    UNKNOWN: 'unknown' as VariableType,
}

export function inferType(value: unknown): VariableType {
    if (value === null || value === undefined) {
        return 'unknown'
    }

    if (typeof value === 'string') return 'string'
    if (typeof value === 'number') {
        return Number.isInteger(value) ? 'int' : 'float'
    }
    if (typeof value === 'boolean') return 'bool'
    if (Array.isArray(value)) return 'array'
    if (typeof value === 'object') return 'object'

    return 'any'
}

// ============================================================================
// Error Codes
// ============================================================================

export const ErrorCodes = {
    INVALID_INPUT: 'INVALID_INPUT',
    VALIDATION_FAILED: 'VALIDATION_FAILED',
    NOT_FOUND: 'NOT_FOUND',
    ALREADY_EXISTS: 'ALREADY_EXISTS',
    INVARIANT_VIOLATED: 'INVARIANT_VIOLATED',
    INVALID_STATE: 'INVALID_STATE',
    CYCLIC_DEPENDENCY: 'CYCLIC_DEPENDENCY',
    INVALID_TYPE: 'INVALID_TYPE',
} as const

export type ErrorCode = (typeof ErrorCodes)[keyof typeof ErrorCodes]

// ============================================================================
// Domain Error
// ============================================================================

export interface DomainError {
    code: ErrorCode
    message: string
    details?: string
}
