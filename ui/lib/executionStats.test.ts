import { describe, expect, it } from 'vitest';
import {
  computeExecutionStats,
  executionDurationMs,
  formatDuration,
  formatPercent,
  formatTimeAgo,
  type ExecutionLike,
} from './executionStats';

const now = new Date(2026, 4, 10, 15, 0, 0); // local time
const at = (h: number, m = 0, day = 10) => new Date(2026, 4, day, h, m, 0).toISOString();

const exec = (overrides: Partial<ExecutionLike>): ExecutionLike => ({
  id: Math.random().toString(36).slice(2),
  workflow_id: 'wf',
  status: 'completed',
  started_at: at(14),
  ...overrides,
});

describe('executionDurationMs', () => {
  it('prefers the API duration', () => {
    expect(executionDurationMs(exec({ duration: 1234 }))).toBe(1234);
  });
  it('falls back to timestamps', () => {
    expect(executionDurationMs(exec({ started_at: at(14), completed_at: at(14, 2) }))).toBe(120_000);
  });
  it('returns null for running or inconsistent executions', () => {
    expect(executionDurationMs(exec({ status: 'running' }))).toBeNull();
    expect(executionDurationMs(exec({ started_at: at(14, 5), completed_at: at(14) }))).toBeNull();
    expect(executionDurationMs(exec({ started_at: 'garbage', completed_at: at(14) }))).toBeNull();
  });
});

describe('computeExecutionStats', () => {
  it('handles an empty sample', () => {
    const stats = computeExecutionStats([], { now });
    expect(stats).toMatchObject({
      sampleSize: 0,
      total: 0,
      active: 0,
      successRate: null,
      avgDurationMs: null,
      today: 0,
      todayIsLowerBound: false,
      recentFailures: [],
    });
  });

  it('counts statuses, success rate, durations and today', () => {
    const list = [
      exec({ status: 'completed', duration: 1000, started_at: at(14) }),
      exec({ status: 'completed', duration: 3000, started_at: at(13) }),
      exec({ status: 'failed', started_at: at(12), completed_at: at(12, 1), error: 'boom' }),
      exec({ status: 'timeout', started_at: at(11) }),
      exec({ status: 'running', started_at: at(14, 30) }),
      exec({ status: 'pending', started_at: at(14, 45) }),
      exec({ status: 'cancelled', started_at: at(10, 0, 9), duration: 2000 }),
      exec({ status: 'weird', started_at: 'not a date' }),
    ];
    const stats = computeExecutionStats(list, { total: 8, now });

    expect(stats.byStatus).toEqual({ pending: 1, running: 1, completed: 2, failed: 1, cancelled: 1, timeout: 1 });
    expect(stats.active).toBe(2);
    // completed / (completed + failed + cancelled + timeout)
    expect(stats.successRate).toBeCloseTo(40);
    // durations: 1000, 3000, 60000, 2000 (timeout has none)
    expect(stats.avgDurationMs).toBe(16_500);
    expect(stats.today).toBe(6);
    expect(stats.todayIsLowerBound).toBe(false);
    expect(stats.recentFailures.map((e) => e.status)).toEqual(['failed', 'timeout']);
  });

  it('flags today as a lower bound when the whole sample is from today but more exist', () => {
    const list = [exec({ started_at: at(9) }), exec({ started_at: at(10) })];
    const stats = computeExecutionStats(list, { total: 500, now });
    expect(stats.total).toBe(500);
    expect(stats.today).toBe(2);
    expect(stats.todayIsLowerBound).toBe(true);
  });

  it('never reports a total smaller than the sample and limits failures', () => {
    const list = Array.from({ length: 4 }, (_, i) => exec({ status: 'failed', started_at: at(10 + i) }));
    const stats = computeExecutionStats(list, { total: 1, now, maxFailures: 2 });
    expect(stats.total).toBe(4);
    expect(stats.recentFailures).toHaveLength(2);
    expect(stats.recentFailures[0].started_at).toBe(at(13));
  });

  it('uses the current time by default', () => {
    const stats = computeExecutionStats([exec({ started_at: new Date().toISOString() })]);
    expect(stats.today).toBe(1);
  });
});

describe('formatDuration', () => {
  it.each([
    [null, '-'],
    [undefined, '-'],
    [-1, '-'],
    [Number.NaN, '-'],
    [850, '850ms'],
    [12_340, '12.3s'],
    [245_000, '4m 05s'],
    [3_720_000, '1h 02m'],
  ])('%s -> %s', (ms, expected) => expect(formatDuration(ms as number | null)).toBe(expected));
});

describe('formatPercent', () => {
  it('formats percentages', () => {
    expect(formatPercent(null)).toBe('-');
    expect(formatPercent(0)).toBe('0%');
    expect(formatPercent(100)).toBe('100%');
    expect(formatPercent(66.666)).toBe('66.7%');
    expect(formatPercent(40)).toBe('40%');
    expect(formatPercent(99.96)).toBe('100%');
  });
});

describe('formatTimeAgo', () => {
  const labels = { justNow: 'now', minutesAgo: '{n}m', hoursAgo: '{n}h', daysAgo: '{n}d' };
  it('buckets past times', () => {
    expect(formatTimeAgo(new Date(now.getTime() - 10_000).toISOString(), labels, now)).toBe('now');
    expect(formatTimeAgo(new Date(now.getTime() + 60_000).toISOString(), labels, now)).toBe('now');
    expect(formatTimeAgo('invalid', labels, now)).toBe('now');
    expect(formatTimeAgo(at(14, 45), labels, now)).toBe('15m');
    expect(formatTimeAgo(at(10), labels, now)).toBe('5h');
    expect(formatTimeAgo(at(15, 0, 7), labels, now)).toBe('3d');
  });
  it('defaults to the current time', () => {
    expect(formatTimeAgo(new Date().toISOString(), labels)).toBe('now');
  });
});
