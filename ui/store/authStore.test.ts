import { beforeEach, describe, expect, it, vi } from 'vitest';
import { initializeAuth, useAuthStore } from './authStore';
import { authService } from '@/services/authService';
import { useToastStore } from '@/lib/toast';
import { Permissions, type User } from '@/types/auth';
import { apiError } from '@/test/apiMock';

const user = (overrides: Partial<User> = {}): User => ({
  id: 'u1',
  email: 'a@b.c',
  username: 'a',
  is_active: true,
  is_admin: false,
  roles: ['user'],
  created_at: '',
  updated_at: '',
  ...overrides,
});

const auth = (u: User = user()) => ({ user: u, access_token: 'a', refresh_token: 'r', expires_in: 1, token_type: 'Bearer' });
const lastToast = () => useToastStore.getState().toasts.at(-1);

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false, isInitialized: false, error: null });
    useToastStore.getState().clearAll();
  });

  describe('login', () => {
    it('stores the user on success', async () => {
      vi.spyOn(authService, 'login').mockResolvedValue(auth());
      expect(await useAuthStore.getState().login({ email: 'a', password: 'b' })).toBe(true);
      expect(useAuthStore.getState()).toMatchObject({ isAuthenticated: true, isLoading: false, user: { id: 'u1' } });
      expect(lastToast()?.type).toBe('success');
    });

    it.each([
      ['INVALID_CREDENTIALS', 'Invalid email or password'],
      ['ACCOUNT_LOCKED', 'Your account has been locked. Please contact support.'],
      ['ACCOUNT_INACTIVE', 'Your account is not active.'],
      ['OTHER', 'msg'],
    ])('maps %s to a specific message', async (code, message) => {
      vi.spyOn(authService, 'login').mockRejectedValue(apiError(401, { code, message: 'msg' }));
      expect(await useAuthStore.getState().login({ email: 'a', password: 'b' })).toBe(false);
      expect(useAuthStore.getState()).toMatchObject({ isAuthenticated: false, isLoading: false, error: 'msg' });
      expect(lastToast()?.message).toBe(message);
    });
  });

  describe('register', () => {
    it('stores the user on success', async () => {
      vi.spyOn(authService, 'register').mockResolvedValue(auth());
      expect(await useAuthStore.getState().register({ email: 'a', password: 'b' })).toBe(true);
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });

    it.each([
      ['EMAIL_ALREADY_TAKEN', 'This email is already registered'],
      ['VALIDATION_FAILED', 'msg'],
      ['OTHER', 'msg'],
    ])('maps %s', async (code, message) => {
      vi.spyOn(authService, 'register').mockRejectedValue(apiError(400, { code, message: 'msg' }));
      expect(await useAuthStore.getState().register({ email: 'a', password: 'b' })).toBe(false);
      expect(lastToast()?.message).toBe(message);
    });
  });

  it('logs out even when the API call fails', async () => {
    useAuthStore.setState({ user: user(), isAuthenticated: true });
    vi.spyOn(authService, 'logout').mockRejectedValue(new Error('x'));
    await useAuthStore.getState().logout();
    expect(useAuthStore.getState()).toMatchObject({ user: null, isAuthenticated: false, isLoading: false });
  });

  describe('checkAuth', () => {
    it('marks unauthenticated users without a token', async () => {
      vi.spyOn(authService, 'isAuthenticated').mockReturnValue(false);
      await initializeAuth();
      expect(useAuthStore.getState()).toMatchObject({ isInitialized: true, isAuthenticated: false });
    });

    it('loads the current user', async () => {
      vi.spyOn(authService, 'isAuthenticated').mockReturnValue(true);
      vi.spyOn(authService, 'getCurrentUser').mockResolvedValue(user({ id: 'me' }));
      await useAuthStore.getState().checkAuth();
      expect(useAuthStore.getState()).toMatchObject({ isInitialized: true, isAuthenticated: true, user: { id: 'me' } });
    });

    it('refreshes the token when /auth/me fails', async () => {
      vi.spyOn(authService, 'isAuthenticated').mockReturnValue(true);
      vi.spyOn(authService, 'getCurrentUser').mockRejectedValue(new Error('expired'));
      vi.spyOn(authService, 'refreshToken').mockResolvedValue(auth(user({ id: 'refreshed' })));
      await useAuthStore.getState().checkAuth();
      expect(useAuthStore.getState().user?.id).toBe('refreshed');
    });

    it('clears tokens when refresh also fails', async () => {
      vi.spyOn(authService, 'isAuthenticated').mockReturnValue(true);
      vi.spyOn(authService, 'getCurrentUser').mockRejectedValue(new Error('expired'));
      vi.spyOn(authService, 'refreshToken').mockRejectedValue(new Error('nope'));
      const clear = vi.spyOn(authService, 'clearTokens');
      await useAuthStore.getState().checkAuth();
      expect(clear).toHaveBeenCalled();
      expect(useAuthStore.getState()).toMatchObject({ isAuthenticated: false, isInitialized: true, user: null });
    });
  });

  describe('changePassword', () => {
    it('logs out after a successful change', async () => {
      vi.spyOn(authService, 'changePassword').mockResolvedValue();
      const logout = vi.spyOn(authService, 'logout').mockResolvedValue();
      useAuthStore.setState({ user: user(), isAuthenticated: true });
      expect(await useAuthStore.getState().changePassword({ old_password: 'a', new_password: 'bbbbbbbb' })).toBe(true);
      expect(logout).toHaveBeenCalled();
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });

    it('reports failures', async () => {
      vi.spyOn(authService, 'changePassword').mockRejectedValue(apiError(400, { message: 'wrong password' }));
      expect(await useAuthStore.getState().changePassword({ old_password: 'a', new_password: 'b' })).toBe(false);
      expect(useAuthStore.getState().error).toBe('wrong password');
      expect(lastToast()).toMatchObject({ title: 'Password Change Failed', message: 'wrong password' });
    });
  });

  it('updates the user and clears errors', () => {
    useAuthStore.getState().updateUser(user({ id: 'x' }));
    expect(useAuthStore.getState().user?.id).toBe('x');
    useAuthStore.setState({ error: 'e' });
    useAuthStore.getState().clearError();
    expect(useAuthStore.getState().error).toBeNull();
  });

  describe('permissions', () => {
    it('denies everything without a user', () => {
      const s = useAuthStore.getState();
      expect(s.hasPermission(Permissions.WORKFLOW_READ)).toBe(false);
      expect(s.hasRole('user')).toBe(false);
      expect(s.isAdmin()).toBe(false);
    });

    it('grants everything to admins', () => {
      useAuthStore.setState({ user: user({ is_admin: true, roles: [] }) });
      expect(useAuthStore.getState().hasPermission(Permissions.SYSTEM_ADMIN)).toBe(true);
      expect(useAuthStore.getState().isAdmin()).toBe(true);
    });

    it('derives permissions from roles', () => {
      useAuthStore.setState({ user: user({ roles: ['Viewer'] }) });
      const s = useAuthStore.getState();
      expect(s.hasPermission(Permissions.WORKFLOW_READ)).toBe(true);
      expect(s.hasPermission(Permissions.WORKFLOW_CREATE)).toBe(false);
      expect(s.hasRole('viewer')).toBe(true);
      expect(s.hasRole('admin')).toBe(false);

      useAuthStore.setState({ user: user({ roles: ['admin'] }) });
      expect(useAuthStore.getState().hasPermission(Permissions.ROLE_MANAGE)).toBe(true);

      useAuthStore.setState({ user: user({ roles: ['unknown'] }) });
      expect(useAuthStore.getState().hasPermission(Permissions.WORKFLOW_READ)).toBe(false);
    });
  });

  it('persists only the user and auth flag', async () => {
    useAuthStore.setState({ user: user({ id: 'persisted' }), isAuthenticated: true, isLoading: true });
    const stored = JSON.parse(localStorage.getItem('auth-storage') || '{}');
    expect(stored.state).toEqual({ user: expect.objectContaining({ id: 'persisted' }), isAuthenticated: true });
  });
});
