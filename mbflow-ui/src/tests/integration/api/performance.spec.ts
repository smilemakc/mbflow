import { describe, it, expect, afterAll } from 'vitest'
import { workflowsApi } from '@/api/workflows.api'
import { executionsApi } from '@/api/executions.api'

import {
    cleanupWorkflows,
    generateTestName,
    PerformanceTimer,
    waitForExecutionCompletion,
} from '../helpers/test-utils'

/**
 * Performance and stress tests for API
 * These tests measure response times and system behavior under load
 * 
 * Note: These tests may take longer to run
 */

describe('API Performance Tests', () => {
    const createdWorkflowIds: string[] = []

    afterAll(async () => {
        await cleanupWorkflows(createdWorkflowIds)
    })

    describe('Response Time Tests', () => {
        it('should create workflow within acceptable time', async () => {
            const timer = new PerformanceTimer()

            const workflow = await workflowsApi.createWorkflow({
                name: generateTestName('perf-create'),
                version: '1.0.0',
                nodes: [
                    { type: 'transform', name: 'placeholder', config: {} },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })

            createdWorkflowIds.push(workflow.id)

            // Should complete within 2 seconds
            timer.assertMaxDuration(2000, 'Workflow creation')
            expect(workflow.id).toBeDefined()
        }, 10000)

        it('should list workflows within acceptable time', async () => {
            const timer = new PerformanceTimer()

            const workflows = await workflowsApi.listWorkflows()

            // Should complete within 1 second
            timer.assertMaxDuration(1000, 'List workflows')
            expect(Array.isArray(workflows)).toBe(true)
        }, 10000)

        it('should execute simple workflow within acceptable time', async () => {
            // Create test workflow
            const workflow = await workflowsApi.createWorkflow({
                name: generateTestName('perf-exec'),
                version: '1.0.0',
                nodes: [
                    {
                        type: 'transform',
                        name: 'calc',
                        config: {
                            transformations: { result: 'input + 1' },
                        },
                    },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })

            createdWorkflowIds.push(workflow.id)

            const timer = new PerformanceTimer()

            const execution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: { input: 42 },
            })

            // Execution start should be fast
            timer.assertMaxDuration(1000, 'Execution start')

            // Wait for completion
            timer.reset()
            const completed = await waitForExecutionCompletion(execution.id, 10000)

            // Simple workflow should complete quickly
            expect(completed.phase).toBe('completed')
            expect(timer.elapsed()).toBeLessThan(5000)
        }, 20000)
    })

    describe('Concurrent Operations', () => {
        it('should handle multiple workflow creations concurrently', async () => {
            const count = 5
            const promises = []

            for (let i = 0; i < count; i++) {
                const promise = workflowsApi.createWorkflow({
                    name: generateTestName(`concurrent-${i}`),
                    version: '1.0.0',
                    nodes: [
                        { type: 'transform', name: 'placeholder', config: {} },
                    ],
                    edges: [],
                    triggers: [{ type: 'manual', config: {} }],
                })
                promises.push(promise)
            }

            const workflows = await Promise.all(promises)

            expect(workflows).toHaveLength(count)
            workflows.forEach((wf) => {
                expect(wf.id).toBeDefined()
                createdWorkflowIds.push(wf.id)
            })
        }, 30000)

        it('should handle multiple executions concurrently', async () => {
            // Create a test workflow
            const workflow = await workflowsApi.createWorkflow({
                name: generateTestName('concurrent-exec'),
                version: '1.0.0',
                nodes: [
                    {
                        type: 'transform',
                        name: 'process',
                        config: {
                            transformations: { result: 'value * 2' },
                        },
                    },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })

            createdWorkflowIds.push(workflow.id)

            // Execute multiple times concurrently
            const count = 5
            const promises = []

            for (let i = 0; i < count; i++) {
                const promise = executionsApi.executeWorkflow({
                    workflow_id: workflow.id,
                    variables: { value: i },
                })
                promises.push(promise)
            }

            const executions = await Promise.all(promises)

            expect(executions).toHaveLength(count)
            executions.forEach((exec) => {
                expect(exec.id).toBeDefined()
                expect(exec.workflow_id).toBe(workflow.id)
            })

            // All executions should have unique IDs
            const ids = executions.map((e) => e.id)
            const uniqueIds = new Set(ids)
            expect(uniqueIds.size).toBe(count)
        }, 40000)
    })

    describe('Large Workflow Tests', () => {
        it('should handle workflow with many nodes', async () => {
            const nodeCount = 20
            const nodes: any[] = []
            const edges: any[] = []

            // Create chain of transform nodes
            for (let i = 0; i < nodeCount; i++) {
                nodes.push({
                    type: 'transform' as const,
                    name: `transform_${i}`,
                    config: {
                        transformations: {
                            [`result_${i}`]: `value + ${i}`,
                        },
                    },
                })

                if (i > 0) {
                    edges.push({
                        from: `transform_${i - 1}`,
                        to: `transform_${i}`,
                        type: 'direct' as const,
                    })
                }
            }

            const workflow = await workflowsApi.createWorkflow({
                name: generateTestName('large-workflow'),
                version: '1.0.0',
                nodes,
                edges,
                triggers: [{ type: 'manual', config: {} }],
            })

            createdWorkflowIds.push(workflow.id)

            expect(workflow.nodes).toHaveLength(nodeCount)
            expect(workflow.edges).toHaveLength(nodeCount - 1)

            // Execute large workflow
            const execution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: { value: 1 },
            })

            expect(execution.id).toBeDefined()
        }, 30000)

        it('should handle workflow with complex parallel branches', async () => {
            const branchCount = 5
            const nodes: any[] = [
                { type: 'parallel', name: 'fork' },
            ]
            const edges: any[] = []

            // Create parallel branches
            for (let i = 0; i < branchCount; i++) {
                nodes.push({
                    type: 'transform' as const,
                    name: `branch_${i}`,
                    config: {
                        transformations: {
                            [`result_${i}`]: `input * ${i + 1}`,
                        },
                    },
                })

                edges.push({
                    from: 'fork',
                    to: `branch_${i}`,
                    type: 'fork' as const,
                })
            }

            // Add join node
            nodes.push({
                type: 'parallel' as const,
                name: 'join',
                config: { join_strategy: 'wait_all' },
            })

            for (let i = 0; i < branchCount; i++) {
                edges.push({
                    from: `branch_${i}`,
                    to: 'join',
                    type: 'join' as const,
                })
            }

            const workflow = await workflowsApi.createWorkflow({
                name: generateTestName('parallel-branches'),
                version: '1.0.0',
                nodes,
                edges,
                triggers: [{ type: 'manual', config: {} }],
            })

            createdWorkflowIds.push(workflow.id)

            expect(workflow.nodes).toHaveLength(branchCount + 2) // branches + fork + join

            // Execute parallel workflow
            const execution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: { input: 10 },
            })

            expect(execution.id).toBeDefined()
        }, 30000)
    })

    describe('Data Volume Tests', () => {
        it('should handle execution with large variable payload', async () => {
            const workflow = await workflowsApi.createWorkflow({
                name: generateTestName('large-payload'),
                version: '1.0.0',
                nodes: [
                    { type: 'transform', name: 'placeholder', config: {} },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })

            createdWorkflowIds.push(workflow.id)

            // Create large payload
            const largeArray = Array.from({ length: 1000 }, (_, i) => ({
                id: i,
                value: Math.random(),
                text: `Item ${i}`,
            }))

            const execution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: {
                    data: largeArray,
                    metadata: {
                        count: largeArray.length,
                        timestamp: new Date().toISOString(),
                    },
                },
            })

            expect(execution.id).toBeDefined()
            expect(execution.variables.data).toHaveLength(1000)
        }, 20000)

        it('should handle many executions for same workflow', async () => {
            const workflow = await workflowsApi.createWorkflow({
                name: generateTestName('many-executions'),
                version: '1.0.0',
                nodes: [
                    {
                        type: 'transform',
                        name: 'process',
                        config: {
                            transformations: { result: 'counter + 1' },
                        },
                    },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })

            createdWorkflowIds.push(workflow.id)

            // Create many executions
            const executionCount = 10
            for (let i = 0; i < executionCount; i++) {
                await executionsApi.executeWorkflow({
                    workflow_id: workflow.id,
                    variables: { counter: i },
                })
            }

            // Wait a bit for executions to process
            await new Promise((resolve) => setTimeout(resolve, 2000))

            // Verify all executions are tracked
            const executions = await executionsApi.listExecutions({
                workflow_id: workflow.id,
            })

            expect(executions.length).toBeGreaterThanOrEqual(executionCount)
        }, 40000)
    })

    describe('Stress Tests', () => {
        it('should handle rapid workflow updates', async () => {
            const workflow = await workflowsApi.createWorkflow({
                name: generateTestName('rapid-updates'),
                version: '1.0.0',
                nodes: [
                    { type: 'transform', name: 'placeholder', config: {} },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })

            createdWorkflowIds.push(workflow.id)

            // Perform rapid updates
            const updateCount = 5
            for (let i = 0; i < updateCount; i++) {
                await workflowsApi.updateWorkflow(workflow.id, {
                    description: `Update ${i}`,
                    version: `1.0.${i}`,
                })
            }

            const finalWorkflow = await workflowsApi.getWorkflow(workflow.id)
            expect(finalWorkflow.version).toBe(`1.0.${updateCount - 1}`)
        }, 30000)

        it('should maintain consistency under concurrent modifications', async () => {
            const workflow = await workflowsApi.createWorkflow({
                name: generateTestName('concurrent-mods'),
                version: '1.0.0',
                nodes: [
                    { type: 'transform', name: 'placeholder', config: {} },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })

            createdWorkflowIds.push(workflow.id)

            // Try concurrent updates (may result in conflicts, which is expected)
            const promises = []
            for (let i = 0; i < 3; i++) {
                promises.push(
                    workflowsApi
                        .updateWorkflow(workflow.id, {
                            description: `Concurrent update ${i}`,
                        })
                        .catch((err) => {
                            // Some updates may fail due to conflicts, which is OK
                            console.log('Expected conflict:', err.message)
                            return null
                        })
                )
            }

            await Promise.all(promises)

            // Workflow should still be valid
            const finalWorkflow = await workflowsApi.getWorkflow(workflow.id)
            expect(finalWorkflow.id).toBe(workflow.id)
            expect(finalWorkflow.description).toBeDefined()
        }, 30000)
    })
})
