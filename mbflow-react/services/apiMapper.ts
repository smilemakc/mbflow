/**
 * API Mapper - Converts between frontend and backend data formats
 * 
 * Backend uses: from/to for edges, uuid IDs, flat node structure
 * Frontend uses: source/target for edges, ReactFlow node format
 */

import { AppNode, AppEdge, NodeType, NodeStatus, VariableType, VariableSource, isValidNodeType } from '@/types';
import { MarkerType } from 'reactflow';

// ============ Type Definitions for API Responses ============

export interface WorkflowApiResponse {
    id: string;
    name: string;
    description: string;
    status: string;
    version: number;
    variables: Record<string, any>;
    metadata: Record<string, any>;
    nodes: NodeApiResponse[];
    edges: EdgeApiResponse[];
    created_at: string;
    updated_at: string;
}

export interface NodeApiResponse {
    id: string;
    node_id: string;  // logical ID used in edges
    name: string;
    type: string;
    config: Record<string, any>;
    position: { x: number; y: number };
}

export interface EdgeApiResponse {
    id: string;
    edge_id: string;  // logical ID
    from: string;     // source node_id
    to: string;       // target node_id
    condition: Record<string, any>;
}

export interface ExecutionApiResponse {
    id: string;
    workflow_id: string;
    status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
    started_at: string;
    completed_at?: string;
    input: Record<string, any>;
    output: Record<string, any>;
    error?: string;
    node_executions: NodeExecutionApiResponse[];
}

export interface NodeExecutionApiResponse {
    id: string;
    node_id: string;
    status: string;
    started_at: string;
    completed_at?: string;
    input: Record<string, any>;
    output: Record<string, any>;
    error?: string;
}

// ============ Node Type Mapping ============

/**
 * Legacy type mappings - only needed for types where NodeType enum key differs from backend value.
 * Modern types (transform, html_clean, etc.) use identical values and don't need mapping.
 *
 * NodeType enum values ARE the backend type strings:
 *   NodeType.API_REQUEST = 'http'      (legacy - needs mapping)
 *   NodeType.HTML_CLEAN = 'html_clean' (modern - no mapping needed)
 */

// Backend -> Frontend: Only legacy types that need remapping
// 'webhook' and 'split' are aliases handled here
const LEGACY_BACKEND_ALIASES: Record<string, NodeType> = {
    'webhook': NodeType.HTTP,  // webhook -> http node
    'split': NodeType.CONDITIONAL,      // split -> conditional node
};

// Frontend -> Backend: No mapping needed!
// NodeType enum values ARE already the backend type strings.
// This function is kept for documentation but always returns the input.

// ============ Workflow Mappers ============

export function workflowFromApi(api: WorkflowApiResponse) {
    // Convert variables from Record to Variable[]
    const variables = api.variables
        ? Object.entries(api.variables).map(([key, value], index) => ({
            id: `var_${index}`,
            key,
            name: key,
            type: typeof value === 'string' ? VariableType.STRING
                : typeof value === 'number' ? VariableType.NUMBER
                : typeof value === 'boolean' ? VariableType.BOOLEAN
                : VariableType.OBJECT,
            source: VariableSource.GLOBAL,
            value,
        }))
        : [];

    return {
        id: api.id,
        name: api.name,
        description: api.description || '',
        version: api.version || 1,
        status: (api.status as 'draft' | 'active' | 'inactive' | 'archived') || 'draft',
        nodes: api.nodes?.map(nodeFromApi) || [],
        edges: api.edges?.map(edgeFromApi) || [],
        variables,
        tags: [],
        metadata: api.metadata || {},
        createdAt: api.created_at,
        updatedAt: api.updated_at,
        ownerId: '',
    };
}

export function workflowToApi(workflow: {
    id?: string;
    name: string;
    description?: string;
    nodes: AppNode[];
    edges: AppEdge[];
    variables?: Record<string, string>;
}) {
    return {
        id: workflow.id,
        name: workflow.name,
        description: workflow.description || '',
        nodes: workflow.nodes.map(nodeToApi),
        edges: workflow.edges.map(edgeToApi),
        variables: workflow.variables || {},
    };
}

// ============ Node Mappers ============

export function nodeFromApi(api: NodeApiResponse): AppNode {
    // Check for legacy aliases first (webhook -> http, split -> conditional)
    // Otherwise, backend type IS the NodeType enum value (they match)
    let mappedType: NodeType = LEGACY_BACKEND_ALIASES[api.type] || api.type as NodeType;

    // Runtime validation - warn if unknown type
    if (!isValidNodeType(mappedType)) {
        console.warn(`[apiMapper] Unknown node type from backend: '${api.type}'. Node ID: ${api.node_id || api.id}`);
    }

    return {
        id: api.node_id || api.id,  // Use logical ID for ReactFlow
        type: 'custom',
        position: api.position || { x: 100, y: 100 },
        data: {
            label: api.name,
            type: mappedType,
            description: api.config?.description || '',
            status: NodeStatus.IDLE,
            config: api.config || {},
        },
    };
}

export function nodeToApi(node: AppNode) {
    // NodeType enum values ARE the backend type strings - no mapping needed!
    // e.g. NodeType.API_REQUEST = 'http', NodeType.HTML_CLEAN = 'html_clean'
    const nodeType = node.data.type as string || 'http';

    return {
        id: node.id,
        name: node.data.label || 'Unnamed Node',
        type: nodeType,  // Direct pass-through - enum values match backend
        config: {
            ...node.data.config,
            description: node.data.description,
        },
        position: {
            x: node.position.x,
            y: node.position.y,
        },
    };
}

// ============ Edge Mappers ============

export function edgeFromApi(api: EdgeApiResponse): AppEdge {
    return {
        id: api.edge_id || api.id,
        source: api.from,
        target: api.to,
        type: 'smoothstep',
        animated: false,
        markerEnd: { type: MarkerType.ArrowClosed },
        data: {
            condition: api.condition,
        },
    };
}

export function edgeToApi(edge: AppEdge) {
    return {
        id: edge.id,
        from: edge.source,
        to: edge.target,
        condition: (edge.data as any)?.condition || {},
    };
}

// ============ Execution Mappers ============

export function executionFromApi(api: ExecutionApiResponse) {
    return {
        id: api.id,
        workflow_id: api.workflow_id,
        status: api.status,
        started_at: api.started_at,
        completed_at: api.completed_at,
        input: api.input || {},
        output: api.output || {},
        error: api.error,
        node_executions: api.node_executions?.map(nodeExecutionFromApi) || [],
        created_at: api.started_at,
        updated_at: api.completed_at || api.started_at,
    };
}

export function nodeExecutionFromApi(api: NodeExecutionApiResponse) {
    return {
        id: api.id,
        execution_id: '',
        node_id: api.node_id,
        status: api.status as any,
        started_at: api.started_at,
        completed_at: api.completed_at,
        input: api.input || {},
        output: api.output || {},
        error: api.error,
    };
}
