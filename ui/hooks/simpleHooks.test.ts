import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import type { Edge, Node } from 'reactflow';
import dagre from 'dagre';
import { useArrayToString } from './useArrayToString';
import { useAutoLayout } from './useAutoLayout';
import { useAutoSave } from './useAutoSave';
import { useCredentialForm } from './useCredentialForm';
import { useNodeConfig } from './useNodeConfig';
import { useSafeConfig } from './useSafeConfig';
import { useTemplateFilter } from './useTemplateFilter';
import { useToast } from './useToast';
import { useWorkflowFilters } from './useWorkflowFilters';
import { useDagStore } from '@/store/dagStore';
import { useToastStore } from '@/lib/toast';
import type { DAG } from '@/types';
import type { WorkflowTemplate } from '@/data/workflowTemplates';

describe('useArrayToString', () => {
  it('joins arrays and parses strings back, trimming blanks', () => {
    const { result, rerender } = renderHook(({ arr, sep }) => useArrayToString(arr, sep), {
      initialProps: { arr: ['a', 'b'] as string[] | undefined, sep: undefined as string | undefined },
    });
    expect(result.current[0]).toBe('a\nb');
    expect(result.current[1](' x \n\n y ')).toEqual(['x', 'y']);
    expect(result.current[1]('   ')).toEqual([]);

    rerender({ arr: ['a', 'b'], sep: ',' });
    expect(result.current[0]).toBe('a,b');
    expect(result.current[1]('1, 2,,3')).toEqual(['1', '2', '3']);

    rerender({ arr: undefined, sep: ',' });
    expect(result.current[0]).toBe('');
    rerender({ arr: [], sep: ',' });
    expect(result.current[0]).toBe('');
  });
});

describe('useAutoLayout', () => {
  const nodes = [
    { id: 'a', position: { x: 0, y: 0 }, data: {} },
    { id: 'b', position: { x: 0, y: 0 }, data: {}, width: 100, height: 40 },
    { id: 'c', position: { x: 0, y: 0 }, data: {}, measured: { width: 300, height: 120 } } as Node,
  ] as Node[];
  const edges: Edge[] = [
    { id: 'e1', source: 'a', target: 'b' },
    { id: 'e2', source: 'b', target: 'c' },
  ];

  it('lays nodes out top-to-bottom by default and left-to-right on request', () => {
    const { result } = renderHook(() => useAutoLayout());
    let out!: { nodes: Node[]; edges: Edge[] };
    act(() => {
      out = result.current.applyLayout(nodes, edges);
    });
    expect(out.edges).toBe(edges);
    const [a, b, c] = out.nodes;
    expect(a.position.y).toBeLessThan(b.position.y);
    expect(b.position.y).toBeLessThan(c.position.y);
    expect(result.current.isLayouting).toBe(false);

    act(() => {
      out = result.current.applyLayout(nodes, edges, 'LR');
    });
    expect(out.nodes[0].position.x).toBeLessThan(out.nodes[1].position.x);
    expect(out.nodes[1].position.x).toBeLessThan(out.nodes[2].position.x);
  });

  it('respects custom spacing', () => {
    const tight = renderHook(() => useAutoLayout({ rankSpacing: 10 })).result.current.applyLayout(nodes, edges);
    const loose = renderHook(() => useAutoLayout({ rankSpacing: 500, nodeSpacing: 10 })).result.current.applyLayout(
      nodes,
      edges,
    );
    const gap = (r: { nodes: Node[] }) => r.nodes[1].position.y - r.nodes[0].position.y;
    expect(gap(loose)).toBeGreaterThan(gap(tight));
  });

  it('returns the input unchanged when dagre fails', () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(dagre, 'layout').mockImplementation(() => {
      throw new Error('boom');
    });
    const { result } = renderHook(() => useAutoLayout());
    const out = result.current.applyLayout(nodes, edges);
    expect(out.nodes).toBe(nodes);
    expect(console.error).toHaveBeenCalled();
  });
});

describe('useAutoSave', () => {
  it('mirrors the store saving flag', () => {
    const { result } = renderHook(() => useAutoSave());
    expect(result.current.isSaving).toBe(false);
    act(() => useDagStore.setState({ isSaving: true }));
    expect(result.current.isSaving).toBe(true);
    act(() => useDagStore.setState({ isSaving: false }));
  });
});

describe('useCredentialForm', () => {
  it('validates and builds requests for every credential type', () => {
    const { result } = renderHook(() => useCredentialForm());
    expect(result.current.isValid(null)).toBe(false);
    expect(result.current.isValid('api_key')).toBe(false);

    act(() => {
      result.current.setField('name', '  My cred ');
      result.current.setField('description', ' desc ');
      result.current.setField('provider', ' openai ');
    });
    const base = { name: 'My cred', description: 'desc', provider: 'openai' };

    expect(result.current.isValid('api_key')).toBe(false);
    act(() => result.current.setField('apiKey', 'sk'));
    expect(result.current.isValid('api_key')).toBe(true);
    expect(result.current.buildRequest('api_key')).toEqual({ ...base, api_key: 'sk' });

    expect(result.current.isValid('basic_auth')).toBe(false);
    act(() => {
      result.current.setField('username', 'u');
      result.current.setField('password', 'p');
    });
    expect(result.current.isValid('basic_auth')).toBe(true);
    expect(result.current.buildRequest('basic_auth')).toEqual({ ...base, username: 'u', password: 'p' });

    expect(result.current.isValid('oauth2')).toBe(false);
    act(() => {
      result.current.setField('clientId', 'id');
      result.current.setField('clientSecret', 'secret');
    });
    expect(result.current.isValid('oauth2')).toBe(true);
    expect(result.current.buildRequest('oauth2')).toEqual({
      ...base,
      client_id: 'id',
      client_secret: 'secret',
      access_token: undefined,
      refresh_token: undefined,
      token_url: undefined,
      scopes: undefined,
    });
    act(() => {
      result.current.setField('accessToken', 'at');
      result.current.setField('refreshToken', 'rt');
      result.current.setField('tokenUrl', 'https://t');
      result.current.setField('scopes', 'a b');
    });
    expect(result.current.buildRequest('oauth2')).toMatchObject({
      access_token: 'at',
      refresh_token: 'rt',
      token_url: 'https://t',
      scopes: 'a b',
    });

    expect(result.current.isValid('service_account')).toBe(false);
    act(() => result.current.setField('jsonKey', '{}'));
    expect(result.current.isValid('service_account')).toBe(true);
    expect(result.current.buildRequest('service_account')).toEqual({ ...base, json_key: '{}' });

    expect(result.current.isValid('custom')).toBe(false);
    act(() =>
      result.current.setField('customFields', [
        { key: ' k1 ', value: 'v1' },
        { key: ' ', value: 'ignored' },
        { key: 'k2', value: '' },
      ]),
    );
    expect(result.current.isValid('custom')).toBe(true);
    expect(result.current.buildRequest('custom')).toEqual({ ...base, data: { k1: 'v1' } });

    expect(result.current.isValid('unknown' as never)).toBe(false);
    expect(result.current.buildRequest('unknown' as never)).toEqual(base);

    act(() => result.current.reset());
    expect(result.current.state.name).toBe('');
    expect(result.current.state.customFields).toEqual([{ key: '', value: '' }]);
  });
});

describe('useNodeConfig', () => {
  it('merges updates, propagates them and resyncs when the input changes', () => {
    const onChange = vi.fn();
    const { result, rerender } = renderHook(({ cfg }) => useNodeConfig(cfg, onChange), {
      initialProps: { cfg: { model: 'a', temp: 1 } as Record<string, unknown> },
    });
    expect(result.current[0]).toEqual({ model: 'a', temp: 1 });

    act(() => result.current[1]({ temp: 2 }));
    expect(result.current[0]).toEqual({ model: 'a', temp: 2 });
    expect(onChange).toHaveBeenCalledWith({ model: 'a', temp: 2 });

    // Same content as local state: no reset.
    rerender({ cfg: { model: 'a', temp: 2 } });
    expect(result.current[0]).toEqual({ model: 'a', temp: 2 });

    rerender({ cfg: { model: 'b' } });
    expect(result.current[0]).toEqual({ model: 'b' });
  });
});

describe('useSafeConfig', () => {
  it('fills defaults under the provided config', () => {
    const defaults = { op: 'read', id: '' };
    const { result, rerender } = renderHook(({ cfg }) => useSafeConfig(cfg, defaults), {
      initialProps: { cfg: undefined as Partial<typeof defaults> | undefined },
    });
    expect(result.current).toEqual(defaults);
    rerender({ cfg: { id: 'x' } });
    expect(result.current).toEqual({ op: 'read', id: 'x' });
  });
});

describe('useTemplateFilter', () => {
  const templates = [
    { id: '1', name: 'Slack Alert', description: 'Send a message', category: 'notifications' },
    { id: '2', name: 'ETL', description: 'Load data into Sheets', category: 'data' },
  ] as unknown as WorkflowTemplate[];

  it('filters by category and case-insensitive search on name or description', () => {
    const { result, rerender } = renderHook(({ cat, q }) => useTemplateFilter(templates, cat, q), {
      initialProps: { cat: 'all', q: '' },
    });
    expect(result.current).toHaveLength(2);
    rerender({ cat: 'data', q: '' });
    expect(result.current.map((t) => t.id)).toEqual(['2']);
    rerender({ cat: 'all', q: 'SLACK' });
    expect(result.current.map((t) => t.id)).toEqual(['1']);
    rerender({ cat: 'all', q: 'sheets' });
    expect(result.current.map((t) => t.id)).toEqual(['2']);
    rerender({ cat: 'notifications', q: 'sheets' });
    expect(result.current).toEqual([]);
  });
});

describe('useToast', () => {
  afterEach(() => {
    useToastStore.getState().clearAll();
    vi.useRealTimers();
  });

  it('adds typed toasts with default durations and dismisses them', () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => useToast());
    act(() => {
      result.current.showToast({ title: 'plain' });
      result.current.showToast({ type: 'warning', title: 'custom', message: 'm', duration: 0 });
      result.current.success('s', 'sm');
      result.current.error('e');
      result.current.warning('w');
      result.current.info('i');
    });
    const toasts = result.current.toasts;
    expect(toasts.map((t) => [t.type, t.title, t.duration])).toEqual([
      ['info', 'plain', 5000],
      ['warning', 'custom', 0],
      ['success', 's', 5000],
      ['error', 'e', 7000],
      ['warning', 'w', 5000],
      ['info', 'i', 5000],
    ]);
    expect(toasts[2].message).toBe('sm');

    act(() => result.current.dismissToast(toasts[0].id));
    expect(result.current.toasts).toHaveLength(5);

    act(() => vi.advanceTimersByTime(5000));
    expect(result.current.toasts.map((t) => t.title)).toEqual(['custom', 'e']);
    act(() => vi.advanceTimersByTime(2000));
    expect(result.current.toasts.map((t) => t.title)).toEqual(['custom']);

    act(() => result.current.clearAll());
    expect(result.current.toasts).toEqual([]);
  });
});

describe('useWorkflowFilters', () => {
  const wf = (id: string, name: string, status: DAG['status'], description = '') =>
    ({ id, name, status, description }) as DAG;
  const workflows = [
    wf('1', 'Alpha', 'active', 'first one'),
    wf('2', 'Beta', 'draft'),
    wf('3', 'Gamma', 'active', 'Contains BETA too'),
  ];

  it('filters by search and status and reports active filters', () => {
    const { result } = renderHook(() => useWorkflowFilters(workflows));
    expect(result.current.filteredWorkflows).toHaveLength(3);
    expect(result.current.hasActiveFilters).toBe(false);

    act(() => result.current.setSearchQuery('beta'));
    expect(result.current.filteredWorkflows.map((w) => w.id)).toEqual(['2', '3']);
    expect(result.current.hasActiveFilters).toBe(true);

    act(() => result.current.setStatusFilter('active'));
    expect(result.current.filteredWorkflows.map((w) => w.id)).toEqual(['3']);
    expect(result.current.statusFilter).toBe('active');

    act(() => result.current.setSearchQuery(''));
    expect(result.current.filteredWorkflows.map((w) => w.id)).toEqual(['1', '3']);
    expect(result.current.searchQuery).toBe('');
    expect(result.current.hasActiveFilters).toBe(true);
  });
});
