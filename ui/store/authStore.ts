/**
 * Authentication Store
 * Manages authentication state using Zustand with persistence
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { authService } from '@/services/authService';
import { toast } from '@/lib/toast';
import { getErrorMessage, isErrorCode, ErrorCodes } from '@/lib/api';
import type {
  User,
  LoginCredentials,
  RegisterData,
  ChangePasswordRequest,
  Permission,
} from '@/types/auth';
import { Permissions } from '@/types/auth';

interface AuthState {
  // State
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isInitialized: boolean;
  error: string | null;

  // Actions
  login: (credentials: LoginCredentials) => Promise<boolean>;
  register: (data: RegisterData) => Promise<boolean>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;
  refreshToken: () => Promise<boolean>;
  changePassword: (data: ChangePasswordRequest) => Promise<boolean>;
  updateUser: (user: User) => void;
  clearError: () => void;

  // Permission helpers
  hasPermission: (permission: Permission) => boolean;
  hasRole: (role: string) => boolean;
  isAdmin: () => boolean;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      isAuthenticated: false,
      isLoading: false,
      isInitialized: false,
      error: null,

      // Login
      login: async (credentials: LoginCredentials): Promise<boolean> => {
        set({ isLoading: true, error: null });
        try {
          const response = await authService.login(credentials);
          set({
            user: response.user,
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
          toast.success('Welcome back!');
          return true;
        } catch (error: unknown) {
          const message = getErrorMessage(error);
          set({ isLoading: false, error: message });

          // Специфичные сообщения для разных ошибок
          if (isErrorCode(error, ErrorCodes.INVALID_CREDENTIALS)) {
            toast.error('Login Failed', 'Invalid email or password');
          } else if (isErrorCode(error, ErrorCodes.ACCOUNT_LOCKED)) {
            toast.error('Account Locked', 'Your account has been locked. Please contact support.');
          } else if (isErrorCode(error, ErrorCodes.ACCOUNT_INACTIVE)) {
            toast.error('Account Inactive', 'Your account is not active.');
          } else {
            toast.error('Login Failed', message);
          }
          return false;
        }
      },

      // Register
      register: async (data: RegisterData): Promise<boolean> => {
        set({ isLoading: true, error: null });
        try {
          const response = await authService.register(data);
          set({
            user: response.user,
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
          toast.success('Registration successful!');
          return true;
        } catch (error: unknown) {
          const message = getErrorMessage(error);
          set({ isLoading: false, error: message });

          if (isErrorCode(error, ErrorCodes.EMAIL_ALREADY_TAKEN)) {
            toast.error('Registration Failed', 'This email is already registered');
          } else if (isErrorCode(error, ErrorCodes.VALIDATION_FAILED)) {
            toast.error('Validation Error', message);
          } else {
            toast.error('Registration Failed', message);
          }
          return false;
        }
      },

      // Logout
      logout: async (): Promise<void> => {
        set({ isLoading: true });
        try {
          await authService.logout();
        } catch {
          // Ignore errors during logout
        } finally {
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
            error: null,
          });
          toast.info('You have been logged out');
        }
      },

      // Check authentication status
      checkAuth: async (): Promise<void> => {
        if (!authService.isAuthenticated()) {
          set({ isInitialized: true, isAuthenticated: false, user: null });
          return;
        }

        set({ isLoading: true });
        try {
          const user = await authService.getCurrentUser();
          set({
            user,
            isAuthenticated: true,
            isLoading: false,
            isInitialized: true,
            error: null,
          });
        } catch (error: any) {
          // Token might be expired, try to refresh
          const refreshed = await get().refreshToken();
          if (!refreshed) {
            authService.clearTokens();
            set({
              user: null,
              isAuthenticated: false,
              isLoading: false,
              isInitialized: true,
              error: null,
            });
          }
        }
      },

      // Refresh token
      refreshToken: async (): Promise<boolean> => {
        try {
          const response = await authService.refreshToken();
          set({
            user: response.user,
            isAuthenticated: true,
            isLoading: false,
            isInitialized: true,
            error: null,
          });
          return true;
        } catch {
          return false;
        }
      },

      // Change password
      changePassword: async (data: ChangePasswordRequest): Promise<boolean> => {
        set({ isLoading: true, error: null });
        try {
          await authService.changePassword(data);
          set({ isLoading: false });
          toast.success('Password changed successfully');
          // After password change, user needs to re-login
          await get().logout();
          return true;
        } catch (error: unknown) {
          const message = getErrorMessage(error);
          set({ isLoading: false, error: message });
          toast.error('Password Change Failed', message);
          return false;
        }
      },

      // Update user in store
      updateUser: (user: User): void => {
        set({ user });
      },

      // Clear error
      clearError: (): void => {
        set({ error: null });
      },

      // Permission check
      hasPermission: (permission: Permission): boolean => {
        const { user } = get();
        if (!user) return false;
        if (user.is_admin) return true;

        // Get permissions from roles
        // Note: In a real app, you might want to fetch role permissions from backend
        const rolePermissions: Record<string, Permission[]> = {
          admin: Object.values(Permissions) as Permission[],
          user: [
            Permissions.WORKFLOW_CREATE,
            Permissions.WORKFLOW_READ,
            Permissions.WORKFLOW_UPDATE,
            Permissions.WORKFLOW_DELETE,
            Permissions.WORKFLOW_EXECUTE,
            Permissions.EXECUTION_READ,
            Permissions.EXECUTION_CANCEL,
            Permissions.TRIGGER_CREATE,
            Permissions.TRIGGER_READ,
            Permissions.TRIGGER_UPDATE,
            Permissions.TRIGGER_DELETE,
          ],
          viewer: [
            Permissions.WORKFLOW_READ,
            Permissions.EXECUTION_READ,
            Permissions.TRIGGER_READ,
          ],
        };

        for (const role of user.roles) {
          const perms = rolePermissions[role.toLowerCase()];
          if (perms?.includes(permission)) {
            return true;
          }
        }

        return false;
      },

      // Role check
      hasRole: (role: string): boolean => {
        const { user } = get();
        if (!user) return false;
        return user.roles.some((r) => r.toLowerCase() === role.toLowerCase());
      },

      // Admin check
      isAdmin: (): boolean => {
        const { user } = get();
        return user?.is_admin ?? false;
      },
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      // Only persist user data, not loading states
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

// Initialize auth on app load
export const initializeAuth = async (): Promise<void> => {
  await useAuthStore.getState().checkAuth();
};

export default useAuthStore;
