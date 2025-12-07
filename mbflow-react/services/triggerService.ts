/**
 * Trigger Service for React application
 * Ported from Vue: /mbflow-ui/src/api/triggers.ts
 */

import apiClient from '../lib/api';
import type {
  Trigger,
  TriggerCreateRequest,
  TriggerUpdateRequest,
  TriggerListParams,
  TriggerListResponse,
  TriggerExecutionResponse,
} from '@/types/triggers';

interface TriggerResponse {
  trigger: Trigger;
}

export const triggerService = {
  /**
   * Get list of triggers with optional filters
   */
  async getTriggers(params?: TriggerListParams): Promise<TriggerListResponse> {
    const response = await apiClient.get<TriggerListResponse>('/triggers', { params });
    return response.data;
  },

  /**
   * Get a single trigger by ID
   */
  async getTrigger(id: string): Promise<Trigger> {
    const response = await apiClient.get<TriggerResponse | Trigger>(`/triggers/${id}`);
    // Handle both wrapped and unwrapped response formats
    const data = response.data;
    return 'trigger' in data ? data.trigger : data;
  },

  /**
   * Create a new trigger
   */
  async createTrigger(data: TriggerCreateRequest): Promise<Trigger> {
    const response = await apiClient.post<TriggerResponse | Trigger>('/triggers', data);
    const result = response.data;
    return 'trigger' in result ? result.trigger : result;
  },

  /**
   * Update an existing trigger
   */
  async updateTrigger(id: string, data: TriggerUpdateRequest): Promise<Trigger> {
    const response = await apiClient.put<TriggerResponse | Trigger>(`/triggers/${id}`, data);
    const result = response.data;
    return 'trigger' in result ? result.trigger : result;
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
    const response = await apiClient.post<TriggerExecutionResponse>(
      `/triggers/${id}/execute`,
      input ? { input } : undefined
    );
    return response.data;
  },

  /**
   * Enable a trigger
   */
  async enableTrigger(id: string): Promise<Trigger> {
    const response = await apiClient.post<TriggerResponse | Trigger>(`/triggers/${id}/enable`);
    const result = response.data;
    return 'trigger' in result ? result.trigger : result;
  },

  /**
   * Disable a trigger
   */
  async disableTrigger(id: string): Promise<Trigger> {
    const response = await apiClient.post<TriggerResponse | Trigger>(`/triggers/${id}/disable`);
    const result = response.data;
    return 'trigger' in result ? result.trigger : result;
  },
};

export default triggerService;
