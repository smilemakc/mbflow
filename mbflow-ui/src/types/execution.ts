export type ExecutionStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface NodeExecution {
  node_id: string;
  status: ExecutionStatus;
  started_at: string;
  completed_at?: string;
  error?: string;
  input?: Record<string, any>;
  output?: Record<string, any>;
  retry_count?: number;
}

export interface Execution {
  id: string;
  workflow_id: string;
  workflow_name?: string;
  status: ExecutionStatus;
  started_at: string;
  completed_at?: string;
  error?: string;
  input?: Record<string, any>;
  output?: Record<string, any>;
  node_executions?: NodeExecution[];
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
