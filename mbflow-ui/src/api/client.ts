import axios, { type AxiosInstance } from "axios";

const API_BASE_URL = import.meta.env.VITE_API_URL || "/api/v1";

export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
  },
});

// Request interceptor
apiClient.interceptors.request.use(
  (config) => {
    const apiKey = localStorage.getItem("mbflow_api_key");
    if (apiKey && config.headers) {
      config.headers.Authorization = `Bearer ${apiKey}`;
    }
    return config;
  },
  (error) => Promise.reject(error),
);

// Response interceptor
apiClient.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response) {
      switch (error.response.status) {
        case 401:
          localStorage.removeItem("mbflow_api_key");
          break;
        case 404:
          console.error("Resource not found:", error.config.url);
          break;
        case 500:
          console.error("Server error:", error.response.data);
          break;
      }
    }
    return Promise.reject(error);
  },
);

export default apiClient;
