/**
 * ExecutionDetailPage falls back to built-in English labels when a
 * dictionary lacks the execution keys, and renders unknown statuses verbatim.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ExecutionDetailPage } from './ExecutionDetailPage';
import { executionService } from '@/services/executionService';
import { workflowService } from '@/services/workflowService';
import { executionWS, type ExecutionEventHandler } from '@/services/executionWebSocket';
import { useToastStore } from '@/lib/toast';
import { apiError } from '@/test/apiMock';
import { renderPage } from '@/test/pagesRouter';
import type { Execution } from '@/types/execution';

vi.mock('@/store/translations', () => ({ useTranslation: () => ({}) }));

const ID = 'e2e2e2e2-0000-4000-8000-000000000001';

const execution = (overrides: Partial<Execution> = {}): Execution => ({
  id: ID,
  workflow_id: 'wf-1',
  status: 'failed',
  started_at: '2026-03-04T10:00:00Z',
  error: 'boom',
  variables: { a: 1 },
  created_at: '',
  updated_at: '',
  ...overrides,
});

const renderDetail = () => renderPage(<ExecutionDetailPage />, { path: '/executions/:id', at: `/executions/${ID}` });
const toasts = () => useToastStore.getState().toasts;

describe('ExecutionDetailPage without translations', () => {
  let getStatus: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    useToastStore.getState().clearAll();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(console, 'log').mockImplementation(() => {});
    getStatus = vi.spyOn(executionService, 'getStatus').mockResolvedValue(execution() as never);
    vi.spyOn(workflowService, 'getById').mockResolvedValue({ id: 'wf-1', name: 'Flow' } as never);
  });

  it('uses default labels across header, tabs and overview', async () => {
    const retry = vi
      .spyOn(executionService, 'retry')
      .mockResolvedValueOnce(execution() as never)
      .mockRejectedValueOnce(apiError(500));
    const user = userEvent.setup();
    renderDetail();

    expect(await screen.findByRole('heading', { name: /Execution Details/ })).toHaveTextContent('failed');
    expect(screen.getByText('Execution Error')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument();
    expect(screen.getByText('Click on a node to view its inputs and outputs')).toBeInTheDocument();
    expect(screen.getByText('No node executions found')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Expand All' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Collapse All' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Retry' }));
    await waitFor(() => expect(toasts().some((t) => t.title === 'Retry started')).toBe(true));
    await user.click(await screen.findByRole('button', { name: 'Retry' }));
    await waitFor(() => expect(toasts().at(-1)?.title).toBe('Failed to retry execution'));
    expect(retry).toHaveBeenCalledTimes(2);

    await user.click(screen.getByRole('button', { name: /Overview/ }));
    for (const label of ['Metadata', 'Triggered By', 'Completed At', 'Input', 'Output', 'Variables']) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
    await user.click(screen.getByRole('button', { name: /Node Executions/ }));
  });

  it('uses default not-found labels', async () => {
    getStatus.mockRejectedValue('not an Error');
    renderDetail();
    expect(await screen.findByText('Execution not found')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Back to Executions' })).toBeInTheDocument();
    expect(toasts().at(-1)?.title).toBe('Failed to load execution details');
  });

  it('adds streamed nodes to an execution that had none', async () => {
    let handler: ExecutionEventHandler | undefined;
    vi.spyOn(executionWS, 'connect').mockImplementation((_id, h) => {
      handler = h;
      return () => {};
    });
    vi.spyOn(executionWS, 'disconnect').mockImplementation(() => {});
    vi.spyOn(executionWS, 'isConnected').mockReturnValue(true);
    getStatus.mockResolvedValue(execution({ status: 'running', error: undefined, node_executions: undefined }));
    renderDetail();
    await screen.findByText('No node executions found');

    act(() =>
      handler!({
        type: 'event',
        timestamp: '',
        event: {
          event_type: 'node.started',
          execution_id: ID,
          workflow_id: 'wf-1',
          timestamp: '2026-03-04T10:00:01Z',
          status: 'running',
          node_id: 'n1',
        },
      }),
    );
    expect(screen.getByText('n1')).toBeInTheDocument();
  });
});
