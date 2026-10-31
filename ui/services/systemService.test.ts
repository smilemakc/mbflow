import { afterEach, describe, expect, it, vi } from 'vitest';
import axios from 'axios';
import { getBackendRoot, systemService } from './systemService';

const response = (status: number, data: unknown) => ({ status, data, statusText: '', headers: {}, config: {} });

describe('getBackendRoot', () => {
  it('strips the API prefix', () => {
    expect(getBackendRoot('/api/v1')).toBe('');
    expect(getBackendRoot('/api/v1/')).toBe('');
    expect(getBackendRoot('https://host:8585/api/v1')).toBe('https://host:8585');
    expect(getBackendRoot('https://host/api')).toBe('https://host');
    expect(getBackendRoot('https://host/custom')).toBe('https://host/custom');
  });
});

describe('systemService', () => {
  afterEach(() => vi.restoreAllMocks());

  it('reports liveness from GET /health', async () => {
    const get = vi.spyOn(axios, 'get').mockResolvedValue(response(200, { status: 'healthy' }));
    expect(await systemService.getLiveness()).toBe(true);
    expect(get).toHaveBeenCalledWith('/health', expect.objectContaining({ timeout: expect.any(Number) }));

    get.mockRejectedValue(new Error('down'));
    expect(await systemService.getLiveness()).toBe(false);
  });

  it('reads readiness checks, including a 503 body', async () => {
    const get = vi.spyOn(axios, 'get');
    get.mockResolvedValueOnce(response(200, { status: 'ready', checks: { database: 'ok', redis: 'ok' } }));
    expect(await systemService.getReadiness()).toEqual({ ready: true, status: 'ready', checks: { database: 'ok', redis: 'ok' } });

    const options = get.mock.calls[0][1] as { validateStatus: (s: number) => boolean };
    expect(get.mock.calls[0][0]).toBe('/ready');
    expect(options.validateStatus(200)).toBe(true);
    expect(options.validateStatus(503)).toBe(true);
    expect(options.validateStatus(500)).toBe(false);

    get.mockResolvedValueOnce(response(503, { status: 'not_ready', checks: { database: 'unavailable' } }));
    expect(await systemService.getReadiness()).toEqual({ ready: false, status: 'not_ready', checks: { database: 'unavailable' } });

    get.mockResolvedValueOnce(response(503, 'Service Unavailable'));
    expect(await systemService.getReadiness()).toEqual({ ready: false, status: 'not_ready', checks: {} });

    get.mockResolvedValueOnce(response(200, null));
    expect(await systemService.getReadiness()).toEqual({ ready: true, status: 'ready', checks: {} });

    get.mockRejectedValueOnce(new Error('network'));
    expect(await systemService.getReadiness()).toEqual({ ready: null, status: 'unreachable', checks: {} });
  });

  it('combines both probes', async () => {
    vi.spyOn(axios, 'get').mockImplementation(async (url: string) =>
      url === '/health' ? response(200, {}) : response(200, { status: 'ready', checks: { database: 'ok' } }),
    );
    const at = new Date('2026-01-01T00:00:00Z');
    expect(await systemService.getSystemHealth(() => at)).toEqual({
      apiUp: true,
      ready: true,
      status: 'ready',
      checks: { database: 'ok' },
      checkedAt: at,
    });
    expect((await systemService.getSystemHealth()).checkedAt).toBeInstanceOf(Date);
  });
});
