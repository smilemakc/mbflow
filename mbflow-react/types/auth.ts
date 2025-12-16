/**
 * Authentication and Authorization Types
 */

// User model
export interface User {
  id: string;
  email: string;
  username: string;
  full_name?: string;
  is_active: boolean;
  is_admin: boolean;
  roles: string[];
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
}

// Role model
export interface Role {
  id: string;
  name: string;
  description?: string;
  permissions: string[];
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

// Login credentials
export interface LoginCredentials {
  email: string;
  password: string;
}

// Registration data
export interface RegisterData {
  email: string;
  username: string;
  password: string;
  full_name?: string;
}

// Auth response from API
export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
}

// Token refresh request
export interface RefreshTokenRequest {
  refresh_token: string;
}

// Change password request
export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

// Admin user create request
export interface AdminCreateUserRequest {
  email: string;
  username: string;
  password: string;
  full_name?: string;
  is_admin?: boolean;
}

// Admin user update request
export interface AdminUpdateUserRequest {
  email: string;
  username: string;
  full_name?: string;
  is_active: boolean;
  is_admin: boolean;
}

// Auth info response
export interface AuthInfo {
  mode: string;
  providers: string[];
  gateway_available: boolean;
}

// JWT Claims (decoded from token)
export interface JWTClaims {
  user_id: string;
  email: string;
  username: string;
  is_admin: boolean;
  roles: string[];
  exp: number;
  iat: number;
}

// Permission constants
export const Permissions = {
  WORKFLOW_CREATE: 'workflow:create',
  WORKFLOW_READ: 'workflow:read',
  WORKFLOW_UPDATE: 'workflow:update',
  WORKFLOW_DELETE: 'workflow:delete',
  WORKFLOW_EXECUTE: 'workflow:execute',
  EXECUTION_READ: 'execution:read',
  EXECUTION_CANCEL: 'execution:cancel',
  EXECUTION_RETRY: 'execution:retry',
  TRIGGER_CREATE: 'trigger:create',
  TRIGGER_READ: 'trigger:read',
  TRIGGER_UPDATE: 'trigger:update',
  TRIGGER_DELETE: 'trigger:delete',
  USER_MANAGE: 'user:manage',
  ROLE_MANAGE: 'role:manage',
  SYSTEM_ADMIN: 'system:admin',
} as const;

export type Permission = typeof Permissions[keyof typeof Permissions];

// Role names
export const RoleNames = {
  ADMIN: 'admin',
  USER: 'user',
  VIEWER: 'viewer',
} as const;

export type RoleName = typeof RoleNames[keyof typeof RoleNames];
