import type {Edge as ReactFlowEdge, Node as ReactFlowNode} from 'reactflow';
import type { WorkflowResource } from './types/workflow';

// --- Enums ---

/**
 * Node types aligned with backend
 * Backend reference: internal/domain/workflow/node.go
 */
export enum NodeType {
    // Basic nodes (existing, values updated for backend compatibility)
    TELEGRAM = 'telegram',
    LLM = 'llm',
    DELAY = 'delay',
    HTTP = 'http',
    CONDITIONAL = 'conditional',
    COMMENT = 'comment',

    // New node types from Vue migration
    TRANSFORM = 'transform',
    FUNCTION_CALL = 'function_call',
    FILE_STORAGE = 'file_storage',
    TELEGRAM_DOWNLOAD = 'telegram_download',
    TELEGRAM_PARSE = 'telegram_parse',
    TELEGRAM_CALLBACK = 'telegram_callback',
    MERGE = 'merge',

    // Adapter nodes
    BASE64_TO_BYTES = 'base64_to_bytes',
    BYTES_TO_BASE64 = 'bytes_to_base64',
    STRING_TO_JSON = 'string_to_json',
    JSON_TO_STRING = 'json_to_string',
    BYTES_TO_JSON = 'bytes_to_json',
    FILE_TO_BYTES = 'file_to_bytes',
    BYTES_TO_FILE = 'bytes_to_file',

    // Content processing
    HTML_CLEAN = 'html_clean',
    RSS_PARSER = 'rss_parser',
    CSV_TO_JSON = 'csv_to_json',

    // External integrations
    GOOGLE_SHEETS = 'google_sheets',
    GOOGLE_DRIVE = 'google_drive'
}

// Type representing all possible NodeType values
export type NodeTypeValue = `${NodeType}`;

// Runtime validation for node types
const nodeTypeValues = new Set(Object.values(NodeType));

export function isValidNodeType(type: string): type is NodeTypeValue {
    return nodeTypeValues.has(type as NodeType);
}

/**
 * Workflow status types
 */
export type WorkflowStatus = 'draft' | 'active' | 'inactive' | 'archived';

export enum NodeStatus {
    IDLE = 'idle',
    PENDING = 'pending',
    RUNNING = 'running',
    SUCCESS = 'success',
    ERROR = 'error',
    SKIPPED = 'skipped'
}

export enum VariableType {
    STRING = 'string',
    NUMBER = 'number',
    BOOLEAN = 'boolean',
    OBJECT = 'object'
}

export enum VariableSource {
    INPUT = 'input',
    NODE = 'node',
    GLOBAL = 'global'
}

// --- Interfaces ---

export interface NodeData extends Record<string, unknown> {
    label: string;
    description?: string;
    config?: Record<string, any>;
    icon?: string;
    status?: NodeStatus; // Visual status indicator
    type?: NodeType; // Explicit type in data for easy access
    lastRun?: {
        duration: number;
        timestamp: number;
    };
    executionState?: 'running' | 'completed' | 'failed';
    animated?: boolean;
    executionInput?: Record<string, any>;
    executionOutput?: Record<string, any>;
    executionError?: string;
}

export type AppNode = ReactFlowNode<NodeData>;
export type AppEdge = ReactFlowEdge;

export interface Variable {
    id: string;
    key: string;
    name: string;
    type: VariableType;
    source: VariableSource;
    nodeId?: string;
    value?: any;
}

export interface DAG {
    id: string;
    name: string;
    description: string;
    version: number;
    status: WorkflowStatus;
    nodes: AppNode[];
    edges: AppEdge[];
    variables: Variable[];
    resources?: WorkflowResource[];
    tags?: string[];
    metadata?: Record<string, any>;
    createdAt: string;
    updatedAt: string;
    ownerId: string;
}

export interface DAGHistoryState {
    nodes: AppNode[];
    edges: AppEdge[];
}

// --- Execution & Monitoring ---

export interface ExecutionLog {
    id: string;
    nodeId: string | null;
    level: 'info' | 'error' | 'success' | 'warning';
    message: string;
    timestamp: Date;
}

export interface NodeExecutionResult {
    nodeId: string;
    status: NodeStatus;
    inputs: Record<string, any>;
    outputs: Record<string, any>;
    startTime: number;
    endTime?: number;
    logs: string[];
}