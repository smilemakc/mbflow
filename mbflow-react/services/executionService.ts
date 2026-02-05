import { apiClient, ApiResponse, ApiListResponse } from '../lib/api';
import { NodeExecutionResult, ExecutionLog } from '@/types';
import {
  executionFromApi,
  ExecutionApiResponse
} from './apiMapper';

export interface ExecutionResponse {
  id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  workflow_id: string;
}

export interface ExecutionStatusResponse {
  status: 'pending' | 'running' | 'completed' | 'failed';
  results: Record<string, NodeExecutionResult>;
  logs: ExecutionLog[];
}

interface LogsResponse {
  logs: {
    timestamp: string;
    event_type: string;
    level: string;
    message: string;
    data: Record<string, any>;
  }[];
  total: number;
}

export const executionService = {
  // Trigger a new execution
  trigger: async (workflowId: string, input?: Record<string, any>) => {
    const response = await apiClient.post<ApiResponse<ExecutionApiResponse>>(`/executions/run/${workflowId}`, {
      input: input || {},
      async: true,
    });
    return executionFromApi(response.data.data);
  },

  // Get status of a running execution
  getStatus: async (executionId: string) => {
    const response = await apiClient.get<ApiResponse<ExecutionApiResponse>>(`/executions/${executionId}`);
    return executionFromApi(response.data.data);
  },

  // Get logs for an execution
  getLogs: async (executionId: string) => {
    const response = await apiClient.get<ApiResponse<LogsResponse>>(`/executions/${executionId}/logs`);
    const logsData = response.data.data;
    return logsData.logs.map(log => ({
      id: `${log.timestamp}_${log.event_type}`,
      nodeId: log.data?.node_id || null,
      level: log.level as 'info' | 'error' | 'success' | 'warning',
      message: log.message,
      timestamp: new Date(log.timestamp),
    }));
  },

  // Get recent executions
  getRecent: async (limit = 5) => {
    const response = await apiClient.get<ApiListResponse<ExecutionApiResponse>>(`/executions?limit=${limit}`);
    return response.data.data.map(executionFromApi);
  },

  // Cancel execution (not implemented on backend yet)
  cancel: async (executionId: string) => {
    return await apiClient.post(`/executions/${executionId}/cancel`);
  },

  // Get all executions with filters
  getAll: async (params?: {
    workflow_id?: string;
    status?: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'timeout';
    limit?: number;
    offset?: number;
    from?: string;
    to?: string;
  }) => {
    const queryParams = new URLSearchParams();
    if (params?.workflow_id) queryParams.append('workflow_id', params.workflow_id);
    if (params?.status) queryParams.append('status', params.status);
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.offset) queryParams.append('offset', params.offset.toString());
    if (params?.from) queryParams.append('from', params.from);
    if (params?.to) queryParams.append('to', params.to);

    const response = await apiClient.get<ApiListResponse<ExecutionApiResponse>>(
      `/executions?${queryParams.toString()}`
    );
    return {
      executions: response.data.data.map(executionFromApi),
      total: response.data.meta.total,
      limit: response.data.meta.limit,
      offset: response.data.meta.offset,
    };
  },

  // Retry a failed execution
  retry: async (executionId: string) => {
    const response = await apiClient.post<ApiResponse<ExecutionApiResponse>>(
      `/executions/${executionId}/retry`
    );
    return executionFromApi(response.data.data);
  }
};
