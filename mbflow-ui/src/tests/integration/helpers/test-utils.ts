/**
 * Test utilities for integration tests
 */

import { apiClient } from '@/api/client'
import type { Workflow, Execution } from '@/types'

/**
 * Check if backend server is available
 */
export async function isBackendAvailable(): Promise<boolean> {
    try {
        const response = await apiClient.get('/health')
        return response.status === 200
    } catch (error) {
        return false
    }
}

/**
 * Wait for backend server to be ready
 */
export async function waitForBackend(
    maxAttempts = 10,
    delayMs = 1000
): Promise<boolean> {
    for (let i = 0; i < maxAttempts; i++) {
        if (await isBackendAvailable()) {
            return true
        }
        await new Promise((resolve) => setTimeout(resolve, delayMs))
    }
    return false
}

/**
 * Wait for execution to complete
 */
export async function waitForExecutionCompletion(
    executionId: string,
    maxWaitMs = 30000,
    pollIntervalMs = 500
): Promise<Execution> {
    const startTime = Date.now()

    while (Date.now() - startTime < maxWaitMs) {
        try {
            const response = await apiClient.get(`/api/v1/executions/${executionId}`)
            const execution: Execution = response.data

            if (
                execution.phase === 'completed' ||
                execution.phase === 'failed' ||
                execution.phase === 'cancelled'
            ) {
                return execution
            }

            await new Promise((resolve) => setTimeout(resolve, pollIntervalMs))
        } catch (error) {
            console.error('Error polling execution:', error)
            throw error
        }
    }

    throw new Error(`Execution ${executionId} did not complete within ${maxWaitMs}ms`)
}

/**
 * Clean up test workflows
 */
export async function cleanupWorkflows(workflowIds: string[]): Promise<void> {
    const deletePromises = workflowIds.map(async (id) => {
        try {
            await apiClient.delete(`/api/v1/workflows/${id}`)
        } catch (error) {
            console.warn(`Failed to delete workflow ${id}:`, error)
        }
    })

    await Promise.all(deletePromises)
}

/**
 * Generate unique test name
 */
export function generateTestName(prefix: string): string {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substring(7)}`
}

/**
 * Retry function with exponential backoff
 */
export async function retryWithBackoff<T>(
    fn: () => Promise<T>,
    maxRetries = 3,
    initialDelayMs = 1000
): Promise<T> {
    let lastError: Error | null = null

    for (let i = 0; i < maxRetries; i++) {
        try {
            return await fn()
        } catch (error) {
            lastError = error as Error
            const delay = initialDelayMs * Math.pow(2, i)
            console.warn(`Retry ${i + 1}/${maxRetries} after ${delay}ms:`, error)
            await new Promise((resolve) => setTimeout(resolve, delay))
        }
    }

    throw lastError || new Error('Max retries exceeded')
}

/**
 * Assert execution completed successfully
 */
export function assertExecutionSuccess(execution: Execution): void {
    if (execution.phase !== 'completed') {
        throw new Error(
            `Expected execution to be completed, but was ${execution.phase}`
        )
    }

    if (!execution.completed_at) {
        throw new Error('Expected execution to have completed_at timestamp')
    }

    if (!execution.duration_ms || execution.duration_ms <= 0) {
        throw new Error('Expected execution to have positive duration_ms')
    }
}

/**
 * Assert workflow structure is valid
 */
export function assertWorkflowValid(workflow: Workflow): void {
    if (!workflow.id) {
        throw new Error('Workflow must have an ID')
    }

    if (!workflow.name) {
        throw new Error('Workflow must have a name')
    }

    if (!workflow.version) {
        throw new Error('Workflow must have a version')
    }

    if (!Array.isArray(workflow.nodes)) {
        throw new Error('Workflow must have nodes array')
    }

    if (!Array.isArray(workflow.edges)) {
        throw new Error('Workflow must have edges array')
    }

    if (!Array.isArray(workflow.triggers)) {
        throw new Error('Workflow must have triggers array')
    }

    // Validate nodes
    workflow.nodes.forEach((node, index) => {
        if (!node.id) {
            throw new Error(`Node at index ${index} must have an ID`)
        }
        if (!node.type) {
            throw new Error(`Node ${node.id} must have a type`)
        }
        if (!node.name) {
            throw new Error(`Node ${node.id} must have a name`)
        }
    })

    // Validate edges
    workflow.edges.forEach((edge, index) => {
        if (!edge.id) {
            throw new Error(`Edge at index ${index} must have an ID`)
        }
        if (!edge.from) {
            throw new Error(`Edge ${edge.id} must have a 'from' node`)
        }
        if (!edge.to) {
            throw new Error(`Edge ${edge.id} must have a 'to' node`)
        }
        if (!edge.type) {
            throw new Error(`Edge ${edge.id} must have a type`)
        }
    })

    // Validate triggers
    if (workflow.triggers.length === 0) {
        throw new Error('Workflow must have at least one trigger')
    }

    workflow.triggers.forEach((trigger, index) => {
        if (!trigger.id) {
            throw new Error(`Trigger at index ${index} must have an ID`)
        }
        if (!trigger.type) {
            throw new Error(`Trigger ${trigger.id} must have a type`)
        }
    })
}

/**
 * Create a simple test workflow
 */
export function createSimpleWorkflowData(name?: string) {
    return {
        name: name || generateTestName('test-workflow'),
        version: '1.0.0',
        description: 'Simple test workflow',
        nodes: [
            { type: 'start' as const, name: 'start' },
            {
                type: 'transform' as const,
                name: 'transform',
                config: {
                    transformations: {
                        result: 'input * 2',
                    },
                },
            },
            { type: 'end' as const, name: 'end' },
        ],
        edges: [
            { from: 'start', to: 'transform', type: 'direct' as const },
            { from: 'transform', to: 'end', type: 'direct' as const },
        ],
        triggers: [{ type: 'manual' as const, config: {} }],
    }
}

/**
 * Performance measurement utility
 */
export class PerformanceTimer {
    private startTime: number

    constructor() {
        this.startTime = Date.now()
    }

    elapsed(): number {
        return Date.now() - this.startTime
    }

    reset(): void {
        this.startTime = Date.now()
    }

    assertMaxDuration(maxMs: number, operation: string): void {
        const elapsed = this.elapsed()
        if (elapsed > maxMs) {
            throw new Error(
                `${operation} took ${elapsed}ms, expected max ${maxMs}ms`
            )
        }
    }
}

/**
 * Test data generators
 */
export const TestDataGenerators = {
    /**
     * Generate random integer
     */
    randomInt(min: number, max: number): number {
        return Math.floor(Math.random() * (max - min + 1)) + min
    },

    /**
     * Generate random string
     */
    randomString(length: number): string {
        const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
        let result = ''
        for (let i = 0; i < length; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length))
        }
        return result
    },

    /**
     * Generate test variables
     */
    randomVariables(): Record<string, any> {
        return {
            input: this.randomInt(1, 100),
            multiplier: this.randomInt(2, 10),
            flag: Math.random() > 0.5,
            text: this.randomString(10),
        }
    },
}
