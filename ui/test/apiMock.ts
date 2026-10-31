/**
 * Test helper: replaces the shared axios client's adapter with an in-memory
 * router so services run their real URL/payload logic without a network.
 */
import { AxiosError, AxiosHeaders, type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios';
import { apiClient } from '@/lib/api';

export interface RecordedCall {
  method: string;
  url: string;
  params?: Record<string, unknown>;
  data?: unknown;
  headers: Record<string, unknown>;
  config: InternalAxiosRequestConfig;
}

export interface MockResponse {
  status?: number;
  data?: unknown;
  headers?: Record<string, string>;
}

type Handler = MockResponse | ((call: RecordedCall) => MockResponse | Promise<MockResponse>);

interface Route {
  method: string;
  url: string | RegExp;
  handler: Handler;
  once: boolean;
}

const parseBody = (data: unknown): unknown => {
  if (typeof data !== 'string') return data;
  try {
    return JSON.parse(data);
  } catch {
    return data;
  }
};

export function mockApi() {
  const calls: RecordedCall[] = [];
  const routes: Route[] = [];
  const original = apiClient.defaults.adapter;

  const adapter: AxiosAdapter = async (config) => {
    const call: RecordedCall = {
      method: (config.method || 'get').toUpperCase(),
      url: config.url || '',
      params: config.params,
      data: parseBody(config.data),
      headers: { ...(config.headers as Record<string, unknown>) },
      config,
    };
    calls.push(call);

    const index = routes.findIndex(
      (r) => r.method === call.method && (typeof r.url === 'string' ? r.url === call.url : r.url.test(call.url)),
    );
    const route = index >= 0 ? routes[index] : undefined;
    if (route?.once) routes.splice(index, 1);

    const result: MockResponse = route
      ? typeof route.handler === 'function'
        ? await route.handler(call)
        : route.handler
      : { status: 404, data: { code: 'NOT_FOUND', message: `No mock for ${call.method} ${call.url}` } };

    const response = {
      data: result.data,
      status: result.status ?? 200,
      statusText: '',
      headers: new AxiosHeaders(result.headers ?? {}),
      config,
      request: {},
    };
    if (response.status >= 400) {
      throw new AxiosError(`Request failed with status code ${response.status}`, 'ERR_BAD_RESPONSE', config, {}, response);
    }
    return response;
  };

  apiClient.defaults.adapter = adapter;

  const api = {
    calls,
    on(method: string, url: string | RegExp, handler: Handler = {}) {
      routes.push({ method: method.toUpperCase(), url, handler, once: false });
      return api;
    },
    /** One-shot route; takes precedence over routes registered with `on`. */
    once(method: string, url: string | RegExp, handler: Handler = {}) {
      routes.unshift({ method: method.toUpperCase(), url, handler, once: true });
      return api;
    },
    last(): RecordedCall | undefined {
      return calls[calls.length - 1];
    },
    restore() {
      apiClient.defaults.adapter = original;
    },
  };
  return api;
}

export type ApiMock = ReturnType<typeof mockApi>;

/** Builds an AxiosError carrying an API error body, for code paths that inspect errors. */
export const apiError = (status: number, data?: unknown): AxiosError => {
  const config = { headers: new AxiosHeaders() } as InternalAxiosRequestConfig;
  return new AxiosError('Request failed', 'ERR_BAD_RESPONSE', config, {}, {
    data,
    status,
    statusText: '',
    headers: new AxiosHeaders(),
    config,
  });
};
