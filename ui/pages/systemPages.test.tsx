/**
 * Monitoring page, dashboard and the shared health panel / overview hook.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, render, renderHook, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { MonitoringPage } from './MonitoringPage';
import { DashboardPage } from './DashboardPage';
import { HealthPanel } from '@/components/system/HealthPanel';
import { useSystemOverview } from '@/hooks/useSystemOverview';
import { systemService, type SystemHealth } from '@/services/systemService';
import { executionService } from '@/services/executionService';
import { workflowService } from '@/services/workflowService';
import { useUIStore } from '@/store/uiStore';
import { apiError } from '@/test/apiMock';

const healthy: SystemHealth = {
  apiUp: true,
  ready: true,
  status: 'ready',
  checks: { database: 'ok', redis: 'ok' },
  checkedAt: new Date('2026-01-01T10:00:00Z'),
};

const ago = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString();

const executions = [
  { id: 'e-running-1', workflow_id: 'wf-1', status: 'running', started_at: ago(1) },
  { id: 'e-failed-1', workflow_id: 'wf-1', workflow_name: 'Billing', status: 'failed', started_at: ago(10), completed_at: ago(9), error: 'HTTP 500 from upstream' },
  { id: 'e-done-1', workflow_id: 'wf-2', status: 'completed', started_at: ago(30), duration: 1500, triggered_by: 'cron' },
  { id: 'e-done-2', workflow_id: 'wf-unknown-123', status: 'completed', started_at: ago(60), duration: 500 },
  { id: 'e-cancel', workflow_id: 'wf-2', status: 'cancelled', started_at: ago(90), duration: 1000 },
  { id: 'e-pending', workflow_id: 'wf-2', status: 'pending', started_at: ago(100) },
  { id: 'e-timeout', workflow_id: 'wf-2', status: 'timeout', started_at: ago(120) },
];

const mockExecutions = (list = executions, total = list.length) =>
  vi.spyOn(executionService, 'getAll').mockResolvedValue({ executions: list as never, total, limit: 100, offset: 0 });

const Location = () => <div data-testid="location">{useLocation().pathname}</div>;
const renderAt = (ui: React.ReactElement) =>
  render(
    <MemoryRouter>
      <Routes>
        <Route path="/" element={ui} />
        <Route path="*" element={<Location />} />
      </Routes>
    </MemoryRouter>,
  );

beforeEach(() => {
  useUIStore.setState({ language: 'en' });
});

describe('HealthPanel', () => {
  it('shows a placeholder before the first check', () => {
    render(<HealthPanel health={null} />);
    expect(screen.getByText('Checking...')).toBeInTheDocument();
  });

  it('lists the API and every readiness check', () => {
    render(<HealthPanel health={{ ...healthy, ready: false, status: 'not_ready', checks: { database: 'ok', redis: 'unavailable', queue: 'ok' } }} />);
    expect(within(screen.getByTestId('health-api')).getByText('Operational')).toBeInTheDocument();
    expect(within(screen.getByTestId('health-database')).getByText('Operational')).toBeInTheDocument();
    expect(within(screen.getByTestId('health-redis')).getByText('Unavailable')).toBeInTheDocument();
    expect(within(screen.getByTestId('health-queue')).getByText('queue')).toBeInTheDocument();
  });

  it('reports an unreachable backend and a draining server', () => {
    const { rerender } = render(<HealthPanel health={{ ...healthy, apiUp: false, ready: null, status: 'unreachable', checks: {} }} />);
    expect(within(screen.getByTestId('health-api')).getByText('Unreachable')).toBeInTheDocument();
    expect(within(screen.getByTestId('health-readiness')).getByText('Unreachable')).toBeInTheDocument();

    rerender(<HealthPanel health={{ ...healthy, ready: false, status: 'shutting_down', checks: {} }} />);
    expect(within(screen.getByTestId('health-api')).getByText('Shutting down')).toBeInTheDocument();
  });
});

describe('useSystemOverview', () => {
  afterEach(() => vi.useRealTimers());

  it('loads health and execution stats', async () => {
    vi.spyOn(systemService, 'getSystemHealth').mockResolvedValue(healthy);
    const getAll = mockExecutions(executions, 250);
    const { result } = renderHook(() => useSystemOverview({ sampleSize: 50 }));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(getAll).toHaveBeenCalledWith({ limit: 50 });
    expect(result.current.health).toBe(healthy);
    expect(result.current.stats).toMatchObject({ sampleSize: 7, total: 250, active: 2 });
    expect(result.current.executions).toHaveLength(7);
    expect(result.current.error).toBeNull();
  });

  it('keeps going when one source fails', async () => {
    vi.spyOn(systemService, 'getSystemHealth').mockRejectedValue(new Error('x'));
    vi.spyOn(executionService, 'getAll').mockRejectedValue(apiError(500, { message: 'db down' }));
    const { result } = renderHook(() => useSystemOverview());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.health).toBeNull();
    expect(result.current.error).toBe('db down');
  });

  it('refreshes on an interval and stops after unmount', async () => {
    vi.useFakeTimers();
    const health = vi.spyOn(systemService, 'getSystemHealth').mockResolvedValue(healthy);
    mockExecutions();
    const { unmount } = renderHook(() => useSystemOverview({ refreshMs: 1000 }));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2500);
    });
    expect(health).toHaveBeenCalledTimes(3);
    unmount();
    await vi.advanceTimersByTimeAsync(5000);
    expect(health).toHaveBeenCalledTimes(3);
  });

  it('ignores results that arrive after unmount', async () => {
    let resolve!: (h: SystemHealth) => void;
    vi.spyOn(systemService, 'getSystemHealth').mockReturnValue(new Promise((r) => (resolve = r)));
    mockExecutions();
    const { result, unmount } = renderHook(() => useSystemOverview());
    unmount();
    await act(async () => resolve(healthy));
    expect(result.current.health).toBeNull();
  });
});

describe('MonitoringPage', () => {
  it('shows real health, metrics, breakdown and recent failures', async () => {
    vi.spyOn(systemService, 'getSystemHealth').mockResolvedValue(healthy);
    mockExecutions(executions, 40);
    const user = userEvent.setup();
    renderAt(<MonitoringPage />);

    expect(await screen.findByText('Computed from the latest 7 of 40 executions.')).toBeInTheDocument();
    expect(screen.queryByText(/CPU Usage|Memory Usage|Active Workers|Redis cache response time/)).toBeNull();
    expect(screen.getByText(/Last checked/)).toBeInTheDocument();
    expect(within(screen.getByTestId('health-redis')).getByText('Operational')).toBeInTheDocument();

    expect(screen.getByTestId('count-completed')).toHaveTextContent('2');
    expect(screen.getByTestId('count-failed')).toHaveTextContent('1');
    expect(screen.getByTestId('count-timeout')).toHaveTextContent('1');
    // failed + timeout
    expect(screen.getByText('Failed', { selector: 'h3' }).parentElement).toHaveTextContent('2');
    // 2 completed of 5 finished
    expect(screen.getByText('40%')).toBeInTheDocument();
    // (1500 + 500 + 60000 + 1000) / 4 finished executions with a duration
    expect(screen.getByText('15.8s')).toBeInTheDocument();

    expect(screen.getByText('HTTP 500 from upstream')).toBeInTheDocument();
    expect(screen.getByText('10 mins ago')).toBeInTheDocument();
    await user.click(screen.getByText('Billing'));
    expect(screen.getByTestId('location')).toHaveTextContent('/executions/e-failed-1');
  });

  it('refreshes on demand', async () => {
    const health = vi.spyOn(systemService, 'getSystemHealth').mockResolvedValue(healthy);
    mockExecutions([]);
    const user = userEvent.setup();
    renderAt(<MonitoringPage />);
    expect(await screen.findByText('No executions yet.')).toBeInTheDocument();
    expect(screen.getByText('No failed executions in the sample.')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    await waitFor(() => expect(health).toHaveBeenCalledTimes(2));
  });

  it('shows execution loading errors', async () => {
    vi.spyOn(systemService, 'getSystemHealth').mockResolvedValue(healthy);
    vi.spyOn(executionService, 'getAll').mockRejectedValue(apiError(500, { message: 'db down' }));
    renderAt(<MonitoringPage />);
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load executions: db down');
  });
});

describe('DashboardPage', () => {
  beforeEach(() => {
    vi.spyOn(systemService, 'getSystemHealth').mockResolvedValue(healthy);
    vi.spyOn(workflowService, 'getAll').mockResolvedValue([
      { id: 'wf-1', name: 'Flow One' },
      { id: 'wf-2', name: 'Flow Two' },
    ] as never);
  });

  it('shows real stats, recent runs and system health', async () => {
    mockExecutions(executions, 7);
    const user = userEvent.setup();
    renderAt(<DashboardPage />);

    expect(await screen.findByText('Flow One')).toBeInTheDocument();
    const card = (label: string) => screen.getByText(label).parentElement!;
    await waitFor(() => expect(card('Total Workflows')).toHaveTextContent('2'));
    expect(card('Executions Today')).toHaveTextContent(/^\d+Executions Today$/);
    expect(card('Success Rate')).toHaveTextContent('40%');
    expect(card('Avg Duration')).toHaveTextContent('15.8s');
    expect(screen.getAllByText('last 7 executions')).toHaveLength(2);

    const rows = screen.getAllByRole('row').slice(1);
    expect(rows).toHaveLength(5);
    expect(within(rows[0]).getByText('Running')).toBeInTheDocument();
    expect(within(rows[1]).getByText('Billing')).toBeInTheDocument();
    expect(within(rows[1]).getByText('Failed')).toBeInTheDocument();
    expect(within(rows[1]).getByText('1m 00s')).toBeInTheDocument();
    expect(within(rows[2]).getByText('cron')).toBeInTheDocument();
    expect(within(rows[3]).getByText('wf-unkno...')).toBeInTheDocument();
    expect(within(rows[4]).getByText('Cancelled')).toBeInTheDocument();
    expect(screen.queryByText(/API Latency|Queue Depth|Worker Utilization/)).toBeNull();
    expect(within(screen.getByTestId('health-database')).getByText('Operational')).toBeInTheDocument();

    await user.click(rows[2]);
    expect(screen.getByTestId('location')).toHaveTextContent('/executions/e-done-1');
  });

  it('shows the empty state, a lower-bound count and navigation buttons', async () => {
    mockExecutions([], 0);
    const user = userEvent.setup();
    renderAt(<DashboardPage />);
    expect(await screen.findByText('No executions yet.')).toBeInTheDocument();
    await user.click(screen.getAllByRole('button', { name: 'View All' })[1]);
    expect(screen.getByTestId('location')).toHaveTextContent('/monitoring');
  });

  it('marks today as a lower bound and handles workflow load failures', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(workflowService, 'getAll').mockRejectedValue(new Error('x'));
    mockExecutions([{ id: 'x', workflow_id: 'wf-1', status: 'pending', started_at: new Date().toISOString() }], 500);
    const user = userEvent.setup();
    renderAt(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Executions Today').parentElement).toHaveTextContent('1+'));
    expect(screen.getByText('Total Workflows').parentElement).toHaveTextContent('0');
    expect(screen.getByText('Pending')).toBeInTheDocument();
    await user.click(screen.getAllByRole('button', { name: 'View All' })[0]);
    expect(screen.getByTestId('location')).toHaveTextContent('/executions');
  });

  it('opens the template picker from quick actions', async () => {
    mockExecutions([]);
    const user = userEvent.setup();
    renderAt(<DashboardPage />);
    await user.click(screen.getByRole('button', { name: 'Create New Workflow' }));
    expect(useUIStore.getState().activeModal).toBe('templates');
    expect(screen.getByTestId('location')).toHaveTextContent('/builder');
  });

  it('shows a spinner row while loading', async () => {
    vi.spyOn(executionService, 'getAll').mockReturnValue(new Promise(() => {}));
    renderAt(<DashboardPage />);
    expect(screen.getAllByRole('row')).toHaveLength(2);
    expect(screen.queryByText('No executions yet.')).toBeNull();
  });
});
