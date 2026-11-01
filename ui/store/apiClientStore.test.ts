import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useApiClientStore } from './apiClientStore';
import { apiClientService, type ApiClient } from '@/services/apiClientService';
import { apiError } from '@/test/apiMock';

const client = (id: string, updated_at = '1'): ApiClient =>
  ({
    id,
    name: `client-${id}`,
    title: 'T',
    spec_version: '3.0',
    auth: { type: 'none' },
    operation_count: 1,
    warnings: [],
    created_at: '0',
    updated_at,
    owner_id: 'o',
    servers: [],
    security_schemes: [],
    default_headers: {},
    operations: [],
    schemas: {},
    spec_size: 1,
  }) as ApiClient;

const ok = <T,>(data: T) => ({ data }) as never;

describe('apiClientStore', () => {
  beforeEach(() => {
    useApiClientStore.setState({ clients: [], details: {}, missing: {}, isLoading: false, isLoaded: false, error: null });
  });

  it('loads the list once and shares in-flight requests', async () => {
    const list = vi.spyOn(apiClientService, 'list').mockResolvedValue(ok({ data: [client('a')] }));
    await Promise.all([useApiClientStore.getState().loadClients(), useApiClientStore.getState().loadClients()]);
    await useApiClientStore.getState().loadClients();
    expect(list).toHaveBeenCalledTimes(1);
    expect(useApiClientStore.getState()).toMatchObject({ isLoaded: true, isLoading: false });
    expect(useApiClientStore.getState().clients.map((c) => c.id)).toEqual(['a']);

    await useApiClientStore.getState().loadClients(true);
    expect(list).toHaveBeenCalledTimes(2);
  });

  it('handles a missing list and list errors', async () => {
    vi.spyOn(apiClientService, 'list').mockResolvedValueOnce(ok({})).mockRejectedValueOnce(apiError(500, { message: 'down' }));
    await useApiClientStore.getState().loadClients(true);
    expect(useApiClientStore.getState().clients).toEqual([]);
    await useApiClientStore.getState().loadClients(true);
    expect(useApiClientStore.getState().error).toBe('down');
  });

  it('caches details, dedupes requests and records missing clients', async () => {
    const get = vi.spyOn(apiClientService, 'get').mockImplementation(async (id: string) => {
      if (id === 'gone') throw apiError(404);
      if (id === 'bad') throw apiError(400);
      if (id === 'err') throw apiError(500);
      return ok(client(id));
    });
    const s = () => useApiClientStore.getState();

    const [a1, a2] = await Promise.all([s().loadClient('a'), s().loadClient('a')]);
    expect(a1?.id).toBe('a');
    expect(a2).toBe(a1);
    expect(await s().loadClient('a')).toBe(a1);
    expect(get).toHaveBeenCalledTimes(1);
    await s().loadClient('a', true);
    expect(get).toHaveBeenCalledTimes(2);

    expect(await s().loadClient('gone')).toBeNull();
    expect(await s().loadClient('bad')).toBeNull();
    expect(await s().loadClient('err')).toBeNull();
    expect(s().missing).toEqual({ gone: true, bad: true });
  });

  it('reloads details only for clients that changed', async () => {
    useApiClientStore.setState({ details: { a: client('a', '1'), b: client('b', '1') } });
    vi.spyOn(apiClientService, 'list').mockResolvedValue(ok({ data: [client('a', '1'), client('b', '2')] }));
    const get = vi.spyOn(apiClientService, 'get').mockImplementation(async (id: string) => ok(client(id, '2')));

    await useApiClientStore.getState().loadAllDetails();

    expect(get).toHaveBeenCalledTimes(1);
    expect(get).toHaveBeenCalledWith('b');
  });

  it('upserts and removes clients', () => {
    const s = () => useApiClientStore.getState();
    useApiClientStore.setState({ missing: { a: true } });
    s().upsertClient(client('a'));
    s().upsertClient(client('b'));
    expect(s().clients.map((c) => c.id)).toEqual(['b', 'a']);
    expect(s().missing).toEqual({});

    s().upsertClient({ ...client('a'), name: 'renamed' });
    expect(s().clients.find((c) => c.id === 'a')?.name).toBe('renamed');
    expect(s().clients).toHaveLength(2);

    s().removeClient('a');
    expect(s().clients.map((c) => c.id)).toEqual(['b']);
    expect(s().details.a).toBeUndefined();
    expect(s().missing.a).toBe(true);
  });
});
