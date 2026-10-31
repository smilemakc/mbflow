/**
 * Aggregates computed client-side from a page of executions
 * (GET /api/v1/executions). The backend has no stats endpoint, so every
 * figure here describes the sample that was fetched, not all history.
 */

export type ExecutionStatusName = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'timeout';

export interface ExecutionLike {
  id: string;
  workflow_id: string;
  workflow_name?: string;
  status: string;
  started_at: string;
  completed_at?: string;
  duration?: number;
  error?: string;
  triggered_by?: string;
}

export interface ExecutionStats {
  /** Executions the figures are computed from. */
  sampleSize: number;
  /** Total executions reported by the API (may exceed the sample). */
  total: number;
  byStatus: Record<ExecutionStatusName, number>;
  /** Executions in the sample that are still pending or running. */
  active: number;
  /** completed / finished, as a percentage; null when nothing finished. */
  successRate: number | null;
  /** Mean duration of finished executions in ms; null when unknown. */
  avgDurationMs: number | null;
  /** Executions in the sample started since local midnight. */
  today: number;
  /** True when the sample may not reach back to midnight, so `today` is a lower bound. */
  todayIsLowerBound: boolean;
  /** Most recent failed or timed-out executions (newest first). */
  recentFailures: ExecutionLike[];
}

const FINISHED: ExecutionStatusName[] = ['completed', 'failed', 'cancelled', 'timeout'];

const emptyByStatus = (): Record<ExecutionStatusName, number> => ({
  pending: 0,
  running: 0,
  completed: 0,
  failed: 0,
  cancelled: 0,
  timeout: 0,
});

/** Duration in ms from the API field or the timestamps; null while running. */
export const executionDurationMs = (execution: ExecutionLike): number | null => {
  if (typeof execution.duration === 'number' && execution.duration > 0) return execution.duration;
  if (!execution.completed_at) return null;
  const ms = new Date(execution.completed_at).getTime() - new Date(execution.started_at).getTime();
  return Number.isFinite(ms) && ms >= 0 ? ms : null;
};

const startOfDay = (now: Date): number => {
  const d = new Date(now);
  d.setHours(0, 0, 0, 0);
  return d.getTime();
};

export const computeExecutionStats = (
  executions: ExecutionLike[],
  options: { total?: number; now?: Date; maxFailures?: number } = {},
): ExecutionStats => {
  const now = options.now ?? new Date();
  const total = Math.max(options.total ?? executions.length, executions.length);
  const midnight = startOfDay(now);

  const byStatus = emptyByStatus();
  let durationSum = 0;
  let durationCount = 0;
  let today = 0;
  let oldestStart = Infinity;

  for (const execution of executions) {
    if (execution.status in byStatus) byStatus[execution.status as ExecutionStatusName]++;

    const started = new Date(execution.started_at).getTime();
    if (Number.isFinite(started)) {
      if (started >= midnight) today++;
      oldestStart = Math.min(oldestStart, started);
    }

    if (FINISHED.includes(execution.status as ExecutionStatusName)) {
      const duration = executionDurationMs(execution);
      if (duration !== null) {
        durationSum += duration;
        durationCount++;
      }
    }
  }

  const finished = FINISHED.reduce((sum, status) => sum + byStatus[status], 0);
  const recentFailures = executions
    .filter((e) => e.status === 'failed' || e.status === 'timeout')
    .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())
    .slice(0, options.maxFailures ?? 5);

  return {
    sampleSize: executions.length,
    total,
    byStatus,
    active: byStatus.pending + byStatus.running,
    successRate: finished > 0 ? (byStatus.completed / finished) * 100 : null,
    avgDurationMs: durationCount > 0 ? durationSum / durationCount : null,
    today,
    // Older executions were not fetched, and the oldest fetched one is from today.
    todayIsLowerBound: total > executions.length && oldestStart >= midnight,
    recentFailures,
  };
};

/** Human-readable duration: 850ms, 12.3s, 4m 05s, 1h 02m. */
export const formatDuration = (ms: number | null | undefined): string => {
  if (ms === null || ms === undefined || !Number.isFinite(ms) || ms < 0) return '-';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const totalSeconds = Math.round(seconds);
  const minutes = Math.floor(totalSeconds / 60);
  if (minutes < 60) return `${minutes}m ${String(totalSeconds % 60).padStart(2, '0')}s`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${String(minutes % 60).padStart(2, '0')}m`;
};

export const formatPercent = (value: number | null): string => {
  if (value === null) return '-';
  const rounded = Math.round(value * 10) / 10;
  return `${Number.isInteger(rounded) ? rounded : rounded.toFixed(1)}%`;
};

/** Relative "time ago" for past timestamps, with i18n templates containing {n}. */
export const formatTimeAgo = (
  date: string,
  labels: { justNow: string; minutesAgo: string; hoursAgo: string; daysAgo: string },
  now: Date = new Date(),
): string => {
  const diffMins = Math.floor((now.getTime() - new Date(date).getTime()) / 60000);
  if (!Number.isFinite(diffMins) || diffMins < 1) return labels.justNow;
  if (diffMins < 60) return labels.minutesAgo.replace('{n}', String(diffMins));
  const hours = Math.floor(diffMins / 60);
  if (hours < 24) return labels.hoursAgo.replace('{n}', String(hours));
  return labels.daysAgo.replace('{n}', String(Math.floor(hours / 24)));
};
