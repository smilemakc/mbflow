/**
 * Workflow Store - manages workflow state
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Workflow, Node, Edge } from '@/types'
import { workflowsApi, nodesApi, edgesApi } from '@/api'

export const useWorkflowStore = defineStore('workflow', () => {
    // State
    const currentWorkflow = ref<Workflow | null>(null)
    const workflows = ref<Workflow[]>([])
    const loading = ref(false)
    const error = ref<string | null>(null)

    // History for undo/redo
    const history = ref<Workflow[]>([])
    const historyIndex = ref(-1)
    const MAX_HISTORY = 50

    // Computed
    const canUndo = computed(() => historyIndex.value > 0)
    const canRedo = computed(() => historyIndex.value < history.value.length - 1)

    // Actions
    async function fetchWorkflows() {
        loading.value = true
        error.value = null
        try {
            workflows.value = await workflowsApi.listWorkflows()
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to fetch workflows'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function fetchWorkflow(id: string) {
        loading.value = true
        error.value = null
        try {
            currentWorkflow.value = await workflowsApi.getWorkflow(id)
            resetHistory()
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to fetch workflow'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function createWorkflow(data: Omit<Workflow, 'id' | 'created_at'>) {
        loading.value = true
        error.value = null
        try {
            const workflow = await workflowsApi.createWorkflow(data)
            workflows.value.push(workflow)
            return workflow
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to create workflow'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function updateWorkflow(id: string, data: Partial<Workflow>) {
        loading.value = true
        error.value = null
        try {
            // Preserve IDs when updating to maintain entity identity
            const updateData: any = { ...data }

            if (data.nodes) {
                updateData.nodes = data.nodes.map(n => ({
                    id: n.id,  // Include ID to preserve node identity
                    type: n.type,
                    name: n.name,
                    config: n.config
                }))
            }

            if (data.edges) {
                updateData.edges = data.edges.map(e => ({
                    id: e.id,  // Include ID to preserve edge identity
                    from: e.from,
                    to: e.to,
                    type: e.type,
                    condition: e.condition,
                    config: e.config
                }))
            }

            if (data.triggers) {
                updateData.triggers = data.triggers.map(t => ({
                    id: t.id,  // Include ID to preserve trigger identity
                    type: t.type,
                    config: t.config
                }))
            }

            const workflow = await workflowsApi.updateWorkflow(id, updateData)
            const index = workflows.value.findIndex((w) => w.id === id)
            if (index !== -1) {
                workflows.value[index] = workflow
            }
            if (currentWorkflow.value?.id === id) {
                currentWorkflow.value = workflow
            }
            return workflow
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to update workflow'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function deleteWorkflow(id: string) {
        loading.value = true
        error.value = null
        try {
            await workflowsApi.deleteWorkflow(id)
            workflows.value = workflows.value.filter((w) => w.id !== id)
            if (currentWorkflow.value?.id === id) {
                currentWorkflow.value = null
            }
        } catch (e) {
            error.value = e instanceof Error ? e.message : 'Failed to delete workflow'
            throw e
        } finally {
            loading.value = false
        }
    }

    // Node operations
    function addNode(node: Node) {
        if (!currentWorkflow.value) return

        currentWorkflow.value.nodes.push(node)
        pushToHistory()
    }

    function updateNode(nodeId: string, updates: Partial<Node>) {
        if (!currentWorkflow.value) return

        const node = currentWorkflow.value.nodes.find((n) => n.id === nodeId)
        if (node) {
            Object.assign(node, updates)
            pushToHistory()
        }
    }

    function removeNode(nodeId: string) {
        if (!currentWorkflow.value) return

        currentWorkflow.value.nodes = currentWorkflow.value.nodes.filter(
            (n) => n.id !== nodeId
        )
        // Also remove edges connected to this node
        currentWorkflow.value.edges = currentWorkflow.value.edges.filter(
            (e) => e.from !== nodeId && e.to !== nodeId
        )
        pushToHistory()
    }

    // Edge operations
    function addEdge(edge: Edge) {
        if (!currentWorkflow.value) return

        currentWorkflow.value.edges.push(edge)
        pushToHistory()
    }

    function updateEdge(edgeId: string, updates: Partial<Edge>) {
        if (!currentWorkflow.value) return

        const edge = currentWorkflow.value.edges.find((e) => e.id === edgeId)
        if (edge) {
            Object.assign(edge, updates)
            pushToHistory()
        }
    }

    function removeEdge(edgeId: string) {
        if (!currentWorkflow.value) return

        currentWorkflow.value.edges = currentWorkflow.value.edges.filter(
            (e) => e.id !== edgeId
        )
        pushToHistory()
    }

    // History management
    function pushToHistory() {
        if (!currentWorkflow.value) return

        // Remove any history after current index
        history.value = history.value.slice(0, historyIndex.value + 1)

        // Add current state
        history.value.push(structuredClone(currentWorkflow.value))

        // Limit history size
        if (history.value.length > MAX_HISTORY) {
            history.value.shift()
        } else {
            historyIndex.value++
        }
    }

    function resetHistory() {
        if (!currentWorkflow.value) return

        history.value = [structuredClone(currentWorkflow.value)]
        historyIndex.value = 0
    }

    function undo() {
        if (!canUndo.value) return

        historyIndex.value--
        currentWorkflow.value = structuredClone(history.value[historyIndex.value])
    }

    function redo() {
        if (!canRedo.value) return

        historyIndex.value++
        currentWorkflow.value = structuredClone(history.value[historyIndex.value])
    }

    return {
        // State
        currentWorkflow,
        workflows,
        loading,
        error,

        // Computed
        canUndo,
        canRedo,

        // Actions
        fetchWorkflows,
        fetchWorkflow,
        createWorkflow,
        updateWorkflow,
        deleteWorkflow,
        addNode,
        updateNode,
        removeNode,
        addEdge,
        updateEdge,
        removeEdge,
        undo,
        redo,
    }
})
