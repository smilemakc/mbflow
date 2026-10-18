/**
 * Single source of truth for auth-related localStorage keys and accessors.
 */

export const TOKEN_KEY = 'auth_token';
export const REFRESH_TOKEN_KEY = 'auth_refresh_token';
/** Persisted zustand auth store key */
export const AUTH_STORAGE_KEY = 'auth-storage';

export const getAccessToken = (): string | null => {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
};

export const getRefreshToken = (): string | null => {
  try {
    return localStorage.getItem(REFRESH_TOKEN_KEY);
  } catch {
    return null;
  }
};
