/**
 * Mock data for development
 */

import type {
    Workflow,
    Node,
    Edge,
    Trigger,
    Execution,
    NodeState,
    ExecutionEvent,
    NodeTypeMetadata,
    EdgeTypeMetadata,
} from '@/types'
import { NodeTypes, EdgeTypes, NodeStatuses, ExecutionPhases } from '@/types'

// ============================================================================
// Mock Workflows
// ============================================================================

export const mockWorkflows: Workflow[] = [
    {
        id: 'wf-1',
        name: 'Simple Transform Workflow',
        version: '1.0.0',
        description: 'A simple workflow that transforms data',
        nodes: [
            {
                id: 'node-1',
                type: NodeTypes.TRANSFORM,
                name: 'double',
                config: {
                    transformations: {
                        result: 'input * 2',
                    },
                },
                metadata: {
                    position: { x: 100, y: 200 },
                },
            },
            {
                id: 'node-2',
                type: NodeTypes.TRANSFORM,
                name: 'square',
                config: {
                    transformations: {
                        final: 'double.result * double.result',
                    },
                },
                metadata: {
                    position: { x: 400, y: 200 },
                },
            },
        ],
        edges: [
            {
                id: 'edge-1',
                from: 'double',
                to: 'square',
                type: EdgeTypes.DIRECT,
            },
        ],
        triggers: [
            {
                id: 'trigger-1',
                type: 'manual',
            },
        ],
        created_at: new Date().toISOString(),
    },
    {
        id: 'wf-2',
        name: 'HTTP API Workflow',
        version: '1.0.0',
        description: 'Fetch data from API and process it',
        nodes: [
            {
                id: 'node-1',
                type: NodeTypes.HTTP,
                name: 'fetch_users',
                config: {
                    url: 'https://jsonplaceholder.typicode.com/users',
                    method: 'GET',
                },
                metadata: {
                    position: { x: 100, y: 250 },
                },
            },
            {
                id: 'node-2',
                type: NodeTypes.JSON_PARSER,
                name: 'parse_response',
                config: {
                    input_field: 'fetch_users.body',
                },
                metadata: {
                    position: { x: 400, y: 250 },
                },
            },
        ],
        edges: [
            {
                id: 'edge-1',
                from: 'fetch_users',
                to: 'parse_response',
                type: EdgeTypes.DIRECT,
            },
        ],
        triggers: [
            {
                id: 'trigger-1',
                type: 'manual',
            },
        ],
        created_at: new Date().toISOString(),
    },
]

// ============================================================================
// Mock Executions
// ============================================================================

export const mockExecutions: Execution[] = [
    {
        id: 'exec-1',
        workflow_id: 'wf-1',
        workflow_name: 'Simple Transform Workflow',
        phase: ExecutionPhases.COMPLETED,
        started_at: new Date(Date.now() - 60000).toISOString(),
        completed_at: new Date().toISOString(),
        duration_ms: 1250,
        sequence_number: 1,
        node_states: {
            'node-1': {
                node_id: 'node-1',
                node_name: 'double',
                status: NodeStatuses.COMPLETED,
                started_at: new Date(Date.now() - 60000).toISOString(),
                completed_at: new Date(Date.now() - 59000).toISOString(),
                duration_ms: 500,
                output: {
                    result: 84,
                },
                retry_count: 0,
            },
            'node-2': {
                node_id: 'node-2',
                node_name: 'square',
                status: NodeStatuses.COMPLETED,
                started_at: new Date(Date.now() - 59000).toISOString(),
                completed_at: new Date().toISOString(),
                duration_ms: 750,
                output: {
                    final: 7056,
                },
                retry_count: 0,
            },
        },
        variables: {
            input: 42,
            result: 84,
        },
    },
]

// ============================================================================
// Mock Events
// ============================================================================

export const mockEvents: ExecutionEvent[] = [
    {
        id: 'event-1',
        event_type: 'execution.started',
        execution_id: 'exec-1',
        workflow_id: 'wf-1',
        sequence: 1,
        timestamp: new Date(Date.now() - 60000).toISOString(),
        data: {
            initial_variables: { input: 42 },
        },
    },
    {
        id: 'event-2',
        event_type: 'node.started',
        execution_id: 'exec-1',
        workflow_id: 'wf-1',
        sequence: 2,
        timestamp: new Date(Date.now() - 60000).toISOString(),
        data: {
            node_id: 'node-1',
            node_name: 'double',
        },
    },
    {
        id: 'event-3',
        event_type: 'node.completed',
        execution_id: 'exec-1',
        workflow_id: 'wf-1',
        sequence: 3,
        timestamp: new Date(Date.now() - 59000).toISOString(),
        data: {
            node_id: 'node-1',
            node_name: 'double',
            output: { result: 84 },
        },
    },
    {
        id: 'event-4',
        event_type: 'node.started',
        execution_id: 'exec-1',
        workflow_id: 'wf-1',
        sequence: 4,
        timestamp: new Date(Date.now() - 59000).toISOString(),
        data: {
            node_id: 'node-2',
            node_name: 'square',
        },
    },
    {
        id: 'event-5',
        event_type: 'node.completed',
        execution_id: 'exec-1',
        workflow_id: 'wf-1',
        sequence: 5,
        timestamp: new Date().toISOString(),
        data: {
            node_id: 'node-2',
            node_name: 'square',
            output: { final: 7056 },
        },
    },
    {
        id: 'event-6',
        event_type: 'execution.completed',
        execution_id: 'exec-1',
        workflow_id: 'wf-1',
        sequence: 6,
        timestamp: new Date().toISOString(),
    },
]

// ============================================================================
// Node Type Metadata
// ============================================================================

export const mockNodeTypes: NodeTypeMetadata[] = [
    {
        type: NodeTypes.TRANSFORM,
        label: 'Transform',
        description: 'Transform data using expressions',
        icon: 'mdi-function-variant',
        category: 'transform',
    },
    {
        type: NodeTypes.HTTP,
        label: 'HTTP Request',
        description: 'Make HTTP requests',
        icon: 'mdi-web',
        category: 'integration',
    },
    {
        type: NodeTypes.OPENAI_COMPLETION,
        label: 'OpenAI Completion',
        description: 'Generate text using OpenAI',
        icon: 'mdi-robot',
        category: 'ai',
    },
    {
        type: NodeTypes.JSON_PARSER,
        label: 'JSON Parser',
        description: 'Parse JSON data',
        icon: 'mdi-code-json',
        category: 'data',
    },
    {
        type: NodeTypes.CONDITIONAL_ROUTER,
        label: 'Conditional Router',
        description: 'Route based on conditions',
        icon: 'mdi-source-branch',
        category: 'control',
    },
    {
        type: NodeTypes.PARALLEL,
        label: 'Parallel',
        description: 'Execute branches in parallel',
        icon: 'mdi-source-fork',
        category: 'control',
    },
]

// ============================================================================
// Edge Type Metadata
// ============================================================================

export const mockEdgeTypes: EdgeTypeMetadata[] = [
    {
        type: EdgeTypes.DIRECT,
        label: 'Direct',
        description: 'Simple sequential connection',
        requiresCondition: false,
    },
    {
        type: EdgeTypes.CONDITIONAL,
        label: 'Conditional',
        description: 'Connection with condition',
        requiresCondition: true,
    },
    {
        type: EdgeTypes.FORK,
        label: 'Fork',
        description: 'Start parallel branches',
        requiresCondition: false,
    },
    {
        type: EdgeTypes.JOIN,
        label: 'Join',
        description: 'Synchronize parallel branches',
        requiresCondition: false,
    },
]
