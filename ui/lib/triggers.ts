/**
 * Pure helpers for working with workflow triggers: converting between the
 * trigger form and the REST payloads, validation and display formatting.
 *
 * API contract (go/pkg/models/trigger.go):
 *   manual   - no config
 *   cron     - config.schedule (cron expression, 5 or 6 fields or @descriptor), config.timezone (IANA, optional)
 *   interval - config.interval (Go duration string such as "5m", or a number of seconds)
 *   webhook  - config.secret (HMAC secret, optional), config.ip_whitelist (IPs / CIDRs, optional)
 *   event    - config.event_type (required), config.source, config.filter (optional)
 *   all      - config.input: default workflow input (optional)
 */

import type {
  Trigger,
  TriggerConfig,
  TriggerCreateRequest,
  TriggerType,
  TriggerUpdateRequest,
} from '@/types/triggers';
import { TRIGGER_TYPES } from '@/types/triggers';

export interface TriggerFormValues {
  workflowId: string;
  name: string;
  description: string;
  type: TriggerType;
  enabled: boolean;
  // cron
  schedule: string;
  timezone: string;
  // interval
  interval: string;
  // webhook
  secret: string;
  ipWhitelist: string;
  // event
  eventType: string;
  eventSource: string;
  eventFilter: string;
  // all types
  input: string;
}

export type TriggerFormErrorCode =
  | 'required'
  | 'invalidCron'
  | 'invalidTimezone'
  | 'invalidDuration'
  | 'invalidJson'
  | 'notAnObject'
  | 'invalidIp';

export type TriggerFormErrors = Partial<Record<keyof TriggerFormValues, TriggerFormErrorCode>>;

/** Config keys the form manages; everything else is preserved on edit. */
const MANAGED_CONFIG_KEYS = [
  'name',
  'description',
  'schedule',
  'timezone',
  'interval',
  'secret',
  'ip_whitelist',
  'event_type',
  'source',
  'filter',
  'input',
] as const;

export const isTriggerType = (value: unknown): value is TriggerType =>
  typeof value === 'string' && (TRIGGER_TYPES as readonly string[]).includes(value);

export const emptyTriggerForm = (workflowId = ''): TriggerFormValues => ({
  workflowId,
  name: '',
  description: '',
  type: 'cron',
  enabled: true,
  schedule: '0 * * * *',
  timezone: '',
  interval: '5m',
  secret: '',
  ipWhitelist: '',
  eventType: '',
  eventSource: '',
  eventFilter: '',
  input: '',
});

const str = (value: unknown): string => (typeof value === 'string' ? value : '');

const jsonText = (value: unknown): string =>
  value && typeof value === 'object' && Object.keys(value as object).length > 0
    ? JSON.stringify(value, null, 2)
    : '';

/** Name shown for a trigger. Older backends only keep the name inside config. */
export const getTriggerName = (trigger: Pick<Trigger, 'id' | 'name' | 'type' | 'config'>): string =>
  trigger.name || str(trigger.config?.name) || `${trigger.type} trigger ${trigger.id.slice(0, 8)}`;

export const getTriggerDescription = (trigger: Pick<Trigger, 'description' | 'config'>): string =>
  trigger.description || str(trigger.config?.description);

export const triggerToForm = (trigger: Trigger): TriggerFormValues => {
  const config: TriggerConfig = trigger.config ?? {};
  const base = emptyTriggerForm(trigger.workflow_id);
  const interval = config.interval;
  return {
    ...base,
    name: trigger.name || str(config.name),
    description: getTriggerDescription(trigger),
    type: isTriggerType(trigger.type) ? trigger.type : 'manual',
    enabled: trigger.enabled,
    schedule: str(config.schedule) || (trigger.type === 'cron' ? '' : base.schedule),
    timezone: str(config.timezone),
    interval:
      typeof interval === 'number'
        ? String(interval)
        : str(interval) || (trigger.type === 'interval' ? '' : base.interval),
    secret: str(config.secret),
    ipWhitelist: Array.isArray(config.ip_whitelist) ? config.ip_whitelist.join('\n') : '',
    eventType: str(config.event_type),
    eventSource: str(config.source),
    eventFilter: jsonText(config.filter),
    input: jsonText(config.input),
  };
};

const CRON_DESCRIPTORS = ['@yearly', '@annually', '@monthly', '@weekly', '@daily', '@midnight', '@hourly'];
const CRON_FIELD = /^[0-9A-Za-z*?/,\-]+$/;
const GO_DURATION = /^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$/;

/**
 * Validates a cron expression the way the backend parser accepts it:
 * 5 fields, 6 fields (leading seconds), a descriptor, or "@every <duration>".
 */
export const isValidCronExpression = (expression: string): boolean => {
  const expr = expression.trim();
  if (!expr) return false;
  if (expr.startsWith('@')) {
    if (CRON_DESCRIPTORS.includes(expr)) return true;
    const every = /^@every\s+(\S+)$/.exec(expr);
    return !!every && isValidDuration(every[1]);
  }
  const fields = expr.split(/\s+/);
  return (fields.length === 5 || fields.length === 6) && fields.every((f) => CRON_FIELD.test(f));
};

/** Positive Go duration ("90s", "1h30m") or a positive whole number of seconds. */
export const isValidDuration = (value: string): boolean => {
  const v = value.trim();
  if (/^\d+$/.test(v)) return Number(v) > 0;
  if (!GO_DURATION.test(v)) return false;
  // Reject all-zero durations such as "0s".
  return /[1-9]/.test(v);
};

export const isValidTimezone = (timezone: string): boolean => {
  const tz = timezone.trim();
  if (!tz) return true;
  try {
    new Intl.DateTimeFormat('en-US', { timeZone: tz });
    return true;
  } catch {
    return false;
  }
};

const IPV4 = /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/;
const IPV6 = /^[0-9a-fA-F:]+$/;

export const isValidIpOrCidr = (value: string): boolean => {
  const [ip, prefix, ...rest] = value.split('/');
  if (rest.length > 0) return false;
  const v4 = IPV4.test(ip);
  const v6 = !v4 && ip.includes(':') && IPV6.test(ip);
  if (!v4 && !v6) return false;
  if (prefix === undefined) return true;
  if (!/^\d+$/.test(prefix)) return false;
  return Number(prefix) <= (v4 ? 32 : 128);
};

export const parseList = (text: string): string[] =>
  text
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean);

export interface JsonObjectResult {
  ok: boolean;
  value?: Record<string, unknown>;
  error?: 'invalidJson' | 'notAnObject';
}

/** Parses optional JSON object text. Empty text yields `undefined`. */
export const parseJsonObject = (text: string): JsonObjectResult => {
  if (!text.trim()) return { ok: true, value: undefined };
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return { ok: false, error: 'invalidJson' };
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return { ok: false, error: 'notAnObject' };
  }
  return { ok: true, value: parsed as Record<string, unknown> };
};

export const validateTriggerForm = (values: TriggerFormValues): TriggerFormErrors => {
  const errors: TriggerFormErrors = {};

  if (!values.workflowId.trim()) errors.workflowId = 'required';
  if (!values.name.trim()) errors.name = 'required';

  switch (values.type) {
    case 'cron':
      if (!values.schedule.trim()) errors.schedule = 'required';
      else if (!isValidCronExpression(values.schedule)) errors.schedule = 'invalidCron';
      if (!isValidTimezone(values.timezone)) errors.timezone = 'invalidTimezone';
      break;
    case 'interval':
      if (!values.interval.trim()) errors.interval = 'required';
      else if (!isValidDuration(values.interval)) errors.interval = 'invalidDuration';
      break;
    case 'webhook':
      if (parseList(values.ipWhitelist).some((ip) => !isValidIpOrCidr(ip))) {
        errors.ipWhitelist = 'invalidIp';
      }
      break;
    case 'event': {
      if (!values.eventType.trim()) errors.eventType = 'required';
      const filter = parseJsonObject(values.eventFilter);
      if (!filter.ok) errors.eventFilter = filter.error;
      break;
    }
    case 'manual':
      break;
  }

  const input = parseJsonObject(values.input);
  if (!input.ok) errors.input = input.error;

  return errors;
};

export const hasErrors = (errors: TriggerFormErrors): boolean => Object.keys(errors).length > 0;

/**
 * Builds the config object for the API. When `baseConfig` is given (editing a
 * trigger of the same type) keys the form does not manage are preserved.
 * Assumes the values passed validation.
 */
export const buildTriggerConfig = (values: TriggerFormValues, baseConfig?: TriggerConfig): TriggerConfig => {
  const config: TriggerConfig = { ...(baseConfig ?? {}) };
  for (const key of MANAGED_CONFIG_KEYS) delete config[key];

  const name = values.name.trim();
  const description = values.description.trim();
  if (name) config.name = name;
  if (description) config.description = description;

  switch (values.type) {
    case 'cron':
      config.schedule = values.schedule.trim();
      if (values.timezone.trim()) config.timezone = values.timezone.trim();
      break;
    case 'interval': {
      const interval = values.interval.trim();
      config.interval = /^\d+$/.test(interval) ? Number(interval) : interval;
      break;
    }
    case 'webhook': {
      if (values.secret) config.secret = values.secret;
      const ips = parseList(values.ipWhitelist);
      if (ips.length > 0) config.ip_whitelist = ips;
      break;
    }
    case 'event': {
      config.event_type = values.eventType.trim();
      if (values.eventSource.trim()) config.source = values.eventSource.trim();
      const filter = parseJsonObject(values.eventFilter);
      if (filter.ok && filter.value) config.filter = filter.value;
      break;
    }
    case 'manual':
      break;
  }

  const input = parseJsonObject(values.input);
  if (input.ok && input.value) config.input = input.value;

  return config;
};

export const buildCreateRequest = (values: TriggerFormValues): TriggerCreateRequest => {
  const request: TriggerCreateRequest = {
    workflow_id: values.workflowId.trim(),
    name: values.name.trim(),
    type: values.type,
    config: buildTriggerConfig(values),
    enabled: values.enabled,
  };
  const description = values.description.trim();
  if (description) request.description = description;
  return request;
};

export const buildUpdateRequest = (values: TriggerFormValues, original: Trigger): TriggerUpdateRequest => ({
  name: values.name.trim(),
  description: values.description.trim(),
  type: values.type,
  // Unknown keys survive only when the type is unchanged.
  config: buildTriggerConfig(values, values.type === original.type ? original.config : undefined),
  enabled: values.enabled,
});

/** Public URL that fires a webhook trigger: POST {api}/webhooks/{id}. */
export const getWebhookUrl = (triggerId: string, apiBaseUrl: string, origin: string): string => {
  const base = apiBaseUrl.replace(/\/+$/, '');
  const absolute = /^https?:\/\//.test(base) ? base : `${origin.replace(/\/+$/, '')}${base.startsWith('/') ? '' : '/'}${base}`;
  return `${absolute}/webhooks/${triggerId}`;
};

/** One-line summary of a trigger's configuration for list views. */
export const summarizeTriggerConfig = (trigger: Pick<Trigger, 'type' | 'config'>): string => {
  const config = trigger.config ?? {};
  switch (trigger.type) {
    case 'cron': {
      const schedule = str(config.schedule);
      const tz = str(config.timezone);
      return tz ? `${schedule} (${tz})` : schedule;
    }
    case 'interval':
      return typeof config.interval === 'number'
        ? `every ${config.interval}s`
        : config.interval
          ? `every ${str(config.interval)}`
          : '';
    case 'event': {
      const eventType = str(config.event_type);
      const source = str(config.source);
      return source ? `${eventType} from ${source}` : eventType;
    }
    case 'webhook':
      return config.secret ? 'POST, HMAC-signed' : 'POST';
    default:
      return '';
  }
};

export type TriggerStatusFilter = 'enabled' | 'disabled';

export const filterTriggersByStatus = <T extends Pick<Trigger, 'enabled'>>(
  triggers: T[],
  status?: TriggerStatusFilter,
): T[] => {
  if (!status) return triggers;
  const wanted = status === 'enabled';
  return triggers.filter((t) => t.enabled === wanted);
};

export type RelativeTime =
  | { unit: 'now' }
  | { unit: 'minutes' | 'hours' | 'days'; value: number; past: boolean };

/** Coarse relative time between `target` and `now`. */
export const relativeTime = (target: string | Date, now: Date = new Date()): RelativeTime => {
  const diffMs = new Date(target).getTime() - now.getTime();
  const past = diffMs < 0;
  const minutes = Math.floor(Math.abs(diffMs) / 60000);
  if (minutes < 1) return { unit: 'now' };
  if (minutes < 60) return { unit: 'minutes', value: minutes, past };
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return { unit: 'hours', value: hours, past };
  return { unit: 'days', value: Math.floor(hours / 24), past };
};
