<script setup lang="ts">
// @ts-nocheck
import { ref, watch, nextTick } from "vue";
import {
  VueFlow,
  useVueFlow,
  type Connection,
  PanOnScrollMode,
} from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import { MiniMap } from "@vue-flow/minimap";
import { useWorkflowStore } from "@/stores/workflow";
import {
  useAutoLayout,
  type LayoutAlgorithm,
} from "@/composables/useAutoLayout";

interface Props {
  readonly?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  readonly: false,
});

const emit = defineEmits<{
  nodeClick: [nodeId: string];
  edgeClick: [edgeId: string];
}>();

const workflowStore = useWorkflowStore();
const { applyLayout } = useAutoLayout();
const vueFlowInstance = useVueFlow();
const { addEdges: _addEdges, setNodes, setEdges, fitView } = vueFlowInstance;

// Layout state
const layoutAlgorithm = ref<LayoutAlgorithm>("dagre");
const isLayouting = ref(false);

// Handle node click
function onNodeClick(event: any) {
  const nodeId = event.node?.id;
  if (nodeId) {
    // Don't automatically open config panel
    // Config panel now opens via settings button in node
    emit("nodeClick", nodeId);
  }
}

// Handle edge click
function onEdgeClick(event: any) {
  const edgeId = event.edge?.id;
  if (edgeId) {
    emit("edgeClick", edgeId);
  }
}

// Handle connection creation
function onConnect(connection: Connection) {
  if (props.readonly) return;

  const edge = {
    id: `e${connection.source}-${connection.target}`,
    source: connection.source,
    target: connection.target,
    sourceHandle: connection.sourceHandle,
    targetHandle: connection.targetHandle,
  };

  workflowStore.addEdge(edge);
}

// Handle node drag stop
function onNodeDragStop(event: any) {
  if (props.readonly) return;

  const { node } = event;
  if (node) {
    workflowStore.updateNode(node.id, {
      position: node.position,
    });
  }
}

// Apply auto-layout
async function triggerAutoLayout() {
  isLayouting.value = true;

  try {
    const { nodes: layoutedNodes } = await applyLayout(
      workflowStore.nodes,
      workflowStore.edges,
      layoutAlgorithm.value,
      {
        direction: "TB",
        spacing: { node: 80, rank: 150 },
      },
    );

    workflowStore.updateNodePositions(layoutedNodes);

    // Wait for DOM update and then fit view
    await nextTick();
    setTimeout(() => {
      fitView({ padding: 0.2, duration: 300 });
    }, 100);
  } catch (error) {
    console.error("Auto-layout failed:", error);
  } finally {
    isLayouting.value = false;
  }
}

// Watch for nodes/edges changes and update Vue Flow
watch(
  () => [workflowStore.nodes, workflowStore.edges],
  ([newNodes, newEdges]) => {
    setNodes(newNodes);
    setEdges(newEdges);
  },
  { deep: true, immediate: true },
);

// Expose methods
defineExpose({
  triggerAutoLayout,
  layoutAlgorithm,
  vueFlowInstance,
});
</script>

<template>
  <div class="workflow-canvas relative size-full">
    <VueFlow
      :nodes="workflowStore.nodes"
      :edges="workflowStore.edges"
      :class="{ 'pointer-events-none': readonly }"
      :min-zoom="0.2"
      :max-zoom="4"
      :zoom-on-scroll="true"
      :zoom-on-pinch="true"
      :pan-on-scroll="true"
      :pan-on-scroll-mode="PanOnScrollMode.Free"
      :pan-on-drag="[1, 2]"
      fit-view-on-init
      @node-click="onNodeClick"
      @edge-click="onEdgeClick"
      @connect="onConnect"
      @node-drag-stop="onNodeDragStop"
    >
      <Background pattern-color="#aaa" :gap="16" />
      <Controls />
      <MiniMap />

      <!-- Loading overlay -->
      <div
        v-if="isLayouting"
        class="absolute inset-0 z-50 flex items-center justify-center bg-white/50"
      >
        <div class="text-center">
          <div
            class="mx-auto size-12 animate-spin rounded-full border-b-2 border-blue-600"
          />
          <p class="mt-2 text-sm text-gray-600">Applying layout...</p>
        </div>
      </div>
    </VueFlow>
  </div>
</template>

<style>
/* Import Vue Flow base styles */
@import "@vue-flow/core/dist/style.css";
@import "@vue-flow/core/dist/theme-default.css";
@import "@vue-flow/controls/dist/style.css";
@import "@vue-flow/minimap/dist/style.css";

.workflow-canvas {
  background-color: #f9fafb;
}

/* Custom node styles */
.vue-flow__node {
  border-radius: 0.5rem;
  border: 2px solid #e5e7eb;
  background: white;
  padding: 0;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
  transition: all 0.2s;
}

.vue-flow__node:hover {
  border-color: #3b82f6;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.vue-flow__node.selected {
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
}

/* Custom edge styles */
.vue-flow__edge-path {
  stroke: #9ca3af;
  stroke-width: 2;
}

.vue-flow__edge.selected .vue-flow__edge-path {
  stroke: #2563eb;
  stroke-width: 3;
}

.vue-flow__edge:hover .vue-flow__edge-path {
  stroke: #3b82f6;
}

/* Selection box */
.vue-flow__selection {
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid #3b82f6;
}
</style>
