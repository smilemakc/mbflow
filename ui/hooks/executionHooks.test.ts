import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import { executionWS, type ExecutionEventHandler } from '@/services/executionWebSocket';
import { executionService } from '@/services/executionService';
import { workflowService } from '@/services/workflowService';
import { useToastStore } from '@/lib/toast';
import { useDagStore } from '@/store/dagStore';
import type { ExecutionEvent } from '@/types/execution';
import type { AppEdge, AppNode, DAG } from '@/types';
import { useExecutionWS } from './useExecutionWS';
import { useExecutionWebSocket } from './useExecutionWebSocket';
import { useExecutionData } from './useExecutionData';
import { useWorkflowExecutionAnimation } from './useWorkflowExecutionAnimation';
import { useWorkflows } from './useWorkflows';

type ExecEvent = NonNullable<ExecutionEvent['event']>;
const ev = (event_type: string, extra: Partial<ExecEvent> = {}): ExecutionEvent => ({
  type: 'event',
  timestamp: '2024-01-01T00:00:00Z',
  event: {
    event_type,
    execution_id: 'e1',
    workflow_id: 'w1',
    timestamp: '2024-01-01T00:00:01Z',
    status: 'x',
    ...extra,
  },
});

/** Replaces the singleton WS service with spies and captures registered handlers. */
function fakeWS() {
  const handlers: Record<string, ExecutionEventHandler> = {};
  const unsubscribes: ReturnType<typeof vi.fn>[] = [];
  let connected = true;
  const connect = vi.spyOn(executionWS, 'connect').mockImplementation((id, handler) => {
    handlers[id] = handler!;
    const unsub = vi.fn();
    unsubscribes.push(unsub);
    return unsub;
  });
  const isConnected = vi.spyOn(executionWS, 'isConnected').mockImplementation(() => connected);
  const disconnect = vi.spyOn(executionWS, 'disconnect').mockImplementation(() => {});
  return {
    handlers,
    unsubscribes,
    connect,
    isConnected,
    disconnect,
    setConnected(v: boolean) {
      connected = v;
    },
  };
}

const toastTitles = () => useToastStore.getState().toasts.map((t) => `${t.type}:${t.title}`);

beforeEach(() => {
  vi.spyOn(console, 'log').mockImplementation(() => {});
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  vi.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  useToastStore.getState().clearAll();
  vi.useRealTimers();
});

describe('useExecutionWS', () => {
  it('auto-connects, records events, polls status and stops polling on disconnect', () => {
    vi.useFakeTimers();
    const ws = fakeWS();
    const onEvent = vi.fn();
    const { result, rerender } = renderHook((props) => useExecutionWS(props), {
      initialProps: { executionId: 'e1', onEvent } as Parameters<typeof useExecutionWS>[0],
    });
    expect(ws.connect).toHaveBeenCalledWith('e1', expect.any(Function));
    expect(result.current.isConnected).toBe(true);

    const event = ev('node.started');
    act(() => ws.handlers.e1(event));
    expect(result.current.lastEvent).toBe(event);
    expect(onEvent).toHaveBeenCalledWith(event);

    // Updated handler ref is used without reconnecting.
    const onEvent2 = vi.fn();
    rerender({ executionId: 'e1', onEvent: onEvent2 });
    act(() => ws.handlers.e1(event));
    expect(onEvent2).toHaveBeenCalledTimes(1);
    expect(ws.connect).toHaveBeenCalledTimes(1);

    act(() => vi.advanceTimersByTime(1000));
    expect(result.current.isConnected).toBe(true);
    ws.setConnected(false);
    act(() => vi.advanceTimersByTime(1000));
    expect(result.current.isConnected).toBe(false);
    const polls = ws.isConnected.mock.calls.length;
    act(() => vi.advanceTimersByTime(5000));
    expect(ws.isConnected.mock.calls.length).toBe(polls);
  });

  it('reconnects manually, disconnects and cleans up on unmount', () => {
    vi.useFakeTimers();
    const ws = fakeWS();
    const { result, unmount } = renderHook(() => useExecutionWS({ executionId: 'e1', autoConnect: false }));
    expect(ws.connect).not.toHaveBeenCalled();

    act(() => result.current.connect());
    act(() => result.current.connect());
    expect(ws.connect).toHaveBeenCalledTimes(2);
    expect(ws.unsubscribes[0]).toHaveBeenCalled();

    act(() => result.current.disconnect());
    expect(ws.unsubscribes[1]).toHaveBeenCalled();
    expect(ws.disconnect).toHaveBeenCalledWith('e1');
    expect(result.current.isConnected).toBe(false);
    // Second disconnect is a no-op for the subscription.
    act(() => result.current.disconnect());
    expect(ws.unsubscribes[1]).toHaveBeenCalledTimes(1);

    act(() => result.current.connect());
    unmount();
    expect(ws.unsubscribes[2]).toHaveBeenCalled();
  });

  it('does nothing without an execution id', () => {
    const ws = fakeWS();
    const { result, unmount } = renderHook(() => useExecutionWS());
    act(() => result.current.connect());
    act(() => result.current.disconnect());
    unmount();
    expect(ws.connect).not.toHaveBeenCalled();
    expect(ws.disconnect).not.toHaveBeenCalled();
  });
});

describe('useExecutionWebSocket', () => {
  it('maps execution and node events to callbacks and toasts once', () => {
    vi.useFakeTimers();
    const ws = fakeWS();
    const onExecutionUpdate = vi.fn();
    const onNodeUpdate = vi.fn();
    const { result } = renderHook(() =>
      useExecutionWebSocket({ executionId: 'e1', onExecutionUpdate, onNodeUpdate }),
    );
    expect(result.current.wsConnected).toBe(false);
    act(() => result.current.connect('e1'));
    expect(result.current.wsConnected).toBe(true);
    const send = (e: ExecutionEvent) => act(() => ws.handlers.e1(e));

    send({ type: 'control', control: { ping: 1 }, timestamp: '' });
    send({ type: 'event', timestamp: '' });
    expect(onExecutionUpdate).not.toHaveBeenCalled();

    send(ev('node.started', { node_id: 'n1', node_name: 'N', node_type: 'http', input: { a: 1 } }));
    expect(onNodeUpdate).toHaveBeenLastCalledWith('n1', {
      id: 'e1_n1',
      execution_id: 'e1',
      node_id: 'n1',
      node_name: 'N',
      node_type: 'http',
      status: 'running',
      started_at: '2024-01-01T00:00:01Z',
      completed_at: undefined,
      duration: undefined,
      error: undefined,
      input: { a: 1 },
      output: undefined,
    });
    send(ev('node.completed', { node_id: 'n1', duration_ms: 5, output: { b: 2 } }));
    expect(onNodeUpdate.mock.lastCall![1]).toMatchObject({
      status: 'completed',
      completed_at: '2024-01-01T00:00:01Z',
      duration: 5,
      output: { b: 2 },
    });
    send(ev('node.failed', { node_id: 'n2', error: 'bad' }));
    expect(onNodeUpdate.mock.lastCall![1]).toMatchObject({ status: 'failed', error: 'bad' });
    send(ev('node.started'));
    expect(onNodeUpdate).toHaveBeenCalledTimes(3);
    send(ev('something.else'));

    send(ev('execution.completed'));
    send(ev('execution.completed'));
    expect(onExecutionUpdate).toHaveBeenCalledWith({ status: 'completed' });
    send(ev('execution.failed', { error: 'boom' }));
    expect(onExecutionUpdate).toHaveBeenLastCalledWith({ status: 'failed', error: 'boom' });
    expect(toastTitles()).toEqual(['success:Execution completed']);

    // Reconnecting resets the toast guard.
    act(() => result.current.connect('e1'));
    expect(ws.unsubscribes[0]).toHaveBeenCalled();
    send(ev('execution.failed'));
    expect(toastTitles()).toEqual(['success:Execution completed', 'error:Execution failed: Unknown error']);
    send(ev('execution.failed', { error: 'again' }));
    expect(toastTitles()).toHaveLength(2);
  });

  it('ignores node events without a node handler', () => {
    const ws = fakeWS();
    const onExecutionUpdate = vi.fn();
    const { result } = renderHook(() => useExecutionWebSocket({ executionId: 'e1', onExecutionUpdate }));
    act(() => result.current.connect('e1'));
    act(() => ws.handlers.e1(ev('node.completed', { node_id: 'n1' })));
    expect(onExecutionUpdate).not.toHaveBeenCalled();
    act(() => ws.handlers.e1(ev('execution.failed', { error: 'x' })));
    expect(toastTitles()).toEqual(['error:Execution failed: x']);
  });

  it('polls the connection and disconnects on unmount', () => {
    vi.useFakeTimers();
    const ws = fakeWS();
    const { result, unmount } = renderHook(() =>
      useExecutionWebSocket({ executionId: 'e1', onExecutionUpdate: vi.fn() }),
    );
    act(() => result.current.connect('e1'));
    act(() => vi.advanceTimersByTime(2000));
    expect(result.current.wsConnected).toBe(true);
    ws.setConnected(false);
    act(() => vi.advanceTimersByTime(2000));
    expect(result.current.wsConnected).toBe(false);
    const polls = ws.isConnected.mock.calls.length;
    act(() => vi.advanceTimersByTime(10000));
    expect(ws.isConnected.mock.calls.length).toBe(polls);

    unmount();
    expect(ws.unsubscribes[0]).toHaveBeenCalled();
    expect(ws.disconnect).toHaveBeenCalledWith('e1');
  });

  it('disconnect without an execution id only clears local state', () => {
    const ws = fakeWS();
    const { result } = renderHook(() =>
      useExecutionWebSocket({ executionId: undefined, onExecutionUpdate: vi.fn() }),
    );
    act(() => result.current.disconnect());
    expect(ws.disconnect).not.toHaveBeenCalled();
  });
});

describe('useExecutionData', () => {
  const execution = { id: 'e1', workflow_id: 'w1', status: 'running' };

  it('loads the execution and its workflow, and refetches silently', async () => {
    const getStatus = vi.spyOn(executionService, 'getStatus').mockResolvedValue(execution as never);
    const getById = vi.spyOn(workflowService, 'getById').mockResolvedValue({ id: 'w1' } as never);
    const { result } = renderHook(() => useExecutionData('e1'));
    expect(result.current.isLoading).toBe(true);
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.execution).toEqual(execution);
    expect(result.current.workflow).toEqual({ id: 'w1' });
    expect(getStatus).toHaveBeenCalledWith('e1');

    getStatus.mockResolvedValueOnce({ ...execution, status: 'completed' } as never);
    await act(() => result.current.refetch(false));
    expect(result.current.execution?.status).toBe('completed');
    expect(getById).toHaveBeenCalledTimes(1);

    act(() => result.current.setExecution(null));
    expect(result.current.execution).toBeNull();
  });

  it('keeps the execution when the workflow fails and skips missing workflow ids', async () => {
    vi.spyOn(executionService, 'getStatus').mockResolvedValue(execution as never);
    vi.spyOn(workflowService, 'getById').mockRejectedValue(new Error('nope'));
    const { result } = renderHook(() => useExecutionData('e1'));
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.execution).toEqual(execution);
    expect(result.current.workflow).toBeNull();
    expect(result.current.error).toBeNull();
  });

  it('reports errors with a toast only for the loud fetch', async () => {
    const getStatus = vi.spyOn(executionService, 'getStatus').mockRejectedValue(new Error('gone'));
    const { result } = renderHook(() => useExecutionData('e1'));
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBe('gone');
    expect(toastTitles()).toEqual(['error:gone']);

    getStatus.mockRejectedValueOnce('weird');
    await act(() => result.current.refetch(false));
    expect(result.current.error).toBe('Failed to load execution details');
    expect(toastTitles()).toHaveLength(1);
  });

  it('does not fetch without an id', async () => {
    const getStatus = vi.spyOn(executionService, 'getStatus');
    const { result } = renderHook(() => useExecutionData(undefined));
    await act(async () => {});
    expect(getStatus).not.toHaveBeenCalled();
    expect(result.current.isLoading).toBe(true);
  });
});

describe('useWorkflowExecutionAnimation', () => {
  const node = (id: string, data: Record<string, unknown> | null = { label: id }) =>
    ({ id, position: { x: 0, y: 0 }, data }) as unknown as AppNode;
  const edge = (source: string, target: string) => ({ id: `${source}-${target}`, source, target }) as AppEdge;

  beforeEach(() => {
    useDagStore.getState().loadGraph(
      [node('a'), node('b'), node('c'), node('x', null)],
      [edge('a', 'b'), edge('b', 'c')],
    );
  });

  const data = (id: string) => useDagStore.getState().nodes.find((n) => n.id === id)!.data as Record<string, unknown>;
  const edgeOf = (id: string) => useDagStore.getState().edges.find((e) => e.id === id)!;

  it('animates nodes and edges through an execution lifecycle', () => {
    const { result } = renderHook(() => useWorkflowExecutionAnimation());
    const send = (e: ExecutionEvent) => act(() => result.current.handleExecutionEvent(e));

    send({ type: 'control', timestamp: '' });
    send({ type: 'event', timestamp: '' });

    send(ev('node.started', { node_id: 'a', input: { q: 1 } }));
    expect(data('a')).toMatchObject({ executionState: 'running', animated: true, executionInput: { q: 1 } });

    send(ev('node.completed', { node_id: 'a', output: { r: 2 } }));
    expect(data('a')).toMatchObject({ executionState: 'completed', animated: false, executionOutput: { r: 2 } });
    expect(edgeOf('a-b')).toMatchObject({ animated: true, style: { stroke: '#3b82f6' } });

    send(ev('node.started', { node_id: 'b' }));
    send(ev('node.completed', { node_id: 'b' }));
    expect(edgeOf('a-b')).toMatchObject({ animated: false, style: { stroke: '#10b981' } });
    expect(edgeOf('b-c').animated).toBe(true);

    send(ev('node.started', { node_id: 'c' }));
    send(ev('node.failed', { node_id: 'c', error: 'kaput' }));
    expect(data('c')).toMatchObject({ executionState: 'failed', animated: false, executionError: 'kaput' });
    send(ev('node.failed', { node_id: 'b' }));
    expect(data('b').executionState).toBe('failed');

    send(ev('node.started', { node_id: 'a' }));
    expect(data('a').animated).toBe(true);
    send(ev('execution.failed'));
    expect(data('a').animated).toBe(false);
    expect(useDagStore.getState().edges.every((e) => !e.animated)).toBe(true);

    send(ev('execution.started'));
    expect(data('a')).toMatchObject({ executionState: undefined, animated: false });
    expect(edgeOf('b-c').style).toBeUndefined();
  });

  it('does not leak edge animation into undo history', () => {
    const { result } = renderHook(() => useWorkflowExecutionAnimation());
    const historyEdge = useDagStore.getState().history[0].edges[0];
    act(() => result.current.animateEdge('a', 'b', 'active'));
    expect(edgeOf('a-b').animated).toBe(true);
    expect(historyEdge.animated).toBeUndefined();
    act(() => result.current.animateEdge('nope', 'b', 'active'));
  });

  it('tracks sub-workflow progress and ignores events for unknown nodes', () => {
    const { result } = renderHook(() => useWorkflowExecutionAnimation());
    const send = (e: ExecutionEvent) => act(() => result.current.handleExecutionEvent(e));

    send(ev('sub_workflow.progress', { node_id: 'a', sub_workflow_total: 10, sub_workflow_completed: 3, sub_workflow_failed: 1 }));
    expect(data('a').subWorkflowProgress).toEqual({ total: 10, completed: 3, failed: 1, running: 6 });
    send(ev('sub_workflow.item_failed', { node_id: 'a' }));
    expect(data('a').subWorkflowProgress).toEqual({ total: 0, completed: 0, failed: 0, running: 0 });
    send(ev('sub_workflow.item_completed', { node_id: 'b', sub_workflow_total: 1, sub_workflow_completed: 2 }));
    expect(data('b').subWorkflowProgress).toMatchObject({ running: 0 });
    send(ev('sub_workflow.progress', { node_id: 'x' }));
    send(ev('sub_workflow.progress'));
    expect(useDagStore.getState().nodes.find((n) => n.id === 'x')!.data).toBeNull();

    send(ev('node.started', { node_id: 'ghost', input: { a: 1 } }));
    send(ev('node.completed', { node_id: 'ghost', output: { a: 1 } }));
    send(ev('node.failed', { node_id: 'ghost', error: 'e' }));
    send(ev('node.started'));
    send(ev('node.completed'));
    send(ev('node.failed'));
    expect(console.warn).toHaveBeenCalledWith('[Animation] Node not found:', 'ghost');

    send(ev('execution.completed'));
    act(() => result.current.resetAnimations());
    expect(data('a').executionState).toBeUndefined();
  });
});

describe('useWorkflows', () => {
  const wf = (id: string, nodes: unknown[] = [], edges: unknown[] = []) =>
    ({ id, name: `WF ${id}`, description: 'd', nodes, edges }) as unknown as DAG;
  const translations = {
    cloneFailed: 'Clone failed',
    cloneMessage: 'cm',
    deleteFailed: 'Delete failed',
    deleteMessage: 'dm',
  };

  it('loads on mount and reports load errors', async () => {
    const getAll = vi.spyOn(workflowService, 'getAll').mockResolvedValue([wf('1')] as never);
    const { result } = renderHook(() => useWorkflows());
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.workflows.map((w) => w.id)).toEqual(['1']);

    getAll.mockRejectedValueOnce(new Error('x'));
    await act(() => result.current.loadWorkflows());
    expect(result.current.error).toBe('Failed to load workflows. Please try again.');
  });

  it('does not autoload when disabled', () => {
    const getAll = vi.spyOn(workflowService, 'getAll');
    const { result } = renderHook(() => useWorkflows({ autoLoad: false }));
    expect(getAll).not.toHaveBeenCalled();
    expect(result.current.isLoading).toBe(true);
  });

  it('clones workflows, copying the graph only when non-empty', async () => {
    vi.spyOn(workflowService, 'getAll').mockResolvedValue([]);
    const create = vi.spyOn(workflowService, 'create').mockResolvedValue(wf('new') as never);
    const save = vi.spyOn(workflowService, 'save').mockResolvedValue(wf('new') as never);
    const { result } = renderHook(() => useWorkflows({ autoLoad: false }));

    await act(() => result.current.cloneWorkflow(wf('1')));
    expect(create).toHaveBeenCalledWith('WF 1 (Copy)', 'd');
    expect(save).not.toHaveBeenCalled();

    await act(() => result.current.cloneWorkflow(wf('2', [{ id: 'n' }])));
    expect(save).toHaveBeenCalledWith({ id: 'new', name: 'WF new', description: 'd', nodes: [{ id: 'n' }], edges: [] });
    await act(() => result.current.cloneWorkflow(wf('3', [], [{ id: 'e' }])));
    expect(save).toHaveBeenCalledTimes(2);
  });

  it('reports clone and delete failures through toasts and callbacks', async () => {
    vi.spyOn(workflowService, 'getAll').mockResolvedValue([]);
    const err = new Error('fail');
    vi.spyOn(workflowService, 'create').mockRejectedValue(err);
    const del = vi.spyOn(workflowService, 'delete').mockRejectedValueOnce(err).mockResolvedValue(undefined as never);
    const onCloneError = vi.fn();
    const onDeleteError = vi.fn();
    const { result } = renderHook(() => useWorkflows({ autoLoad: false, onCloneError, onDeleteError, translations }));

    await act(() => result.current.cloneWorkflow(wf('1')));
    await act(() => result.current.deleteWorkflow('1'));
    expect(onCloneError).toHaveBeenCalledWith(err);
    expect(onDeleteError).toHaveBeenCalledWith(err);
    expect(toastTitles()).toEqual(['error:Clone failed', 'error:Delete failed']);

    await act(() => result.current.deleteWorkflow('1'));
    expect(del).toHaveBeenCalledTimes(2);
    expect(onDeleteError).toHaveBeenCalledTimes(1);
  });

  it('swallows failures silently without translations or callbacks', async () => {
    vi.spyOn(workflowService, 'create').mockRejectedValue(new Error('x'));
    vi.spyOn(workflowService, 'delete').mockRejectedValue(new Error('x'));
    const { result } = renderHook(() => useWorkflows({ autoLoad: false }));
    await act(() => result.current.cloneWorkflow(wf('1')));
    await act(() => result.current.deleteWorkflow('1'));
    expect(toastTitles()).toEqual([]);
  });
});
