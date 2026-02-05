import { apiClient, ApiResponse, ApiListResponse } from '../lib/api';
import { DAG, AppNode, AppEdge } from '@/types';
import type { WorkflowResource } from '@/types/workflow';
import {
  workflowFromApi,
  workflowToApi,
  WorkflowApiResponse
} from './apiMapper';

export interface WorkflowPayload {
  name: string;
  description?: string;
  nodes: AppNode[];
  edges: AppEdge[];
}

// DAG save payload with variables as Record (used by dagStore)
export interface DAGSavePayload {
  id?: string;
  name?: string;
  description?: string;
  nodes?: AppNode[];
  edges?: AppEdge[];
  variables?: Record<string, string>;
  resources?: WorkflowResource[];
}

export const workflowService = {
  // Get all workflows
  getAll: async () => {
    const response = await apiClient.get<ApiListResponse<WorkflowApiResponse>>('/workflows');
    return response.data.data.map(workflowFromApi);
  },

  // Get a single workflow by ID
  getById: async (id: string) => {
    const response = await apiClient.get<ApiResponse<WorkflowApiResponse>>(`/workflows/${id}`);
    return workflowFromApi(response.data.data);
  },

  // Create new workflow
  create: async (name: string, description?: string) => {
    const response = await apiClient.post<ApiResponse<WorkflowApiResponse>>('/workflows', {
      name,
      description: description || '',
    });
    return workflowFromApi(response.data.data);
  },

  // Save (Update existing workflow with nodes and edges)
  save: async (dag: DAGSavePayload) => {
    if (!dag.id || dag.id === 'temp_id') {
      // Create new workflow first
      const created = await workflowService.create(dag.name || 'New Workflow', dag.description);
      dag.id = created.id;
    }

    // Update workflow with nodes, edges, variables, and resources
    const payload = workflowToApi({
      id: dag.id,
      name: dag.name || 'Unnamed Workflow',
      description: dag.description,
      nodes: dag.nodes || [],
      edges: dag.edges || [],
      variables: dag.variables,
      resources: dag.resources,
    });

    const response = await apiClient.put<ApiResponse<WorkflowApiResponse>>(`/workflows/${dag.id}`, payload);
    return workflowFromApi(response.data.data);
  },

  // Delete workflow
  delete: async (id: string) => {
    return await apiClient.delete(`/workflows/${id}`);
  },

  // Add resource to workflow
  attachResource: (workflowId: string, resourceId: string, alias: string, accessType?: string) =>
    apiClient.post<ApiResponse<WorkflowResource>>(`/workflows/${workflowId}/resources`, {
      resource_id: resourceId,
      alias,
      access_type: accessType || 'read'
    }),

  // Remove resource from workflow
  detachResource: (workflowId: string, resourceId: string) =>
    apiClient.delete(`/workflows/${workflowId}/resources/${resourceId}`),

  // Get workflow resources
  getResources: (workflowId: string) =>
    apiClient.get<ApiResponse<{ resources: WorkflowResource[] }>>(`/workflows/${workflowId}/resources`),

  // Update resource alias
  updateResourceAlias: (workflowId: string, resourceId: string, alias: string) =>
    apiClient.put<ApiResponse<WorkflowResource>>(`/workflows/${workflowId}/resources/${resourceId}`, { alias }),
};
