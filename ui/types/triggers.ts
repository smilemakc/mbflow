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

  // Legacy: triggers created before the backend persisted name/description
  // (migration 020) kept them here. Read as a fallback only; never written.
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
  /** Next scheduled run; set by the API for enabled cron and interval triggers. */
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

/**
 * POST /api/v1/triggers/:id/execute (202 Accepted): the queued execution.
 * Body `{input}` is merged over the trigger's default config.input.
 */
export interface TriggerExecutionResponse {
  id: string;
  workflow_id: string;
  status: string;
  [key: string]: unknown;
}
