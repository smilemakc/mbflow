import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import { executionsApi } from '@/api/executions.api'
import { workflowsApi } from '@/api/workflows.api'
import type { Execution, CreateWorkflowRequest } from '@/types'

/**
 * Integration tests for Executions API
 * These tests require the backend server to be running on http://localhost:8181
 * 
 * To run these tests:
 * 1. Start the backend: cd /Users/balashov/PycharmProjects/mbflow && go run cmd/server/main.go
 * 2. Run tests: npm run test -- executions.api.spec.ts
 */

describe('Executions API Integration', () => {
    let testWorkflowId: string | null = null
    let testExecutionId: string | null = null
    const testWorkflowName = `exec-test-workflow-${Date.now()}`

    // Setup: Create a test workflow
    beforeAll(async () => {
        // Note: Variables are passed between nodes using namespaced access
        // Parent node output is available as: parent_node_name.field_name
        // Initial variables are available directly by name
        const workflowData: CreateWorkflowRequest = {
            name: testWorkflowName,
            version: '1.0.0',
            description: 'Workflow for execution testing',
            nodes: [
                {
                    type: 'transform',
                    name: 'multiply',
                    config: {
                        transformations: {
                            // 'input' comes from initial variables (global context)
                            result: 'input * 3',
                        },
                    },
                },
                {
                    type: 'transform',
                    name: 'add',
                    config: {
                        transformations: {
                            // 'multiply.result' refers to output from 'multiply' node
                            final: 'multiply.result + 10',
                        },
                    },
                },
            ],
            edges: [
                { from: 'multiply', to: 'add', type: 'direct' },
            ],
            triggers: [
                {
                    type: 'manual',
                    config: {},
                },
            ],
        }

        const workflow = await workflowsApi.createWorkflow(workflowData)
        testWorkflowId = workflow.id
    }, 15000)

    // Cleanup: Delete test workflow
    afterAll(async () => {
        if (testWorkflowId) {
            try {
                await workflowsApi.deleteWorkflow(testWorkflowId)
            } catch (error) {
                console.warn('Failed to cleanup test workflow:', error)
            }
        }
    })

    describe('Execution Operations', () => {
        it('should execute a workflow', async () => {
            if (!testWorkflowId) {
                throw new Error('Test workflow not created')
            }

            const execution = await executionsApi.executeWorkflow({
                workflow_id: testWorkflowId,
                variables: {
                    input: 5,
                },
            })

            expect(execution).toBeDefined()
            expect(execution.id).toBeDefined()
            expect(execution.workflow_id).toBe(testWorkflowId)
            expect(execution.phase).toBeDefined()
            expect(execution.started_at).toBeDefined()
            // Variables contain both initial inputs and node outputs
            expect(execution.variables?.input).toBe(5)

            testExecutionId = execution.id
        }, 15000)

        it('should list all executions', async () => {
            const executions = await executionsApi.listExecutions()

            expect(Array.isArray(executions)).toBe(true)
            expect(executions.length).toBeGreaterThan(0)
        }, 10000)

        it('should filter executions by workflow ID', async () => {
            if (!testWorkflowId) {
                throw new Error('Test workflow not created')
            }

            const executions = await executionsApi.listExecutions({
                workflow_id: testWorkflowId,
            })

            expect(Array.isArray(executions)).toBe(true)
            expect(executions.every((e) => e.workflow_id === testWorkflowId)).toBe(true)
        }, 10000)

        it('should get execution by ID', async () => {
            if (!testExecutionId) {
                throw new Error('Test execution not created')
            }

            const execution = await executionsApi.getExecution(testExecutionId)

            expect(execution).toBeDefined()
            expect(execution.id).toBe(testExecutionId)
            expect(execution.workflow_id).toBe(testWorkflowId)
        }, 10000)

        it('should get execution events', async () => {
            if (!testExecutionId) {
                throw new Error('Test execution not created')
            }

            const events = await executionsApi.getExecutionEvents(testExecutionId)

            expect(Array.isArray(events)).toBe(true)
            // Events should be ordered by timestamp
            if (events.length > 1) {
                for (let i = 1; i < events.length; i++) {
                    const prev = new Date(events[i - 1].timestamp).getTime()
                    const curr = new Date(events[i].timestamp).getTime()
                    expect(curr).toBeGreaterThanOrEqual(prev)
                }
            }
        }, 10000)

        it('should execute workflow with complex variables', async () => {
            if (!testWorkflowId) {
                throw new Error('Test workflow not created')
            }

            const execution = await executionsApi.executeWorkflow({
                workflow_id: testWorkflowId,
                variables: {
                    input: 10,
                    metadata: {
                        user: 'test-user',
                        timestamp: new Date().toISOString(),
                    },
                },
            })

            expect(execution).toBeDefined()
            expect(execution.variables.input).toBe(10)
            expect(execution.variables.metadata).toBeDefined()
        }, 15000)

        it('should filter executions by phase', async () => {
            const executions = await executionsApi.listExecutions({
                phase: 'completed',
            })

            expect(Array.isArray(executions)).toBe(true)
            if (executions.length > 0) {
                expect(executions.every((e) => e.phase === 'completed')).toBe(true)
            }
        }, 10000)
    })

    describe('Execution Lifecycle', () => {
        let lifecycleExecutionId: string | null = null

        it('should execute workflow and track lifecycle', async () => {
            if (!testWorkflowId) {
                throw new Error('Test workflow not created')
            }

            // Start execution
            const execution = await executionsApi.executeWorkflow({
                workflow_id: testWorkflowId,
                variables: { input: 7 },
            })

            lifecycleExecutionId = execution.id

            expect(execution.phase).toBeDefined()
            expect(['pending', 'executing', 'completed']).toContain(execution.phase)

            // Wait a bit for execution to complete
            await new Promise((resolve) => setTimeout(resolve, 2000))

            // Check final state
            const finalExecution = await executionsApi.getExecution(execution.id)
            expect(finalExecution.phase).toBeDefined()

            // If completed, should have completion timestamp
            if (finalExecution.phase === 'completed') {
                expect(finalExecution.completed_at).toBeDefined()
                expect(finalExecution.duration_ms).toBeDefined()
                expect(finalExecution.duration_ms).toBeGreaterThan(0)
            }
        }, 20000)

        it('should have execution events for lifecycle', async () => {
            if (!lifecycleExecutionId) {
                throw new Error('Lifecycle execution not created')
            }

            const events = await executionsApi.getExecutionEvents(lifecycleExecutionId)

            expect(events.length).toBeGreaterThan(0)

            // Should have at least execution started event
            const startEvent = events.find((e) => e.event_type === 'execution_started')
            expect(startEvent).toBeDefined()

            // Check event structure
            events.forEach((event) => {
                expect(event.id).toBeDefined()
                expect(event.execution_id).toBe(lifecycleExecutionId)
                expect(event.event_type).toBeDefined()
                expect(event.timestamp).toBeDefined()
            })
        }, 10000)
    })

    describe('Parallel Workflow Execution', () => {
        let parallelWorkflowId: string | null = null

        beforeAll(async () => {
            // Create a workflow with parallel branches
            const parallelWorkflow: CreateWorkflowRequest = {
                name: `parallel-workflow-${Date.now()}`,
                version: '1.0.0',
                description: 'Workflow with parallel execution',
                nodes: [
                    { type: 'parallel', name: 'fork' },
                    {
                        type: 'transform',
                        name: 'branch_a',
                        config: {
                            transformations: { result_a: 'input * 2' },
                        },
                    },
                    {
                        type: 'transform',
                        name: 'branch_b',
                        config: {
                            transformations: { result_b: 'input * 3' },
                        },
                    },
                    {
                        type: 'parallel',
                        name: 'join',
                        config: { join_strategy: 'wait_all' },
                    },
                ],
                edges: [
                    { from: 'fork', to: 'branch_a', type: 'fork' },
                    { from: 'fork', to: 'branch_b', type: 'fork' },
                    { from: 'branch_a', to: 'join', type: 'join' },
                    { from: 'branch_b', to: 'join', type: 'join' },
                ],
                triggers: [{ type: 'manual', config: {} }],
            }

            const workflow = await workflowsApi.createWorkflow(parallelWorkflow)
            parallelWorkflowId = workflow.id
        }, 15000)

        afterAll(async () => {
            if (parallelWorkflowId) {
                try {
                    await workflowsApi.deleteWorkflow(parallelWorkflowId)
                } catch (error) {
                    console.warn('Failed to cleanup parallel workflow:', error)
                }
            }
        })

        it('should execute parallel workflow', async () => {
            if (!parallelWorkflowId) {
                throw new Error('Parallel workflow not created')
            }

            const execution = await executionsApi.executeWorkflow({
                workflow_id: parallelWorkflowId,
                variables: { input: 4 },
            })

            expect(execution).toBeDefined()
            expect(execution.workflow_id).toBe(parallelWorkflowId)

            // Wait for execution to complete
            await new Promise((resolve) => setTimeout(resolve, 3000))

            const finalExecution = await executionsApi.getExecution(execution.id)
            const events = await executionsApi.getExecutionEvents(execution.id)

            expect(events.length).toBeGreaterThan(0)

            // Should have events for both branches
            const nodeEvents = events.filter((e) =>
                e.event_type.includes('node')
            )
            expect(nodeEvents.length).toBeGreaterThan(0)
        }, 20000)
    })

    describe('Error Handling', () => {
        it('should handle invalid workflow ID', async () => {
            await expect(
                executionsApi.executeWorkflow({
                    workflow_id: 'non-existent-workflow',
                    variables: {},
                })
            ).rejects.toThrow()
        }, 10000)

        it('should handle invalid execution ID', async () => {
            await expect(
                executionsApi.getExecution('non-existent-execution')
            ).rejects.toThrow()
        }, 10000)

        it('should handle missing variables gracefully', async () => {
            if (!testWorkflowId) {
                throw new Error('Test workflow not created')
            }

            // Execute without required variables - should fail since transform needs 'input'
            // This tests that the backend handles expression errors gracefully
            await expect(
                executionsApi.executeWorkflow({
                    workflow_id: testWorkflowId,
                    variables: {},
                })
            ).rejects.toThrow()
        }, 15000)
    })
})
