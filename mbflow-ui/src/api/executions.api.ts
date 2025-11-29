/**
 * Executions API - operations for workflow executions
 */

import { apiClient, request } from './client'
import type {
    Execution,
    ExecuteWorkflowRequest,
    ExecutionEvent,
    ExecutionFilters,
} from '@/types'
import { mockExecutions, mockEvents } from './mock-data'

const USE_MOCK = import.meta.env.VITE_USE_MOCK_API === 'true'

export const executionsApi = {
    /**
     * Get all executions with optional filters
     */
    async listExecutions(filters?: ExecutionFilters): Promise<Execution[]> {
        if (USE_MOCK) {
            let filtered = [...mockExecutions]

            if (filters?.workflow_id) {
                filtered = filtered.filter((e) => e.workflow_id === filters.workflow_id)
            }

            if (filters?.phase) {
                filtered = filtered.filter((e) => e.phase === filters.phase)
            }

            return Promise.resolve(filtered)
        }

        const params = new URLSearchParams()
        if (filters?.workflow_id) params.append('workflow_id', filters.workflow_id)
        if (filters?.phase) params.append('phase', filters.phase)
        if (filters?.limit) params.append('limit', filters.limit.toString())
        if (filters?.offset) params.append('offset', filters.offset.toString())

        return request<Execution[]>(
            apiClient.get(`/api/v1/executions?${params.toString()}`)
        )
    },

    /**
     * Get execution by ID
     */
    async getExecution(id: string): Promise<Execution> {
        if (USE_MOCK) {
            const execution = mockExecutions.find((e) => e.id === id)
            if (!execution) {
                throw new Error(`Execution ${id} not found`)
            }
            return Promise.resolve(execution)
        }
        return request<Execution>(apiClient.get(`/api/v1/executions/${id}`))
    },

    /**
     * Execute workflow
     */
    async executeWorkflow(data: ExecuteWorkflowRequest): Promise<Execution> {
        if (USE_MOCK) {
            const newExecution: Execution = {
                id: `exec-${Date.now()}`,
                workflow_id: data.workflow_id,
                phase: 'executing',
                started_at: new Date().toISOString(),
                sequence_number: mockExecutions.length + 1,
                node_states: {},
                variables: data.variables || {},
            }
            mockExecutions.push(newExecution)

            // Simulate completion after 2 seconds
            setTimeout(() => {
                newExecution.phase = 'completed'
                newExecution.completed_at = new Date().toISOString()
                newExecution.duration_ms = 2000
            }, 2000)

            return Promise.resolve(newExecution)
        }
        return request<Execution>(apiClient.post('/api/v1/executions', data))
    },

    /**
     * Get execution events
     */
    async getExecutionEvents(executionId: string): Promise<ExecutionEvent[]> {
        if (USE_MOCK) {
            const events = mockEvents.filter((e) => e.execution_id === executionId)
            return Promise.resolve(events)
        }
        return request<ExecutionEvent[]>(
            apiClient.get(`/api/v1/executions/${executionId}/events`)
        )
    },

    /**
     * Cancel execution
     */
    async cancelExecution(id: string): Promise<void> {
        if (USE_MOCK) {
            const execution = mockExecutions.find((e) => e.id === id)
            if (execution) {
                execution.phase = 'cancelled'
                execution.completed_at = new Date().toISOString()
            }
            return Promise.resolve()
        }
        return request<void>(apiClient.post(`/api/v1/executions/${id}/cancel`))
    },

    /**
     * Pause execution
     */
    async pauseExecution(id: string): Promise<void> {
        if (USE_MOCK) {
            const execution = mockExecutions.find((e) => e.id === id)
            if (execution) {
                execution.phase = 'paused'
            }
            return Promise.resolve()
        }
        return request<void>(apiClient.post(`/api/v1/executions/${id}/pause`))
    },

    /**
     * Resume execution
     */
    async resumeExecution(id: string): Promise<void> {
        if (USE_MOCK) {
            const execution = mockExecutions.find((e) => e.id === id)
            if (execution) {
                execution.phase = 'executing'
            }
            return Promise.resolve()
        }
        return request<void>(apiClient.post(`/api/v1/executions/${id}/resume`))
    },
}
