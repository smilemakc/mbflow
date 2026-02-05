/**
 * Trigger Service for React application
 * Ported from Vue: /mbflow-ui/src/api/triggers.ts
 */

import { apiClient, ApiResponse, ApiListResponse } from '../lib/api';
import type {
  Trigger,
  TriggerCreateRequest,
  TriggerUpdateRequest,
  TriggerListParams,
  TriggerExecutionResponse,
} from '@/types/triggers';

export const triggerService = {
  /**
   * Get list of triggers with optional filters
   */
  async getTriggers(params?: TriggerListParams): Promise<{ triggers: Trigger[]; total: number }> {
    const response = await apiClient.get<ApiListResponse<Trigger>>('/triggers', { params });
    return {
      triggers: response.data.data,
      total: response.data.meta.total,
    };
  },

  /**
   * Get a single trigger by ID
   */
  async getTrigger(id: string): Promise<Trigger> {
    const response = await apiClient.get<ApiResponse<Trigger>>(`/triggers/${id}`);
    return response.data.data;
  },

  /**
   * Create a new trigger
   */
  async createTrigger(data: TriggerCreateRequest): Promise<Trigger> {
    const response = await apiClient.post<ApiResponse<Trigger>>('/triggers', data);
    return response.data.data;
  },

  /**
   * Update an existing trigger
   */
  async updateTrigger(id: string, data: TriggerUpdateRequest): Promise<Trigger> {
    const response = await apiClient.put<ApiResponse<Trigger>>(`/triggers/${id}`, data);
    return response.data.data;
  },

  /**
   * Delete a trigger
   */
  async deleteTrigger(id: string): Promise<void> {
    await apiClient.delete(`/triggers/${id}`);
  },

  /**
   * Manually execute a trigger
   */
  async executeTrigger(id: string, input?: Record<string, any>): Promise<TriggerExecutionResponse> {
    const response = await apiClient.post<ApiResponse<TriggerExecutionResponse>>(
      `/triggers/${id}/execute`,
      input ? { input } : undefined
    );
    return response.data.data;
  },

  /**
   * Enable a trigger
   */
  async enableTrigger(id: string): Promise<Trigger> {
    const response = await apiClient.post<ApiResponse<Trigger>>(`/triggers/${id}/enable`);
    return response.data.data;
  },

  /**
   * Disable a trigger
   */
  async disableTrigger(id: string): Promise<Trigger> {
    const response = await apiClient.post<ApiResponse<Trigger>>(`/triggers/${id}/disable`);
    return response.data.data;
  },
};

export default triggerService;
