/**
 * Fixtures for node editors backed by imported API clients (LLM API tools, OpenAPI node).
 */
import { mockApi, type ApiMock } from './apiMock';
import { useApiClientStore } from '@/store/apiClientStore';
import type { ApiClient, ApiOperation } from '@/services/apiClientService';

export const nodesOperation = (id: string, overrides: Partial<ApiOperation> = {}): ApiOperation => ({
  id,
  method: 'GET',
  path: `/${id}`,
  ...overrides,
});

export const nodesApiClient = (id: string, operations: ApiOperation[], overrides: Partial<ApiClient> = {}): ApiClient => ({
  id,
  name: `Client ${id}`,
  title: `Title ${id}`,
  spec_version: '3.0.3',
  base_url: 'https://api.example.test',
  auth: { type: 'none' },
  operation_count: operations.length,
  warnings: [],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  owner_id: 'u1',
  servers: [],
  security_schemes: [],
  default_headers: {},
  operations,
  schemas: {},
  spec_size: 100,
  ...overrides,
});

/** Resets the API client store and routes list/detail requests to the given clients. */
export function nodesMockApiClients(clients: ApiClient[]): ApiMock {
  useApiClientStore.setState({ clients: [], details: {}, missing: {}, isLoading: false, isLoaded: false, error: null });
  const api = mockApi();
  api.on('GET', '/api-clients', {
    data: { data: clients.map(({ operations: _o, schemas: _s, ...summary }) => summary), total: clients.length },
  });
  api.on('GET', /^\/api-clients\/[^/]+$/, (call) => {
    const id = call.url.split('/').pop();
    const client = clients.find((c) => c.id === id);
    return client ? { data: client } : { status: 404, data: { code: 'NOT_FOUND', message: 'not found' } };
  });
  return api;
}
