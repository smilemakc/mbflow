// @ts-nocheck
import {defineStore} from "pinia";
import {computed, ref} from "vue";
import type {Edge as VueFlowEdge, Node as VueFlowNode} from "@vue-flow/core";
import type {Edge, Node, Workflow} from "@/types/workflow";

export const useWorkflowStore = defineStore("workflow", () => {
    // State
    const currentWorkflow = ref<Workflow | null>(null);
    const nodes = ref<VueFlowNode[]>([]);
    const edges = ref<VueFlowEdge[]>([]);
    const selectedNodeId = ref<string | null>(null);
    const isDirty = ref(false);

    // Computed
    const selectedNode = computed(() => {
        if (!selectedNodeId.value) return null;
        return nodes.value.find((n) => n.id === selectedNodeId.value);
    });

    const nodeCount = computed(() => nodes.value.length);
    const edgeCount = computed(() => edges.value.length);

    // Actions

    /**
     * Load workflow into store
     */
    function loadWorkflow(workflow: Workflow) {
        currentWorkflow.value = workflow;

        // Convert backend nodes to Vue Flow nodes
        nodes.value = (workflow.nodes || []).map((node: Node) => ({
            id: node.id,
            type: node.type,
            position: node.position || {x: 0, y: 0},
            data: {
                label: node.name,
                config: node.config,
                metadata: node.metadata,
            },
            dimensions: {width: 200, height: 100},
        }));

        // Convert backend edges to Vue Flow edges
        edges.value = workflow.edges.map((edge: Edge) => ({
            id: edge.id,
            source: edge.from,
            target: edge.to,
            sourceHandle: edge.source_handle,
            targetHandle: edge.target_handle,
            data: {
                condition: edge.condition,
            },
            type: edge.condition ? "conditional" : "default",
        }));

        isDirty.value = false;
    }

    /**
     * Clear workflow from store
     */
    function clearWorkflow() {
        currentWorkflow.value = null;
        nodes.value = [];
        edges.value = [];
        selectedNodeId.value = null;
        isDirty.value = false;
    }

    /**
     * Add node to workflow
     */
    function addNode(node: VueFlowNode) {
        nodes.value.push(node);
        isDirty.value = true;
    }

    /**
     * Update node in workflow
     */
    function updateNode(nodeId: string, updates: Partial<VueFlowNode>) {
        const index = nodes.value.findIndex((n) => n.id === nodeId);
        if (index !== -1) {
            nodes.value[index] = {...nodes.value[index], ...updates};
            isDirty.value = true;
        }
    }

    /**
     * Remove node from workflow
     */
    function removeNode(nodeId: string) {
        nodes.value = nodes.value.filter((n) => n.id !== nodeId);
        // Remove connected edges
        edges.value = edges.value.filter(
            (e) => e.source !== nodeId && e.target !== nodeId,
        );
        if (selectedNodeId.value === nodeId) {
            selectedNodeId.value = null;
        }
        isDirty.value = true;
    }

    /**
     * Add edge to workflow
     */
    function addEdge(edge: VueFlowEdge) {
        // Check if edge already exists
        const exists = edges.value.some(
            (e) => e.source === edge.source && e.target === edge.target,
        );
        if (!exists) {
            edges.value.push(edge);
            isDirty.value = true;
        }
    }

    /**
     * Remove edge from workflow
     */
    function removeEdge(edgeId: string) {
        edges.value = edges.value.filter((e) => e.id !== edgeId);
        isDirty.value = true;
    }

    /**
     * Select node
     */
    function selectNode(nodeId: string | null) {
        selectedNodeId.value = nodeId;
    }

    /**
     * Update node positions after layout
     */
    function updateNodePositions(layoutedNodes: VueFlowNode[]) {
        layoutedNodes.forEach((layoutedNode) => {
            const index = nodes.value.findIndex((n) => n.id === layoutedNode.id);
            if (index !== -1) {
                nodes.value[index].position = layoutedNode.position;
            }
        });
        isDirty.value = true;
    }

    /**
     * Convert store state to backend format
     */
    function toBackendFormat(): Partial<Workflow> {
        if (!currentWorkflow.value) {
            throw new Error("No workflow loaded");
        }

        const edges1 = edges.value.map((edge) => ({
            id: edge.id,
            from: edge.source,
            to: edge.target,
            source_handle: edge.sourceHandle,
            target_handle: edge.targetHandle,
            condition: edge.data?.condition,
        }));

        const nodes1 = nodes.value.map((node) => ({
            id: node.id,
            name: node.data.label,
            type: node.type || "http",
            config: node.data.config || {},
            position: node.position,
            metadata: node.data.metadata || {},
        }));

        return {
            id: currentWorkflow.value.id,
            name: currentWorkflow.value.name,
            description: currentWorkflow.value.description,
            nodes: nodes1,
            edges: edges1,
            status: currentWorkflow.value.status,
            metadata: currentWorkflow.value.metadata,
        };
    }

    return {
        // State
        currentWorkflow,
        nodes,
        edges,
        selectedNodeId,
        isDirty,

        // Computed
        selectedNode,
        nodeCount,
        edgeCount,

        // Actions
        loadWorkflow,
        clearWorkflow,
        addNode,
        updateNode,
        removeNode,
        addEdge,
        removeEdge,
        selectNode,
        updateNodePositions,
        toBackendFormat,
    };
});
