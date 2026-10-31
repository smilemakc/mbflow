/**
 * Thin REST wrappers: credentials, API clients, rental keys and resources.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { credentialsApi, getCredentialTypeLabel, COMMON_PROVIDERS } from './credentialsService';
import { apiClientService } from './apiClientService';
import {
  calculateUsagePercent,
  formatTokenCount,
  getProviderLabel,
  getProvisionerLabel,
  getStatusColor,
  isNearLimit,
  LLM_PROVIDERS,
  PROVISIONER_TYPES,
  rentalKeyAdminApi,
  rentalKeyApi,
} from './rentalKeyService';
import { resourcesApi } from './resources';
import { mockApi, type ApiMock } from '@/test/apiMock';

let api: ApiMock;
beforeEach(() => {
  api = mockApi();
  api.on('GET', /.*/, { data: {} }).on('POST', /.*/, { data: {} }).on('PUT', /.*/, { data: {} }).on('DELETE', /.*/, { data: {} });
});
afterEach(() => api.restore());

const last = () => {
  const call = api.last()!;
  return { method: call.method, url: call.url, data: call.data, params: call.params };
};

describe('credentialsApi', () => {
  it('hits the credential endpoints', async () => {
    await credentialsApi.listCredentials();
    expect(last()).toMatchObject({ method: 'GET', url: '/credentials' });
    await credentialsApi.listCredentials('open ai');
    expect(last().url).toBe('/credentials?provider=open%20ai');
    await credentialsApi.getCredential('c');
    expect(last().url).toBe('/credentials/c');
    await credentialsApi.getCredentialSecrets('c');
    expect(last().url).toBe('/credentials/c/secrets');

    const creates: Array<[() => Promise<unknown>, string]> = [
      [() => credentialsApi.createAPIKey({ name: 'n', api_key: 'k' }), '/credentials/api-key'],
      [() => credentialsApi.createBasicAuth({ name: 'n', username: 'u', password: 'p' }), '/credentials/basic-auth'],
      [() => credentialsApi.createOAuth2({ name: 'n', client_id: 'i', client_secret: 's' }), '/credentials/oauth2'],
      [() => credentialsApi.createServiceAccount({ name: 'n', json_key: '{}' }), '/credentials/service-account'],
      [() => credentialsApi.createCustom({ name: 'n', data: { a: 'b' } }), '/credentials/custom'],
    ];
    for (const [fn, url] of creates) {
      await fn();
      expect(last()).toMatchObject({ method: 'POST', url });
    }
    await credentialsApi.updateCredential('c', { name: 'x' });
    expect(last()).toMatchObject({ method: 'PUT', url: '/credentials/c', data: { name: 'x' } });
    await credentialsApi.deleteCredential('c');
    expect(last()).toMatchObject({ method: 'DELETE', url: '/credentials/c' });
  });

  it('labels credential types', () => {
    expect(getCredentialTypeLabel('api_key')).toBe('API Key');
    expect(getCredentialTypeLabel('service_account')).toBe('Service Account');
    expect(getCredentialTypeLabel('other' as never)).toBe('other');
    expect(COMMON_PROVIDERS.some((p) => p.value === 'openai')).toBe(true);
  });
});

describe('apiClientService', () => {
  const file = new File(['openapi: 3.0.0'], 'spec.yaml');

  it('uploads specs as multipart forms with settings', async () => {
    await apiClientService.preview(file);
    let call = api.last()!;
    expect(call.url).toBe('/api-clients/preview');
    expect(call.data).toBeInstanceOf(FormData);
    expect((call.data as FormData).get('file')).toBeInstanceOf(File);
    expect(call.config.timeout).toBe(120_000);

    await apiClientService.create(file, {
      name: 'n',
      description: 'd',
      base_url: 'https://x',
      auth: { type: 'bearer' },
      default_headers: { A: 'b' },
    });
    call = api.last()!;
    const form = call.data as FormData;
    expect(form.get('name')).toBe('n');
    expect(form.get('description')).toBe('d');
    expect(form.get('base_url')).toBe('https://x');
    expect(form.get('auth')).toBe('{"type":"bearer"}');
    expect(form.get('default_headers')).toBe('{"A":"b"}');

    await apiClientService.create(file, {});
    expect((api.last()!.data as FormData).get('name')).toBeNull();
  });

  it('lists, reads, updates and removes clients', async () => {
    await apiClientService.list();
    expect(last()).toMatchObject({ url: '/api-clients', params: { limit: 100 } });
    await apiClientService.get('a');
    expect(last().url).toBe('/api-clients/a');
    await apiClientService.update('a', { name: 'x' });
    expect(last()).toMatchObject({ method: 'PUT', data: { name: 'x' } });
    await apiClientService.replaceSpec('a', file);
    expect(last()).toMatchObject({ method: 'PUT', url: '/api-clients/a/spec' });
    await apiClientService.downloadSpec('a');
    expect(api.last()!.config.responseType).toBe('blob');
    await apiClientService.remove('a');
    expect(last()).toMatchObject({ method: 'DELETE', url: '/api-clients/a' });
  });
});

describe('rentalKeyApi', () => {
  it('hits the user endpoints', async () => {
    await rentalKeyApi.listRentalKeys();
    expect(last().url).toBe('/rental-keys');
    await rentalKeyApi.listRentalKeys('openai');
    expect(last().url).toBe('/rental-keys?provider=openai');
    await rentalKeyApi.getRentalKey('k');
    expect(last().url).toBe('/rental-keys/k');
    await rentalKeyApi.getUsageHistory('k');
    expect(last().url).toBe('/rental-keys/k/usage');
    await rentalKeyApi.getUsageHistory('k', 10, 0);
    expect(last().url).toBe('/rental-keys/k/usage?limit=10&offset=0');
    await rentalKeyApi.getUsageSummary('k');
    expect(last().url).toBe('/rental-keys/k/summary');
  });

  it('hits the admin endpoints', async () => {
    await rentalKeyAdminApi.createRentalKey({ owner_id: 'o', name: 'n', provider: 'openai', api_key: 'k' });
    expect(last()).toMatchObject({ method: 'POST', url: '/admin/rental-keys' });
    await rentalKeyAdminApi.listAllRentalKeys();
    expect(last().url).toBe('/admin/rental-keys');
    await rentalKeyAdminApi.listAllRentalKeys({ provider: 'anthropic', status: 'active', owner_id: 'o', created_by: 'c', limit: 5, offset: 0 });
    expect(last().url).toBe('/admin/rental-keys?provider=anthropic&status=active&owner_id=o&created_by=c&limit=5&offset=0');
    await rentalKeyAdminApi.getRentalKey('k');
    await rentalKeyAdminApi.updateRentalKey('k', { name: 'x' });
    expect(last()).toMatchObject({ method: 'PUT', url: '/admin/rental-keys/k' });
    await rentalKeyAdminApi.rotateAPIKey('k', { new_api_key: 'n' });
    expect(last()).toMatchObject({ url: '/admin/rental-keys/k/rotate-key', data: { new_api_key: 'n' } });
    await rentalKeyAdminApi.deleteRentalKey('k');
    expect(last().method).toBe('DELETE');
    await rentalKeyAdminApi.resetDailyUsage();
    expect(last().url).toBe('/admin/rental-keys/reset-daily');
    await rentalKeyAdminApi.resetMonthlyUsage();
    expect(last().url).toBe('/admin/rental-keys/reset-monthly');
  });

  it('formats and labels', () => {
    expect(getProviderLabel('google_ai')).toBe('Google AI');
    expect(getProviderLabel('x' as never)).toBe('x');
    expect(getProvisionerLabel('auto_openai')).toBe('Auto (OpenAI)');
    expect(getProvisionerLabel('y' as never)).toBe('y');
    expect(getStatusColor('suspended')).toBe('yellow');
    expect(getStatusColor('z' as never)).toBe('gray');
    expect(formatTokenCount(999)).toBe('999');
    expect(formatTokenCount(1500)).toBe('1.50K');
    expect(formatTokenCount(2_500_000)).toBe('2.50M');
    expect(formatTokenCount(3_000_000_000)).toBe('3.00B');
    expect(calculateUsagePercent(5)).toBeNull();
    expect(calculateUsagePercent(5, 0)).toBeNull();
    expect(calculateUsagePercent(5, 10)).toBe(50);
    expect(calculateUsagePercent(50, 10)).toBe(100);
    expect(isNearLimit(85, 100)).toBe(true);
    expect(isNearLimit(50, 100)).toBe(false);
    expect(isNearLimit(50, 100, 50)).toBe(true);
    expect(isNearLimit(50)).toBe(false);
    expect(LLM_PROVIDERS).toHaveLength(3);
    expect(PROVISIONER_TYPES).toHaveLength(4);
  });
});

describe('resourcesApi', () => {
  it('hits the resource and account endpoints', async () => {
    await resourcesApi.listResources();
    expect(last().url).toBe('/resources');
    await resourcesApi.getResource('r');
    await resourcesApi.createFileStorage('n', 'd');
    expect(last()).toMatchObject({ url: '/resources/file-storage', data: { name: 'n', description: 'd' } });
    await resourcesApi.deleteResource('r');
    await resourcesApi.updateResource('r', 'n');
    expect(last()).toMatchObject({ method: 'PUT', url: '/resources/r' });
    await resourcesApi.listPricingPlans();
    await resourcesApi.uploadFile('r', new File(['x'], 'x.txt'));
    expect(api.last()!.data).toBeInstanceOf(FormData);
    await resourcesApi.listFiles('r');
    expect(last().url).toBe('/resources/r/files?limit=50&offset=0');
    await resourcesApi.listFiles('r', 10, 20);
    expect(last().url).toBe('/resources/r/files?limit=10&offset=20');
    await resourcesApi.getFileMetadata('r', 'f');
    await resourcesApi.deleteFile('r', 'f');
    expect(last()).toMatchObject({ method: 'DELETE', url: '/resources/r/files/f' });
    await resourcesApi.getAccount();
    await resourcesApi.deposit(5, 'key');
    expect(last().data).toEqual({ amount: 5, idempotency_key: 'key' });
    await resourcesApi.listTransactions();
    expect(last().url).toBe('/account/transactions?limit=20&offset=0');
    await resourcesApi.listTransactions(5, 5);
    expect(last().url).toBe('/account/transactions?limit=5&offset=5');
  });

  describe('downloadFile', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      URL.createObjectURL = vi.fn(() => 'blob:url');
      URL.revokeObjectURL = vi.fn();
    });
    afterEach(() => vi.useRealTimers());

    const download = async (headers: Record<string, string>, fileName?: string) => {
      api.once('GET', '/resources/r/files/f/download', { data: new Blob(['x']), headers });
      let downloaded = '';
      const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
        downloaded = this.download;
      });
      await resourcesApi.downloadFile('r', 'f', fileName);
      click.mockRestore();
      return downloaded;
    };

    it('uses the explicit file name', async () => {
      expect(await download({}, 'given.txt')).toBe('given.txt');
      vi.runAllTimers();
      expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:url');
      expect(document.querySelector('a')).toBeNull();
    });

    it('parses Content-Disposition (RFC 5987 and plain)', async () => {
      expect(await download({ 'content-disposition': "attachment; filename*=UTF-8''r%C3%A9sum%C3%A9.pdf" })).toBe('résumé.pdf');
      expect(await download({ 'content-disposition': 'attachment; filename="report.csv"' })).toBe('report.csv');
      expect(await download({ 'content-disposition': "attachment; filename*=UTF-8''%E0%A4%A; filename=\"fallback.bin\"" })).toBe('fallback.bin');
    });

    it('falls back to the file id', async () => {
      expect(await download({})).toBe('f');
      expect(await download({ 'content-disposition': 'inline' })).toBe('f');
    });
  });
});
