import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';

// API URL: use relative path (proxied by Vite) or env variable for production
const API_URL = (import.meta as any).env?.VITE_API_URL || '/api/v1';

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
    const token = localStorage.getItem('auth_token');
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
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      // Handle Unauthorized (e.g., redirect to login)
      console.warn('Unauthorized access. Redirecting to login...');
      // window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default apiClient;