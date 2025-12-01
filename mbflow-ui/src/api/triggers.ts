import { apiClient } from "./client";
import type {
  Trigger,
  TriggerCreateRequest,
  TriggerUpdateRequest,
  TriggerListParams,
} from "@/types/trigger";

export interface TriggerListResponse {
  triggers: Trigger[];
  total: number;
}

export interface TriggerResponse {
  trigger: Trigger;
}

/**
 * Get list of triggers
 */
export async function getTriggers(
  params?: TriggerListParams,
): Promise<TriggerListResponse> {
  const data = await apiClient.get<TriggerListResponse>("/triggers", {
    params,
  });
  return data as unknown as TriggerListResponse;
}

/**
 * Get trigger by ID
 */
export async function getTrigger(id: string): Promise<TriggerResponse> {
  const data = await apiClient.get<TriggerResponse>(`/triggers/${id}`);
  return data as unknown as TriggerResponse;
}

/**
 * Create new trigger
 */
export async function createTrigger(
  trigger: TriggerCreateRequest,
): Promise<TriggerResponse> {
  const data = await apiClient.post<TriggerResponse>("/triggers", trigger);
  return data as unknown as TriggerResponse;
}

/**
 * Update trigger
 */
export async function updateTrigger(
  id: string,
  trigger: TriggerUpdateRequest,
): Promise<TriggerResponse> {
  const data = await apiClient.put<TriggerResponse>(`/triggers/${id}`, trigger);
  return data as unknown as TriggerResponse;
}

/**
 * Delete trigger
 */
export async function deleteTrigger(id: string): Promise<void> {
  await apiClient.delete(`/triggers/${id}`);
}

/**
 * Execute trigger manually
 */
export async function executeTrigger(
  id: string,
  input?: Record<string, any>,
): Promise<{ execution_id: string }> {
  const data = await apiClient.post(`/triggers/${id}/execute`, { input });
  return data as unknown as { execution_id: string };
}

/**
 * Enable trigger
 */
export async function enableTrigger(id: string): Promise<TriggerResponse> {
  const data = await apiClient.post<TriggerResponse>(`/triggers/${id}/enable`);
  return data as unknown as TriggerResponse;
}

/**
 * Disable trigger
 */
export async function disableTrigger(id: string): Promise<TriggerResponse> {
  const data = await apiClient.post<TriggerResponse>(`/triggers/${id}/disable`);
  return data as unknown as TriggerResponse;
}
