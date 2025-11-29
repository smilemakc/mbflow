/**
 * HTTP Client - Axios instance with interceptors
 */

import axios, { type AxiosInstance, type AxiosError } from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8181'

// Create axios instance
export const apiClient: AxiosInstance = axios.create({
    baseURL: API_BASE_URL,
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json',
    },
})

// Request interceptor
apiClient.interceptors.request.use(
    (config) => {
        // Add auth token if available
        const token = localStorage.getItem('api_token')
        if (token) {
            config.headers.Authorization = `Bearer ${token}`
        }

        console.log(`[API] ${config.method?.toUpperCase()} ${config.url}`)
        return config
    },
    (error) => {
        console.error('[API] Request error:', error)
        return Promise.reject(error)
    }
)

// Response interceptor
apiClient.interceptors.response.use(
    (response) => {
        console.log(`[API] Response ${response.status}:`, response.data)
        return response
    },
    (error: AxiosError) => {
        console.error('[API] Response error:', error.response?.data || error.message)

        // Import notification store dynamically to avoid circular dependencies
        // Only show notifications in browser context (not during tests)
        if (typeof window !== 'undefined' && !import.meta.env.VITEST) {
            import('@/stores/notification.store').then(({ useNotificationStore }) => {
                try {
                    const notificationStore = useNotificationStore()

                    // Handle common errors
                    if (error.code === 'ERR_NETWORK' || error.message === 'Network Error') {
                        notificationStore.error('Backend server is not available. Please start the server on http://localhost:8181')
                    } else if (error.response?.status === 401) {
                        // Unauthorized - clear token and redirect to login
                        localStorage.removeItem('api_token')
                        notificationStore.error('Unauthorized. Please log in again.')
                    } else if (error.response?.status === 404) {
                        notificationStore.error('Resource not found')
                    } else if (error.response?.status === 500) {
                        notificationStore.error('Server error. Please try again later.')
                    } else if (error.response?.data) {
                        // Try to extract error message from response
                        const errorData = error.response.data as any
                        const message = errorData.message || errorData.error || 'An error occurred'
                        notificationStore.error(message)
                    }
                } catch (e) {
                    // Pinia not initialized, skip notification
                    console.warn('[API] Could not show notification:', e)
                }
            }).catch(() => {
                // Module import failed, skip notification
            })
        }

        return Promise.reject(error)
    }
)

// Helper function to extract data from response
export async function request<T>(promise: Promise<any>): Promise<T> {
    const response = await promise
    return response.data
}
