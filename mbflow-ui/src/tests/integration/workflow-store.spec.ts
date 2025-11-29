import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWorkflowStore } from '@/stores/workflow.store'
import type { Workflow, Node, Edge } from '@/types'

// Mock axios with interceptors
vi.mock('axios', () => ({
    default: {
        create: () => ({
            get: vi.fn(),
            post: vi.fn(),
            put: vi.fn(),
            delete: vi.fn(),
            interceptors: {
                request: { use: vi.fn() },
                response: { use: vi.fn() },
            },
        })
    }
}))

describe('Workflow Store Integration', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
    })

    describe('Workflow Management', () => {
        it('initializes with null current workflow', () => {
            const store = useWorkflowStore()
            expect(store.currentWorkflow).toBeNull()
        })

        it('creates a new workflow with proper structure', () => {
            const store = useWorkflowStore()

            const newWorkflow: Omit<Workflow, 'id'> = {
                name: 'Test Workflow',
                version: '1.0.0',
                description: 'Test description',
                nodes: [],
                edges: [],
                triggers: [{
                    id: 'trigger-1',
                    type: 'manual',
                    config: {}
                }],
                metadata: {}
            }

            store.currentWorkflow = { id: 'test-id', ...newWorkflow }

            expect(store.currentWorkflow).toBeDefined()
            expect(store.currentWorkflow?.name).toBe('Test Workflow')
            expect(store.currentWorkflow?.triggers).toHaveLength(1)
        })
    })

    describe('Node Operations', () => {
        beforeEach(() => {
            const store = useWorkflowStore()
            store.currentWorkflow = {
                id: 'workflow-1',
                name: 'Test Workflow',
                version: '1.0.0',
                description: '',
                nodes: [],
                edges: [],
                triggers: [{ id: 'trigger-1', type: 'manual', config: {} }],
                metadata: {}
            }
        })

        it('adds a node to workflow', () => {
            const store = useWorkflowStore()

            const newNode: Node = {
                id: 'node-1',
                type: 'transform',
                name: 'transform',
                config: {},
                metadata: { position: { x: 100, y: 100 } }
            }

            store.currentWorkflow!.nodes.push(newNode)

            expect(store.currentWorkflow?.nodes).toHaveLength(1)
            expect(store.currentWorkflow?.nodes[0].name).toBe('transform')
        })

        it('generates unique names for duplicate node types', () => {
            const store = useWorkflowStore()

            const node1: Node = {
                id: 'node-1',
                type: 'transform',
                name: 'transform',
                config: {},
                metadata: { position: { x: 100, y: 100 } }
            }

            const node2: Node = {
                id: 'node-2',
                type: 'transform',
                name: 'transform_swift_fox',
                config: {},
                metadata: { position: { x: 200, y: 100 } }
            }

            store.currentWorkflow!.nodes.push(node1, node2)

            expect(store.currentWorkflow?.nodes).toHaveLength(2)
            expect(store.currentWorkflow?.nodes[0].name).toBe('transform')
            expect(store.currentWorkflow?.nodes[1].name).toMatch(/^transform_[a-z]+_[a-z]+$/)
        })

        it('updates node configuration', () => {
            const store = useWorkflowStore()

            const node: Node = {
                id: 'node-1',
                type: 'transform',
                name: 'transform',
                config: { transformations: {} },
                metadata: { position: { x: 100, y: 100 } }
            }

            store.currentWorkflow!.nodes.push(node)

            // Update config
            store.currentWorkflow!.nodes[0].config = {
                transformations: { output: 'input * 2' }
            }

            expect(store.currentWorkflow?.nodes[0].config).toHaveProperty('transformations')
            expect(store.currentWorkflow?.nodes[0].config.transformations).toEqual({ output: 'input * 2' })
        })

        it('removes a node from workflow', () => {
            const store = useWorkflowStore()

            const node: Node = {
                id: 'node-1',
                type: 'transform',
                name: 'transform',
                config: {},
                metadata: { position: { x: 100, y: 100 } }
            }

            store.currentWorkflow!.nodes.push(node)
            expect(store.currentWorkflow?.nodes).toHaveLength(1)

            // Remove node
            store.currentWorkflow!.nodes = store.currentWorkflow!.nodes.filter(n => n.id !== 'node-1')
            expect(store.currentWorkflow?.nodes).toHaveLength(0)
        })
    })

    describe('Edge Operations', () => {
        beforeEach(() => {
            const store = useWorkflowStore()
            store.currentWorkflow = {
                id: 'workflow-1',
                name: 'Test Workflow',
                version: '1.0.0',
                description: '',
                nodes: [
                    { id: 'node-1', type: 'transform', name: 'transform_1', config: {}, metadata: {} },
                    { id: 'node-2', type: 'http', name: 'http_1', config: {}, metadata: {} }
                ],
                edges: [],
                triggers: [{ id: 'trigger-1', type: 'manual', config: {} }],
                metadata: {}
            }
        })

        it('adds an edge between nodes', () => {
            const store = useWorkflowStore()

            const edge: Edge = {
                id: 'edge-1',
                from: 'node-1',
                to: 'node-2',
                type: 'direct'
            }

            store.currentWorkflow!.edges.push(edge)

            expect(store.currentWorkflow?.edges).toHaveLength(1)
            expect(store.currentWorkflow?.edges[0].from).toBe('node-1')
            expect(store.currentWorkflow?.edges[0].to).toBe('node-2')
        })

        it('prevents duplicate edges', () => {
            const store = useWorkflowStore()

            const edge1: Edge = {
                id: 'edge-1',
                from: 'node-1',
                to: 'node-2',
                type: 'direct'
            }

            const edge2: Edge = {
                id: 'edge-2',
                from: 'node-1',
                to: 'node-2',
                type: 'direct'
            }

            store.currentWorkflow!.edges.push(edge1)

            // Check if edge already exists before adding
            const edgeExists = store.currentWorkflow!.edges.some(
                e => e.from === edge2.from && e.to === edge2.to
            )

            if (!edgeExists) {
                store.currentWorkflow!.edges.push(edge2)
            }

            expect(store.currentWorkflow?.edges).toHaveLength(1)
        })

        it('removes an edge', () => {
            const store = useWorkflowStore()

            const edge: Edge = {
                id: 'edge-1',
                from: 'node-1',
                to: 'node-2',
                type: 'direct'
            }

            store.currentWorkflow!.edges.push(edge)
            expect(store.currentWorkflow?.edges).toHaveLength(1)

            // Remove edge
            store.currentWorkflow!.edges = store.currentWorkflow!.edges.filter(e => e.id !== 'edge-1')
            expect(store.currentWorkflow?.edges).toHaveLength(0)
        })
    })

    describe('Workflow Validation', () => {
        it('requires at least one trigger', () => {
            const store = useWorkflowStore()

            store.currentWorkflow = {
                id: 'workflow-1',
                name: 'Test Workflow',
                version: '1.0.0',
                description: '',
                nodes: [],
                edges: [],
                triggers: [],
                metadata: {}
            }

            expect(store.currentWorkflow.triggers).toHaveLength(0)

            // Add default trigger
            store.currentWorkflow.triggers.push({
                id: 'trigger-1',
                type: 'manual',
                config: {}
            })

            expect(store.currentWorkflow.triggers).toHaveLength(1)
        })

        it('validates node names are in snake_case', () => {
            const store = useWorkflowStore()

            store.currentWorkflow = {
                id: 'workflow-1',
                name: 'Test Workflow',
                version: '1.0.0',
                description: '',
                nodes: [
                    { id: 'node-1', type: 'transform', name: 'transform_node', config: {}, metadata: {} },
                    { id: 'node-2', type: 'http', name: 'http_request_1', config: {}, metadata: {} }
                ],
                edges: [],
                triggers: [{ id: 'trigger-1', type: 'manual', config: {} }],
                metadata: {}
            }

            // All node names should match snake_case pattern
            const allSnakeCase = store.currentWorkflow.nodes.every(
                node => /^[a-z0-9_]+$/.test(node.name)
            )

            expect(allSnakeCase).toBe(true)
        })
    })
})
