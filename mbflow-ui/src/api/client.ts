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
      const status = error.response.status;
      const data = error.response.data;

      // Create a more informative error message
      const errorMessage =
        data?.message || data?.error || error.message || "An error occurred";

      switch (status) {
        case 401:
          localStorage.removeItem("mbflow_api_key");
          error.message = "Unauthorized. Please check your API key.";
          break;
        case 403:
          error.message =
            "Forbidden. You don't have permission to perform this action.";
          break;
        case 404:
          error.message =
            data?.message || `Resource not found: ${error.config?.url}`;
          break;
        case 422:
          // Validation errors
          if (data?.errors && Array.isArray(data.errors)) {
            error.message = data.errors.join(", ");
          } else {
            error.message = errorMessage;
          }
          error.validationErrors = data?.errors;
          break;
        case 500:
          error.message =
            data?.message || "Internal server error. Please try again later.";
          break;
        case 503:
          error.message = "Service unavailable. Please try again later.";
          break;
        default:
          error.message = errorMessage;
      }

      // Attach response data for detailed error handling
      error.responseData = data;
    } else if (error.request) {
      // Network error
      error.message = "Network error. Please check your connection.";
    } else {
      // Request setup error
      error.message = error.message || "An unexpected error occurred";
    }

    return Promise.reject(error);
  },
);

export default apiClient;
