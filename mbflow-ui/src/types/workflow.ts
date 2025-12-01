export type WorkflowStatus = "draft" | "active" | "inactive" | "archived";

export interface Position {
  x: number;
  y: number;
}

export interface Node {
  id: string;
  name: string;
  type: string;
  description?: string;
  config: Record<string, any>;
  position?: Position;
  metadata?: Record<string, any>;
}

export interface Edge {
  id: string;
  from: string;
  to: string;
  condition?: string;
  metadata?: Record<string, any>;
}

export interface Workflow {
  id: string;
  name: string;
  description?: string;
  version: number;
  status: WorkflowStatus;
  tags?: string[];
  nodes: Node[];
  edges: Edge[];
  variables?: Record<string, any>;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface WorkflowCreateRequest {
  name: string;
  description?: string;
  variables?: Record<string, any>;
  metadata?: Record<string, any>;
}

export interface WorkflowUpdateRequest {
  name?: string;
  description?: string;
  variables?: Record<string, any>;
  metadata?: Record<string, any>;
  status?: WorkflowStatus;
}
