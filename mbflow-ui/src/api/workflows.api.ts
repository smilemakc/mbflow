/**
 * Workflows API - CRUD operations for workflows
 */

import { apiClient, request } from './client'
import type {
    Workflow,
    CreateWorkflowRequest,
    UpdateWorkflowRequest,
    WorkflowGraph,
    Node,
    Edge,
    NodeTypeMetadata,
    EdgeTypeMetadata,
} from '@/types'
import {
    mockWorkflows,
    mockNodeTypes,
    mockEdgeTypes,
} from './mock-data'

const USE_MOCK = import.meta.env.VITE_USE_MOCK_API === 'true'

// ============================================================================
// Workflows
// ============================================================================

export const workflowsApi = {
    /**
     * Get all workflows
     */
    async listWorkflows(): Promise<Workflow[]> {
        if (USE_MOCK) {
            return Promise.resolve([...mockWorkflows])
        }
        return request<Workflow[]>(apiClient.get('/api/v1/workflows'))
    },

    /**
     * Get workflow by ID
     */
    async getWorkflow(id: string): Promise<Workflow> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === id)
            if (!workflow) {
                throw new Error(`Workflow ${id} not found`)
            }
            return Promise.resolve(workflow)
        }
        return request<Workflow>(apiClient.get(`/api/v1/workflows/${id}`))
    },

    /**
     * Create new workflow
     */
    async createWorkflow(data: CreateWorkflowRequest): Promise<Workflow> {
        if (USE_MOCK) {
            const newWorkflow: Workflow = {
                id: `wf-${Date.now()}`,
                ...data,
                nodes: data.nodes.map((n, i) => ({ id: `node-${i}`, ...n })),
                edges: data.edges.map((e, i) => ({ id: `edge-${i}`, ...e })),
                triggers: data.triggers.map((t, i) => ({ id: `trigger-${i}`, ...t })),
                created_at: new Date().toISOString(),
            }
            mockWorkflows.push(newWorkflow)
            return Promise.resolve(newWorkflow)
        }
        return request<Workflow>(apiClient.post('/api/v1/workflows', data))
    },

    /**
     * Update workflow
     */
    async updateWorkflow(
        id: string,
        data: UpdateWorkflowRequest
    ): Promise<Workflow> {
        if (USE_MOCK) {
            const index = mockWorkflows.findIndex((w) => w.id === id)
            if (index === -1) {
                throw new Error(`Workflow ${id} not found`)
            }
            mockWorkflows[index] = {
                ...mockWorkflows[index],
                ...data,
                updated_at: new Date().toISOString(),
            }
            return Promise.resolve(mockWorkflows[index])
        }
        return request<Workflow>(apiClient.put(`/api/v1/workflows/${id}`, data))
    },

    /**
     * Delete workflow
     */
    async deleteWorkflow(id: string): Promise<void> {
        if (USE_MOCK) {
            const index = mockWorkflows.findIndex((w) => w.id === id)
            if (index !== -1) {
                mockWorkflows.splice(index, 1)
            }
            return Promise.resolve()
        }
        return request<void>(apiClient.delete(`/api/v1/workflows/${id}`))
    },

    /**
     * Get workflow graph (nodes + edges)
     */
    async getWorkflowGraph(workflowId: string): Promise<WorkflowGraph> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            if (!workflow) {
                throw new Error(`Workflow ${workflowId} not found`)
            }
            return Promise.resolve({
                nodes: workflow.nodes,
                edges: workflow.edges,
            })
        }
        return request<WorkflowGraph>(
            apiClient.get(`/api/v1/workflows/${workflowId}/graph`)
        )
    },
}

// ============================================================================
// Nodes
// ============================================================================

export const nodesApi = {
    /**
     * Get all nodes in workflow
     */
    async listNodes(workflowId: string): Promise<Node[]> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            return Promise.resolve(workflow?.nodes || [])
        }
        return request<Node[]>(
            apiClient.get(`/api/v1/workflows/${workflowId}/nodes`)
        )
    },

    /**
     * Get node by ID
     */
    async getNode(workflowId: string, nodeId: string): Promise<Node> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            const node = workflow?.nodes.find((n) => n.id === nodeId)
            if (!node) {
                throw new Error(`Node ${nodeId} not found`)
            }
            return Promise.resolve(node)
        }
        return request<Node>(
            apiClient.get(`/api/v1/workflows/${workflowId}/nodes/${nodeId}`)
        )
    },

    /**
     * Create node
     */
    async createNode(
        workflowId: string,
        data: Omit<Node, 'id'>
    ): Promise<Node> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            if (!workflow) {
                throw new Error(`Workflow ${workflowId} not found`)
            }
            const newNode: Node = {
                id: `node-${Date.now()}`,
                ...data,
            }
            workflow.nodes.push(newNode)
            return Promise.resolve(newNode)
        }
        return request<Node>(
            apiClient.post(`/api/v1/workflows/${workflowId}/nodes`, data)
        )
    },

    /**
     * Update node
     */
    async updateNode(
        workflowId: string,
        nodeId: string,
        data: Partial<Node>
    ): Promise<Node> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            if (!workflow) {
                throw new Error(`Workflow ${workflowId} not found`)
            }
            const index = workflow.nodes.findIndex((n) => n.id === nodeId)
            if (index === -1) {
                throw new Error(`Node ${nodeId} not found`)
            }
            workflow.nodes[index] = { ...workflow.nodes[index], ...data }
            return Promise.resolve(workflow.nodes[index])
        }
        return request<Node>(
            apiClient.put(`/api/v1/workflows/${workflowId}/nodes/${nodeId}`, data)
        )
    },

    /**
     * Delete node
     */
    async deleteNode(workflowId: string, nodeId: string): Promise<void> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            if (!workflow) {
                throw new Error(`Workflow ${workflowId} not found`)
            }
            const index = workflow.nodes.findIndex((n) => n.id === nodeId)
            if (index !== -1) {
                workflow.nodes.splice(index, 1)
            }
            return Promise.resolve()
        }
        return request<void>(
            apiClient.delete(`/api/v1/workflows/${workflowId}/nodes/${nodeId}`)
        )
    },

    /**
     * Get available node types
     */
    async getNodeTypes(): Promise<NodeTypeMetadata[]> {
        if (USE_MOCK) {
            return Promise.resolve([...mockNodeTypes])
        }
        return request<NodeTypeMetadata[]>(apiClient.get('/api/v1/node-types'))
    },
}

// ============================================================================
// Edges
// ============================================================================

export const edgesApi = {
    /**
     * Get all edges in workflow
     */
    async listEdges(workflowId: string): Promise<Edge[]> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            return Promise.resolve(workflow?.edges || [])
        }
        return request<Edge[]>(
            apiClient.get(`/api/v1/workflows/${workflowId}/edges`)
        )
    },

    /**
     * Get edge by ID
     */
    async getEdge(workflowId: string, edgeId: string): Promise<Edge> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            const edge = workflow?.edges.find((e) => e.id === edgeId)
            if (!edge) {
                throw new Error(`Edge ${edgeId} not found`)
            }
            return Promise.resolve(edge)
        }
        return request<Edge>(
            apiClient.get(`/api/v1/workflows/${workflowId}/edges/${edgeId}`)
        )
    },

    /**
     * Create edge
     */
    async createEdge(
        workflowId: string,
        data: Omit<Edge, 'id'>
    ): Promise<Edge> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            if (!workflow) {
                throw new Error(`Workflow ${workflowId} not found`)
            }
            const newEdge: Edge = {
                id: `edge-${Date.now()}`,
                ...data,
            }
            workflow.edges.push(newEdge)
            return Promise.resolve(newEdge)
        }
        return request<Edge>(
            apiClient.post(`/api/v1/workflows/${workflowId}/edges`, data)
        )
    },

    /**
     * Update edge
     */
    async updateEdge(
        workflowId: string,
        edgeId: string,
        data: Partial<Edge>
    ): Promise<Edge> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            if (!workflow) {
                throw new Error(`Workflow ${workflowId} not found`)
            }
            const index = workflow.edges.findIndex((e) => e.id === edgeId)
            if (index === -1) {
                throw new Error(`Edge ${edgeId} not found`)
            }
            workflow.edges[index] = { ...workflow.edges[index], ...data }
            return Promise.resolve(workflow.edges[index])
        }
        return request<Edge>(
            apiClient.put(`/api/v1/workflows/${workflowId}/edges/${edgeId}`, data)
        )
    },

    /**
     * Delete edge
     */
    async deleteEdge(workflowId: string, edgeId: string): Promise<void> {
        if (USE_MOCK) {
            const workflow = mockWorkflows.find((w) => w.id === workflowId)
            if (!workflow) {
                throw new Error(`Workflow ${workflowId} not found`)
            }
            const index = workflow.edges.findIndex((e) => e.id === edgeId)
            if (index !== -1) {
                workflow.edges.splice(index, 1)
            }
            return Promise.resolve()
        }
        return request<void>(
            apiClient.delete(`/api/v1/workflows/${workflowId}/edges/${edgeId}`)
        )
    },

    /**
     * Get available edge types
     */
    async getEdgeTypes(): Promise<EdgeTypeMetadata[]> {
        if (USE_MOCK) {
            return Promise.resolve([...mockEdgeTypes])
        }
        return request<EdgeTypeMetadata[]>(apiClient.get('/api/v1/edge-types'))
    },
}
