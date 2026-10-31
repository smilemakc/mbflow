import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { triggerService } from './triggerService';
import { mockApi, type ApiMock } from '@/test/apiMock';
import type { Trigger } from '@/types/triggers';

const trigger = (id: string, overrides: Partial<Trigger> = {}): Trigger => ({
  id,
  workflow_id: 'wf-1',
  name: `t-${id}`,
  type: 'cron',
  config: { schedule: '@daily' },
  enabled: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  ...overrides,
});

describe('triggerService', () => {
  let api: ApiMock;
  beforeEach(() => {
    api = mockApi();
  });
  afterEach(() => api.restore());

  it('lists triggers from the paginated envelope and forwards filters', async () => {
    api.on('GET', '/triggers', { data: { data: [trigger('1')], total: 1, limit: 10, offset: 0 } });

    const result = await triggerService.getTriggers({ workflow_id: 'wf-1', type: 'cron', limit: 10, offset: 0 });

    expect(result).toEqual({ triggers: [trigger('1')], total: 1, limit: 10, offset: 0 });
    expect(api.last()?.params).toEqual({ workflow_id: 'wf-1', type: 'cron', limit: 10, offset: 0 });
  });

  it('omits unset filters and tolerates a sparse body', async () => {
    api.on('GET', '/triggers', { data: {} });
    const result = await triggerService.getTriggers();
    expect(api.last()?.params).toEqual({});
    expect(result).toEqual({ triggers: [], total: 0, limit: 0, offset: 0 });
  });

  it('walks every page in getAllTriggers', async () => {
    api.on('GET', '/triggers', (call) => {
      const offset = Number(call.params?.offset);
      const page = offset === 0 ? [trigger('1'), trigger('2')] : offset === 2 ? [trigger('3'), trigger('4')] : [trigger('5')];
      return { data: { data: page, total: 5, limit: 2, offset } };
    });

    const all = await triggerService.getAllTriggers({ type: 'webhook' }, 2);

    expect(all.map((t) => t.id)).toEqual(['1', '2', '3', '4', '5']);
    expect(api.calls.map((c) => c.params)).toEqual([
      { type: 'webhook', limit: 2, offset: 0 },
      { type: 'webhook', limit: 2, offset: 2 },
      { type: 'webhook', limit: 2, offset: 4 },
    ]);
  });

  it('stops when the total is reached even on a full page', async () => {
    api.on('GET', '/triggers', { data: { data: [trigger('1'), trigger('2')], total: 2 } });
    expect(await triggerService.getAllTriggers({}, 2)).toHaveLength(2);
    expect(api.calls).toHaveLength(1);
  });

  it('keeps fetching full pages while more exist, up to maxPages', async () => {
    let n = 0;
    api.on('GET', '/triggers', () => ({ data: { data: [trigger(String(++n))], total: 100 } }));
    const result = await triggerService.getAllTriggers({}, 1, 3);
    expect(result).toHaveLength(3);
  });

  it('gets, creates, updates and deletes', async () => {
    api
      .on('GET', '/triggers/t1', { data: trigger('t1') })
      .on('POST', '/triggers', (call) => ({ status: 201, data: { ...trigger('new'), ...(call.data as object) } }))
      .on('PUT', '/triggers/t1', (call) => ({ data: { ...trigger('t1'), ...(call.data as object) } }))
      .on('DELETE', '/triggers/t1', { data: { message: 'trigger deleted successfully' } });

    expect((await triggerService.getTrigger('t1')).id).toBe('t1');

    const created = await triggerService.createTrigger({
      workflow_id: 'wf-1',
      name: 'n',
      type: 'interval',
      config: { interval: '5m' },
      enabled: true,
    });
    expect(created.type).toBe('interval');
    expect(api.last()?.data).toEqual({ workflow_id: 'wf-1', name: 'n', type: 'interval', config: { interval: '5m' }, enabled: true });

    const updated = await triggerService.updateTrigger('t1', { enabled: false });
    expect(updated.enabled).toBe(false);
    expect(api.last()?.data).toEqual({ enabled: false });

    await expect(triggerService.deleteTrigger('t1')).resolves.toBeUndefined();
    expect(api.last()).toMatchObject({ method: 'DELETE', url: '/triggers/t1' });
  });

  it('enables, disables and executes', async () => {
    api
      .on('POST', '/triggers/t1/enable', { data: trigger('t1', { enabled: true }) })
      .on('POST', '/triggers/t1/disable', { data: trigger('t1', { enabled: false }) })
      .on('POST', '/triggers/t1/execute', { data: { id: 'e1', workflow_id: 'wf-1', status: 'pending' } });

    expect((await triggerService.enableTrigger('t1')).enabled).toBe(true);
    expect((await triggerService.disableTrigger('t1')).enabled).toBe(false);

    expect(await triggerService.executeTrigger('t1')).toEqual({ id: 'e1', workflow_id: 'wf-1', status: 'pending' });
    expect(api.last()?.data).toEqual({ input: {} });

    await triggerService.executeTrigger('t1', { a: 1 });
    expect(api.last()?.data).toEqual({ input: { a: 1 } });
  });

  it('propagates API errors such as a disabled trigger', async () => {
    api.on('POST', '/triggers/t1/execute', { status: 400, data: { code: 'TRIGGER_DISABLED', message: 'disabled' } });
    await expect(triggerService.executeTrigger('t1')).rejects.toMatchObject({ response: { status: 400 } });
  });
});
