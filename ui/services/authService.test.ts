import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { authService } from './authService';
import { REFRESH_TOKEN_KEY, TOKEN_KEY } from '@/lib/authStorage';
import { mockApi, type ApiMock } from '@/test/apiMock';

const user = { id: 'u1', email: 'a@b.c', username: 'a', is_active: true, is_admin: false, roles: ['user'], created_at: '', updated_at: '' };
const authResponse = { user, access_token: 'acc', refresh_token: 'ref', expires_in: 3600, token_type: 'Bearer' };

describe('authService', () => {
  let api: ApiMock;
  beforeEach(() => {
    api = mockApi();
  });
  afterEach(() => api.restore());

  it('stores tokens on login and register', async () => {
    api.on('POST', '/auth/login', { data: authResponse }).on('POST', '/auth/register', { data: { ...authResponse, access_token: 'acc2', refresh_token: undefined } });

    await authService.login({ email: 'a@b.c', password: 'pw' });
    expect(api.last()?.data).toEqual({ email: 'a@b.c', password: 'pw' });
    expect(localStorage.getItem(TOKEN_KEY)).toBe('acc');
    expect(localStorage.getItem(REFRESH_TOKEN_KEY)).toBe('ref');
    expect(authService.isAuthenticated()).toBe(true);

    await authService.register({ email: 'a@b.c', password: 'pw' });
    expect(authService.getAccessToken()).toBe('acc2');
    expect(authService.getRefreshToken()).toBe('ref');
  });

  it('clears tokens on logout even when the request fails', async () => {
    authService.setTokens('a', 'r');
    api.on('POST', '/auth/logout', { status: 500 });
    await expect(authService.logout()).rejects.toBeTruthy();
    expect(authService.getAccessToken()).toBeNull();
    expect(authService.getRefreshToken()).toBeNull();
    expect(authService.isAuthenticated()).toBe(false);
  });

  it('refreshes tokens', async () => {
    await expect(authService.refreshToken()).rejects.toThrow('No refresh token available');

    authService.setTokens('a', 'r1');
    api.on('POST', '/auth/refresh', { data: { ...authResponse, access_token: 'a2', refresh_token: 'r2' } });
    await authService.refreshToken();
    expect(api.last()?.data).toEqual({ refresh_token: 'r1' });
    expect(authService.getAccessToken()).toBe('a2');
  });

  it('reads the current user, auth info and changes password', async () => {
    api
      .on('GET', '/auth/me', { data: { user } })
      .on('GET', '/auth/info', { data: { mode: 'builtin', providers: [] } })
      .on('POST', '/auth/password', { data: { message: 'ok' } });

    expect(await authService.getCurrentUser()).toEqual(user);
    expect(await authService.getAuthInfo()).toEqual({ mode: 'builtin', providers: [] });
    await authService.changePassword({ old_password: 'o', new_password: 'newpassword' });
    expect(api.last()?.data).toEqual({ old_password: 'o', new_password: 'newpassword' });
    expect(authService.getOAuthAuthorizeUrl()).toBe('/api/v1/auth/oauth/authorize');
  });

  it('calls the admin user endpoints', async () => {
    api
      .on('GET', '/admin/users', { data: { data: [user], total: 1 } })
      .on('GET', '/admin/users/u1', { data: { user } })
      .on('POST', '/admin/users', { data: { user } })
      .on('PUT', '/admin/users/u1', { data: { user } })
      .on('DELETE', '/admin/users/u1', { data: {} })
      .on('POST', '/admin/users/u1/reset-password', { data: {} });

    expect(await authService.listUsers()).toEqual({ data: [user], total: 1 });
    expect(api.last()?.params).toEqual({ limit: 50, offset: 0 });
    await authService.listUsers(5, 10);
    expect(api.last()?.params).toEqual({ limit: 5, offset: 10 });
    expect(await authService.getUser('u1')).toEqual(user);
    expect(await authService.createUser({ email: 'x', password: 'y' })).toEqual(user);
    expect(await authService.updateUser('u1', { email: 'x' } as never)).toEqual(user);
    await authService.deleteUser('u1');
    await authService.resetUserPassword('u1', 'pw123456');
    expect(api.last()?.data).toEqual({ new_password: 'pw123456' });
  });

  it('calls the role endpoints', async () => {
    const roles = [{ id: 'r', name: 'admin' }];
    api
      .on('GET', '/admin/roles', { data: { roles } })
      .on('GET', '/admin/users/u1/roles', { data: { roles } })
      .on('POST', '/admin/users/u1/roles', { data: {} })
      .on('DELETE', '/admin/users/u1/roles/r', { data: {} });

    expect(await authService.listRoles()).toEqual(roles);
    expect(await authService.getUserRoles('u1')).toEqual(roles);
    await authService.assignRole('u1', 'r');
    expect(api.last()?.data).toEqual({ role_id: 'r' });
    await authService.removeRole('u1', 'r');
    expect(api.last()?.method).toBe('DELETE');
  });
});
