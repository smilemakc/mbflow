import { describe, expect, it } from 'vitest';
import {
  buildCreateRequest,
  buildTriggerConfig,
  buildUpdateRequest,
  emptyTriggerForm,
  filterTriggersByStatus,
  getTriggerDescription,
  getTriggerName,
  getWebhookUrl,
  hasErrors,
  isTriggerType,
  isValidCronExpression,
  isValidDuration,
  isValidIpOrCidr,
  isValidTimezone,
  parseJsonObject,
  parseList,
  relativeTime,
  summarizeTriggerConfig,
  triggerToForm,
  validateTriggerForm,
  type TriggerFormValues,
} from './triggers';
import type { Trigger } from '@/types/triggers';

const WF = '11111111-1111-4111-8111-111111111111';

const trigger = (overrides: Partial<Trigger> = {}): Trigger => ({
  id: 'abcdef12-0000-4000-8000-000000000000',
  workflow_id: WF,
  name: 'Nightly',
  type: 'cron',
  config: { schedule: '0 0 * * *' },
  enabled: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  ...overrides,
});

const form = (overrides: Partial<TriggerFormValues> = {}): TriggerFormValues => ({
  ...emptyTriggerForm(WF),
  name: 'My trigger',
  ...overrides,
});

describe('isTriggerType', () => {
  it('accepts every API trigger type and rejects legacy UI values', () => {
    for (const type of ['manual', 'cron', 'interval', 'webhook', 'event']) {
      expect(isTriggerType(type)).toBe(true);
    }
    expect(isTriggerType('schedule')).toBe(false);
    expect(isTriggerType(42)).toBe(false);
  });
});

describe('getTriggerName / getTriggerDescription', () => {
  it('prefers the top-level name and falls back to config, then to a generated label', () => {
    expect(getTriggerName(trigger())).toBe('Nightly');
    expect(getTriggerName(trigger({ name: '', config: { name: 'From config' } }))).toBe('From config');
    expect(getTriggerName(trigger({ name: '', config: {} }))).toBe('cron trigger abcdef12');
    expect(getTriggerName(trigger({ name: '', config: undefined as never }))).toBe('cron trigger abcdef12');
  });

  it('reads description from the field or config', () => {
    expect(getTriggerDescription(trigger({ description: 'top' }))).toBe('top');
    expect(getTriggerDescription(trigger({ config: { description: 'cfg' } }))).toBe('cfg');
    expect(getTriggerDescription(trigger())).toBe('');
  });
});

describe('cron validation', () => {
  it.each([
    '0 * * * *',
    '*/5 9-17 * * MON-FRI',
    '0 30 9 * * 1-5',
    '0 0 1,15 * ?',
    '@daily',
    '@hourly',
    '@every 10m',
    '@every 1h30m',
  ])('accepts %s', (expr) => expect(isValidCronExpression(expr)).toBe(true));

  it.each(['', '   ', '* * * *', '* * * * * * *', '@sometimes', '@every', '@every soon', '0 0 * * $'])(
    'rejects %j',
    (expr) => expect(isValidCronExpression(expr)).toBe(false),
  );
});

describe('duration validation', () => {
  it.each(['30s', '5m', '1h30m', '1.5h', '250ms', '90', '1'])('accepts %s', (d) =>
    expect(isValidDuration(d)).toBe(true),
  );
  it.each(['', '0', '0s', '-5m', '5 minutes', 'm', '1d'])('rejects %j', (d) =>
    expect(isValidDuration(d)).toBe(false),
  );
});

describe('timezone validation', () => {
  it('treats empty as UTC and validates IANA names', () => {
    expect(isValidTimezone('')).toBe(true);
    expect(isValidTimezone('Europe/Berlin')).toBe(true);
    expect(isValidTimezone('UTC')).toBe(true);
    expect(isValidTimezone('Mars/Olympus')).toBe(false);
  });
});

describe('IP whitelist validation', () => {
  it.each(['10.0.0.1', '10.0.0.0/8', '0.0.0.0/0', '::1', '2001:db8::/32'])('accepts %s', (ip) =>
    expect(isValidIpOrCidr(ip)).toBe(true),
  );
  it.each(['256.0.0.1', '10.0.0', '10.0.0.0/33', '10.0.0.0/x', '::1/129', 'host', '1.2.3.4/8/9', 'abcd'])(
    'rejects %s',
    (ip) => expect(isValidIpOrCidr(ip)).toBe(false),
  );

  it('splits lists on commas, spaces and new lines', () => {
    expect(parseList(' 1.1.1.1, 2.2.2.2\n3.3.3.3  ')).toEqual(['1.1.1.1', '2.2.2.2', '3.3.3.3']);
    expect(parseList('')).toEqual([]);
  });
});

describe('parseJsonObject', () => {
  it('returns undefined for empty input', () => {
    expect(parseJsonObject('  ')).toEqual({ ok: true, value: undefined });
  });
  it('parses objects', () => {
    expect(parseJsonObject('{"a":1}')).toEqual({ ok: true, value: { a: 1 } });
  });
  it('reports invalid JSON and non-objects', () => {
    expect(parseJsonObject('{a:1}')).toEqual({ ok: false, error: 'invalidJson' });
    expect(parseJsonObject('[1]')).toEqual({ ok: false, error: 'notAnObject' });
    expect(parseJsonObject('null')).toEqual({ ok: false, error: 'notAnObject' });
    expect(parseJsonObject('"x"')).toEqual({ ok: false, error: 'notAnObject' });
  });
});

describe('validateTriggerForm', () => {
  it('requires workflow and name', () => {
    const errors = validateTriggerForm(form({ workflowId: ' ', name: '' }));
    expect(errors).toMatchObject({ workflowId: 'required', name: 'required' });
    expect(hasErrors(errors)).toBe(true);
  });

  it('validates cron fields', () => {
    expect(validateTriggerForm(form({ type: 'cron', schedule: '' })).schedule).toBe('required');
    expect(validateTriggerForm(form({ type: 'cron', schedule: 'nope' })).schedule).toBe('invalidCron');
    expect(validateTriggerForm(form({ type: 'cron', timezone: 'Nowhere/City' })).timezone).toBe('invalidTimezone');
    expect(hasErrors(validateTriggerForm(form({ type: 'cron', schedule: '@daily', timezone: 'UTC' })))).toBe(false);
  });

  it('validates interval', () => {
    expect(validateTriggerForm(form({ type: 'interval', interval: '' })).interval).toBe('required');
    expect(validateTriggerForm(form({ type: 'interval', interval: 'soon' })).interval).toBe('invalidDuration');
    expect(hasErrors(validateTriggerForm(form({ type: 'interval', interval: '15m' })))).toBe(false);
  });

  it('validates webhook IP whitelist', () => {
    expect(validateTriggerForm(form({ type: 'webhook', ipWhitelist: '10.0.0.1, bad' })).ipWhitelist).toBe('invalidIp');
    expect(hasErrors(validateTriggerForm(form({ type: 'webhook', ipWhitelist: '10.0.0.0/8' })))).toBe(false);
  });

  it('validates event type and filter', () => {
    const errors = validateTriggerForm(form({ type: 'event', eventType: '', eventFilter: '{' }));
    expect(errors).toMatchObject({ eventType: 'required', eventFilter: 'invalidJson' });
    expect(hasErrors(validateTriggerForm(form({ type: 'event', eventType: 'user.created', eventFilter: '{"a":1}' })))).toBe(false);
  });

  it('accepts manual triggers without config and validates input JSON for any type', () => {
    expect(hasErrors(validateTriggerForm(form({ type: 'manual' })))).toBe(false);
    expect(validateTriggerForm(form({ type: 'manual', input: '[]' })).input).toBe('notAnObject');
  });
});

describe('buildTriggerConfig', () => {
  it('builds cron config with optional timezone and mirrors name/description', () => {
    expect(buildTriggerConfig(form({ type: 'cron', schedule: ' 0 9 * * * ', timezone: ' Europe/Berlin ', description: ' d ' }))).toEqual({
      name: 'My trigger',
      description: 'd',
      schedule: '0 9 * * *',
      timezone: 'Europe/Berlin',
    });
    expect(buildTriggerConfig(form({ type: 'cron', name: '' }))).toEqual({ schedule: '0 * * * *' });
  });

  it('sends whole-second intervals as numbers and durations as strings', () => {
    expect(buildTriggerConfig(form({ type: 'interval', interval: '90' })).interval).toBe(90);
    expect(buildTriggerConfig(form({ type: 'interval', interval: '1h' })).interval).toBe('1h');
  });

  it('builds webhook config only with the values that were set', () => {
    expect(buildTriggerConfig(form({ type: 'webhook' }))).toEqual({ name: 'My trigger' });
    expect(buildTriggerConfig(form({ type: 'webhook', secret: 's3cret', ipWhitelist: '1.1.1.1\n2.2.2.0/24' }))).toEqual({
      name: 'My trigger',
      secret: 's3cret',
      ip_whitelist: ['1.1.1.1', '2.2.2.0/24'],
    });
  });

  it('builds event config with source and filter', () => {
    expect(
      buildTriggerConfig(form({ type: 'event', eventType: 'order.paid', eventSource: 'shop', eventFilter: '{"plan":"pro"}' })),
    ).toEqual({ name: 'My trigger', event_type: 'order.paid', source: 'shop', filter: { plan: 'pro' } });
    expect(buildTriggerConfig(form({ type: 'event', eventType: 'x' }))).toEqual({ name: 'My trigger', event_type: 'x' });
  });

  it('adds default input and drops stale keys of other types while keeping unknown keys', () => {
    const config = buildTriggerConfig(form({ type: 'manual', input: '{"k":"v"}' }), {
      schedule: 'old',
      secret: 'old',
      custom_flag: true,
    });
    expect(config).toEqual({ name: 'My trigger', input: { k: 'v' }, custom_flag: true });
  });
});

describe('buildCreateRequest / buildUpdateRequest', () => {
  it('creates the POST /triggers payload', () => {
    expect(buildCreateRequest(form({ type: 'interval', interval: '5m', enabled: false }))).toEqual({
      workflow_id: WF,
      name: 'My trigger',
      type: 'interval',
      config: { name: 'My trigger', interval: '5m' },
      enabled: false,
    });
    expect(buildCreateRequest(form({ type: 'manual', description: 'why' })).description).toBe('why');
  });

  it('preserves unknown config keys only when the type is unchanged', () => {
    const original = trigger({ config: { schedule: '0 0 * * *', extra: 1 } });
    const same = buildUpdateRequest(form({ type: 'cron', schedule: '@daily' }), original);
    expect(same).toEqual({
      name: 'My trigger',
      description: '',
      type: 'cron',
      config: { name: 'My trigger', schedule: '@daily', extra: 1 },
      enabled: true,
    });
    const changed = buildUpdateRequest(form({ type: 'manual' }), original);
    expect(changed.config).toEqual({ name: 'My trigger' });
  });
});

describe('triggerToForm', () => {
  it('round-trips every type', () => {
    const cases: Trigger[] = [
      trigger({ type: 'cron', config: { schedule: '@daily', timezone: 'UTC' } }),
      trigger({ type: 'interval', config: { interval: 30 } }),
      trigger({ type: 'interval', config: { interval: '2h' } }),
      trigger({ type: 'webhook', config: { secret: 's', ip_whitelist: ['1.1.1.1', '2.2.2.2'] } }),
      trigger({ type: 'event', config: { event_type: 'a.b', source: 'src', filter: { x: 1 } } }),
      trigger({ type: 'manual', config: { input: { a: 1 } }, enabled: false, description: 'desc' }),
    ];
    for (const t of cases) {
      const values = triggerToForm(t);
      expect(values.workflowId).toBe(WF);
      expect(values.type).toBe(t.type);
      expect(values.enabled).toBe(t.enabled);
      const request = buildUpdateRequest(values, t);
      expect(request.config).toEqual({ name: 'Nightly', ...(t.description ? { description: t.description } : {}), ...t.config });
    }
  });

  it('fills form fields from config', () => {
    const values = triggerToForm(
      trigger({ name: '', type: 'webhook', config: { name: 'cfg-name', ip_whitelist: ['1.1.1.1', '2.2.2.2'], filter: {} } }),
    );
    expect(values.name).toBe('cfg-name');
    expect(values.ipWhitelist).toBe('1.1.1.1\n2.2.2.2');
    expect(values.eventFilter).toBe('');
    expect(values.schedule).toBe('0 * * * *'); // default kept for non-cron types
    expect(values.interval).toBe('5m');
  });

  it('leaves the primary field empty when a trigger of that type lacks it', () => {
    expect(triggerToForm(trigger({ type: 'cron', config: {} })).schedule).toBe('');
    expect(triggerToForm(trigger({ type: 'interval', config: {} })).interval).toBe('');
  });

  it('falls back to manual for unknown types and handles a missing config', () => {
    const values = triggerToForm(trigger({ type: 'schedule' as never, config: undefined as never }));
    expect(values.type).toBe('manual');
    expect(values.input).toBe('');
  });
});

describe('getWebhookUrl', () => {
  it('joins a relative API base with the page origin', () => {
    expect(getWebhookUrl('t1', '/api/v1', 'https://app.example.com/')).toBe('https://app.example.com/api/v1/webhooks/t1');
    expect(getWebhookUrl('t1', 'api/v1/', 'http://localhost:3435')).toBe('http://localhost:3435/api/v1/webhooks/t1');
  });
  it('uses an absolute API base as-is', () => {
    expect(getWebhookUrl('t1', 'https://api.example.com/api/v1', 'https://app')).toBe(
      'https://api.example.com/api/v1/webhooks/t1',
    );
  });
});

describe('summarizeTriggerConfig', () => {
  it('summarizes each type', () => {
    expect(summarizeTriggerConfig({ type: 'cron', config: { schedule: '@daily' } })).toBe('@daily');
    expect(summarizeTriggerConfig({ type: 'cron', config: { schedule: '@daily', timezone: 'UTC' } })).toBe('@daily (UTC)');
    expect(summarizeTriggerConfig({ type: 'interval', config: { interval: 30 } })).toBe('every 30s');
    expect(summarizeTriggerConfig({ type: 'interval', config: { interval: '5m' } })).toBe('every 5m');
    expect(summarizeTriggerConfig({ type: 'interval', config: {} })).toBe('');
    expect(summarizeTriggerConfig({ type: 'event', config: { event_type: 'a' } })).toBe('a');
    expect(summarizeTriggerConfig({ type: 'event', config: { event_type: 'a', source: 'b' } })).toBe('a from b');
    expect(summarizeTriggerConfig({ type: 'webhook', config: {} })).toBe('POST');
    expect(summarizeTriggerConfig({ type: 'webhook', config: { secret: 'x' } })).toBe('POST, HMAC-signed');
    expect(summarizeTriggerConfig({ type: 'manual', config: undefined as never })).toBe('');
  });
});

describe('filterTriggersByStatus', () => {
  const list = [{ enabled: true }, { enabled: false }, { enabled: true }];
  it('filters by enabled flag', () => {
    expect(filterTriggersByStatus(list)).toHaveLength(3);
    expect(filterTriggersByStatus(list, 'enabled')).toHaveLength(2);
    expect(filterTriggersByStatus(list, 'disabled')).toEqual([{ enabled: false }]);
  });
});

describe('relativeTime', () => {
  const now = new Date('2026-05-01T12:00:00Z');
  it('buckets differences', () => {
    expect(relativeTime('2026-05-01T12:00:30Z', now)).toEqual({ unit: 'now' });
    expect(relativeTime('2026-05-01T12:10:00Z', now)).toEqual({ unit: 'minutes', value: 10, past: false });
    expect(relativeTime('2026-05-01T09:00:00Z', now)).toEqual({ unit: 'hours', value: 3, past: true });
    expect(relativeTime(new Date('2026-05-04T12:00:00Z'), now)).toEqual({ unit: 'days', value: 3, past: false });
  });
  it('defaults to the current time', () => {
    expect(relativeTime(new Date())).toEqual({ unit: 'now' });
  });
});
