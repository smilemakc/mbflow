/**
 * Execution types - mirrors execution domain models and API DTOs
 */

import type { ExecutionPhase, NodeStatus } from './domain.types'

// ============================================================================
// Execution
// ============================================================================

export interface Execution {
    id: string
    workflow_id: string
    workflow_name?: string
    phase: ExecutionPhase
    started_at?: string
    completed_at?: string
    duration_ms?: number
    node_states: Record<string, NodeState>
    variables: Record<string, unknown>
    error?: string
    current_node_id?: string
    sequence_number: number
}

export interface NodeState {
    node_id: string
    node_name: string
    status: NodeStatus
    started_at?: string
    completed_at?: string
    duration_ms?: number
    output?: Record<string, unknown>
    error?: string
    retry_count: number
}

export interface ExecuteWorkflowRequest {
    workflow_id: string
    trigger_id?: string
    variables?: Record<string, unknown>
}

export interface ExecutionResponse extends Execution { }

export interface NodeStateResponse extends NodeState { }

// ============================================================================
// Execution Events
// ============================================================================

export type ExecutionEventType =
    | 'execution.started'
    | 'execution.completed'
    | 'execution.failed'
    | 'execution.cancelled'
    | 'node.started'
    | 'node.completed'
    | 'node.failed'
    | 'node.skipped'
    | 'variable.set'
    | 'variable.updated'

export interface ExecutionEvent {
    id: string
    event_type: ExecutionEventType
    execution_id: string
    workflow_id?: string
    sequence?: number
    timestamp: string
    data?: Record<string, unknown>
}

export interface ExecutionEventsResponse {
    execution_id: string
    events: ExecutionEvent[]
}

// ============================================================================
// Execution Filters
// ============================================================================

export interface ExecutionFilters {
    workflow_id?: string
    phase?: ExecutionPhase
    limit?: number
    offset?: number
}

// ============================================================================
// Execution Statistics
// ============================================================================

export interface ExecutionStatistics {
    total_executions: number
    completed: number
    failed: number
    running: number
    average_duration_ms: number
}
