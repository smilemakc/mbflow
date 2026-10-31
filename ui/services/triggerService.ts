/**
 * Trigger Service: thin client for /api/v1/triggers.
 */

import { apiClient, ApiListResponse } from '../lib/api';
import type {
  Trigger,
  TriggerCreateRequest,
  TriggerUpdateRequest,
  TriggerListParams,
  TriggerListResponse,
  TriggerExecutionResponse,
} from '@/types/triggers';

export const triggerService = {
  /** List triggers. Supports workflow_id, type, limit and offset filters. */
  async getTriggers(params: TriggerListParams = {}): Promise<TriggerListResponse> {
    const query: Record<string, string | number> = {};
    if (params.workflow_id) query.workflow_id = params.workflow_id;
    if (params.type) query.type = params.type;
    if (params.limit !== undefined) query.limit = params.limit;
    if (params.offset !== undefined) query.offset = params.offset;

    const response = await apiClient.get<ApiListResponse<Trigger>>('/triggers', { params: query });
    const body = response.data;
    return {
      triggers: body.data ?? [],
      total: body.total ?? 0,
      limit: body.limit ?? 0,
      offset: body.offset ?? 0,
    };
  },

  /**
   * Fetch every trigger matching the filters by walking the paginated list.
   * Stops after `maxPages` pages as a safety net.
   */
  async getAllTriggers(
    params: Omit<TriggerListParams, 'limit' | 'offset'> = {},
    pageSize = 100,
    maxPages = 50,
  ): Promise<Trigger[]> {
    const all: Trigger[] = [];
    for (let page = 0; page < maxPages; page++) {
      const { triggers, total } = await triggerService.getTriggers({
        ...params,
        limit: pageSize,
        offset: page * pageSize,
      });
      all.push(...triggers);
      if (triggers.length < pageSize || all.length >= total) break;
    }
    return all;
  },

  async getTrigger(id: string): Promise<Trigger> {
    const response = await apiClient.get<Trigger>(`/triggers/${id}`);
    return response.data;
  },

  async createTrigger(data: TriggerCreateRequest): Promise<Trigger> {
    const response = await apiClient.post<Trigger>('/triggers', data);
    return response.data;
  },

  async updateTrigger(id: string, data: TriggerUpdateRequest): Promise<Trigger> {
    const response = await apiClient.put<Trigger>(`/triggers/${id}`, data);
    return response.data;
  },

  async deleteTrigger(id: string): Promise<void> {
    await apiClient.delete(`/triggers/${id}`);
  },

  /** Fire a trigger by hand. The backend rejects disabled triggers. */
  async executeTrigger(id: string, input?: Record<string, unknown>): Promise<TriggerExecutionResponse> {
    const response = await apiClient.post<TriggerExecutionResponse>(`/triggers/${id}/execute`, {
      input: input ?? {},
    });
    return response.data;
  },

  async enableTrigger(id: string): Promise<Trigger> {
    const response = await apiClient.post<Trigger>(`/triggers/${id}/enable`);
    return response.data;
  },

  async disableTrigger(id: string): Promise<Trigger> {
    const response = await apiClient.post<Trigger>(`/triggers/${id}/disable`);
    return response.data;
  },
};

export default triggerService;
