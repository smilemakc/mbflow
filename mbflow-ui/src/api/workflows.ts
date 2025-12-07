import { apiClient } from "./client";
import type {
  Workflow,
  WorkflowCreateRequest,
  WorkflowUpdateRequest,
} from "@/types/workflow";

export interface WorkflowListResponse {
  workflows: Workflow[];
  total: number;
}

export interface WorkflowResponse {
  workflow: Workflow;
}

export interface WorkflowListParams {
  page?: number;
  limit?: number;
  status?: string;
  search?: string;
}

/**
 * Get list of workflows
 */
export async function getWorkflows(
  params?: WorkflowListParams,
): Promise<WorkflowListResponse> {
  const data = await apiClient.get<WorkflowListResponse>("/workflows", {
    params,
  });
  return data as unknown as WorkflowListResponse;
}

/**
 * Get workflow by ID
 */
export async function getWorkflow(id: string): Promise<Workflow> {
  const data = await apiClient.get<WorkflowResponse>(`/workflows/${id}`);
  const response = data as unknown as WorkflowResponse;
  return response.workflow || (response as unknown as Workflow);
}

/**
 * Create new workflow
 */
export async function createWorkflow(
  workflow: WorkflowCreateRequest,
): Promise<Workflow> {
  const data = await apiClient.post<WorkflowResponse>("/workflows", workflow);
  const response = data as unknown as WorkflowResponse;
  return response.workflow || (data as unknown as Workflow);
}

/**
 * Update workflow
 */
export async function updateWorkflow(
  id: string,
  workflow: WorkflowUpdateRequest,
): Promise<WorkflowResponse> {
  const data = await apiClient.put<WorkflowResponse>(
    `/workflows/${id}`,
    workflow,
  );
  return data as unknown as WorkflowResponse;
}

/**
 * Delete workflow
 */
export async function deleteWorkflow(id: string): Promise<void> {
  await apiClient.delete(`/workflows/${id}`);
}

/**
 * Validate workflow DAG
 */
export async function validateWorkflow(id: string): Promise<{
  valid: boolean;
  errors?: string[];
}> {
  const data = await apiClient.post(`/workflows/${id}/validate`);
  return data as unknown as {
    valid: boolean;
    errors?: string[];
  };
}

/**
 * Get workflow execution history
 */
export async function getWorkflowExecutions(
  id: string,
  params?: { page?: number; limit?: number },
): Promise<any> {
  return apiClient.get(`/workflows/${id}/executions`, { params });
}

export interface ExecuteWorkflowOptions {
  input?: Record<string, any>;
  variables?: Record<string, any>; // Execution variables that override workflow variables
  strict_mode?: boolean; // If true, missing template variables cause execution to fail
}

/**
 * Execute workflow manually
 */
export async function executeWorkflow(
  id: string,
  options?: ExecuteWorkflowOptions,
): Promise<{ execution_id: string }> {
  const data = await apiClient.post(`/workflows/${id}/execute`, options || {});
  return data as unknown as {
    execution_id: string;
  };
}

/**
 * Get workflow diagram (Mermaid or ASCII)
 */
export async function getWorkflowDiagram(
  id: string,
  format: "mermaid" | "ascii" = "mermaid",
): Promise<{ diagram: string }> {
  const data = await apiClient.get(`/workflows/${id}/diagram`, {
    params: { format },
  });
  return data as unknown as { diagram: string };
}
