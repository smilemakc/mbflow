import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ExecutionDetailPage } from './ExecutionDetailPage';
import { executionService } from '@/services/executionService';
import { workflowService } from '@/services/workflowService';
import { executionWS, type ExecutionEventHandler } from '@/services/executionWebSocket';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';
import { apiError } from '@/test/apiMock';
import { renderPage } from '@/test/pagesRouter';
import type { Execution, ExecutionEvent, NodeExecution } from '@/types/execution';

const ID = 'e1e1e1e1-0000-4000-8000-000000000001';
const WF = 'f0f0f0f0-0000-4000-8000-000000000002';

const node = (node_id: string, overrides: Partial<NodeExecution> = {}): NodeExecution => ({
  id: `ne-${node_id}`,
  execution_id: ID,
  node_id,
  node_name: `Node ${node_id}`,
  status: 'completed',
  started_at: '2026-03-04T10:00:01Z',
  completed_at: '2026-03-04T10:00:02Z',
  ...overrides,
});

const execution = (overrides: Partial<Execution> = {}): Execution => ({
  id: ID,
  workflow_id: WF,
  status: 'completed',
  started_at: '2026-03-04T10:00:00Z',
  completed_at: '2026-03-04T10:00:05Z',
  input: { order_id: 42 },
  output: { ok: true },
  variables: { region: 'eu' },
  triggered_by: 'webhook',
  node_executions: [
    node('second', { started_at: '2026-03-04T10:00:03Z', node_type: 'http' }),
    node('first', { input: { a: 1 }, output: { b: 2 } }),
  ],
  created_at: '',
  updated_at: '',
  ...overrides,
});

const toasts = () => useToastStore.getState().toasts;
const renderDetail = () => renderPage(<ExecutionDetailPage />, { path: '/executions/:id', at: `/executions/${ID}` });
const nodeNames = () => screen.getAllByText(/^Node (first|second|third)$/).map((el) => el.textContent);

describe('ExecutionDetailPage', () => {
  let getStatus: ReturnType<typeof vi.spyOn>;
  let wsHandler: ExecutionEventHandler | undefined;
  let unsubscribe: ReturnType<typeof vi.fn>;
  let connect: ReturnType<typeof vi.spyOn>;
  let disconnect: ReturnType<typeof vi.spyOn>;

  const emit = (event: Partial<NonNullable<ExecutionEvent['event']>> | null, type: ExecutionEvent['type'] = 'event') =>
    act(() => {
      wsHandler!({
        type,
        timestamp: '2026-03-04T10:00:09Z',
        control: type === 'control' ? { action: 'ping' } : undefined,
        event: event
          ? { execution_id: ID, workflow_id: WF, timestamp: '2026-03-04T10:00:09Z', status: '', event_type: '', ...event }
          : undefined,
      });
    });

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    useToastStore.getState().clearAll();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(console, 'log').mockImplementation(() => {});
    getStatus = vi.spyOn(executionService, 'getStatus').mockResolvedValue(execution() as never);
    vi.spyOn(workflowService, 'getById').mockResolvedValue({ id: WF, name: 'Order pipeline' } as never);
    wsHandler = undefined;
    unsubscribe = vi.fn();
    connect = vi.spyOn(executionWS, 'connect').mockImplementation((_id, handler) => {
      wsHandler = handler;
      return unsubscribe;
    });
    disconnect = vi.spyOn(executionWS, 'disconnect').mockImplementation(() => {});
    vi.spyOn(executionWS, 'isConnected').mockReturnValue(true);
  });

  it('shows a not-found state when the execution cannot be loaded', async () => {
    getStatus.mockRejectedValue(new Error('execution missing'));
    const user = userEvent.setup();
    renderDetail();

    expect(await screen.findByText('Execution not found')).toBeInTheDocument();
    expect(toasts().at(-1)).toMatchObject({ type: 'error', title: 'execution missing' });
    await user.click(screen.getByRole('button', { name: 'Back to Executions' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/executions');
  });

  it('renders a completed execution with its nodes in start order', async () => {
    const user = userEvent.setup();
    renderDetail();

    expect(await screen.findByText('Order pipeline')).toBeInTheDocument();
    expect(getStatus).toHaveBeenCalledWith(ID);
    expect(screen.getByRole('heading', { name: /Execution Details/ })).toHaveTextContent('Completed');
    expect(screen.getByText('e1e1e1e1...')).toBeInTheDocument();
    expect(nodeNames()).toEqual(['Node first', 'Node second']);
    expect(screen.queryByText('Live Updates')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(connect).not.toHaveBeenCalled();
    expect(disconnect).toHaveBeenCalledWith(ID);

    // Expand a single node, then all, then collapse all.
    expect(screen.queryByText('Node ID')).toBeNull();
    await user.click(screen.getByText('Node first'));
    expect(screen.getAllByText('Node ID')).toHaveLength(1);
    await user.click(screen.getByText('Node first'));
    expect(screen.queryByText('Node ID')).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Expand All' }));
    expect(screen.getAllByText('Node ID')).toHaveLength(2);
    await user.click(screen.getByRole('button', { name: 'Collapse All' }));
    expect(screen.queryByText('Node ID')).toBeNull();

    // Back the user leaves to the list.
    await user.click(screen.getByRole('button', { name: 'Back to Executions' }));
    expect(screen.getByTestId('location')).toHaveTextContent('/executions');
  });

  it('shows metadata, input, output and variables on the overview tab', async () => {
    const user = userEvent.setup();
    renderDetail();
    await screen.findByText('Order pipeline');

    await user.click(screen.getByRole('button', { name: /Overview/ }));
    expect(screen.getByText('Metadata')).toBeInTheDocument();
    expect(screen.getByText(ID)).toBeInTheDocument();
    expect(screen.getByText(WF)).toBeInTheDocument();
    expect(screen.getByText('webhook')).toBeInTheDocument();
    expect(screen.getByText('Variables')).toBeInTheDocument();
    expect(screen.getByText(/order_id/)).toBeInTheDocument();
    expect(screen.getByText(/region/)).toBeInTheDocument();
    expect(screen.getByText('Completed At').nextElementSibling?.textContent).not.toBe('-');
    expect(screen.queryByText('Expand All')).toBeNull();

    await user.click(screen.getByRole('button', { name: /Node Executions/ }));
    expect(screen.getByRole('button', { name: 'Expand All' })).toBeInTheDocument();
  });

  it('handles a sparse execution: unknown workflow, no nodes, no variables', async () => {
    vi.spyOn(workflowService, 'getById').mockRejectedValue(apiError(404));
    getStatus.mockResolvedValue(
      execution({ node_executions: undefined, variables: {}, completed_at: undefined, triggered_by: undefined, status: 'cancelled' }),
    );
    const user = userEvent.setup();
    renderDetail();

    expect(await screen.findByText('f0f0f0f0')).toBeInTheDocument();
    expect(screen.getByText('No node executions found')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Overview/ }));
    expect(screen.getByText('System')).toBeInTheDocument();
    expect(screen.getByText('Completed At').nextElementSibling).toHaveTextContent('-');
    expect(screen.queryByText('Variables')).toBeNull();
  });

  it('shows the error banner and retries failed executions', async () => {
    getStatus.mockResolvedValue(execution({ status: 'failed', error: 'node second: HTTP 502' }));
    const retry = vi
      .spyOn(executionService, 'retry')
      .mockResolvedValueOnce(execution() as never)
      .mockRejectedValueOnce(apiError(500));
    const user = userEvent.setup();
    renderDetail();

    expect(await screen.findByText('Execution Error')).toBeInTheDocument();
    expect(screen.getByText('node second: HTTP 502')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Retry' }));
    expect(retry).toHaveBeenCalledWith(ID);
    await waitFor(() => expect(toasts().some((t) => t.title === 'Retry started')).toBe(true));
    await waitFor(() => expect(getStatus).toHaveBeenCalledTimes(2));

    await user.click(await screen.findByRole('button', { name: 'Retry' }));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'error', title: 'Failed to retry execution' }));
  });

  it('reloads the execution on refresh', async () => {
    const user = userEvent.setup();
    renderDetail();
    await screen.findByText('Order pipeline');
    getStatus.mockResolvedValue(execution({ status: 'failed' }));
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    await waitFor(() => expect(getStatus).toHaveBeenCalledTimes(2));
    expect(await screen.findByRole('button', { name: 'Retry' })).toBeInTheDocument();
  });

  it('streams live updates for a running execution', async () => {
    getStatus.mockResolvedValue(
      execution({
        status: 'running',
        completed_at: undefined,
        node_executions: [node('first', { status: 'running', completed_at: undefined, input: { a: 1 } })],
      }),
    );
    const view = renderDetail();

    expect(await screen.findByText('Live Updates')).toBeInTheDocument();
    expect(connect).toHaveBeenCalledWith(ID, expect.any(Function));
    expect(screen.getByRole('heading', { name: /Execution Details/ })).toHaveTextContent('Running');

    emit(null, 'control');
    emit({ event_type: 'execution.started' });
    emit(null);

    // A known node finishing keeps its earlier input; a new node is appended.
    emit({ event_type: 'node.completed', node_id: 'first', node_name: 'Node first', output: { b: 2 }, duration_ms: 5 });
    emit({ event_type: 'node.started', node_id: 'third', node_name: 'Node third', node_type: 'llm', timestamp: '2026-03-04T10:00:08Z' });
    expect(nodeNames()).toEqual(['Node first', 'Node third']);
    // Duration is measured from the original start (10:00:01) to the completion event (10:00:09).
    expect(screen.getByText('8.00s')).toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole('button', { name: 'Expand All' }));
    expect(screen.getByText(/"a"/)).toBeInTheDocument();
    expect(screen.getByText(/"b"/)).toBeInTheDocument();
    emit({ event_type: 'node.failed', node_id: 'third', error: 'quota exceeded' });
    expect(screen.getByText('quota exceeded')).toBeInTheDocument();

    emit({ event_type: 'execution.completed' });
    expect(screen.getByRole('heading', { name: /Execution Details/ })).toHaveTextContent('Completed');
    expect(toasts().at(-1)).toMatchObject({ type: 'success', title: 'Execution completed' });
    expect(unsubscribe).toHaveBeenCalled();
    expect(screen.queryByText('Live Updates')).toBeNull();

    view.unmount();
    expect(disconnect).toHaveBeenCalledWith(ID);
  });

  it('reports a failure pushed over the socket', async () => {
    getStatus.mockResolvedValue(execution({ status: 'pending', completed_at: undefined, node_executions: [] }));
    renderDetail();
    await screen.findByText('Live Updates');

    emit({ event_type: 'execution.failed', error: 'timeout' });
    expect(await screen.findByText('Execution Error')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /Execution Details/ })).toHaveTextContent('Failed');
    expect(toasts().at(-1)).toMatchObject({ type: 'error', title: 'Execution failed: timeout' });
  });
});
