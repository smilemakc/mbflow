import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MonitoringPanel } from './MonitoringPanel';
import { useDagStore } from '@/store/dagStore';
import { NodeStatus, type ExecutionLog } from '@/types';
import { makeNode, resetBuilderStores } from '@/test/builderFlowMocks';

const setCenter = vi.fn();
vi.mock('reactflow', async (importOriginal) => ({
  ...(await importOriginal<typeof import('reactflow')>()),
  useReactFlow: () => ({ setCenter }),
}));

const log = (id: string, level: ExecutionLog['level'], message: string, nodeId: string | null = null): ExecutionLog => ({
  id,
  level,
  message,
  nodeId,
  timestamp: new Date(2024, 0, 1, 12, 30, 45),
});

const s = () => useDagStore.getState();

beforeEach(() => {
  resetBuilderStores();
  setCenter.mockClear();
});

const expand = async () => userEvent.click(screen.getByText('Monitoring'));

describe('MonitoringPanel', () => {
  it('starts collapsed with a log counter and expands', async () => {
    useDagStore.setState({ logs: [log('1', 'info', 'hello')] });
    render(<MonitoringPanel />);
    expect(screen.getByText('1')).toBeInTheDocument();
    await expand();
    expect(screen.getByText('Execution Monitor')).toBeInTheDocument();
    expect(screen.getByText('hello')).toBeInTheDocument();
    expect(screen.getByText('12:30:45')).toBeInTheDocument();
  });

  it('shows the empty state and collapses again', async () => {
    render(<MonitoringPanel />);
    expect(screen.queryByText('1')).not.toBeInTheDocument();
    await expand();
    expect(screen.getByText("Ready to execute. Click 'Run' to start.")).toBeInTheDocument();
    const buttons = screen.getAllByRole('button');
    await userEvent.click(buttons[buttons.length - 1]);
    expect(screen.queryByText('Execution Monitor')).not.toBeInTheDocument();
  });

  it('colours log levels, centres on a node when its log is clicked and clears', async () => {
    useDagStore.setState({
      nodes: [makeNode('n1')],
      logs: [
        log('1', 'info', 'info msg'),
        log('2', 'success', 'ok msg', 'n1'),
        log('3', 'error', 'bad msg', 'ghost'),
        log('4', 'warning', 'warn msg'),
      ],
    });
    render(<MonitoringPanel />);
    await expand();
    expect(screen.getByText('ok msg').className).toContain('text-green-600');
    expect(screen.getByText('bad msg').className).toContain('text-red-600');
    expect(screen.getByText('warn msg').previousElementSibling!.className).toContain('bg-yellow-500');

    await userEvent.click(screen.getByText('info msg'));
    expect(s().selectedNodeId).toBeNull();

    await userEvent.click(screen.getByText('ok msg'));
    expect(s().selectedNodeId).toBe('n1');
    expect(setCenter).toHaveBeenCalledWith(110, 70, { zoom: 1.2, duration: 800 });

    await userEvent.click(screen.getByText('bad msg'));
    expect(s().selectedNodeId).toBe('ghost');
    expect(setCenter).toHaveBeenCalledTimes(1);

    await userEvent.click(screen.getByTitle('Clear Logs'));
    expect(s().logs).toEqual([]);
  });

  it('shows live / offline connection state while running', async () => {
    useDagStore.setState({ isRunning: true, wsConnected: true });
    render(<MonitoringPanel />);
    await expand();
    expect(screen.getByText('Live')).toBeInTheDocument();
    useDagStore.setState({ wsConnected: false });
    expect(await screen.findByText('Offline')).toBeInTheDocument();
  });

  it('shows inputs and outputs of the selected node', async () => {
    useDagStore.setState({
      nodes: [makeNode('n1'), makeNode('n2'), makeNode('n3')],
      selectedNodeId: 'n1',
      executionResults: {
        n1: { nodeId: 'n1', status: NodeStatus.SUCCESS, inputs: { a: 1 }, outputs: { b: 2 }, startTime: 100, endTime: 350, logs: [] },
        n2: { nodeId: 'n2', status: NodeStatus.ERROR, inputs: {}, outputs: {}, startTime: 0, logs: [] },
        n3: { nodeId: 'n3', status: NodeStatus.RUNNING, inputs: {}, outputs: {}, startTime: 0, logs: [] },
      },
    });
    render(<MonitoringPanel />);
    await expand();
    await userEvent.click(screen.getByRole('button', { name: /Inputs \/ Outputs/ }));
    expect(screen.getByText('Selected')).toBeInTheDocument();
    expect(screen.getByText('Node: Label n1')).toBeInTheDocument();
    expect(screen.getByText(/"a": 1/)).toBeInTheDocument();
    expect(screen.getByText(/"b": 2/)).toBeInTheDocument();
    expect(screen.getByText('250ms')).toBeInTheDocument();
    expect(screen.getByText('success').className).toContain('bg-green-100');

    useDagStore.setState({ selectedNodeId: 'n2' });
    expect((await screen.findByText('error')).className).toContain('bg-red-100');
    useDagStore.setState({ selectedNodeId: 'n3' });
    expect((await screen.findByText('running')).className).toContain('bg-slate-100');

    await userEvent.click(screen.getByRole('button', { name: 'Console Logs' }));
    expect(screen.getByText("Ready to execute. Click 'Run' to start.")).toBeInTheDocument();
  });

  it('asks to select a node when there is no result', async () => {
    useDagStore.setState({ logs: [log('1', 'info', 'x')] });
    render(<MonitoringPanel />);
    await expand();
    await userEvent.click(screen.getByRole('button', { name: /Inputs \/ Outputs/ }));
    expect(screen.getByText('Select a node to view its inputs and outputs.')).toBeInTheDocument();
    expect(screen.queryByText(/\(Ready to execute/)).not.toBeInTheDocument();
    useDagStore.setState({ logs: [] });
    expect(await screen.findByText("(Ready to execute. Click 'Run' to start.)")).toBeInTheDocument();
  });
});
