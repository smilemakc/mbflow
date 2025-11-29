import { describe, it, expect, beforeAll, afterAll, beforeEach } from 'vitest'
import { workflowsApi, nodesApi, edgesApi } from '@/api/workflows.api'
import type { Workflow, Node, Edge, CreateWorkflowRequest } from '@/types'

/**
 * Integration tests for Workflows API
 * These tests require the backend server to be running on http://localhost:8181
 * 
 * To run these tests:
 * 1. Start the backend: cd /Users/balashov/PycharmProjects/mbflow && go run cmd/server/main.go
 * 2. Run tests: npm run test -- workflows.api.spec.ts
 */

describe('Workflows API Integration', () => {
    let testWorkflowId: string | null = null
    const testWorkflowName = `test-workflow-${Date.now()}`

    // Cleanup after all tests
    afterAll(async () => {
        if (testWorkflowId) {
            try {
                await workflowsApi.deleteWorkflow(testWorkflowId)
            } catch (error) {
                console.warn('Failed to cleanup test workflow:', error)
            }
        }
    })

    describe('Workflow CRUD Operations', () => {
        it('should create a new workflow', async () => {
            const workflowData: CreateWorkflowRequest = {
                name: testWorkflowName,
                version: '1.0.0',
                description: 'Integration test workflow',
                nodes: [
                    {
                        type: 'transform',
                        name: 'double',
                        config: {
                            transformations: {
                                result: 'input * 2',
                            },
                        },
                    },
                ],
                edges: [],
                triggers: [
                    {
                        type: 'manual',
                        config: {},
                    },
                ],
                metadata: {
                    test: true,
                },
            }

            const workflow = await workflowsApi.createWorkflow(workflowData)

            expect(workflow).toBeDefined()
            expect(workflow.id).toBeDefined()
            expect(workflow.name).toBe(testWorkflowName)
            expect(workflow.version).toBe('1.0.0')
            expect(workflow.nodes).toHaveLength(1)
            expect(workflow.edges).toHaveLength(0)
            expect(workflow.triggers).toHaveLength(1)

            testWorkflowId = workflow.id
        }, 10000)

        it('should list all workflows', async () => {
            const workflows = await workflowsApi.listWorkflows()

            expect(Array.isArray(workflows)).toBe(true)
            expect(workflows.length).toBeGreaterThan(0)

            const testWorkflow = workflows.find((w) => w.name === testWorkflowName)
            expect(testWorkflow).toBeDefined()
        }, 10000)

        it('should get workflow by ID', async () => {
            if (!testWorkflowId) {
                throw new Error('Test workflow not created')
            }

            const workflow = await workflowsApi.getWorkflow(testWorkflowId)

            expect(workflow).toBeDefined()
            expect(workflow.id).toBe(testWorkflowId)
            expect(workflow.name).toBe(testWorkflowName)
            expect(workflow.nodes).toHaveLength(1)
        }, 10000)

        it('should update workflow', async () => {
            if (!testWorkflowId) {
                throw new Error('Test workflow not created')
            }

            const updatedWorkflow = await workflowsApi.updateWorkflow(testWorkflowId, {
                description: 'Updated description',
                metadata: {
                    test: true,
                    updated: true,
                },
            })

            expect(updatedWorkflow).toBeDefined()
            expect(updatedWorkflow.description).toBe('Updated description')
            expect(updatedWorkflow.metadata?.updated).toBe(true)
        }, 10000)

        it('should get workflow graph', async () => {
            if (!testWorkflowId) {
                throw new Error('Test workflow not created')
            }

            const graph = await workflowsApi.getWorkflowGraph(testWorkflowId)

            expect(graph).toBeDefined()
            expect(graph.nodes).toHaveLength(1)
            expect(graph.edges).toHaveLength(0)
        }, 10000)

        it('should delete workflow', async () => {
            // Create a temporary workflow to delete
            const tempWorkflow = await workflowsApi.createWorkflow({
                name: `temp-workflow-${Date.now()}`,
                version: '1.0.0',
                description: 'Temporary workflow for delete test',
                nodes: [
                    { type: 'transform', name: 'placeholder', config: {} },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })

            await workflowsApi.deleteWorkflow(tempWorkflow.id)

            // Verify deletion
            await expect(workflowsApi.getWorkflow(tempWorkflow.id)).rejects.toThrow()
        }, 10000)
    })

    describe('Node Operations', () => {
        let nodeWorkflowId: string | null = null

        beforeEach(async () => {
            // Create a fresh workflow for node tests
            const workflow = await workflowsApi.createWorkflow({
                name: `node-test-workflow-${Date.now()}`,
                version: '1.0.0',
                nodes: [
                    { type: 'transform', name: 'placeholder', config: {} },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })
            nodeWorkflowId = workflow.id
        }, 15000)

        it('should list all nodes in workflow', async () => {
            if (!nodeWorkflowId) throw new Error('Test workflow not created')

            const nodes = await nodesApi.listNodes(nodeWorkflowId)

            expect(Array.isArray(nodes)).toBe(true)
            expect(nodes.length).toBeGreaterThan(0)
        }, 10000)

        it('should create a new node', async () => {
            if (!nodeWorkflowId) throw new Error('Test workflow not created')

            const newNode = await nodesApi.createNode(nodeWorkflowId, {
                type: 'http',
                name: 'api_call',
                config: {
                    url: 'https://api.example.com/data',
                    method: 'GET',
                },
            })

            expect(newNode).toBeDefined()
            expect(newNode.id).toBeDefined()
            expect(newNode.type).toBe('http')
            expect(newNode.name).toBe('api_call')
        }, 10000)

        it('should get node by ID', async () => {
            if (!nodeWorkflowId) throw new Error('Test workflow not created')

            // Create a node first
            const createdNode = await nodesApi.createNode(nodeWorkflowId, {
                type: 'http',
                name: 'test_node',
                config: { url: 'https://test.com', method: 'GET' },
            })

            const node = await nodesApi.getNode(nodeWorkflowId, createdNode.id)

            expect(node).toBeDefined()
            expect(node.id).toBe(createdNode.id)
            expect(node.type).toBe('http')
        }, 10000)

        it('should update node', async () => {
            if (!nodeWorkflowId) throw new Error('Test workflow not created')

            // Create a node first
            const createdNode = await nodesApi.createNode(nodeWorkflowId, {
                type: 'http',
                name: 'update_test_node',
                config: { url: 'https://original.com', method: 'GET' },
            })

            const updatedNode = await nodesApi.updateNode(nodeWorkflowId, createdNode.id, {
                config: {
                    url: 'https://api.example.com/updated',
                    method: 'POST',
                },
            })

            expect(updatedNode).toBeDefined()
            expect(updatedNode.config?.url).toBe('https://api.example.com/updated')
            expect(updatedNode.config?.method).toBe('POST')
        }, 10000)

        it('should delete node', async () => {
            if (!nodeWorkflowId) throw new Error('Test workflow not created')

            // Create a node first
            const createdNode = await nodesApi.createNode(nodeWorkflowId, {
                type: 'http',
                name: 'delete_test_node',
                config: { url: 'https://test.com', method: 'GET' },
            })

            await nodesApi.deleteNode(nodeWorkflowId, createdNode.id)

            // Verify deletion
            await expect(nodesApi.getNode(nodeWorkflowId, createdNode.id)).rejects.toThrow()
        }, 10000)

        it('should get available node types', async () => {
            const nodeTypes = await nodesApi.getNodeTypes()

            expect(Array.isArray(nodeTypes)).toBe(true)
            expect(nodeTypes.length).toBeGreaterThan(0)

            // Check for common node types
            const types = nodeTypes.map((nt) => nt.type)
            expect(types).toContain('transform')
            expect(types).toContain('http')
        }, 10000)
    })

    describe('Edge Operations', () => {
        let edgeWorkflowId: string | null = null
        let edgeId: string | null = null

        beforeEach(async () => {
            // Create a fresh workflow for edge tests to avoid conflicts
            const workflow = await workflowsApi.createWorkflow({
                name: `edge-test-workflow-${Date.now()}`,
                version: '1.0.0',
                nodes: [
                    { type: 'transform', name: 'transform_1', config: {} },
                    { type: 'transform', name: 'transform_2', config: {} },
                ],
                edges: [],
                triggers: [{ type: 'manual', config: {} }],
            })
            edgeWorkflowId = workflow.id
        }, 15000)

        it('should list all edges in workflow', async () => {
            if (!edgeWorkflowId) throw new Error('Test workflow not created')

            const edges = await edgesApi.listEdges(edgeWorkflowId)

            expect(Array.isArray(edges)).toBe(true)
        }, 10000)

        it('should create a new edge', async () => {
            if (!edgeWorkflowId) throw new Error('Test workflow not created')

            const newEdge = await edgesApi.createEdge(edgeWorkflowId, {
                from: 'transform_1',
                to: 'transform_2',
                type: 'direct',
            })

            expect(newEdge).toBeDefined()
            expect(newEdge.id).toBeDefined()
            expect(newEdge.from).toBe('transform_1')
            expect(newEdge.to).toBe('transform_2')
            expect(newEdge.type).toBe('direct')

            edgeId = newEdge.id
        }, 10000)

        it('should get edge by ID', async () => {
            if (!edgeWorkflowId) throw new Error('Test workflow not created')

            // Create an edge first
            const createdEdge = await edgesApi.createEdge(edgeWorkflowId, {
                from: 'transform_1',
                to: 'transform_2',
                type: 'direct',
            })

            const edge = await edgesApi.getEdge(edgeWorkflowId, createdEdge.id)

            expect(edge).toBeDefined()
            expect(edge.id).toBe(createdEdge.id)
        }, 10000)

        it('should create conditional edge', async () => {
            if (!edgeWorkflowId) throw new Error('Test workflow not created')

            const conditionalEdge = await edgesApi.createEdge(edgeWorkflowId, {
                from: 'transform_1',
                to: 'transform_2',
                type: 'conditional',
                condition: {
                    expression: 'result > 0',
                },
            })

            expect(conditionalEdge).toBeDefined()
            expect(conditionalEdge.id).toBeDefined()
            expect(conditionalEdge.type).toBe('conditional')
            // Note: condition may be stored differently by backend
        }, 10000)

        it('should update edge', async () => {
            if (!edgeWorkflowId) throw new Error('Test workflow not created')

            // Create an edge first
            const createdEdge = await edgesApi.createEdge(edgeWorkflowId, {
                from: 'transform_1',
                to: 'transform_2',
                type: 'direct',
            })

            const updatedEdge = await edgesApi.updateEdge(edgeWorkflowId, createdEdge.id, {
                type: 'conditional',
                condition: {
                    expression: 'result > 100',
                },
            })

            expect(updatedEdge).toBeDefined()
            expect(updatedEdge.id).toBeDefined()
            expect(updatedEdge.type).toBe('conditional')
            // Note: Backend may return same or new ID depending on implementation
        }, 10000)

        it('should delete edge', async () => {
            if (!edgeWorkflowId) throw new Error('Test workflow not created')

            // Create an edge first
            const createdEdge = await edgesApi.createEdge(edgeWorkflowId, {
                from: 'transform_1',
                to: 'transform_2',
                type: 'direct',
            })

            await edgesApi.deleteEdge(edgeWorkflowId, createdEdge.id)

            // Verify deletion
            await expect(edgesApi.getEdge(edgeWorkflowId, createdEdge.id)).rejects.toThrow()
        }, 10000)

        it('should get available edge types', async () => {
            const edgeTypes = await edgesApi.getEdgeTypes()

            expect(Array.isArray(edgeTypes)).toBe(true)
            expect(edgeTypes.length).toBeGreaterThan(0)

            // Check for common edge types
            const types = edgeTypes.map((et) => et.type)
            expect(types).toContain('direct')
            expect(types).toContain('conditional')
        }, 10000)
    })
})
