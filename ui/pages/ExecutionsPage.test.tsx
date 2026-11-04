import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ExecutionsPage } from './ExecutionsPage';
import { executionService } from '@/services/executionService';
import { workflowService } from '@/services/workflowService';
import { useUIStore } from '@/store/uiStore';
import { apiError } from '@/test/apiMock';
import { renderPage } from '@/test/pagesRouter';
import type { Execution } from '@/types/execution';

const WF1 = 'aaaaaaaa-1111-4111-8111-111111111111';
const WF2 = 'bbbbbbbb-2222-4222-8222-222222222222';

const exec = (id: string, overrides: Partial<Execution> = {}): Execution => ({
  id,
  workflow_id: WF1,
  status: 'completed',
  started_at: '2026-03-04T10:00:00Z',
  created_at: '',
  updated_at: '',
  ...overrides,
});

const EXECUTIONS: Execution[] = [
  exec('11111111-done', { completed_at: '2026-03-04T10:00:00.250Z', triggered_by: 'cron' }),
  exec('22222222-fail', { status: 'failed', completed_at: '2026-03-04T10:00:12.500Z' }),
  exec('33333333-run', { status: 'running', workflow_id: WF2 }),
  exec('44444444-pend', { status: 'pending', workflow_id: 'cccccccc-unknown' }),
  exec('55555555-canc', { status: 'cancelled', completed_at: '2026-03-04T10:03:00Z' }),
  exec('66666666-time', { status: 'timeout' as never }),
];

const rows = () => screen.getAllByRole('row').slice(1);
const rowFor = (shortId: string) => rows().find((r) => within(r).queryByText(shortId))!;

describe('ExecutionsPage', () => {
  let getAll: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    vi.spyOn(console, 'error').mockImplementation(() => {});
    getAll = vi
      .spyOn(executionService, 'getAll')
      .mockResolvedValue({ executions: EXECUTIONS as never, total: EXECUTIONS.length, limit: 20, offset: 0 });
    vi.spyOn(workflowService, 'getAll').mockResolvedValue([
      { id: WF1, name: 'Billing sync' },
      { id: WF2, name: '' },
    ] as never);
  });

  it('renders executions with workflow names, statuses and durations', async () => {
    renderPage(<ExecutionsPage />);
    expect(await screen.findByText('11111111')).toBeInTheDocument();
    expect(getAll).toHaveBeenCalledWith({ limit: 20, offset: 0 });

    const done = rowFor('11111111');
    await within(done).findByText('Billing sync');
    expect(within(done).getByText('Completed')).toBeInTheDocument();
    expect(within(done).getByText('250ms')).toBeInTheDocument();
    expect(within(done).getByText('cron')).toBeInTheDocument();
    expect(within(done).getByText(/Mar 4/)).toBeInTheDocument();

    const failed = rowFor('22222222');
    expect(within(failed).getByText('Failed')).toBeInTheDocument();
    expect(within(failed).getByText('12.5s')).toBeInTheDocument();
    expect(within(failed).getByText('System')).toBeInTheDocument();
    expect(within(failed).getByTitle('Retry')).toBeInTheDocument();

    const running = rowFor('33333333');
    expect(within(running).getByText('Running')).toBeInTheDocument();
    expect(within(running).getByText('-')).toBeInTheDocument();
    // Workflow with an empty name falls back to the short id.
    expect(within(running).getAllByText('bbbbbbbb')).toHaveLength(2);
    expect(within(running).queryByTitle('Retry')).toBeNull();

    expect(within(rowFor('44444444')).getByText('Pending')).toBeInTheDocument();
    expect(within(rowFor('44444444')).getAllByText('cccccccc')).toHaveLength(2);
    expect(within(rowFor('55555555')).getByText('Cancelled')).toBeInTheDocument();
    expect(within(rowFor('55555555')).getByText('3.0m')).toBeInTheDocument();
    expect(within(rowFor('66666666')).getByText('timeout')).toBeInTheDocument();
  });

  it('filters by workflow and status on the server', async () => {
    const user = userEvent.setup();
    renderPage(<ExecutionsPage />);
    await screen.findByText('11111111');
    await screen.findAllByText('Billing sync');

    const [workflowSelect, statusSelect] = screen.getAllByRole('combobox');
    await user.selectOptions(workflowSelect, WF1);
    await waitFor(() => expect(getAll).toHaveBeenLastCalledWith(expect.objectContaining({ workflow_id: WF1 })));

    await user.selectOptions(statusSelect, 'failed');
    await waitFor(() =>
      expect(getAll).toHaveBeenLastCalledWith(expect.objectContaining({ workflow_id: WF1, status: 'failed' })),
    );

    await user.selectOptions(workflowSelect, '');
    await user.selectOptions(statusSelect, '');
    await waitFor(() => expect(getAll).toHaveBeenLastCalledWith({ workflow_id: undefined, status: undefined, limit: 20, offset: 0 }));
  });

  it('pages through results', async () => {
    getAll.mockResolvedValue({ executions: EXECUTIONS, total: 45, limit: 20, offset: 0 });
    const user = userEvent.setup();
    renderPage(<ExecutionsPage />);
    await screen.findByText('11111111');
    expect(screen.getByText('Showing 1 - 20 of 45')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Next' }));
    await waitFor(() => expect(getAll).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 20 })));
    await user.click(await screen.findByRole('button', { name: 'Previous' }));
    await waitFor(() => expect(getAll).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 0 })));
  });

  it('opens execution details from a row or the view button', async () => {
    const user = userEvent.setup();
    const view = renderPage(<ExecutionsPage />);
    await screen.findByText('11111111');

    await user.click(within(rowFor('22222222')).getByTitle('View Details'));
    expect(screen.getByTestId('location')).toHaveTextContent('/executions/22222222-fail');
    view.unmount();

    renderPage(<ExecutionsPage />);
    await user.click(await screen.findByText('11111111'));
    expect(screen.getByTestId('location')).toHaveTextContent('/executions/11111111-done');
  });

  it('retries a failed execution and refreshes', async () => {
    const retry = vi
      .spyOn(executionService, 'retry')
      .mockResolvedValueOnce(exec('new') as never)
      .mockRejectedValueOnce(apiError(500));
    const user = userEvent.setup();
    renderPage(<ExecutionsPage />);
    await screen.findByText('22222222');

    await user.click(within(rowFor('22222222')).getByTitle('Retry'));
    expect(retry).toHaveBeenCalledWith('22222222-fail');
    await waitFor(() => expect(getAll).toHaveBeenCalledTimes(2));

    await user.click(within(rowFor('22222222')).getByTitle('Retry'));
    await waitFor(() => expect(console.error).toHaveBeenCalledWith('Failed to retry execution:', expect.anything()));
    expect(getAll).toHaveBeenCalledTimes(2);
  });

  it('refreshes on demand', async () => {
    const user = userEvent.setup();
    renderPage(<ExecutionsPage />);
    await screen.findByText('11111111');
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    await waitFor(() => expect(getAll).toHaveBeenCalledTimes(2));
  });

  it('shows the empty state and still works when workflows fail to load', async () => {
    vi.spyOn(workflowService, 'getAll').mockRejectedValue(apiError(500));
    getAll.mockResolvedValue({ executions: [], total: 0, limit: 20, offset: 0 });
    renderPage(<ExecutionsPage />);
    expect(await screen.findByText('No executions found')).toBeInTheDocument();
    expect(screen.getByText('No executions found with the current filters')).toBeInTheDocument();
    await waitFor(() => expect(console.error).toHaveBeenCalledWith('Failed to fetch workflows:', expect.anything()));
  });

  it('shows load errors', async () => {
    getAll.mockRejectedValue(apiError(500, { message: 'executions unavailable' }));
    renderPage(<ExecutionsPage />);
    expect(await screen.findByText('executions unavailable')).toBeInTheDocument();
  });
});
