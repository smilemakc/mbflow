import { apiClient } from "./client";
import type { Execution, ExecutionListParams } from "@/types/execution";

export interface ExecutionListResponse {
  executions: Execution[];
  total: number;
}

export interface ExecutionResponse {
  execution: Execution;
}

/**
 * Get list of executions
 */
export async function getExecutions(
  params?: ExecutionListParams,
): Promise<ExecutionListResponse> {
  const data = await apiClient.get<ExecutionListResponse>("/executions", {
    params,
  });
  return data as unknown as ExecutionListResponse;
}

/**
 * Get execution by ID
 */
export async function getExecution(id: string): Promise<ExecutionResponse> {
  const data = await apiClient.get<ExecutionResponse>(`/executions/${id}`);
  return data as unknown as ExecutionResponse;
}

/**
 * Cancel execution
 */
export async function cancelExecution(id: string): Promise<void> {
  await apiClient.post(`/executions/${id}/cancel`);
}

/**
 * Get execution logs
 */
export async function getExecutionLogs(id: string): Promise<any> {
  return apiClient.get(`/executions/${id}/logs`);
}

/**
 * Get execution statistics
 */
export async function getExecutionStats(params?: {
  workflow_id?: string;
  from?: string;
  to?: string;
}): Promise<any> {
  return apiClient.get("/executions/stats", { params });
}
