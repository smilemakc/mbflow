import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { toast } from './toast';

// API URL: use relative path (proxied by Vite) or env variable for production
const API_URL = (import.meta as any).env?.VITE_API_URL || '/api/v1';

/**
 * APIError - структура ошибки от backend
 * Новый формат: { code: string, message: string, details?: object }
 * Legacy формат: { error: string }
 */
export interface APIError {
  code: string;
  message: string;
  details?: {
    request_id?: string;
    field?: string;
    [key: string]: unknown;
  };
}

/**
 * Извлечь сообщение об ошибке из ответа API
 * Поддерживает новый формат APIError и legacy формат
 */
export function getErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data;

    // Новый формат APIError: { code, message, details }
    if (data?.message) {
      return data.message;
    }

    // Legacy формат: { error: "message" }
    if (data?.error) {
      return data.error;
    }

    // Fallback на HTTP статус
    if (error.response?.status) {
      return getHttpStatusMessage(error.response.status);
    }

    // Network error
    if (error.message) {
      return error.message;
    }
  }

  // Unknown error
  if (error instanceof Error) {
    return error.message;
  }

  return 'An unexpected error occurred';
}

/**
 * Извлечь код ошибки из ответа API
 */
export function getErrorCode(error: unknown): string | null {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data;
    if (data?.code) {
      return data.code;
    }
  }
  return null;
}

/**
 * Извлечь request_id из ответа API (для поддержки)
 */
export function getRequestId(error: unknown): string | null {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data;
    if (data?.details?.request_id) {
      return data.details.request_id;
    }
  }
  return null;
}

/**
 * Проверить, является ли ошибка определённым кодом
 */
export function isErrorCode(error: unknown, code: string): boolean {
  return getErrorCode(error) === code;
}

/**
 * Получить человекочитаемое сообщение по HTTP статусу
 */
function getHttpStatusMessage(status: number): string {
  const messages: Record<number, string> = {
    400: 'Invalid request',
    401: 'Authentication required',
    403: 'Access denied',
    404: 'Not found',
    409: 'Conflict',
    422: 'Validation error',
    429: 'Too many requests',
    500: 'Server error',
    502: 'Bad gateway',
    503: 'Service unavailable',
  };
  return messages[status] || `Error ${status}`;
}

/**
 * Показать toast с ошибкой API
 * Автоматически извлекает сообщение и показывает request_id если есть
 */
export function showApiError(error: unknown, fallbackTitle = 'Error'): void {
  const message = getErrorMessage(error);
  const requestId = getRequestId(error);

  if (requestId) {
    toast.error(fallbackTitle, `${message} (ID: ${requestId})`);
  } else {
    toast.error(fallbackTitle, message);
  }
}

/**
 * Коды ошибок API для проверки в коде
 */
export const ErrorCodes = {
  // Not Found
  WORKFLOW_NOT_FOUND: 'WORKFLOW_NOT_FOUND',
  EXECUTION_NOT_FOUND: 'EXECUTION_NOT_FOUND',
  TRIGGER_NOT_FOUND: 'TRIGGER_NOT_FOUND',
  NODE_NOT_FOUND: 'NODE_NOT_FOUND',
  EDGE_NOT_FOUND: 'EDGE_NOT_FOUND',
  RESOURCE_NOT_FOUND: 'RESOURCE_NOT_FOUND',
  USER_NOT_FOUND: 'USER_NOT_FOUND',
  NOT_FOUND: 'NOT_FOUND',

  // Validation
  VALIDATION_FAILED: 'VALIDATION_FAILED',
  INVALID_ID: 'INVALID_ID',
  INVALID_JSON: 'INVALID_JSON',

  // Auth
  UNAUTHORIZED: 'UNAUTHORIZED',
  FORBIDDEN: 'FORBIDDEN',
  INVALID_CREDENTIALS: 'INVALID_CREDENTIALS',
  INVALID_TOKEN: 'INVALID_TOKEN',
  TOKEN_EXPIRED: 'TOKEN_EXPIRED',
  ACCOUNT_LOCKED: 'ACCOUNT_LOCKED',
  ACCOUNT_INACTIVE: 'ACCOUNT_INACTIVE',

  // Conflict
  WORKFLOW_EXISTS: 'WORKFLOW_EXISTS',
  USER_EXISTS: 'USER_EXISTS',
  EMAIL_ALREADY_TAKEN: 'EMAIL_ALREADY_TAKEN',

  // Limits
  RATE_LIMIT_EXCEEDED: 'RATE_LIMIT_EXCEEDED',
  INSUFFICIENT_BALANCE: 'INSUFFICIENT_BALANCE',
  RESOURCE_LIMIT_EXCEEDED: 'RESOURCE_LIMIT_EXCEEDED',

  // Trigger
  TRIGGER_DISABLED: 'TRIGGER_DISABLED',
} as const;

// Storage keys
const TOKEN_KEY = 'auth_token';
const REFRESH_TOKEN_KEY = 'auth_refresh_token';
const AUTH_STORAGE_KEY = 'auth-storage';

// Clear all auth data and redirect to login
const clearAuthAndRedirect = () => {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(AUTH_STORAGE_KEY);
  window.location.href = '/login';
};

export const apiClient = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 10000,
});

// Request Interceptor: Attach Auth Token
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem(TOKEN_KEY);
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response Interceptor: Handle Errors globally
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    // Handle 401 Unauthorized
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      // Try to refresh the token
      const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
      if (refreshToken) {
        try {
          const response = await axios.post(`${API_URL}/auth/refresh`, {
            refresh_token: refreshToken,
          });

          const { access_token, refresh_token: newRefreshToken } = response.data;

          // Update tokens
          localStorage.setItem(TOKEN_KEY, access_token);
          if (newRefreshToken) {
            localStorage.setItem(REFRESH_TOKEN_KEY, newRefreshToken);
          }

          // Retry original request with new token
          originalRequest.headers.Authorization = `Bearer ${access_token}`;
          return apiClient(originalRequest);
        } catch (refreshError) {
          // Refresh failed, clear auth and redirect to login
          clearAuthAndRedirect();
          return Promise.reject(refreshError);
        }
      } else {
        // No refresh token, clear auth and redirect to login
        clearAuthAndRedirect();
      }
    }

    return Promise.reject(error);
  }
);

export default apiClient;