import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { isValidElement, type ReactElement } from 'react';
import { CopyButton } from './CopyButton';
import { JsonViewer } from './JsonViewer';
import { NodeExecutionCard } from './NodeExecutionCard';
import { ExecutionStatsGrid } from './ExecutionStatsGrid';
import { SubWorkflowDrillDown } from './SubWorkflowDrillDown';
import { WebSocketStatusBadge } from './WebSocketStatusBadge';
import { calculateStats, formatDate, formatDuration, getStatusBadgeClass, getStatusIcon } from './executionUtils';
import type { Execution, ExecutionStatus, NodeExecution } from '@/types/execution';
import { useUIStore } from '@/store/uiStore';

const writeText = vi.fn();
beforeEach(() => {
  useUIStore.setState({ language: 'en' });
  writeText.mockReset().mockResolvedValue(undefined);
  Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
});
afterEach(() => {
  vi.useRealTimers();
});

describe('executionUtils', () => {
  it('maps statuses to icons and badge classes', () => {
    const statuses: ExecutionStatus[] = ['completed', 'failed', 'running', 'pending', 'cancelled', 'timeout'];
    const classes = statuses.map((s) => (getStatusIcon(s, 10) as ReactElement<{ className: string; size: number }>).props);
    expect(classes.map((p) => p.size)).toEqual(Array(6).fill(10));
    expect(classes[0].className).toContain('green');
    expect(classes[1].className).toContain('red');
    expect(classes[2].className).toContain('animate-spin');
    expect(classes[3].className).toContain('yellow');
    expect(classes[4].className).toContain('gray');
    expect(isValidElement(getStatusIcon('timeout'))).toBe(true);

    expect(statuses.map((s) => getStatusBadgeClass(s).split(' ')[0])).toEqual([
      'bg-green-50',
      'bg-red-50',
      'bg-blue-50',
      'bg-yellow-50',
      'bg-gray-50',
      'bg-gray-50',
    ]);
  });

  it('formats durations and dates', () => {
    const t0 = '2024-01-01T00:00:00.000Z';
    expect(formatDuration(t0)).toBe('-');
    expect(formatDuration(t0, '2024-01-01T00:00:00.250Z')).toBe('250ms');
    expect(formatDuration(t0, '2024-01-01T00:00:01.500Z')).toBe('1.50s');
    expect(formatDuration(t0, '2024-01-01T00:01:30.000Z')).toBe('1.50m');
    expect(formatDate(t0)).toMatch(/2024/);
  });

  it('counts node statuses', () => {
    expect(calculateStats(undefined)).toEqual({ total: 0, completed: 0, failed: 0, running: 0, pending: 0 });
    const s = (status: ExecutionStatus) => ({ status });
    expect(
      calculateStats([s('completed'), s('completed'), s('failed'), s('running'), s('pending'), s('cancelled')]),
    ).toEqual({ total: 6, completed: 2, failed: 1, running: 1, pending: 1 });
  });
});

describe('CopyButton', () => {
  it('copies text, shows feedback and resets after two seconds', async () => {
    vi.useFakeTimers();
    const parentClick = vi.fn();
    render(
      <div onClick={parentClick}>
        <CopyButton text="hello" label="Copy JSON" />
      </div>,
    );
    const btn = screen.getByRole('button', { name: /copy json/i });
    await act(async () => {
      fireEvent.click(btn);
    });
    expect(writeText).toHaveBeenCalledWith('hello');
    expect(parentClick).not.toHaveBeenCalled();
    expect(screen.getByText('Copied!')).toBeInTheDocument();
    act(() => vi.advanceTimersByTime(2000));
    expect(screen.getByText('Copy JSON')).toBeInTheDocument();
  });

  it('logs and stays idle when the clipboard rejects', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    writeText.mockRejectedValue(new Error('denied'));
    render(<CopyButton text="x" />);
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Copy' }));
    });
    expect(console.error).toHaveBeenCalled();
    expect(screen.queryByText('Copied!')).toBeNull();
  });
});

describe('JsonViewer', () => {
  it('shows a placeholder for empty data', () => {
    const { rerender } = render(<JsonViewer data={null} title="Input" />);
    expect(screen.getByText('No data')).toBeInTheDocument();
    rerender(<JsonViewer data={{}} title="Input" />);
    expect(screen.getByText('No data')).toBeInTheDocument();
  });

  it('renders pretty JSON with field counts and toggles', () => {
    const { container, rerender } = render(<JsonViewer data={{ a: 1, b: 2 }} title="Output" maxHeight="10px" />);
    expect(screen.getByText('(2 fields)')).toBeInTheDocument();
    const pre = container.querySelector('pre')!;
    expect(pre.textContent).toBe(JSON.stringify({ a: 1, b: 2 }, null, 2));
    expect((pre.parentElement as HTMLElement).style.maxHeight).toBe('10px');
    fireEvent.click(screen.getByText('Output'));
    expect(container.querySelector('pre')).toBeNull();
    fireEvent.click(screen.getByText('Output'));
    expect(container.querySelector('pre')).not.toBeNull();

    rerender(<JsonViewer data="scalar" title="" defaultExpanded={false} />);
    expect(screen.getByText('(1 field)')).toBeInTheDocument();
  });
});

describe('NodeExecutionCard', () => {
  const base: NodeExecution = {
    id: 'ne1',
    execution_id: 'e1',
    node_id: 'node-1',
    status: 'completed',
    started_at: '2024-01-01T00:00:00.000Z',
    completed_at: '2024-01-01T00:00:00.120Z',
  };

  it('renders a collapsed header and toggles', () => {
    const onToggle = vi.fn();
    render(<NodeExecutionCard nodeExec={{ ...base, node_name: 'Fetch', node_type: 'http' }} index={2} isExpanded={false} onToggle={onToggle} />);
    expect(screen.getByText('3')).toBeInTheDocument();
    expect(screen.getByText('Fetch')).toBeInTheDocument();
    expect(screen.getByText('http')).toBeInTheDocument();
    expect(screen.getByText('120ms')).toBeInTheDocument();
    expect(screen.queryByText('Node ID')).toBeNull();
    fireEvent.click(screen.getByText('Fetch'));
    expect(onToggle).toHaveBeenCalled();
  });

  it('shows details, errors, retries and payloads when expanded', () => {
    render(
      <NodeExecutionCard
        nodeExec={{ ...base, status: 'failed', error: 'Timeout!', retry_count: 2, input: { q: 1 }, completed_at: undefined }}
        index={0}
        isExpanded
        onToggle={() => {}}
      />,
    );
    expect(screen.getAllByText('node-1')).toHaveLength(2);
    expect(screen.getByText('Timeout!')).toBeInTheDocument();
    expect(screen.getByText('Retry Count')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByText('Input')).toBeInTheDocument();
    expect(screen.getByText('No data')).toBeInTheDocument();
    expect(screen.getByText('Completed At').nextSibling!.textContent).toBe('-');
  });

  it('renders an icon for every node type', () => {
    for (const node_type of ['llm', 'transform', 'conditional', 'merge', 'openapi', 'custom', undefined]) {
      const { container, unmount } = render(
        <NodeExecutionCard nodeExec={{ ...base, node_type, status: 'running' }} index={0} isExpanded={false} onToggle={() => {}} />,
      );
      expect(container.querySelectorAll('svg').length).toBeGreaterThanOrEqual(3);
      unmount();
    }
  });
});

describe('ExecutionStatsGrid', () => {
  it('summarises node results and timing', () => {
    const execution = {
      id: 'e1',
      workflow_id: 'w',
      status: 'completed',
      started_at: '2024-01-01T00:00:00.000Z',
      completed_at: '2024-01-01T00:00:02.000Z',
      node_executions: [{ status: 'completed' }, { status: 'completed' }, { status: 'failed' }],
    } as Execution;
    render(<ExecutionStatsGrid execution={execution} />);
    expect(screen.getByText('Total Nodes').closest('div')!.nextSibling!.textContent).toBe('3');
    expect(screen.getByText('2.00s')).toBeInTheDocument();
    expect(screen.getByText('1')).toBeInTheDocument();
  });
});

describe('SubWorkflowDrillDown', () => {
  const children = [
    { index: 0, status: 'completed', execution_id: 'c0', item_key: 'alpha', duration_ms: 500 },
    { index: 1, status: 'failed', execution_id: 'c1', error: 'bad item', duration_ms: 2500 },
    { index: 2, status: 'running', execution_id: 'c2' },
    { index: 3, status: 'pending', execution_id: 'c3' },
    { index: 4, status: 'failed', execution_id: 'c4' },
  ];

  it('lists child executions with status, duration and retry', () => {
    const onClose = vi.fn();
    const onRetryFailed = vi.fn();
    render(<SubWorkflowDrillDown nodeName="Loop" children={children} onClose={onClose} onRetryFailed={onRetryFailed} />);
    expect(screen.getByText('5 items')).toBeInTheDocument();
    expect(screen.getByText('alpha')).toBeInTheDocument();
    expect(screen.getByText('Item 1')).toBeInTheDocument();
    expect(screen.getByText('500ms')).toBeInTheDocument();
    expect(screen.getByText('2.5s')).toBeInTheDocument();
    expect(screen.getByTitle('bad item')).toBeInTheDocument();
    expect(screen.getByText('✓ completed')).toBeInTheDocument();
    expect(screen.getByText('⟳ running')).toBeInTheDocument();
    expect(screen.getByText('◻ pending')).toBeInTheDocument();
    expect(screen.getAllByText('✗ failed')).toHaveLength(2);

    fireEvent.click(screen.getByText('Retry 2 failed'));
    expect(onRetryFailed).toHaveBeenCalled();
    fireEvent.click(screen.getByText('×'));
    expect(onClose).toHaveBeenCalled();
  });

  it('hides retry when nothing failed or no handler', () => {
    const { rerender } = render(<SubWorkflowDrillDown nodeName="L" children={children.slice(0, 1)} onClose={() => {}} onRetryFailed={() => {}} />);
    expect(screen.queryByText(/Retry/)).toBeNull();
    rerender(<SubWorkflowDrillDown nodeName="L" children={children} onClose={() => {}} />);
    expect(screen.queryByText(/Retry/)).toBeNull();
  });
});

describe('WebSocketStatusBadge', () => {
  it('reflects connection state', () => {
    const { rerender } = render(<WebSocketStatusBadge connected />);
    expect(screen.getByText('Live Updates')).toBeInTheDocument();
    rerender(<WebSocketStatusBadge connected={false} />);
    expect(screen.getByText('Connecting...')).toBeInTheDocument();
  });
});
