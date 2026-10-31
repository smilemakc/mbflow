/**
 * System health: wraps the backend's unauthenticated probes.
 *
 *   GET /health -> 200 {"status":"healthy"}                       (liveness)
 *   GET /ready  -> 200 {"status":"ready","checks":{...}}         (readiness)
 *                  503 {"status":"not_ready"|"shutting_down","checks":{...}}
 *
 * The probes live at the server root, not under /api/v1.
 */

import axios from 'axios';

export type CheckState = 'ok' | 'unavailable' | string;

export interface SystemHealth {
  /** Liveness probe answered. */
  apiUp: boolean;
  /** Readiness result; null when the probe could not be reached. */
  ready: boolean | null;
  /** Raw readiness status: ready, not_ready, shutting_down or unreachable. */
  status: string;
  /** Per-dependency results, e.g. { database: 'ok', redis: 'unavailable' }. */
  checks: Record<string, CheckState>;
  checkedAt: Date;
}

const PROBE_TIMEOUT_MS = 8000;

/**
 * Derives the backend root from the API base URL:
 * "/api/v1" -> "", "https://host/api/v1" -> "https://host".
 */
export const getBackendRoot = (apiUrl: string): string =>
  apiUrl.replace(/\/+$/, '').replace(/\/api(\/v\d+)?$/, '');

const backendRoot = (): string => getBackendRoot(import.meta.env.VITE_API_URL || '/api/v1');

export const systemService = {
  async getLiveness(): Promise<boolean> {
    try {
      const response = await axios.get<{ status?: string }>(`${backendRoot()}/health`, {
        timeout: PROBE_TIMEOUT_MS,
      });
      return response.status === 200;
    } catch {
      return false;
    }
  },

  async getReadiness(): Promise<{ ready: boolean | null; status: string; checks: Record<string, CheckState> }> {
    try {
      const response = await axios.get<{ status?: string; checks?: Record<string, CheckState> }>(
        `${backendRoot()}/ready`,
        {
          timeout: PROBE_TIMEOUT_MS,
          // 503 carries the failing checks in its body.
          validateStatus: (status) => status === 200 || status === 503,
        },
      );
      const body = response.data && typeof response.data === 'object' ? response.data : {};
      return {
        ready: response.status === 200,
        status: body.status || (response.status === 200 ? 'ready' : 'not_ready'),
        checks: body.checks ?? {},
      };
    } catch {
      return { ready: null, status: 'unreachable', checks: {} };
    }
  },

  async getSystemHealth(now: () => Date = () => new Date()): Promise<SystemHealth> {
    const [apiUp, readiness] = await Promise.all([systemService.getLiveness(), systemService.getReadiness()]);
    return { apiUp, ...readiness, checkedAt: now() };
  },
};

export default systemService;
