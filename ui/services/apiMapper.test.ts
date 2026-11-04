import { describe, expect, it, vi } from 'vitest';
import { MarkerType } from 'reactflow';
import {
  edgeFromApi,
  edgeToApi,
  executionFromApi,
  nodeExecutionFromApi,
  nodeFromApi,
  nodeToApi,
  workflowFromApi,
  workflowToApi,
  type WorkflowApiResponse,
} from './apiMapper';
import { NodeStatus, NodeType, VariableSource, VariableType, type AppEdge, type AppNode } from '@/types';

const apiWorkflow = (overrides: Partial<WorkflowApiResponse> = {}): WorkflowApiResponse => ({
  id: 'wf1',
  name: 'Flow',
  description: 'desc',
  status: 'active',
  version: 3,
  variables: { s: 'x', n: 1, b: true, o: { k: 1 } },
  metadata: { m: 1 },
  nodes: [{ id: 'db-1', node_id: 'http', name: 'HTTP', type: 'http', config: { url: 'u', description: 'd' }, position: { x: 1, y: 2 } }],
  edges: [{ id: 'db-e', edge_id: 'e1', from: 'http', to: 'llm', condition: { expr: 'x' } }],
  resources: [{ resource_id: 'r1', alias: 'files', access_type: 'read', resource_name: 'Files', resource_type: 'file_storage' }],
  created_at: 'c',
  updated_at: 'u',
  ...overrides,
});

describe('workflowFromApi', () => {
  it('maps a full workflow', () => {
    const wf = workflowFromApi(apiWorkflow());
    expect(wf).toMatchObject({ id: 'wf1', name: 'Flow', description: 'desc', version: 3, status: 'active', createdAt: 'c', updatedAt: 'u' });
    expect(wf.variables.map((v) => [v.key, v.type, v.source])).toEqual([
      ['s', VariableType.STRING, VariableSource.GLOBAL],
      ['n', VariableType.NUMBER, VariableSource.GLOBAL],
      ['b', VariableType.BOOLEAN, VariableSource.GLOBAL],
      ['o', VariableType.OBJECT, VariableSource.GLOBAL],
    ]);
    expect(wf.nodes[0].id).toBe('http');
    expect(wf.edges[0]).toMatchObject({ id: 'e1', source: 'http', target: 'llm' });
    expect(wf.resources[0]).toEqual({ resource_id: 'r1', alias: 'files', access_type: 'read', resource_name: 'Files', resource_type: 'file_storage' });
  });

  it('fills defaults for a sparse workflow', () => {
    const wf = workflowFromApi({ id: 'x', name: 'n' } as WorkflowApiResponse);
    expect(wf).toMatchObject({ description: '', version: 1, status: 'draft', nodes: [], edges: [], variables: [], resources: [], metadata: {} });
  });
});

describe('workflowToApi', () => {
  it('maps nodes, edges, variables and resources', () => {
    const node: AppNode = { id: 'n1', type: 'custom', position: { x: 5, y: 6 }, data: { label: 'L', type: NodeType.LLM, description: 'D', status: NodeStatus.IDLE, config: { model: 'm' } } } as AppNode;
    const edge = { id: 'e', source: 'n1', target: 'n2', data: { condition: { a: 1 } } } as AppEdge;
    expect(
      workflowToApi({ id: 'w', name: 'N', nodes: [node], edges: [edge], variables: { a: 'b' }, resources: [{ resource_id: 'r', alias: 'x' } as never] }),
    ).toEqual({
      id: 'w',
      name: 'N',
      description: '',
      nodes: [{ id: 'n1', name: 'L', type: 'llm', config: { model: 'm', description: 'D' }, position: { x: 5, y: 6 } }],
      edges: [{ id: 'e', from: 'n1', to: 'n2', condition: { a: 1 } }],
      variables: { a: 'b' },
      resources: [{ resource_id: 'r', alias: 'x', access_type: 'read' }],
    });
  });

  it('defaults optional collections', () => {
    expect(workflowToApi({ name: 'N', description: 'd', nodes: [], edges: [] })).toMatchObject({ description: 'd', variables: {}, resources: [] });
  });
});

describe('node and edge mappers', () => {
  it('maps legacy aliases and warns on unknown types', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    expect(nodeFromApi({ id: 'a', node_id: '', name: 'W', type: 'webhook', config: {}, position: { x: 0, y: 0 } }).data.type).toBe(NodeType.HTTP);
    expect(nodeFromApi({ id: 'a', node_id: 'b', name: 'S', type: 'split', config: {}, position: { x: 0, y: 0 } }).data.type).toBe(NodeType.CONDITIONAL);
    const unknown = nodeFromApi({ id: 'a', node_id: '', name: 'U', type: 'mystery', config: undefined as never, position: undefined as never });
    expect(unknown).toMatchObject({ id: 'a', position: { x: 100, y: 100 }, data: { description: '', config: {} } });
    expect(warn).toHaveBeenCalledOnce();
  });

  it('defaults node fields when mapping to the API', () => {
    const node = { id: 'n', position: { x: 0, y: 0 }, data: {} } as AppNode;
    expect(nodeToApi(node)).toMatchObject({ name: 'Unnamed Node', type: 'http' });
  });

  it('maps edges both ways', () => {
    const edge = edgeFromApi({ id: 'db', edge_id: '', from: 'a', to: 'b', condition: {} });
    expect(edge).toMatchObject({ id: 'db', source: 'a', target: 'b', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed } });
    expect(edgeToApi({ id: 'e', source: 'a', target: 'b' } as AppEdge)).toEqual({ id: 'e', from: 'a', to: 'b', condition: {} });
  });

  it('keeps the branch and loop of an edge across a save and reload', () => {
    const edge = edgeFromApi({
      id: 'db', edge_id: 'e1', from: 'cond', to: 'b', condition: {}, source_handle: 'false', loop: { max_iterations: 3 },
    });
    expect(edge).toMatchObject({ id: 'e1', sourceHandle: 'false', label: 'false', data: { loop: { max_iterations: 3 } } });
    expect(edgeToApi(edge)).toEqual({
      id: 'e1', from: 'cond', to: 'b', condition: {}, source_handle: 'false', loop: { max_iterations: 3 },
    });
  });
});

describe('execution mappers', () => {
  it('maps executions with optional fields', () => {
    const exec = executionFromApi({
      id: 'e',
      workflow_id: 'w',
      workflow_name: 'Flow',
      status: 'completed',
      started_at: 's',
      completed_at: 'c',
      duration: 42,
      triggered_by: 'cron',
      input: { a: 1 },
      output: { b: 2 },
      error: undefined,
      node_executions: [{ id: 'ne', node_id: 'n', status: 'completed', started_at: 's', input: undefined as never, output: undefined as never }],
    });
    expect(exec).toMatchObject({ workflow_name: 'Flow', duration: 42, triggered_by: 'cron', created_at: 's', updated_at: 'c' });
    expect(exec.node_executions[0]).toMatchObject({ execution_id: '', input: {}, output: {} });
  });

  it('defaults missing collections', () => {
    const exec = executionFromApi({ id: 'e', workflow_id: 'w', status: 'running', started_at: 's' } as never);
    expect(exec).toMatchObject({ input: {}, output: {}, node_executions: [], updated_at: 's' });
    expect(nodeExecutionFromApi({ id: 'x', node_id: 'n', status: 'failed', started_at: 's', input: { i: 1 }, output: { o: 1 }, error: 'e' })).toMatchObject({ error: 'e', input: { i: 1 } });
  });
});
