import { useCallback, useEffect, useRef, useState } from 'react';
import { executionService } from '@/services/executionService';
import { systemService, type SystemHealth } from '@/services/systemService';
import { computeExecutionStats, type ExecutionLike, type ExecutionStats } from '@/lib/executionStats';
import { getErrorMessage } from '@/lib/api';

export interface SystemOverview {
  health: SystemHealth | null;
  stats: ExecutionStats | null;
  /** Latest executions (newest first, as returned by the API). */
  executions: ExecutionLike[];
  loading: boolean;
  /** Error loading executions; health failures are reported inside `health`. */
  error: string | null;
  refresh: () => Promise<void>;
}

export interface UseSystemOverviewOptions {
  /** How many recent executions the stats are computed from. */
  sampleSize?: number;
  /** Auto-refresh period in ms; 0 disables it. */
  refreshMs?: number;
}

export function useSystemOverview({ sampleSize = 100, refreshMs = 0 }: UseSystemOverviewOptions = {}): SystemOverview {
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [stats, setStats] = useState<ExecutionStats | null>(null);
  const [executions, setExecutions] = useState<ExecutionLike[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mounted = useRef(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    const [healthResult, executionResult] = await Promise.allSettled([
      systemService.getSystemHealth(),
      executionService.getAll({ limit: sampleSize }),
    ]);
    if (!mounted.current) return;

    if (healthResult.status === 'fulfilled') setHealth(healthResult.value);

    if (executionResult.status === 'fulfilled') {
      const { executions: list, total } = executionResult.value;
      setExecutions(list);
      setStats(computeExecutionStats(list, { total }));
      setError(null);
    } else {
      setError(getErrorMessage(executionResult.reason));
    }
    setLoading(false);
  }, [sampleSize]);

  useEffect(() => {
    mounted.current = true;
    void refresh();
    if (refreshMs <= 0) {
      return () => {
        mounted.current = false;
      };
    }
    const timer = setInterval(() => void refresh(), refreshMs);
    return () => {
      mounted.current = false;
      clearInterval(timer);
    };
  }, [refresh, refreshMs]);

  return { health, stats, executions, loading, error, refresh };
}

export default useSystemOverview;
