/**
 * API types - request/response types for REST API
 */

export * from './domain.types'
export * from './workflow.types'
export * from './execution.types'
export * from './variable.types'
export * from './collaboration.types'

// ============================================================================
// API Response Wrapper
// ============================================================================

export interface ApiResponse<T> {
    data: T
    error?: string
}

export interface ApiError {
    error: string
    code?: string
    details?: string
}

// ============================================================================
// Pagination
// ============================================================================

export interface PaginatedResponse<T> {
    items: T[]
    total: number
    limit: number
    offset: number
}

export interface PaginationParams {
    limit?: number
    offset?: number
}
