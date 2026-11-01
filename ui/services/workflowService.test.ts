import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { workflowService } from './workflowService';
import { executionService } from './executionService';
import { mockApi, type ApiMock } from '@/test/apiMock';

const wf = (id: string, name = 'Flow') => ({ id, name, description: '', status: 'draft', version: 1, variables: {}, metadata: {}, nodes: [], edges: [], created_at: '', updated_at: '' });

describe('workflowService', () => {
  let api: ApiMock;
  beforeEach(() => {
    api = mockApi();
  });
  afterEach(() => api.restore());

  it('lists, gets, creates and deletes workflows', async () => {
    api
      .on('GET', '/workflows', { data: { data: [wf('1'), wf('2')], total: 2 } })
      .on('GET', '/workflows/1', { data: wf('1', 'One') })
      .on('POST', '/workflows', (c) => ({ data: wf('new', (c.data as { name: string }).name) }))
      .on('DELETE', '/workflows/1', { data: {} });

    expect((await workflowService.getAll()).map((w) => w.id)).toEqual(['1', '2']);
    expect((await workflowService.getById('1')).name).toBe('One');
    expect((await workflowService.create('Made')).name).toBe('Made');
    expect(api.last()?.data).toEqual({ name: 'Made', description: '' });
    await workflowService.create('With desc', 'd');
    expect(api.last()?.data).toEqual({ name: 'With desc', description: 'd' });
    await workflowService.delete('1');
    expect(api.last()?.method).toBe('DELETE');
  });

  it('saves an existing workflow with PUT', async () => {
    api.on('PUT', '/workflows/1', (c) => ({ data: { ...wf('1'), ...(c.data as object) } }));
    const saved = await workflowService.save({ id: '1', name: 'N', nodes: [], edges: [], variables: { k: 'v' } });
    expect(saved.name).toBe('N');
    expect(api.last()?.data).toMatchObject({ id: '1', name: 'N', variables: { k: 'v' } });
  });

  it('creates a new workflow before saving when there is no id', async () => {
    api.on('POST', '/workflows', { data: wf('created') }).on('PUT', '/workflows/created', (c) => ({ data: { ...wf('created'), ...(c.data as object) } }));
    const dag: Parameters<typeof workflowService.save>[0] = { id: 'temp_id' };
    const saved = await workflowService.save(dag);
    expect(saved.id).toBe('created');
    expect(api.calls.map((c) => c.method)).toEqual(['POST', 'PUT']);
    expect(api.calls[0].data).toEqual({ name: 'New Workflow', description: '' });
    expect(api.calls[1].data).toMatchObject({ name: 'Unnamed Workflow', nodes: [], edges: [] });
  });

  it('manages workflow resources', async () => {
    api
      .on('POST', '/workflows/w/resources', { data: {} })
      .on('DELETE', '/workflows/w/resources/r', { data: {} })
      .on('GET', '/workflows/w/resources', { data: { resources: [] } })
      .on('PUT', '/workflows/w/resources/r', { data: {} });

    await workflowService.attachResource('w', 'r', 'files');
    expect(api.last()?.data).toEqual({ resource_id: 'r', alias: 'files', access_type: 'read' });
    await workflowService.attachResource('w', 'r', 'files', 'write');
    expect(api.last()?.data).toMatchObject({ access_type: 'write' });
    await workflowService.detachResource('w', 'r');
    expect((await workflowService.getResources('w')).data).toEqual({ resources: [] });
    await workflowService.updateResourceAlias('w', 'r', 'docs');
    expect(api.last()?.data).toEqual({ alias: 'docs' });
  });
});

describe('executionService', () => {
  let api: ApiMock;
  const exec = { id: 'e1', workflow_id: 'w', status: 'completed', started_at: 's', node_executions: [] };
  beforeEach(() => {
    api = mockApi();
  });
  afterEach(() => api.restore());

  it('triggers and reads executions', async () => {
    api
      .on('POST', '/executions/run/w', { data: exec })
      .on('GET', '/executions/e1', { data: exec })
      .on('POST', '/executions/e1/cancel', { data: {} })
      .on('POST', '/executions/e1/retry', { data: { ...exec, id: 'e2' } });

    expect((await executionService.trigger('w')).id).toBe('e1');
    expect(api.last()?.data).toEqual({ input: {}, async: true });
    await executionService.trigger('w', { a: 1 });
    expect(api.last()?.data).toEqual({ input: { a: 1 }, async: true });
    expect((await executionService.getStatus('e1')).status).toBe('completed');
    expect(await executionService.getRaw('e1')).toEqual(exec);
    await executionService.cancel('e1');
    expect((await executionService.retry('e1')).id).toBe('e2');
  });

  it('maps logs', async () => {
    api.on('GET', '/executions/e1/logs', {
      data: { logs: [{ timestamp: '2026-01-01T00:00:00Z', event_type: 'node.started', level: 'info', message: 'm', data: { node_id: 'n' } }, { timestamp: '2026-01-01T00:00:01Z', event_type: 'x', level: 'error', message: 'm2', data: null }], total: 2 },
    });
    const logs = await executionService.getLogs('e1');
    expect(logs[0]).toMatchObject({ id: '2026-01-01T00:00:00Z_node.started', nodeId: 'n', level: 'info' });
    expect(logs[0].timestamp).toBeInstanceOf(Date);
    expect(logs[1].nodeId).toBeNull();
  });

  it('lists recent and filtered executions', async () => {
    api.on('GET', /^\/executions\?/, { data: { data: [exec], total: 9, limit: 1, offset: 2 } });

    expect(await executionService.getRecent()).toHaveLength(1);
    expect(api.last()?.url).toBe('/executions?limit=5');

    const all = await executionService.getAll({ workflow_id: 'w', status: 'failed', limit: 1, offset: 2, from: 'a', to: 'b' });
    expect(all).toMatchObject({ total: 9, limit: 1, offset: 2 });
    expect(api.last()?.url).toBe('/executions?workflow_id=w&status=failed&limit=1&offset=2&from=a&to=b');

    await executionService.getAll();
    expect(api.last()?.url).toBe('/executions?');
  });
});
