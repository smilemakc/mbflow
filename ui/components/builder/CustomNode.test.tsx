import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { ReactFlowProvider, type NodeProps } from 'reactflow';
import { CustomNode } from './CustomNode';
import { OpenAPINodeSummary } from './OpenAPINodeSummary';
import { useApiClientStore } from '@/store/apiClientStore';
import { NodeStatus, NodeType, type NodeData } from '@/types';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { resetBuilderStores } from '@/test/builderFlowMocks';

const initialApiClients = useApiClientStore.getState();

const renderNode = (data: Partial<NodeData>, selected = false) =>
  render(
    <ReactFlowProvider>
      <CustomNode {...({ id: 'n', type: 'custom', data: { label: 'My node', ...data }, selected } as unknown as NodeProps<NodeData>)} />
    </ReactFlowProvider>,
  );

const client = (operations: { id: string; method: string; path: string }[], id = 'c1') => ({
  id,
  name: 'Pets',
  operations,
  updated_at: '1',
});

let api: ApiMock;
beforeEach(() => {
  resetBuilderStores();
  useApiClientStore.setState(initialApiClients, true);
  api = mockApi();
});
afterEach(() => api.restore());

describe('CustomNode', () => {
  it('gives conditional nodes separate true and false outputs', () => {
    const { container } = renderNode({ type: NodeType.CONDITIONAL });
    const sources = container.querySelectorAll('.react-flow__handle.source');
    expect([...sources].map((h) => h.getAttribute('data-handleid'))).toEqual(['true', 'false']);
    expect(screen.getByText('true')).toBeInTheDocument();
    expect(screen.getByText('false')).toBeInTheDocument();
  });

  it('gives other nodes a single output', () => {
    const { container } = renderNode({ type: NodeType.HTTP });
    expect(container.querySelectorAll('.react-flow__handle.source')).toHaveLength(1);
  });

  it.each([...Object.values(NodeType).filter((t) => t !== NodeType.OPENAPI), 'unknown'])('renders a %s node', (type) => {
    const { container } = renderNode({ type: type as NodeType, description: 'Does work' });
    expect(screen.getByText('My node')).toBeInTheDocument();
    expect(screen.getByText(type)).toBeInTheDocument();
    expect(screen.getByText('Does work')).toBeInTheDocument();
    expect(container.querySelectorAll('.react-flow__handle')).toHaveLength(type === NodeType.CONDITIONAL ? 3 : 2);
  });

  it('falls back to defaults without type and description', () => {
    renderNode({});
    expect(screen.getByText('Node')).toBeInTheDocument();
    expect(screen.getByText('No description provided.')).toBeInTheDocument();
  });

  it.each([
    [NodeStatus.RUNNING, '.animate-spin', 'ring-blue-400'],
    [NodeStatus.SUCCESS, '.text-green-500', 'ring-green-500/50'],
    [NodeStatus.ERROR, '.text-red-500', 'ring-red-500/50'],
    [NodeStatus.SKIPPED, '.rounded-full', ''],
  ])('shows the %s status', (status, iconSelector, ring) => {
    const { container } = renderNode({ type: NodeType.HTTP, status });
    expect(container.querySelector(iconSelector)).not.toBeNull();
    if (ring) expect((container.firstChild as HTMLElement).className).toContain(ring);
  });

  it('shows nothing for idle/pending/unknown statuses and highlights selection', () => {
    const { container } = renderNode({ type: NodeType.HTTP, status: NodeStatus.PENDING }, true);
    expect(container.querySelector('.animate-spin')).toBeNull();
    expect((container.firstChild as HTMLElement).className).toContain('ring-blue-500');
    renderNode({ type: NodeType.HTTP, status: 'weird' as NodeStatus });
  });

  it('shows the cron of delay nodes and the last run duration', () => {
    renderNode({ type: NodeType.DELAY, config: { cron: '*/5 * * * *' }, lastRun: { duration: 42 } as never, status: NodeStatus.SUCCESS });
    expect(screen.getByText('*/5 * * * *')).toBeInTheDocument();
    expect(screen.getByText('42ms')).toBeInTheDocument();
  });

  it('hides the last run duration while running', () => {
    renderNode({ type: NodeType.HTTP, lastRun: { duration: 42 } as never, status: NodeStatus.RUNNING });
    expect(screen.queryByText('42ms')).not.toBeInTheDocument();
  });

  it('shows sub-workflow progress', () => {
    renderNode({ type: NodeType.SUB_WORKFLOW, subWorkflowProgress: { total: 4, completed: 1, failed: 1, running: 1 } } as never);
    expect(screen.getByText('2/4')).toBeInTheDocument();
  });

  it('shows the openapi operation summary loaded from the API', async () => {
    api.on('GET', '/api-clients/c1', { data: client([{ id: 'op1', method: 'get', path: '/pets' }]) });
    renderNode({ type: NodeType.OPENAPI, config: { client_id: 'c1', operation_id: 'op1' } });
    expect(await screen.findByText('/pets')).toBeInTheDocument();
    expect(screen.getByText('GET')).toBeInTheDocument();
    expect(screen.getByText('openapi')).toBeInTheDocument();
  });
});

describe('OpenAPINodeSummary', () => {
  it('renders nothing without a client or operation', async () => {
    api.on('GET', '/api-clients/c2', { data: client([], 'c2') });
    const { container } = render(<OpenAPINodeSummary config={{ client_id: 'c2' }} />);
    render(<OpenAPINodeSummary />);
    await waitFor(() => expect(api.calls.filter((c) => c.url === '/api-clients/c2')).toHaveLength(1));
    expect(container).toBeEmptyDOMElement();
  });

  it('renders nothing while the client loads', () => {
    api.on('GET', '/api-clients/pending', () => new Promise(() => {}));
    const { container } = render(<OpenAPINodeSummary config={{ client_id: 'pending', operation_id: 'op1' }} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('warns when the operation is gone from the client', async () => {
    api.on('GET', '/api-clients/c3', { data: client([], 'c3') });
    render(<OpenAPINodeSummary config={{ client_id: 'c3', operation_id: 'op9' }} />);
    expect(await screen.findByText('op9')).toBeInTheDocument();
    expect(screen.getByTitle('Operation "op9" is no longer in the specification. Choose another one.')).toBeInTheDocument();
  });

  it('warns when the client does not exist', async () => {
    api.on('GET', '/api-clients/c4', { status: 404, data: {} });
    render(<OpenAPINodeSummary config={{ client_id: 'c4', operation_id: 'op1' }} />);
    expect(await screen.findByTitle('The API client is not found: it may have been deleted.')).toBeInTheDocument();
  });
});
