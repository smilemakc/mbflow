import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import axios from 'axios';
import {
  apiClient,
  ErrorCodes,
  getErrorCode,
  getErrorMessage,
  getRequestId,
  isErrorCode,
  showApiError,
} from './api';
import { AUTH_STORAGE_KEY, REFRESH_TOKEN_KEY, TOKEN_KEY } from './authStorage';
import { useToastStore } from './toast';
import { apiError, mockApi, type ApiMock } from '@/test/apiMock';

describe('getErrorMessage', () => {
  it('reads the structured API error message', () => {
    expect(getErrorMessage(apiError(400, { code: 'X', message: 'bad input' }))).toBe('bad input');
  });
  it('reads the legacy error field', () => {
    expect(getErrorMessage(apiError(400, { error: 'legacy' }))).toBe('legacy');
  });
  it('falls back to HTTP status text', () => {
    expect(getErrorMessage(apiError(404))).toBe('Not found');
    expect(getErrorMessage(apiError(503, {}))).toBe('Service unavailable');
    expect(getErrorMessage(apiError(418))).toBe('Error 418');
  });
  it('uses the axios message for network errors', () => {
    const err = new axios.AxiosError('Network Error');
    expect(getErrorMessage(err)).toBe('Network Error');
  });
  it('falls through for an axios error without any information', () => {
    const err = new axios.AxiosError('');
    expect(getErrorMessage(err)).toBe('');
  });
  it('handles plain errors and unknown values', () => {
    expect(getErrorMessage(new Error('plain'))).toBe('plain');
    expect(getErrorMessage('???')).toBe('An unexpected error occurred');
  });
});

describe('error code helpers', () => {
  const err = apiError(409, { code: 'USER_EXISTS', message: 'exists', details: { request_id: 'req-1' } });

  it('extracts the code and request id', () => {
    expect(getErrorCode(err)).toBe('USER_EXISTS');
    expect(getRequestId(err)).toBe('req-1');
    expect(isErrorCode(err, ErrorCodes.USER_EXISTS)).toBe(true);
    expect(isErrorCode(err, ErrorCodes.NOT_FOUND)).toBe(false);
  });

  it('returns null when absent', () => {
    expect(getErrorCode(apiError(500, {}))).toBeNull();
    expect(getRequestId(apiError(500, {}))).toBeNull();
    expect(getErrorCode(new Error('x'))).toBeNull();
    expect(getRequestId(new Error('x'))).toBeNull();
  });
});

describe('showApiError', () => {
  beforeEach(() => useToastStore.getState().clearAll());

  it('shows the message with the request id', () => {
    showApiError(apiError(500, { message: 'oops', details: { request_id: 'r9' } }), 'Saving');
    const [toast] = useToastStore.getState().toasts;
    expect(toast).toMatchObject({ type: 'error', title: 'Saving', message: 'oops (ID: r9)' });
  });

  it('shows the message without a request id and with the default title', () => {
    showApiError(new Error('nope'));
    const [toast] = useToastStore.getState().toasts;
    expect(toast).toMatchObject({ title: 'Error', message: 'nope' });
  });
});

describe('apiClient interceptors', () => {
  let api: ApiMock;
  let locationHref: string;
  const originalLocation = window.location;

  beforeEach(() => {
    api = mockApi();
    locationHref = '';
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        ...originalLocation,
        get href() {
          return locationHref;
        },
        set href(v: string) {
          locationHref = v;
        },
      },
    });
  });

  afterEach(() => {
    api.restore();
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation });
  });

  it('uses /api/v1 as the default base URL', () => {
    expect(apiClient.defaults.baseURL).toBe('/api/v1');
  });

  it('attaches the bearer token when present', async () => {
    api.on('GET', '/ping', { data: 'pong' });
    await apiClient.get('/ping');
    expect(api.last()?.headers.Authorization).toBeUndefined();

    localStorage.setItem(TOKEN_KEY, 'tok');
    await apiClient.get('/ping');
    expect(api.last()?.headers.Authorization).toBe('Bearer tok');
  });

  it('passes non-401 errors through', async () => {
    api.on('GET', '/boom', { status: 500, data: { message: 'down' } });
    await expect(apiClient.get('/boom')).rejects.toMatchObject({ response: { status: 500 } });
  });

  it('refreshes the token on 401 and retries the request', async () => {
    localStorage.setItem(TOKEN_KEY, 'old');
    localStorage.setItem(REFRESH_TOKEN_KEY, 'refresh-1');
    api.once('GET', '/me', { status: 401 }).on('GET', '/me', (call) => ({ data: { auth: call.headers.Authorization } }));
    const post = vi.spyOn(axios, 'post').mockResolvedValue({
      data: { access_token: 'new', refresh_token: 'refresh-2' },
    });

    const response = await apiClient.get('/me');

    expect(post).toHaveBeenCalledWith('/api/v1/auth/refresh', { refresh_token: 'refresh-1' });
    expect(response.data).toEqual({ auth: 'Bearer new' });
    expect(localStorage.getItem(TOKEN_KEY)).toBe('new');
    expect(localStorage.getItem(REFRESH_TOKEN_KEY)).toBe('refresh-2');
  });

  it('keeps the old refresh token when the server does not rotate it', async () => {
    localStorage.setItem(REFRESH_TOKEN_KEY, 'keep');
    api.once('GET', '/me', { status: 401 }).on('GET', '/me', { data: 'ok' });
    vi.spyOn(axios, 'post').mockResolvedValue({ data: { access_token: 'new' } });

    await apiClient.get('/me');
    expect(localStorage.getItem(REFRESH_TOKEN_KEY)).toBe('keep');
  });

  it('clears auth and redirects when refresh fails', async () => {
    localStorage.setItem(TOKEN_KEY, 'old');
    localStorage.setItem(REFRESH_TOKEN_KEY, 'bad');
    localStorage.setItem(AUTH_STORAGE_KEY, '{}');
    api.on('GET', '/me', { status: 401 });
    vi.spyOn(axios, 'post').mockRejectedValue(new Error('refresh failed'));

    await expect(apiClient.get('/me')).rejects.toThrow('refresh failed');
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBeNull();
    expect(locationHref).toBe('/login');
  });

  it('redirects to login on 401 without a refresh token', async () => {
    localStorage.setItem(TOKEN_KEY, 'old');
    api.on('GET', '/me', { status: 401 });

    await expect(apiClient.get('/me')).rejects.toMatchObject({ response: { status: 401 } });
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(locationHref).toBe('/login');
  });

  it('rejects request interceptor errors', async () => {
    const handlers = (apiClient.interceptors.request as unknown as {
      handlers: Array<{ rejected: (e: unknown) => Promise<never> }>;
    }).handlers;
    const err = new Error('pre-flight');
    await expect(handlers[0].rejected(err)).rejects.toBe(err);
  });
});
