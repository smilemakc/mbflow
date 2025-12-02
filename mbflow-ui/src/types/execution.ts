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
  duration?: number; // milliseconds
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
  duration?: number; // milliseconds
  error?: string;
  input?: Record<string, any>;
  output?: Record<string, any>;
  node_executions?: NodeExecution[];
  variables?: Record<string, any>; // Runtime variables
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
