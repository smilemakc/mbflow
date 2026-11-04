import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import { credentialsApi } from '@/services/credentialsService';
import { resourcesApi, type FileStorageResource } from '@/services/resources';
import { rentalKeyApi } from '@/services/rentalKeyService';
import { useToastStore } from '@/lib/toast';
import { apiError } from '@/test/apiMock';
import { useCredentials } from './useCredentials';
import { useFileStorage } from './useFileStorage';
import { useRentalKeys } from './useRentalKeys';
import { useResources } from './useResources';

// The hooks only read `response.data`; wrap payloads like axios does.
const ok = <T,>(data: T) => ({ data }) as never;
const fail = (message: string) => apiError(500, { message });
const toasts = () => useToastStore.getState().toasts.map((t) => `${t.type}:${t.title}:${t.message ?? ''}`);

beforeEach(() => {
  vi.spyOn(console, 'error').mockImplementation(() => {});
});
afterEach(() => {
  useToastStore.getState().clearAll();
});

describe('useCredentials', () => {
  it('loads credentials on mount and with a provider filter', async () => {
    const list = vi
      .spyOn(credentialsApi, 'listCredentials')
      .mockResolvedValueOnce(ok({ credentials: [{ id: 'c1' }] }))
      .mockResolvedValueOnce(ok({}));
    const { result } = renderHook(() => useCredentials());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.credentials).toEqual([{ id: 'c1' }]);

    await act(() => result.current.loadCredentials('openai'));
    expect(list).toHaveBeenLastCalledWith('openai');
    expect(result.current.credentials).toEqual([]);
  });

  it('surfaces load errors', async () => {
    vi.spyOn(credentialsApi, 'listCredentials').mockRejectedValue(fail('db down'));
    const { result } = renderHook(() => useCredentials());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe('db down');
    expect(toasts()).toEqual(['error:Load Failed:db down']);
  });

  it('creates each credential type through the matching endpoint', async () => {
    vi.spyOn(credentialsApi, 'listCredentials').mockResolvedValue(ok({ credentials: [] }));
    const spies = {
      api_key: vi.spyOn(credentialsApi, 'createAPIKey').mockResolvedValue(ok({})),
      basic_auth: vi.spyOn(credentialsApi, 'createBasicAuth').mockResolvedValue(ok({})),
      oauth2: vi.spyOn(credentialsApi, 'createOAuth2').mockResolvedValue(ok({})),
      service_account: vi.spyOn(credentialsApi, 'createServiceAccount').mockResolvedValue(ok({})),
      custom: vi.spyOn(credentialsApi, 'createCustom').mockResolvedValue(ok({})),
    };
    const { result } = renderHook(() => useCredentials());
    await waitFor(() => expect(result.current.loading).toBe(false));

    for (const [type, spy] of Object.entries(spies)) {
      let created = false;
      await act(async () => {
        created = await result.current.createCredential(type as never, { name: type });
      });
      expect(created).toBe(true);
      expect(spy).toHaveBeenCalledWith({ name: type });
    }
    expect(toasts()).toHaveLength(5);

    let created = true;
    await act(async () => {
      created = await result.current.createCredential('bogus' as never, {});
    });
    expect(created).toBe(false);
    expect(toasts().at(-1)).toBe('error:Create Failed:Unknown credential type: bogus');
  });

  it('updates, deletes and reads secrets, handling failures', async () => {
    const list = vi.spyOn(credentialsApi, 'listCredentials').mockResolvedValue(ok({ credentials: [] }));
    const update = vi.spyOn(credentialsApi, 'updateCredential').mockResolvedValueOnce(ok({})).mockRejectedValueOnce(fail('u'));
    vi.spyOn(credentialsApi, 'deleteCredential').mockResolvedValueOnce(ok({})).mockRejectedValueOnce(fail('d'));
    vi.spyOn(credentialsApi, 'getCredentialSecrets')
      .mockResolvedValueOnce(ok({ id: 'c1', data: { k: 'v' } }))
      .mockRejectedValueOnce(fail('s'));
    const { result } = renderHook(() => useCredentials());
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(await act(() => result.current.updateCredential('c1', 'n', 'd'))).toBe(true);
    expect(update).toHaveBeenCalledWith('c1', { name: 'n', description: 'd' });
    expect(await act(() => result.current.updateCredential('c1', 'n'))).toBe(false);
    expect(await act(() => result.current.deleteCredential('c1'))).toBe(true);
    expect(await act(() => result.current.deleteCredential('c1'))).toBe(false);
    expect(await act(() => result.current.getSecrets('c1'))).toEqual({ id: 'c1', data: { k: 'v' } });
    expect(await act(() => result.current.getSecrets('c1'))).toBeNull();
    expect(list).toHaveBeenCalledTimes(3);
    expect(toasts()).toEqual([
      'success:Success:Credential updated successfully.',
      'error:Update Failed:u',
      'success:Success:Credential deleted successfully.',
      'error:Delete Failed:d',
      'error:Failed:s',
    ]);
  });
});

describe('useFileStorage', () => {
  const resource = { id: 'r1' } as FileStorageResource;

  it('requires a selected resource for mutations', async () => {
    const { result } = renderHook(() => useFileStorage());
    expect(await act(() => result.current.uploadFile(new File(['x'], 'x.txt')))).toBe(false);
    expect(await act(() => result.current.deleteFile('f', 'x'))).toBe(false);
    await act(() => result.current.downloadFile('f'));
    expect(toasts()).toEqual(Array(3).fill('error:Error:No resource selected.'));
  });

  it('lists, uploads, deletes, downloads and clears files', async () => {
    const onResourceUpdate = vi.fn().mockResolvedValue(undefined);
    const listFiles = vi
      .spyOn(resourcesApi, 'listFiles')
      .mockResolvedValue(ok({ files: [{ id: 'f1', name: 'a.txt' }], total: 1 }));
    const upload = vi.spyOn(resourcesApi, 'uploadFile').mockResolvedValue(ok({}));
    const del = vi.spyOn(resourcesApi, 'deleteFile').mockResolvedValue(ok({}));
    const download = vi.spyOn(resourcesApi, 'downloadFile').mockResolvedValue(undefined);
    const { result } = renderHook(() => useFileStorage(onResourceUpdate));

    await act(() => result.current.loadFiles(resource));
    expect(result.current).toMatchObject({ selectedResource: resource, filesTotal: 1, filesLoading: false });
    expect(result.current.files).toEqual([{ id: 'f1', name: 'a.txt' }]);

    const file = new File(['x'], 'x.txt');
    expect(await act(() => result.current.uploadFile(file))).toBe(true);
    expect(upload).toHaveBeenCalledWith('r1', file);
    expect(await act(() => result.current.deleteFile('f1', 'a.txt'))).toBe(true);
    expect(del).toHaveBeenCalledWith('r1', 'f1');
    expect(onResourceUpdate).toHaveBeenCalledTimes(2);
    expect(listFiles).toHaveBeenCalledTimes(3);

    await act(() => result.current.downloadFile('f1'));
    expect(download).toHaveBeenCalledWith('r1', 'f1', 'a.txt');
    await act(() => result.current.downloadFile('missing'));
    expect(download).toHaveBeenLastCalledWith('r1', 'missing', undefined);

    act(() => result.current.clearSelection());
    expect(result.current).toMatchObject({ selectedResource: null, files: [], filesTotal: 0 });
  });

  it('reports failures for each operation and works without an update callback', async () => {
    const listFiles = vi.spyOn(resourcesApi, 'listFiles').mockResolvedValueOnce(ok({}));
    vi.spyOn(resourcesApi, 'uploadFile').mockResolvedValueOnce(ok({})).mockRejectedValueOnce(fail('up'));
    vi.spyOn(resourcesApi, 'deleteFile').mockResolvedValueOnce(ok({})).mockRejectedValueOnce(fail('del'));
    vi.spyOn(resourcesApi, 'downloadFile').mockRejectedValue(fail('dl'));
    const { result } = renderHook(() => useFileStorage());
    await act(() => result.current.loadFiles(resource));
    expect(result.current.files).toEqual([]);
    expect(result.current.filesTotal).toBe(0);

    listFiles.mockRejectedValueOnce(fail('list'));
    expect(await act(() => result.current.uploadFile(new File([], 'f')))).toBe(true);
    listFiles.mockResolvedValue(ok({ files: [] }));
    expect(await act(() => result.current.uploadFile(new File([], 'f')))).toBe(false);
    expect(await act(() => result.current.deleteFile('f', 'f'))).toBe(true);
    expect(await act(() => result.current.deleteFile('f', 'f'))).toBe(false);
    await act(() => result.current.downloadFile('f'));
    expect(toasts().filter((t) => t.startsWith('error'))).toEqual([
      'error:Load Failed:list',
      'error:Upload Failed:up',
      'error:Delete Failed:del',
      'error:Download Failed:dl',
    ]);
  });
});

describe('useRentalKeys', () => {
  it('loads with the initial provider and remembers explicit filters for refresh', async () => {
    const list = vi.spyOn(rentalKeyApi, 'listRentalKeys').mockResolvedValue(ok({ rental_keys: [{ id: 'k1' }] }));
    const { result } = renderHook(() => useRentalKeys('openai'));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(list).toHaveBeenLastCalledWith('openai');
    expect(result.current.rentalKeys).toEqual([{ id: 'k1' }]);

    list.mockResolvedValueOnce(ok({}));
    await act(() => result.current.loadRentalKeys('anthropic'));
    expect(list).toHaveBeenLastCalledWith('anthropic');
    expect(result.current.rentalKeys).toEqual([]);

    await act(() => result.current.refresh());
    expect(list).toHaveBeenLastCalledWith('anthropic');
  });

  it('reports load errors', async () => {
    vi.spyOn(rentalKeyApi, 'listRentalKeys').mockRejectedValue(fail('nope'));
    const { result } = renderHook(() => useRentalKeys());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe('nope');
    expect(toasts()).toEqual(['error:Load Failed:nope']);
  });

  it('fetches key details, usage history and summary with fallbacks', async () => {
    vi.spyOn(rentalKeyApi, 'listRentalKeys').mockResolvedValue(ok({ rental_keys: [] }));
    vi.spyOn(rentalKeyApi, 'getRentalKey').mockResolvedValueOnce(ok({ id: 'k1' })).mockRejectedValueOnce(fail('a'));
    const history = vi
      .spyOn(rentalKeyApi, 'getUsageHistory')
      .mockResolvedValueOnce(ok({ usage: [{ id: 'u1' }] }))
      .mockResolvedValueOnce(ok({}))
      .mockRejectedValueOnce(fail('b'));
    vi.spyOn(rentalKeyApi, 'getUsageSummary')
      .mockResolvedValueOnce(ok({ summary: { total: 1 } }))
      .mockRejectedValueOnce(fail('c'));
    const { result } = renderHook(() => useRentalKeys());
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(await act(() => result.current.getRentalKey('k1'))).toEqual({ id: 'k1' });
    expect(await act(() => result.current.getRentalKey('k1'))).toBeNull();
    expect(await act(() => result.current.getUsageHistory('k1', 10, 5))).toEqual([{ id: 'u1' }]);
    expect(history).toHaveBeenCalledWith('k1', 10, 5);
    expect(await act(() => result.current.getUsageHistory('k1'))).toEqual([]);
    expect(await act(() => result.current.getUsageHistory('k1'))).toEqual([]);
    expect(await act(() => result.current.getUsageSummary('k1'))).toEqual({ total: 1 });
    expect(await act(() => result.current.getUsageSummary('k1'))).toBeNull();
    expect(toasts()).toEqual(['error:Failed:a', 'error:Failed:b', 'error:Failed:c']);
  });
});

describe('useResources', () => {
  const mockLoad = () => {
    vi.spyOn(resourcesApi, 'listResources').mockResolvedValue(
      ok({ resources: [{ id: 'r1', type: 'file_storage' }, { id: 'r2', type: 'other' }] }),
    );
    vi.spyOn(resourcesApi, 'getAccount').mockResolvedValue(ok({ balance: 10 }));
    return vi.spyOn(resourcesApi, 'listTransactions').mockResolvedValue(ok({ transactions: [{ id: 't1' }], total: 7 }));
  };

  it('loads file-storage resources, account and recent transactions', async () => {
    const tx = mockLoad();
    const { result } = renderHook(() => useResources());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.resources).toEqual([{ id: 'r1', type: 'file_storage' }]);
    expect(result.current.account).toEqual({ balance: 10 });
    expect(result.current.transactions).toEqual([{ id: 't1' }]);
    expect(result.current.transactionsTotal).toBe(7);
    expect(tx).toHaveBeenCalledWith(10, 0);
  });

  it('handles empty payloads and load failures', async () => {
    vi.spyOn(resourcesApi, 'listResources').mockResolvedValueOnce(ok({})).mockRejectedValueOnce(fail('down'));
    vi.spyOn(resourcesApi, 'getAccount').mockResolvedValue(ok(null));
    vi.spyOn(resourcesApi, 'listTransactions').mockResolvedValue(ok({}));
    const { result } = renderHook(() => useResources());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.resources).toEqual([]);
    expect(result.current.transactions).toEqual([]);
    expect(result.current.transactionsTotal).toBe(0);

    await act(() => result.current.loadData());
    expect(toasts()).toEqual(['error:Load Failed:down']);
  });

  it('creates storage, deletes resources and deposits funds', async () => {
    mockLoad();
    const create = vi.spyOn(resourcesApi, 'createFileStorage').mockResolvedValueOnce(ok({})).mockRejectedValueOnce(fail('c'));
    const del = vi.spyOn(resourcesApi, 'deleteResource').mockResolvedValueOnce(ok({})).mockRejectedValueOnce(fail('d'));
    const deposit = vi.spyOn(resourcesApi, 'deposit').mockResolvedValueOnce(ok({})).mockRejectedValueOnce(fail('p'));
    const { result } = renderHook(() => useResources());
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(await act(() => result.current.createStorage('n', 'd'))).toBe(true);
    expect(create).toHaveBeenCalledWith('n', 'd');
    expect(await act(() => result.current.createStorage('n', 'd'))).toBe(false);
    expect(await act(() => result.current.deleteResource('r1'))).toBe(true);
    expect(del).toHaveBeenCalledWith('r1');
    expect(await act(() => result.current.deleteResource('r1'))).toBe(false);
    expect(await act(() => result.current.deposit(5))).toBe(true);
    expect(deposit.mock.calls[0][0]).toBe(5);
    expect(deposit.mock.calls[0][1]).toMatch(/^deposit-\d+-[a-z0-9]+$/);
    expect(await act(() => result.current.deposit(5))).toBe(false);
    expect(toasts()).toEqual([
      'success:Success:File storage created successfully.',
      'error:Create Failed:c',
      'success:Success:Resource deleted successfully.',
      'error:Delete Failed:d',
      'success:Success:Deposited 5.00 successfully.',
      'error:Deposit Failed:p',
    ]);
  });
});
