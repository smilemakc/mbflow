/**
 * Trigger types for React application
 * Ported from Vue: /mbflow-ui/src/types/trigger.ts
 */

export type TriggerType = "manual" | "schedule" | "webhook" | "event";
export type TriggerStatus = "enabled" | "disabled";

export interface TriggerConfig {
  // Schedule trigger config
  cron?: string;
  timezone?: string;

  // Webhook trigger config
  webhook_path?: string;
  http_method?: string;
  auth_type?: string;
  auth_config?: Record<string, any>;

  // Event trigger config
  event_type?: string;
  event_filter?: Record<string, any>;

  [key: string]: any;
}

export interface Trigger {
  id: string;
  workflow_id: string;
  workflow_name?: string;
  name: string;
  description?: string;
  type: TriggerType;
  status: TriggerStatus;
  config: TriggerConfig;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
  last_triggered_at?: string;
  next_trigger_at?: string;
}

export interface TriggerCreateRequest {
  workflow_id: string;
  name: string;
  description?: string;
  type: TriggerType;
  status?: TriggerStatus;
  config: TriggerConfig;
  metadata?: Record<string, any>;
}

export interface TriggerUpdateRequest {
  name?: string;
  description?: string;
  status?: TriggerStatus;
  config?: TriggerConfig;
  metadata?: Record<string, any>;
}

export interface TriggerListParams {
  workflow_id?: string;
  type?: TriggerType;
  status?: TriggerStatus;
  page?: number;
  limit?: number;
}

export interface TriggerListResponse {
  triggers: Trigger[];
  total: number;
}

export interface TriggerExecutionResponse {
  execution_id: string;
}
