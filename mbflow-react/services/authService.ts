/**
 * Authentication Service
 * Handles all authentication-related API calls
 */

import apiClient from '@/lib/api';
import type {
  User,
  Role,
  LoginCredentials,
  RegisterData,
  AuthResponse,
  RefreshTokenRequest,
  ChangePasswordRequest,
  AdminCreateUserRequest,
  AdminUpdateUserRequest,
  AuthInfo,
} from '@/types/auth';

// Storage keys
const TOKEN_KEY = 'auth_token';
const REFRESH_TOKEN_KEY = 'auth_refresh_token';

class AuthService {
  /**
   * Register a new user
   */
  async register(data: RegisterData): Promise<AuthResponse> {
    const response = await apiClient.post<AuthResponse>('/auth/register', data);
    this.setTokens(response.data.access_token, response.data.refresh_token);
    return response.data;
  }

  /**
   * Login with email and password
   */
  async login(credentials: LoginCredentials): Promise<AuthResponse> {
    const response = await apiClient.post<AuthResponse>('/auth/login', credentials);
    this.setTokens(response.data.access_token, response.data.refresh_token);
    return response.data;
  }

  /**
   * Logout current user
   */
  async logout(): Promise<void> {
    try {
      await apiClient.post('/auth/logout');
    } finally {
      this.clearTokens();
    }
  }

  /**
   * Refresh access token
   */
  async refreshToken(): Promise<AuthResponse> {
    const refreshToken = this.getRefreshToken();
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }

    const response = await apiClient.post<AuthResponse>('/auth/refresh', {
      refresh_token: refreshToken,
    } as RefreshTokenRequest);

    this.setTokens(response.data.access_token, response.data.refresh_token);
    return response.data;
  }

  /**
   * Get current user info
   */
  async getCurrentUser(): Promise<User> {
    const response = await apiClient.get<{ user: User }>('/auth/me');
    return response.data.user;
  }

  /**
   * Change password
   */
  async changePassword(data: ChangePasswordRequest): Promise<void> {
    await apiClient.post('/auth/password', data);
  }

  /**
   * Get auth info (available providers, mode)
   */
  async getAuthInfo(): Promise<AuthInfo> {
    const response = await apiClient.get<AuthInfo>('/auth/info');
    return response.data;
  }

  /**
   * Get OAuth authorization URL
   */
  getOAuthAuthorizeUrl(): string {
    return `${apiClient.defaults.baseURL}/auth/oauth/authorize`;
  }

  // Token management
  setTokens(accessToken: string, refreshToken?: string): void {
    localStorage.setItem(TOKEN_KEY, accessToken);
    if (refreshToken) {
      localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
    }
  }

  getAccessToken(): string | null {
    return localStorage.getItem(TOKEN_KEY);
  }

  getRefreshToken(): string | null {
    return localStorage.getItem(REFRESH_TOKEN_KEY);
  }

  clearTokens(): void {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  }

  isAuthenticated(): boolean {
    return !!this.getAccessToken();
  }

  // ============================================================================
  // Admin User Management
  // ============================================================================

  /**
   * List all users (admin only)
   */
  async listUsers(limit = 50, offset = 0): Promise<{ data: User[]; meta: { total: number } }> {
    const response = await apiClient.get('/admin/users', {
      params: { limit, offset },
    });
    return response.data;
  }

  /**
   * Get user by ID (admin only)
   */
  async getUser(userId: string): Promise<User> {
    const response = await apiClient.get<{ user: User }>(`/admin/users/${userId}`);
    return response.data.user;
  }

  /**
   * Create new user (admin only)
   */
  async createUser(data: AdminCreateUserRequest): Promise<User> {
    const response = await apiClient.post<{ user: User }>('/admin/users', data);
    return response.data.user;
  }

  /**
   * Update user (admin only)
   */
  async updateUser(userId: string, data: AdminUpdateUserRequest): Promise<User> {
    const response = await apiClient.put<{ user: User }>(`/admin/users/${userId}`, data);
    return response.data.user;
  }

  /**
   * Delete user (admin only)
   */
  async deleteUser(userId: string): Promise<void> {
    await apiClient.delete(`/admin/users/${userId}`);
  }

  /**
   * Reset user password (admin only)
   */
  async resetUserPassword(userId: string, newPassword: string): Promise<void> {
    await apiClient.post(`/admin/users/${userId}/reset-password`, {
      new_password: newPassword,
    });
  }

  // ============================================================================
  // Role Management
  // ============================================================================

  /**
   * List all roles
   */
  async listRoles(): Promise<Role[]> {
    const response = await apiClient.get<{ roles: Role[] }>('/admin/roles');
    return response.data.roles;
  }

  /**
   * Get user roles
   */
  async getUserRoles(userId: string): Promise<Role[]> {
    const response = await apiClient.get<{ roles: Role[] }>(`/admin/users/${userId}/roles`);
    return response.data.roles;
  }

  /**
   * Assign role to user
   */
  async assignRole(userId: string, roleId: string): Promise<void> {
    await apiClient.post(`/admin/users/${userId}/roles`, { role_id: roleId });
  }

  /**
   * Remove role from user
   */
  async removeRole(userId: string, roleId: string): Promise<void> {
    await apiClient.delete(`/admin/users/${userId}/roles/${roleId}`);
  }
}

export const authService = new AuthService();
export default authService;
