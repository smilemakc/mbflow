import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useDagStore } from './dagStore';
import { workflowService } from '@/services/workflowService';
import { executionService } from '@/services/executionService';
import { executionWS } from '@/services/executionWebSocket';
import { useToastStore } from '@/lib/toast';
import { NodeStatus, NodeType, type AppEdge, type AppNode } from '@/types';
import type { ExecutionEvent } from '@/types/execution';

const initialState = useDagStore.getState();
const s = () => useDagStore.getState();

const node = (id: string, type: NodeType = NodeType.HTTP): AppNode =>
  ({ id, type: 'custom', position: { x: 0, y: 0 }, data: { label: id, type, status: NodeStatus.IDLE, config: { a: 1 } } }) as AppNode;
const edge = (id: string, source: string, target: string, style?: Record<string, unknown>): AppEdge =>
  ({ id, source, target, style }) as AppEdge;

const event = (event_type: string, extra: Record<string, unknown> = {}): ExecutionEvent => ({
  type: 'event',
  timestamp: 't',
  event: { event_type, execution_id: 'e1', workflow_id: 'w', timestamp: 't', status: '', ...extra },
});

const workflow = (overrides: Record<string, unknown> = {}) =>
  ({
    id: 'w1',
    name: 'Flow',
    description: '',
    version: 1,
    status: 'draft',
    nodes: [node('a')],
    edges: [],
    variables: [{ key: 'k', value: 1 }, { key: 'empty', value: undefined }],
    resources: [{ resource_id: 'r', alias: 'files', access_type: 'read' }],
    tags: [],
    metadata: {},
    createdAt: '',
    updatedAt: '',
    ownerId: '',
    ...overrides,
  }) as Awaited<ReturnType<typeof workflowService.getById>>;

beforeEach(() => {
  useDagStore.setState(initialState, true);
  useToastStore.getState().clearAll();
  vi.spyOn(console, 'log').mockImplementation(() => {});
  vi.spyOn(console, 'error').mockImplementation(() => {});
});

describe('graph editing', () => {
  it('adds nodes with unique ids, default labels and history', () => {
    const first = s().addNode(NodeType.HTTP, { x: 1, y: 2 });
    const second = s().addNode(NodeType.HTTP, { x: 3, y: 4 });
    const custom = s().addNode(NodeType.COMMENT, { x: 0, y: 0 }, { label: 'Note', description: 'D', config: { text: 'x' } });

    expect(first.id).toBe('http');
    expect(second.id).toBe('http_2');
    expect(first.data.label).toBe('HTTP Request');
    expect(custom.data).toMatchObject({ label: 'Note', description: 'D', config: { text: 'x' } });
    expect(s().addNode(NodeType.SUB_WORKFLOW, { x: 0, y: 0 }).data.label).toBe('New Node');
    expect(s().isDirty).toBe(true);
    expect(s().history).toHaveLength(5);
  });

  it('skips taken numeric suffixes', () => {
    useDagStore.setState({ nodes: [node('llm', NodeType.LLM), node('llm_2', NodeType.HTTP)] });
    expect(s().addNode(NodeType.LLM, { x: 0, y: 0 }).id).toBe('llm_3');
  });

  it('duplicates nodes with a copied config', () => {
    useDagStore.setState({ nodes: [node('a')] });
    s().duplicateNode('a');
    const copy = s().nodes[1];
    expect(copy.position).toEqual({ x: 50, y: 50 });
    expect(copy.data.label).toBe('a (Copy)');
    expect(copy.data.config).toEqual({ a: 1 });
    expect(copy.data.config).not.toBe(s().nodes[0].data.config);
    s().duplicateNode('missing');
    expect(s().nodes).toHaveLength(2);
  });

  it('handles a node without config when duplicating', () => {
    useDagStore.setState({ nodes: [{ ...node('a'), data: { ...node('a').data, config: undefined } } as AppNode] });
    s().duplicateNode('a');
    expect(s().nodes[1].data.config).toEqual({});
  });

  it('deletes nodes with their edges', () => {
    useDagStore.setState({ nodes: [node('a'), node('b')], edges: [edge('e', 'a', 'b')], selectedNodeId: 'a' });
    s().deleteNode('a');
    expect(s().nodes.map((n) => n.id)).toEqual(['b']);
    expect(s().edges).toEqual([]);
    expect(s().selectedNodeId).toBeNull();
  });

  it('connects, updates and deletes edges', () => {
    useDagStore.setState({ nodes: [node('a'), node('b')] });
    s().onConnect({ source: 'a', target: 'b', sourceHandle: null, targetHandle: null });
    expect(s().edges).toHaveLength(1);
    const id = s().edges[0].id;
    s().updateEdge(id, { label: 'L' });
    s().updateEdge('other', { label: 'X' });
    expect(s().edges[0].label).toBe('L');
    s().setSelectedEdgeId(id);
    s().deleteEdge(id);
    expect(s().edges).toEqual([]);
    expect(s().selectedEdgeId).toBeNull();
  });

  it('applies React Flow changes', () => {
    useDagStore.setState({ nodes: [node('a')], edges: [edge('e', 'a', 'a')] });
    s().onNodesChange([{ type: 'position', id: 'a', position: { x: 9, y: 9 } }]);
    expect(s().nodes[0].position).toEqual({ x: 9, y: 9 });
    s().onEdgesChange([{ type: 'remove', id: 'e' }]);
    expect(s().edges).toEqual([]);
  });

  it('updates node data and selection', () => {
    useDagStore.setState({ nodes: [node('a'), node('b')] });
    s().updateNodeData('a', { label: 'renamed' });
    expect(s().nodes[0].data.label).toBe('renamed');
    expect(s().nodes[1].data.label).toBe('b');
    s().setSelectedNodeId('b');
    expect(s().selectedNodeId).toBe('b');
  });

  it('loads graphs, applies layouts and supports undo/redo', () => {
    s().loadGraph([node('a')], []);
    expect(s()).toMatchObject({ isDirty: false, historyIndex: 0 });
    s().applyLayoutedNodes([{ ...node('a'), position: { x: 100, y: 100 } }]);
    expect(s().nodes[0].position.x).toBe(100);

    s().undo();
    expect(s().nodes[0].position.x).toBe(0);
    s().undo(); // at the start: no-op
    expect(s().historyIndex).toBe(0);
    s().redo();
    expect(s().nodes[0].position.x).toBe(100);
    s().redo(); // at the end: no-op
    expect(s().historyIndex).toBe(1);
  });
});

describe('variables', () => {
  it('manages workflow variables', () => {
    s().updateVariable('a', '1');
    s().updateVariable('b', '2');
    s().deleteVariable('a');
    expect(s().workflowVariables).toEqual({ b: '2' });
    s().updateWorkflowVariables({ c: '3' });
    expect(s().workflowVariables).toEqual({ c: '3' });
    s().setDAGName('Renamed');
    expect(s().dagName).toBe('Renamed');
  });

  it('lists system, custom and node output variables', () => {
    useDagStore.setState({ nodes: [node('a')] });
    s().addVariable({ id: 'v', key: 'custom', name: 'Custom' } as never);
    const keys = s().getAvailableVariables().map((v) => v.key);
    expect(keys).toEqual(['execution_id', 'timestamp', 'custom', 'step_a.output']);
  });
});

describe('resources', () => {
  it('validates aliases and attaches locally for unsaved workflows', async () => {
    await expect(s().attachResource('r', '1bad')).rejects.toThrow('Invalid alias format');
    await s().attachResource('r', 'files');
    await expect(s().attachResource('r2', 'files')).rejects.toThrow('already in use');
    expect(s().resources).toEqual([{ resource_id: 'r', alias: 'files', access_type: 'read' }]);
    expect(s().getResourceAliases()).toEqual(['files']);
    expect(s().getResourceByAlias('files')?.resource_id).toBe('r');
    expect(s().getResourcesMap()).toEqual({ files: { id: 'r', alias: 'files', type: undefined, name: undefined } });
  });

  it('syncs with the backend and rolls back on failure', async () => {
    useDagStore.setState({ dagId: 'w1' });
    const attach = vi.spyOn(workflowService, 'attachResource').mockResolvedValueOnce({} as never).mockRejectedValueOnce(new Error('x'));
    await s().attachResource('r', 'files', 'write');
    expect(attach).toHaveBeenCalledWith('w1', 'r', 'files', 'write');
    await expect(s().attachResource('r2', 'docs')).rejects.toThrow('x');
    expect(s().resources.map((r) => r.alias)).toEqual(['files']);

    vi.spyOn(workflowService, 'updateResourceAlias').mockResolvedValueOnce({} as never).mockRejectedValueOnce(new Error('y'));
    await s().updateResourceAlias('r', 'renamed');
    expect(s().resources[0].alias).toBe('renamed');
    await expect(s().updateResourceAlias('r', 'again')).rejects.toThrow('y');
    expect(s().resources[0].alias).toBe('renamed');

    vi.spyOn(workflowService, 'detachResource').mockRejectedValueOnce(new Error('z')).mockResolvedValueOnce({} as never);
    await expect(s().detachResource('r')).rejects.toThrow('z');
    expect(s().resources).toHaveLength(1);
    await s().detachResource('r');
    expect(s().resources).toHaveLength(0);
  });

  it('updates and detaches locally without a workflow id', async () => {
    useDagStore.setState({ resources: [{ resource_id: 'r', alias: 'a' }, { resource_id: 'q', alias: 'b' }] as never });
    await s().updateResourceAlias('r', 'c');
    await s().detachResource('q');
    expect(s().resources).toEqual([{ resource_id: 'r', alias: 'c' }]);
  });
});

describe('persistence', () => {
  it('saves new and existing workflows', async () => {
    const save = vi.spyOn(workflowService, 'save').mockResolvedValue(workflow({ id: 'new' }));
    useDagStore.setState({ dagName: 'N', isDirty: true });
    expect(await s().saveDAG()).toBe('new');
    expect(s()).toMatchObject({ dagId: 'new', isDirty: false, isSaving: false });
    expect(s().lastSavedAt).toBeInstanceOf(Date);
    expect(await s().saveDAG()).toBeNull();
    expect(save).toHaveBeenLastCalledWith(expect.objectContaining({ id: 'new', name: 'N' }));
  });

  it('reports save failures', async () => {
    vi.spyOn(workflowService, 'save').mockRejectedValue(new Error('disk full'));
    expect(await s().saveDAG()).toBeNull();
    expect(s().isSaving).toBe(false);
    expect(useToastStore.getState().toasts[0]).toMatchObject({ title: 'Save Failed', message: 'disk full' });
  });

  it('fetches a workflow and converts variables', async () => {
    vi.spyOn(workflowService, 'getById').mockResolvedValueOnce(workflow()).mockResolvedValueOnce(workflow({ variables: undefined, resources: undefined })).mockRejectedValueOnce(new Error('404'));
    await s().fetchWorkflow('w1');
    expect(s()).toMatchObject({ dagId: 'w1', dagName: 'Flow', workflowVariables: { k: '1', empty: '' }, isLoading: false });
    expect(s().resources).toHaveLength(1);
    await s().fetchWorkflow('w1');
    expect(s()).toMatchObject({ workflowVariables: {}, resources: [] });
    await s().fetchWorkflow('w1');
    expect(s().isLoading).toBe(false);
  });

  it('ignores an empty fetch result', async () => {
    vi.spyOn(workflowService, 'getById').mockResolvedValue(null as never);
    await s().fetchWorkflow('w1');
    expect(s().isLoading).toBe(true);
  });

  it('creates workflows and lists them', async () => {
    vi.spyOn(workflowService, 'create').mockResolvedValueOnce(workflow({ id: 'c', name: 'Created' })).mockRejectedValueOnce(new Error('x'));
    await s().createNewWorkflow('Created');
    expect(s()).toMatchObject({ dagId: 'c', dagName: 'Created', nodes: [] });
    await s().createNewWorkflow('fail');
    expect(s().isLoading).toBe(false);

    vi.spyOn(workflowService, 'getAll').mockResolvedValueOnce([workflow()]).mockRejectedValueOnce(new Error('x'));
    expect(await s().fetchWorkflowList()).toEqual([{ id: 'w1', name: 'Flow' }]);
    expect(await s().fetchWorkflowList()).toEqual([]);
  });

  it('resets to a new workflow', () => {
    useDagStore.setState({ dagId: 'x', nodes: [node('a')], isRunning: true });
    s().resetToNew();
    expect(s()).toMatchObject({ dagId: '', dagName: 'New Workflow', nodes: [], isRunning: false });
  });
});

describe('execution', () => {
  afterEach(() => vi.useRealTimers());

  it('refuses to run when saving yields no id', async () => {
    vi.spyOn(workflowService, 'save').mockRejectedValue(new Error('x'));
    await s().runWorkflow();
    expect(useToastStore.getState().toasts.at(-1)?.message).toBe('Please save the workflow first.');
    expect(s().isRunning).toBe(false);
  });

  it('reports when saving throws before a run', async () => {
    const saveDAG = vi.fn().mockRejectedValue(new Error('boom'));
    useDagStore.setState({ saveDAG });
    await s().runWorkflow();
    expect(useToastStore.getState().toasts.at(-1)?.message).toBe('Please save the workflow before running.');
  });

  it('saves first, triggers the execution and connects the socket', async () => {
    vi.useFakeTimers();
    vi.spyOn(workflowService, 'save').mockResolvedValue(workflow({ id: 'w1' }));
    vi.spyOn(executionService, 'trigger').mockResolvedValue({ id: 'e1' } as never);
    vi.spyOn(executionService, 'getRaw').mockResolvedValue({ id: 'e1', status: 'running' } as never);
    const connect = vi.spyOn(executionWS, 'connect').mockReturnValue(() => {});
    vi.spyOn(executionWS, 'isConnected').mockReturnValue(true);
    useDagStore.setState({ nodes: [node('a')], edges: [edge('e', 'a', 'a', { stroke: 'red' })] });

    await s().runWorkflow();

    expect(s()).toMatchObject({ isRunning: true, executionId: 'e1', wsConnected: true });
    expect(s().nodes[0].data.status).toBe(NodeStatus.PENDING);
    expect(s().edges[0].style?.stroke).toBeUndefined();
    expect(s().logs.map((l) => l.message)).toEqual(['Starting workflow execution...', 'Execution started: e1']);
    expect(connect).toHaveBeenCalledWith('e1', expect.any(Function));

    // Events from the socket are routed to handleExecutionEvent.
    connect.mock.calls[0][1]!(event('execution.started'));
    expect(s().logs.at(-1)?.message).toBe('Execution started');

    s().clearExecution();
    await vi.runOnlyPendingTimersAsync();
  });

  it('logs trigger failures', async () => {
    useDagStore.setState({ dagId: 'w1' });
    vi.spyOn(executionService, 'trigger').mockRejectedValueOnce(new Error('no quota')).mockRejectedValueOnce('weird');
    await s().runWorkflow();
    expect(s().isRunning).toBe(false);
    expect(s().logs.at(-1)).toMatchObject({ level: 'error', message: 'Execution error: no quota' });
    await s().runWorkflow();
    expect(s().logs.at(-1)?.message).toBe('Execution error: Unknown error');
  });

  it('marks the socket as disconnected when it drops', async () => {
    vi.useFakeTimers();
    vi.spyOn(executionWS, 'connect').mockReturnValue(() => {});
    vi.spyOn(executionWS, 'isConnected').mockReturnValue(false);
    useDagStore.setState({ isRunning: false });
    s().connectToExecution('e1');
    expect(s().wsConnected).toBe(true);
    await vi.advanceTimersByTimeAsync(2000);
    expect(s().wsConnected).toBe(false);
  });

  it('polls the API and replays the final state when no socket event arrives', async () => {
    vi.useFakeTimers();
    vi.spyOn(executionWS, 'connect').mockReturnValue(() => {});
    vi.spyOn(executionWS, 'isConnected').mockReturnValue(true);
    const disconnect = vi.spyOn(executionWS, 'disconnect').mockImplementation(() => {});
    const getRaw = vi
      .spyOn(executionService, 'getRaw')
      .mockRejectedValueOnce(new Error('transient'))
      .mockResolvedValueOnce({ id: 'e1', status: 'running' } as never)
      .mockResolvedValueOnce({
        id: 'e1',
        workflow_id: 'w',
        status: 'failed',
        error: 'bad',
        node_executions: [
          { node_id: 'a', status: 'completed', output: { x: 1 } },
          { node_id: 'b', status: 'failed', error: 'b broke' },
          { node_id: 'c', status: 'pending' },
        ],
      } as never);
    useDagStore.setState({ nodes: [node('a'), node('b'), node('c')], edges: [edge('ab', 'a', 'b')], isRunning: true, executionId: 'e1' });

    s().connectToExecution('e1');
    await vi.advanceTimersByTimeAsync(750 + 2000 + 2000);

    expect(getRaw).toHaveBeenCalledTimes(3);
    expect(s().nodes.map((n) => n.data.status)).toEqual([NodeStatus.SUCCESS, NodeStatus.ERROR, NodeStatus.IDLE]);
    expect(s().isRunning).toBe(false);
    expect(s().logs.at(-1)?.message).toBe('Workflow failed: bad');
    expect(disconnect).toHaveBeenCalledWith('e1');
  });

  it('stops polling once the run is over or replaced', async () => {
    vi.useFakeTimers();
    vi.spyOn(executionWS, 'connect').mockReturnValue(() => {});
    vi.spyOn(executionWS, 'isConnected').mockReturnValue(true);
    const getRaw = vi.spyOn(executionService, 'getRaw').mockImplementation(async () => {
      // The run is replaced while the request is in flight.
      useDagStore.setState({ executionId: 'other' });
      return { id: 'e1', status: 'completed', node_executions: undefined } as never;
    });
    useDagStore.setState({ isRunning: true, executionId: 'e1' });
    s().connectToExecution('e1');
    await vi.advanceTimersByTimeAsync(750);
    expect(getRaw).toHaveBeenCalledTimes(1);
    expect(s().isRunning).toBe(true);

    await vi.advanceTimersByTimeAsync(10_000);
    expect(getRaw).toHaveBeenCalledTimes(1);
  });

  it('replays a completed execution without node executions', async () => {
    vi.useFakeTimers();
    vi.spyOn(executionWS, 'connect').mockReturnValue(() => {});
    vi.spyOn(executionWS, 'isConnected').mockReturnValue(true);
    vi.spyOn(executionWS, 'disconnect').mockImplementation(() => {});
    vi.spyOn(executionService, 'getRaw').mockResolvedValue({ id: 'e1', workflow_id: 'w', status: 'completed', completed_at: 'c' } as never);
    useDagStore.setState({ isRunning: true, executionId: 'e1' });
    s().connectToExecution('e1');
    await vi.advanceTimersByTimeAsync(750);
    expect(s().isRunning).toBe(false);
    expect(s().logs.at(-1)?.message).toBe('Workflow completed successfully');
  });

  it('clears execution state and disconnects', () => {
    const disconnect = vi.spyOn(executionWS, 'disconnect').mockImplementation(() => {});
    useDagStore.setState({ executionId: 'e1', isRunning: true, nodes: [{ ...node('a'), data: { ...node('a').data, status: NodeStatus.SUCCESS } }], edges: [edge('e', 'a', 'a', { stroke: 'x' })], logs: [{} as never] });
    s().clearExecution();
    expect(disconnect).toHaveBeenCalledWith('e1');
    expect(s()).toMatchObject({ isRunning: false, wsConnected: false, logs: [], executionResults: {} });
    expect(s().nodes[0].data.status).toBe(NodeStatus.IDLE);

    disconnect.mockClear();
    useDagStore.setState({ executionId: null });
    s().clearExecution();
    s().disconnectFromExecution();
    expect(disconnect).not.toHaveBeenCalled();
  });
});

describe('handleExecutionEvent', () => {
  beforeEach(() => {
    vi.spyOn(executionWS, 'disconnect').mockImplementation(() => {});
    useDagStore.setState({
      nodes: [node('a'), node('b'), node('c')],
      edges: [edge('ab', 'a', 'b'), edge('bc', 'b', 'c'), edge('xa', 'x', 'a')],
      executionId: 'e1',
      isRunning: true,
    });
  });

  const handle = (e: ExecutionEvent) => s().handleExecutionEvent(e);
  const status = (id: string) => s().nodes.find((n) => n.id === id)!.data.status;
  const stroke = (id: string) => s().edges.find((e) => e.id === id)!.style?.stroke;

  it('ignores control and malformed messages', () => {
    handle({ type: 'control', control: { a: 1 }, timestamp: 't' });
    handle({ type: 'event', timestamp: 't' });
    expect(s().logs).toEqual([]);
  });

  it('tracks a node from start to completion', () => {
    handle(event('node.started', { node_id: 'b', node_name: 'B', input: { i: 1 } }));
    expect(status('b')).toBe(NodeStatus.RUNNING);
    expect(s().edges.find((e) => e.id === 'ab')?.animated).toBe(true);
    expect(s().executionResults.b).toMatchObject({ status: NodeStatus.RUNNING, inputs: { i: 1 } });

    handle(event('node.completed', { node_id: 'b', node_name: 'B', output: { o: 1 }, duration_ms: 12 }));
    expect(status('b')).toBe(NodeStatus.SUCCESS);
    expect(stroke('ab')).toBe('#10b981');
    expect(stroke('bc')).toBe('#3b82f6');
    expect(s().executionResults.b).toMatchObject({ status: NodeStatus.SUCCESS, outputs: { o: 1 } });
    expect(s().logs.at(-1)?.message).toBe('Node "B" completed (12ms)');
  });

  it('handles nodes without edges or names and defaults', () => {
    useDagStore.setState({ edges: [] });
    handle(event('node.started', { node_id: 'a' }));
    expect(s().executionResults.a.inputs).toEqual({});
    handle(event('node.completed', { node_id: 'a' }));
    expect(s().logs.at(-1)?.message).toBe('Node "a" completed');
    handle(event('node.failed', { node_id: 'a' }));
    expect(s().logs.at(-1)?.message).toBe('Node "a" failed: Unknown error');
  });

  it('marks failures and skips', () => {
    handle(event('node.failed', { node_id: 'b', error: 'boom' }));
    expect(status('b')).toBe(NodeStatus.ERROR);
    expect(stroke('ab')).toBe('#ef4444');
    handle(event('node.skipped', { node_id: 'c' }));
    expect(status('c')).toBe(NodeStatus.SKIPPED);
    expect(s().logs.map((l) => l.level)).toEqual(['error', 'warning']);
  });

  it('ignores node events without a node id', () => {
    for (const type of ['node.started', 'node.completed', 'node.failed', 'node.skipped']) handle(event(type));
    expect(s().logs).toEqual([]);
  });

  it('logs waves', () => {
    handle(event('wave.started', { wave_index: 1, node_count: 2 }));
    handle(event('wave.completed', { wave_index: 1 }));
    expect(s().logs.map((l) => l.message)).toEqual(['Wave 1 started (2 nodes)', 'Wave 1 completed']);
  });

  it('finalizes edge colours on completion', () => {
    handle(event('node.completed', { node_id: 'a' }));
    handle(event('node.completed', { node_id: 'b' }));
    handle(event('node.failed', { node_id: 'c' }));
    handle(event('execution.completed'));
    expect(stroke('ab')).toBe('#22c55e');
    expect(stroke('bc')).toBe('#ef4444');
    expect(stroke('xa')).toBeUndefined();
    expect(s().isRunning).toBe(false);
    expect(executionWS.disconnect).toHaveBeenCalledWith('e1');
  });

  it.each([
    ['execution.failed', 'Workflow failed: Unknown error'],
    ['execution.cancelled', 'Workflow was cancelled'],
  ])('%s keeps finished edge colours and resets in-progress ones', (type, message) => {
    useDagStore.setState({
      edges: [edge('ab', 'a', 'b', { stroke: '#10b981' }), edge('bc', 'b', 'c', { stroke: '#3b82f6' }), edge('xa', 'x', 'a', { stroke: '#ef4444' })],
    });
    handle(event(type));
    expect(stroke('ab')).toBe('#10b981');
    expect(stroke('bc')).toBeUndefined();
    expect(stroke('xa')).toBe('#ef4444');
    expect(s().logs.at(-1)?.message).toBe(message);
    expect(s().isRunning).toBe(false);
  });

  it('ignores unknown event types', () => {
    handle(event('something.else'));
    expect(s().logs).toEqual([]);
  });
});
