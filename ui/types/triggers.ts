/**
 * Trigger types. These mirror the REST API contract served by the Go backend
 * (go/pkg/models/trigger.go and go/internal/infrastructure/api/rest/handlers_triggers.go).
 */

export const TRIGGER_TYPES = ['manual', 'cron', 'interval', 'webhook', 'event'] as const;

export type TriggerType = (typeof TRIGGER_TYPES)[number];

/**
 * Type-specific configuration. The backend stores it as a free-form JSON object;
 * the keys below are the ones the trigger manager reads.
 */
export interface TriggerConfig {
  // cron
  schedule?: string;
  timezone?: string;

  // interval: Go duration string ("30s", "5m", "1h") or a number of seconds
  interval?: string | number;

  // webhook
  secret?: string;
  ip_whitelist?: string[];

  // event
  event_type?: string;
  source?: string;
  filter?: Record<string, unknown>;

  // all types: default workflow input
  input?: Record<string, unknown>;

  // name/description are mirrored into config so they survive a reload
  // (the trigger table has no dedicated columns for them).
  name?: string;
  description?: string;

  [key: string]: unknown;
}

export interface Trigger {
  id: string;
  workflow_id: string;
  name: string;
  description?: string;
  type: TriggerType;
  config: TriggerConfig;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_run?: string;
  next_run?: string;
  metadata?: Record<string, unknown>;
}

/** POST /api/v1/triggers */
export interface TriggerCreateRequest {
  workflow_id: string;
  name: string;
  description?: string;
  type: TriggerType;
  config: TriggerConfig;
  enabled: boolean;
  metadata?: Record<string, unknown>;
}

/** PUT /api/v1/triggers/:id (config replaces the stored config as a whole) */
export interface TriggerUpdateRequest {
  name?: string;
  description?: string;
  type?: TriggerType;
  config?: TriggerConfig;
  enabled?: boolean;
  metadata?: Record<string, unknown>;
}

/** GET /api/v1/triggers query parameters */
export interface TriggerListParams {
  workflow_id?: string;
  type?: TriggerType;
  limit?: number;
  offset?: number;
}

export interface TriggerListResponse {
  triggers: Trigger[];
  total: number;
  limit: number;
  offset: number;
}

/** POST /api/v1/triggers/:id/execute */
export interface TriggerExecutionResponse {
  execution_id?: string;
  [key: string]: unknown;
}
