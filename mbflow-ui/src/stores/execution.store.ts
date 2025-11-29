/**
 * Execution Store - manages execution state and events
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Execution, ExecutionEvent, ExecutionFilters } from '@/types'
import { executionsApi } from '@/api'

export const useExecutionStore = defineStore('execution', () => {
    // State
    const executions = ref<Map<string, Execution>>(new Map())
    const events = ref<Map<string, ExecutionEvent[]>>(new Map())
    const currentExecution = ref<Execution | null>(null)
    const loading = ref(false)
    const error = ref<string | null>(null)

    // Computed
    const executionsList = computed(() => Array.from(executions.value.values()))

    const runningExecutions = computed(() =>
        executionsList.value.filter((e) => e.phase === 'executing')
    )

    const completedExecutions = computed(() =>
        executionsList.value.filter((e) => e.phase === 'completed')
    )

    const failedExecutions = computed(() =>
        executionsList.value.filter((e) => e.phase === 'failed')
    )

    // Actions
    async function fetchExecutions(filters?: ExecutionFilters) {
        loading.value = true
        error.value = null
        try {
            const list = await executionsApi.listExecutions(filters)
            list.forEach((exec) => {
                executions.value.set(exec.id, exec)
            })
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to fetch executions'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function fetchExecution(id: string) {
        loading.value = true
        error.value = null
        try {
            const execution = await executionsApi.getExecution(id)
            executions.value.set(execution.id, execution)
            currentExecution.value = execution
            return execution
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to fetch execution'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function executeWorkflow(workflowId: string, variables?: Record<string, unknown>) {
        loading.value = true
        error.value = null
        try {
            const execution = await executionsApi.executeWorkflow({
                workflow_id: workflowId,
                variables,
            })
            executions.value.set(execution.id, execution)
            currentExecution.value = execution
            return execution
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to execute workflow'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function fetchExecutionEvents(executionId: string) {
        loading.value = true
        error.value = null
        try {
            const eventsList = await executionsApi.getExecutionEvents(executionId)
            events.value.set(executionId, eventsList)

            // Apply events to update execution state
            eventsList.forEach((event) => {
                applyEventToExecution(executionId, event)
            })

            return eventsList
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to fetch events'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function cancelExecution(id: string) {
        try {
            await executionsApi.cancelExecution(id)
            const execution = executions.value.get(id)
            if (execution) {
                execution.phase = 'cancelled'
                execution.completed_at = new Date().toISOString()
            }
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to cancel execution'
            throw e
        }
    }

    async function pauseExecution(id: string) {
        try {
            await executionsApi.pauseExecution(id)
            const execution = executions.value.get(id)
            if (execution) {
                execution.phase = 'paused'
            }
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to pause execution'
            throw e
        }
    }

    async function resumeExecution(id: string) {
        try {
            await executionsApi.resumeExecution(id)
            const execution = executions.value.get(id)
            if (execution) {
                execution.phase = 'executing'
            }
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to resume execution'
            throw e
        }
    }

    // Event handling (Event Sourcing pattern)
    function addEvent(executionId: string, event: ExecutionEvent) {
        const executionEvents = events.value.get(executionId) || []
        executionEvents.push(event)
        events.value.set(executionId, executionEvents)

        applyEventToExecution(executionId, event)
    }

    function applyEventToExecution(executionId: string, event: ExecutionEvent) {
        const execution = executions.value.get(executionId)
        if (!execution) return

        switch (event.event_type) {
            case 'execution.started':
                execution.phase = 'executing'
                execution.started_at = event.timestamp
                break

            case 'execution.completed':
                execution.phase = 'completed'
                execution.completed_at = event.timestamp
                break

            case 'execution.failed':
                execution.phase = 'failed'
                execution.completed_at = event.timestamp
                execution.error = event.data?.error as string
                break

            case 'node.started':
                if (event.data?.node_id) {
                    const nodeId = event.data.node_id as string
                    execution.node_states[nodeId] = {
                        node_id: nodeId,
                        node_name: event.data.node_name as string,
                        status: 'running',
                        started_at: event.timestamp,
                        retry_count: 0,
                    }
                    execution.current_node_id = nodeId
                }
                break

            case 'node.completed':
                if (event.data?.node_id) {
                    const nodeId = event.data.node_id as string
                    const nodeState = execution.node_states[nodeId]
                    if (nodeState) {
                        nodeState.status = 'completed'
                        nodeState.completed_at = event.timestamp
                        nodeState.output = event.data.output as Record<string, unknown>
                    }
                }
                break

            case 'node.failed':
                if (event.data?.node_id) {
                    const nodeId = event.data.node_id as string
                    const nodeState = execution.node_states[nodeId]
                    if (nodeState) {
                        nodeState.status = 'failed'
                        nodeState.completed_at = event.timestamp
                        nodeState.error = event.data.error as string
                    }
                }
                break
        }
    }

    return {
        // State
        executions,
        events,
        currentExecution,
        loading,
        error,

        // Computed
        executionsList,
        runningExecutions,
        completedExecutions,
        failedExecutions,

        // Actions
        fetchExecutions,
        fetchExecution,
        executeWorkflow,
        fetchExecutionEvents,
        cancelExecution,
        pauseExecution,
        resumeExecution,
        addEvent,
    }
})
