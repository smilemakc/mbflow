/**
 * Execution types for React application
 * Ported from Vue: /mbflow-ui/src/types/execution.ts
 */

export type ExecutionStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled"
  | "timeout";

export interface NodeExecution {
  id: string;
  execution_id: string;
  node_id: string;
  node_name?: string;
  node_type?: string;
  status: ExecutionStatus;
  started_at: string;
  completed_at?: string;
  duration?: number;
  error?: string;
  input?: Record<string, any>;
  output?: Record<string, any>;
  retry_count?: number;
  metadata?: Record<string, any>;
}

export interface Execution {
  id: string;
  workflow_id: string;
  workflow_name?: string;
  status: ExecutionStatus;
  started_at: string;
  completed_at?: string;
  duration?: number;
  error?: string;
  input?: Record<string, any>;
  output?: Record<string, any>;
  node_executions?: NodeExecution[];
  variables?: Record<string, any>;
  strict_mode?: boolean;
  triggered_by?: string;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface ExecutionListParams {
  workflow_id?: string;
  status?: ExecutionStatus;
  page?: number;
  limit?: number;
  from?: string;
  to?: string;
}

export interface ExecutionListResponse {
  executions: Execution[];
  total: number;
  page?: number;
  limit?: number;
}

export interface ExecutionEvent {
  type: "event" | "control";
  event?: {
    event_type: string;
    execution_id: string;
    workflow_id: string;
    timestamp: string;
    status: string;
    node_id?: string;
    node_name?: string;
    node_type?: string;
    wave_index?: number;
    node_count?: number;
    duration_ms?: number;
    error?: string;
    input?: Record<string, any>;
    output?: Record<string, any>;
  };
  control?: Record<string, any>;
  timestamp: string;
}

export interface ExecutionStatsParams {
  workflow_id?: string;
  from?: string;
  to?: string;
}

export interface ExecutionStats {
  total: number;
  completed: number;
  failed: number;
  cancelled: number;
  running: number;
  pending: number;
  avg_duration_ms?: number;
  success_rate?: number;
}
