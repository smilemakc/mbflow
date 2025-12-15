import { apiClient } from '../lib/api';
import { DAG, AppNode, AppEdge } from '@/types';
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

interface WorkflowListResponse {
  workflows: WorkflowApiResponse[];
  total: number;
  limit: number;
  offset: number;
}

// DAG save payload with variables as Record (used by dagStore)
export interface DAGSavePayload {
  id?: string;
  name?: string;
  description?: string;
  nodes?: AppNode[];
  edges?: AppEdge[];
  variables?: Record<string, string>;
}

export const workflowService = {
  // Get all workflows
  getAll: async () => {
    const response = await apiClient.get<WorkflowListResponse>('/workflows');
    return response.data.workflows.map(workflowFromApi);
  },

  // Get a single workflow by ID
  getById: async (id: string) => {
    const response = await apiClient.get<WorkflowApiResponse>(`/workflows/${id}`);
    return workflowFromApi(response.data);
  },

  // Create new workflow
  create: async (name: string, description?: string) => {
    const response = await apiClient.post<WorkflowApiResponse>('/workflows', {
      name,
      description: description || '',
    });
    return workflowFromApi(response.data);
  },

  // Save (Update existing workflow with nodes and edges)
  save: async (dag: DAGSavePayload) => {
    if (!dag.id || dag.id === 'temp_id') {
      // Create new workflow first
      const created = await workflowService.create(dag.name || 'New Workflow', dag.description);
      dag.id = created.id;
    }

    // Update workflow with nodes, edges, and variables
    const payload = workflowToApi({
      id: dag.id,
      name: dag.name || 'Unnamed Workflow',
      description: dag.description,
      nodes: dag.nodes || [],
      edges: dag.edges || [],
      variables: dag.variables,
    });

    const response = await apiClient.put<WorkflowApiResponse>(`/workflows/${dag.id}`, payload);
    return workflowFromApi(response.data);
  },

  // Delete workflow
  delete: async (id: string) => {
    return await apiClient.delete(`/workflows/${id}`);
  }
};