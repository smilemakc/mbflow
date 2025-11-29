/**
 * Workflow types - mirrors API DTOs and domain models
 */

import type {
    NodeType,
    EdgeType,
    TriggerType,
    NodeStatus,
    ExecutionPhase,
    JoinStrategy,
    ErrorStrategy,
} from './domain.types'

// ============================================================================
// Node Configuration Types
// ============================================================================

export interface BaseNodeConfig {
    [key: string]: unknown
}

export interface TransformConfig extends BaseNodeConfig {
    transformations: Record<string, string> // key -> expression
}

export interface HttpConfig extends BaseNodeConfig {
    url: string
    method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
    headers?: Record<string, string>
    body?: unknown
    timeout?: number
}

export interface ConditionalRouteConfig extends BaseNodeConfig {
    routes: Array<{
        condition: string
        target: string
    }>
    default?: string
}

export interface ParallelConfig extends BaseNodeConfig {
    join_strategy?: JoinStrategy
    wait_count?: number
    error_strategy?: ErrorStrategy
}

export interface OpenAICompletionConfig extends BaseNodeConfig {
    model: string
    prompt: string
    temperature?: number
    max_tokens?: number
    api_key?: string
}

export interface JSONParserConfig extends BaseNodeConfig {
    input_field: string
    output_field?: string
}

export interface ScriptExecutorConfig extends BaseNodeConfig {
    language: 'javascript' | 'python'
    script: string
}

export type NodeConfig =
    | TransformConfig
    | HttpConfig
    | ConditionalRouteConfig
    | ParallelConfig
    | OpenAICompletionConfig
    | JSONParserConfig
    | ScriptExecutorConfig
    | BaseNodeConfig

// ============================================================================
// Node
// ============================================================================

export interface Node {
    id: string
    type: NodeType
    name: string
    config?: NodeConfig
    metadata?: Record<string, unknown>
}

export interface NodeRequest {
    type: NodeType
    name: string
    config?: NodeConfig
}

export interface NodeResponse extends Node { }

// ============================================================================
// Edge
// ============================================================================

export interface Edge {
    id: string
    from: string // node name
    to: string // node name
    type: EdgeType
    condition?: {
        expression: string
    }
    config?: {
        include_outputs_from?: string[]
        [key: string]: unknown
    }
}

export interface EdgeRequest {
    from: string
    to: string
    type: EdgeType
    condition?: {
        expression: string
    }
    config?: {
        include_outputs_from?: string[]
        [key: string]: unknown
    }
}

export interface EdgeResponse extends Edge { }

// ============================================================================
// Trigger
// ============================================================================

export interface Trigger {
    id: string
    type: TriggerType
    config?: Record<string, unknown>
}

export interface TriggerRequest {
    type: TriggerType
    config?: Record<string, unknown>
}

export interface TriggerResponse extends Trigger { }

// ============================================================================
// Workflow
// ============================================================================

export interface Workflow {
    id: string
    name: string
    version: string
    description?: string
    nodes: Node[]
    edges: Edge[]
    triggers: Trigger[]
    metadata?: Record<string, unknown>
    created_at?: string
    updated_at?: string
}

export interface CreateWorkflowRequest {
    name: string
    version: string
    description?: string
    nodes: NodeRequest[]
    edges: EdgeRequest[]
    triggers: TriggerRequest[]
    metadata?: Record<string, unknown>
}

export interface UpdateWorkflowRequest {
    name?: string
    version?: string
    description?: string
    nodes?: NodeRequest[]
    edges?: EdgeRequest[]
    triggers?: TriggerRequest[]
    metadata?: Record<string, unknown>
}

export interface WorkflowResponse extends Workflow { }

// ============================================================================
// Workflow Graph (for visualization)
// ============================================================================

export interface WorkflowGraph {
    nodes: NodeResponse[]
    edges: EdgeResponse[]
}

// ============================================================================
// Node Type Metadata (for UI)
// ============================================================================

export interface NodeTypeMetadata {
    type: NodeType
    label: string
    description: string
    icon: string
    category: 'control' | 'transform' | 'integration' | 'ai' | 'data'
    configSchema?: Record<string, unknown> // JSON Schema
    outputSchema?: Record<string, unknown> // JSON Schema
}

export interface EdgeTypeMetadata {
    type: EdgeType
    label: string
    description: string
    requiresCondition: boolean
}
