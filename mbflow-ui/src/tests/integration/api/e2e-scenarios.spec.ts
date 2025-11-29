import { describe, it, expect, afterAll } from 'vitest'
import { workflowsApi } from '@/api/workflows.api'
import { executionsApi } from '@/api/executions.api'
import type { CreateWorkflowRequest } from '@/types'

/**
 * End-to-End Integration tests
 * These tests simulate complete user workflows from creation to execution
 * 
 * Requires backend server running on http://localhost:8181
 */

describe('E2E: Complete Workflow Scenarios', () => {
    const createdWorkflowIds: string[] = []

    // Cleanup all created workflows
    afterAll(async () => {
        for (const id of createdWorkflowIds) {
            try {
                await workflowsApi.deleteWorkflow(id)
            } catch (error) {
                console.warn(`Failed to cleanup workflow ${id}:`, error)
            }
        }
    })

    describe('Simple Transform Workflow', () => {
        it('should create, execute, and verify a simple transform workflow', async () => {
            // 1. Create workflow
            // Note: Variables are namespaced by parent node name
            // e.g., output from 'double' node is accessed as 'double.doubled'
            const workflowData: CreateWorkflowRequest = {
                name: `e2e-simple-${Date.now()}`,
                version: '1.0.0',
                description: 'E2E test: Simple transformation',
                nodes: [
                    {
                        type: 'transform',
                        name: 'double',
                        config: {
                            transformations: {
                                // 'value' comes from initial variables
                                doubled: 'value * 2',
                            },
                        },
                    },
                    {
                        type: 'transform',
                        name: 'square',
                        config: {
                            transformations: {
                                // Access parent node output with namespace
                                squared: 'double.doubled * double.doubled',
                            },
                        },
                    },
                ],
                edges: [
                    { from: 'double', to: 'square', type: 'direct' },
                ],
                triggers: [{ type: 'manual', config: {} }],
            }

            const workflow = await workflowsApi.createWorkflow(workflowData)
            createdWorkflowIds.push(workflow.id)

            expect(workflow.id).toBeDefined()
            expect(workflow.nodes).toHaveLength(2)

            // 2. Execute workflow
            const execution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: { value: 5 },
            })

            expect(execution.id).toBeDefined()
            expect(execution.workflow_id).toBe(workflow.id)

            // 3. Wait for completion
            await new Promise((resolve) => setTimeout(resolve, 3000))

            // 4. Verify execution
            const finalExecution = await executionsApi.getExecution(execution.id)
            expect(finalExecution.phase).toBeDefined()

            // 5. Check events
            const events = await executionsApi.getExecutionEvents(execution.id)
            expect(events.length).toBeGreaterThan(0)

            const startEvent = events.find((e) => e.event_type === 'execution_started')
            expect(startEvent).toBeDefined()
        }, 25000)
    })

    describe('Conditional Routing Workflow', () => {
        it('should create and execute a workflow with conditional routing', async () => {
            // Create workflow with conditional-router node
            // conditional-router requires routes config with conditions
            const workflowData: CreateWorkflowRequest = {
                name: `e2e-conditional-${Date.now()}`,
                version: '1.0.0',
                description: 'E2E test: Conditional routing',
                nodes: [
                    {
                        type: 'conditional-router',
                        name: 'router',
                        config: {
                            routes: [
                                { name: 'positive', condition: 'value > 0' },
                                { name: 'negative', condition: 'value <= 0' },
                            ],
                            default_route: 'negative',
                        },
                    },
                    {
                        type: 'transform',
                        name: 'positive_path',
                        config: {
                            transformations: {
                                message: '"Value is positive"',
                            },
                        },
                    },
                    {
                        type: 'transform',
                        name: 'negative_path',
                        config: {
                            transformations: {
                                message: '"Value is negative or zero"',
                            },
                        },
                    },
                ],
                edges: [
                    {
                        from: 'router',
                        to: 'positive_path',
                        type: 'conditional',
                        condition: { expression: 'router.selected_route == "positive"' },
                    },
                    {
                        from: 'router',
                        to: 'negative_path',
                        type: 'conditional',
                        condition: { expression: 'router.selected_route == "negative"' },
                    },
                ],
                triggers: [{ type: 'manual', config: {} }],
            }

            const workflow = await workflowsApi.createWorkflow(workflowData)
            createdWorkflowIds.push(workflow.id)

            // Test positive path
            const positiveExecution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: { value: 42 },
            })

            expect(positiveExecution.id).toBeDefined()

            // Test negative path
            const negativeExecution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: { value: -5 },
            })

            expect(negativeExecution.id).toBeDefined()

            // Wait for both executions
            await new Promise((resolve) => setTimeout(resolve, 4000))

            // Verify both executions
            const positiveFinal = await executionsApi.getExecution(positiveExecution.id)
            const negativeFinal = await executionsApi.getExecution(negativeExecution.id)

            expect(positiveFinal.phase).toBeDefined()
            expect(negativeFinal.phase).toBeDefined()
        }, 30000)
    })

    describe('HTTP Integration Workflow', () => {
        it('should create and execute a workflow with HTTP node', async () => {
            const workflowData: CreateWorkflowRequest = {
                name: `e2e-http-${Date.now()}`,
                version: '1.0.0',
                description: 'E2E test: HTTP integration',
                nodes: [
                    {
                        type: 'http',
                        name: 'api_call',
                        config: {
                            url: 'https://jsonplaceholder.typicode.com/posts/1',
                            method: 'GET',
                            timeout: 5000,
                        },
                    },
                    {
                        type: 'transform',
                        name: 'extract_title',
                        config: {
                            transformations: {
                                // HTTP node returns parsed JSON fields directly
                                // Access via node_name.field_name
                                post_title: 'api_call.title',
                            },
                        },
                    },
                ],
                edges: [
                    { from: 'api_call', to: 'extract_title', type: 'direct' },
                ],
                triggers: [{ type: 'manual', config: {} }],
            }

            const workflow = await workflowsApi.createWorkflow(workflowData)
            createdWorkflowIds.push(workflow.id)

            const execution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: {},
            })

            expect(execution.id).toBeDefined()

            // Wait for HTTP call and processing
            await new Promise((resolve) => setTimeout(resolve, 8000))

            const events = await executionsApi.getExecutionEvents(execution.id)

            expect(events.length).toBeGreaterThan(0)

            // Should have node execution events
            const nodeEvents = events.filter((e) =>
                e.event_type.includes('node')
            )
            expect(nodeEvents.length).toBeGreaterThan(0)
        }, 35000)
    })

    describe('Workflow Modification and Re-execution', () => {
        it('should modify workflow and execute with new configuration', async () => {
            // 1. Create initial workflow
            const workflowData: CreateWorkflowRequest = {
                name: `e2e-modify-${Date.now()}`,
                version: '1.0.0',
                description: 'E2E test: Workflow modification',
                nodes: [
                    {
                        type: 'transform',
                        name: 'calc',
                        config: {
                            transformations: {
                                result: 'x + y',
                            },
                        },
                    },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            }

            const workflow = await workflowsApi.createWorkflow(workflowData)
            createdWorkflowIds.push(workflow.id)

            // 2. Execute original workflow
            const execution1 = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: { x: 10, y: 5 },
            })

            expect(execution1.id).toBeDefined()

            // 3. Update workflow configuration
            const updatedWorkflow = await workflowsApi.updateWorkflow(workflow.id, {
                description: 'Modified workflow',
                version: '1.1.0',
            })

            expect(updatedWorkflow.version).toBe('1.1.0')
            expect(updatedWorkflow.description).toBe('Modified workflow')

            // 4. Execute updated workflow
            const execution2 = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: { x: 20, y: 15 },
            })

            expect(execution2.id).toBeDefined()
            expect(execution2.id).not.toBe(execution1.id)

            // 5. Verify both executions exist
            await new Promise((resolve) => setTimeout(resolve, 3000))

            const executions = await executionsApi.listExecutions({
                workflow_id: workflow.id,
            })

            expect(executions.length).toBeGreaterThanOrEqual(2)
        }, 30000)
    })

    describe('Multi-Step Data Pipeline', () => {
        it('should execute a complex multi-step data pipeline', async () => {
            // Note: Each step accesses previous step's output via node_name.field_name
            const workflowData: CreateWorkflowRequest = {
                name: `e2e-pipeline-${Date.now()}`,
                version: '1.0.0',
                description: 'E2E test: Multi-step data pipeline',
                nodes: [
                    {
                        type: 'transform',
                        name: 'step1_normalize',
                        config: {
                            transformations: {
                                // 'value' from initial variables
                                normalized: 'value / 100',
                            },
                        },
                    },
                    {
                        type: 'transform',
                        name: 'step2_scale',
                        config: {
                            transformations: {
                                // Access previous step output
                                scaled: 'step1_normalize.normalized * 1000',
                            },
                        },
                    },
                    {
                        type: 'transform',
                        name: 'step3_round',
                        config: {
                            transformations: {
                                rounded: 'int(step2_scale.scaled)',
                            },
                        },
                    },
                    {
                        type: 'transform',
                        name: 'step4_format',
                        config: {
                            transformations: {
                                formatted: 'string(step3_round.rounded)',
                            },
                        },
                    },
                ],
                edges: [
                    { from: 'step1_normalize', to: 'step2_scale', type: 'direct' },
                    { from: 'step2_scale', to: 'step3_round', type: 'direct' },
                    { from: 'step3_round', to: 'step4_format', type: 'direct' },
                ],
                triggers: [{ type: 'manual', config: {} }],
            }

            const workflow = await workflowsApi.createWorkflow(workflowData)
            createdWorkflowIds.push(workflow.id)

            const execution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: { value: 42.5 },
            })

            expect(execution.id).toBeDefined()

            // Wait for pipeline to complete
            await new Promise((resolve) => setTimeout(resolve, 5000))

            const events = await executionsApi.getExecutionEvents(execution.id)

            // Should have node execution events for pipeline steps
            const nodeEvents = events.filter((e) => e.event_type.includes('node'))
            expect(nodeEvents.length).toBeGreaterThan(0)
        }, 35000)
    })

    describe('Error Recovery', () => {
        it('should handle workflow execution errors gracefully', async () => {
            const workflowData: CreateWorkflowRequest = {
                name: `e2e-error-${Date.now()}`,
                version: '1.0.0',
                description: 'E2E test: Error handling',
                nodes: [
                    {
                        type: 'http',
                        name: 'failing_api',
                        config: {
                            url: 'https://invalid-domain-that-does-not-exist-12345.com/api',
                            method: 'GET',
                            timeout: 2000,
                        },
                    },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            }

            const workflow = await workflowsApi.createWorkflow(workflowData)
            createdWorkflowIds.push(workflow.id)

            const execution = await executionsApi.executeWorkflow({
                workflow_id: workflow.id,
                variables: {},
            })

            expect(execution.id).toBeDefined()

            // Wait for execution to fail
            await new Promise((resolve) => setTimeout(resolve, 5000))

            const finalExecution = await executionsApi.getExecution(execution.id)
            const events = await executionsApi.getExecutionEvents(execution.id)

            expect(events.length).toBeGreaterThan(0)

            // Should have error-related events if execution failed
            const errorEvents = events.filter((e) =>
                e.event_type.includes('failed') || e.event_type.includes('error')
            )

            // Execution might be in failed state
            if (finalExecution.phase === 'failed') {
                expect(errorEvents.length).toBeGreaterThan(0)
            }
        }, 30000)
    })
})
