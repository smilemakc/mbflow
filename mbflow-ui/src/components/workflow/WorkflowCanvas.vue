<template>
  <div class="workflow-canvas" :class="{ 'drag-over': isDragOver }">
    <VueFlow
      id="workflow-editor"
      v-model:nodes="nodes"
      v-model:edges="edges"
      :default-viewport="{ zoom: 1 }"
      :min-zoom="0.2"
      :max-zoom="4"
      :nodes-draggable="true"
      :nodes-connectable="true"
      :elements-selectable="true"
      :snap-to-grid="true"
      :snap-grid="[15, 15]"
      :connection-mode="'loose'"
      :default-edge-options="{ type: 'smoothstep', animated: false }"
      @node-drag-stop="onNodeDragStop"
      @node-click="onNodeClick"
      @edge-click="onEdgeClick"
      @connect="onConnect"
      @dragover.prevent="onDragOver"
      @drop.prevent="onDrop"
      @dragleave="onDragLeave"
    >
      <Background pattern-color="#aaa" :gap="16" />
      <Controls />
      <MiniMap />

      <!-- Custom node templates -->
      <template #node-transform="nodeProps">
        <TransformNode v-bind="nodeProps" />
      </template>

      <template #node-http="nodeProps">
        <HttpNode v-bind="nodeProps" />
      </template>

      <!-- All other node types use CustomNode -->
      <template #node-conditional-router="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-parallel="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-code="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-script-executor="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-http-request="nodeProps">
        <HttpNode v-bind="nodeProps" />
      </template>

      <template #node-telegram-message="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-openai-completion="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-openai-responses="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-llm="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-function-call="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-json-parser="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-data-merger="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <template #node-data-aggregator="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>

      <!-- Fallback for any custom node type -->
      <template #node-custom="nodeProps">
        <CustomNode v-bind="nodeProps" />
      </template>
    </VueFlow>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import type { Node as FlowNode, Edge as FlowEdge, Connection, NodeDragEvent } from '@vue-flow/core'
import type { Node as WorkflowNode, Edge as WorkflowEdge, NodeStatus } from '@/types'
import TransformNode from '@/components/nodes/TransformNode.vue'
import HttpNode from '@/components/nodes/HttpNode.vue'
import CustomNode from '@/components/nodes/CustomNode.vue'
import { toSnakeCase } from '@/utils/formatting'
import { generateRandomName } from '@/utils/name-generator'
import { generateUUID } from '@/utils/uuid'

// Import Vue Flow styles globally for this component
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

interface Props {
  workflowNodes?: WorkflowNode[]
  workflowEdges?: WorkflowEdge[]
  fitViewTrigger?: number
  executionStatus?: Record<string, NodeStatus>
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:nodes': [nodes: FlowNode[]]
  'update:edges': [edges: FlowEdge[]]
  'node-selected': [node: WorkflowNode | null]
  'edge-selected': [edge: WorkflowEdge | null]
}>()

const { fitView, project, addEdges, addNodes } = useVueFlow({ id: 'workflow-editor' })
const isDragOver = ref(false)

// Internal state for Vue Flow
const nodes = ref<FlowNode[]>([])
const edges = ref<FlowEdge[]>([])

// Watch for fit view trigger
watch(
  () => props.fitViewTrigger,
  () => {
    fitView({ padding: 0.2 })
  }
)

// Watch for prop changes to sync DOWN to Vue Flow
// We need to be careful not to overwrite local state during interactions if possible,
// but since we removed onNodesChange, we rely on onNodeDragStop to sync UP.
// So syncing DOWN should be fine as long as it doesn't happen *during* a drag initiated by us.
// However, since we only emit on drag stop, the props won't change during drag, so this watch won't trigger.
watch(
  [() => props.workflowNodes, () => props.executionStatus],
  ([newNodes, newStatus]) => {
    if (newNodes) {
      // Map workflow nodes to Vue Flow nodes
      nodes.value = newNodes.map((node) => ({
        id: node.id,
        type: node.type,
        position: node.metadata?.position as { x: number; y: number } || { x: 100, y: 100 },
        class: newStatus?.[node.id] ? `status-${newStatus[node.id]}` : '',
        data: {
          label: node.name,
          config: node.config,
          metadata: node.metadata,
          status: newStatus?.[node.id],
          ...node,
        },
      }))
    }
  },
  { immediate: true, deep: true }
)

watch(
  () => props.workflowEdges,
  (newEdges) => {
    if (newEdges) {
      edges.value = newEdges.map((edge) => ({
        id: edge.id,
        source: edge.from,
        target: edge.to,
        type: edge.type === 'conditional' ? 'smoothstep' : 'default',
        animated: edge.type === 'fork',
        label: edge.condition?.expression || '',
        data: edge,
      }))
    }
  },
  { immediate: true }
)

// Handlers

function onNodeDragStop(event: NodeDragEvent) {
  // Sync UP to parent
  syncNodesToParent()
}

function syncNodesToParent() {
  const updatedNodes = nodes.value.map((node) => ({
    id: node.id,
    type: node.type as any,
    name: node.data.label,
    config: node.data.config,
    metadata: {
      ...node.data.metadata,
      position: node.position
    }
  })) as any[]
  emit('update:nodes', updatedNodes)
}

function onNodeClick(event: { node: FlowNode }) {
  const node = props.workflowNodes?.find((n) => n.id === event.node.id)
  emit('node-selected', node || null)
}

function onEdgeClick(event: { edge: FlowEdge }) {
  const edge = props.workflowEdges?.find((e) => e.id === event.edge.id)
  emit('edge-selected', edge || null)
}

function onConnect(connection: Connection) {
  // Validate connection
  if (!connection.source || !connection.target) return
  
  // Prevent self-connections
  if (connection.source === connection.target) {
    console.warn('Cannot connect node to itself')
    return
  }

  // Validate that nodes exist in our data model
  const sourceNode = props.workflowNodes?.find(n => n.id === connection.source)
  const targetNode = props.workflowNodes?.find(n => n.id === connection.target)
  
  if (!sourceNode || !targetNode) {
    console.warn('Cannot connect: source or target node not found in workflow data')
    return
  }
  
  // Check if edge already exists
  const edgeExists = edges.value.some(
    (e) => e.source === connection.source && e.target === connection.target
  )
  
  if (edgeExists) {
    console.warn('Edge already exists between these nodes')
    return
  }
  
  // Create new edge locally for immediate feedback
  const newEdgeId = generateUUID()
  const newFlowEdge: FlowEdge = {
    id: newEdgeId,
    source: connection.source,
    target: connection.target,
    type: 'default'
  }
  
  // Add to local state immediately
  addEdges([newFlowEdge])
  
  // Sync UP to parent
  const newWorkflowEdge: WorkflowEdge = {
    id: newEdgeId,
    from: connection.source,
    to: connection.target,
    type: 'direct',
  }

  const updatedEdges = [...(props.workflowEdges || []), newWorkflowEdge]
  emit('update:edges', updatedEdges as any)
}

function onDragOver(event: DragEvent) {
  event.preventDefault()
  
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
  isDragOver.value = true
}

function onDragLeave(event: DragEvent) {
  isDragOver.value = false
}

function onDrop(event: DragEvent) {
  isDragOver.value = false
  
  // Try to get data as JSON first, then fallback to plain text
  let nodeType = ''
  try {
    const jsonData = event.dataTransfer?.getData('application/json')
    if (jsonData) {
      const data = JSON.parse(jsonData)
      nodeType = data.type
    }
  } catch (e) {
    console.warn('Failed to parse JSON drag data:', e)
  }

  if (!nodeType) {
    // Fallback to plain text or custom type
    nodeType = event.dataTransfer?.getData('text/plain') || 
               event.dataTransfer?.getData('application/vueflow-node-type') || ''
  }

  if (!nodeType) return

  // Get canvas bounds
  const wrapper = (event.currentTarget as Element).closest('.vue-flow') || (event.target as Element).closest('.vue-flow')
  
  // If we can't find wrapper, try to use the event target itself if it's the canvas
  const targetElement = wrapper || (event.target as Element)
  const canvasBounds = targetElement.getBoundingClientRect()

  // Calculate position relative to canvas and apply zoom/pan
  const position = project({
    x: event.clientX - canvasBounds.left,
    y: event.clientY - canvasBounds.top,
  })

  // Create new node with unique ID and proper naming
  const nodeId = generateUUID()
  
  // Generate smart name
  const baseName = toSnakeCase(nodeType)
  let name = baseName
  
  // Check if a node of this type already exists
  const typeExists = props.workflowNodes?.some(n => n.type === nodeType)
  
  if (typeExists) {
    // Generate a unique random name
    do {
      name = `${baseName}_${generateRandomName()}`
    } while (props.workflowNodes?.some(n => n.name === name))
  } else {
    // Even if type doesn't exist, check if name is taken (edge case)
    if (props.workflowNodes?.some(n => n.name === name)) {
       do {
        name = `${baseName}_${generateRandomName()}`
      } while (props.workflowNodes?.some(n => n.name === name))
    }
  }

  const newNode: WorkflowNode = {
    id: nodeId,
    type: nodeType as any,
    name: name,
    config: {},
    metadata: {
      position: { 
        x: Math.round(position.x), 
        y: Math.round(position.y) 
      }
    }
  }

  const updatedNodes = [...(props.workflowNodes || []), newNode]
  // Add to local Vue Flow state immediately to prevent "Node not found" errors
  addNodes([{
    id: newNode.id,
    type: newNode.type,
    position: newNode.metadata.position,
    data: {
      label: newNode.name,
      config: newNode.config,
      metadata: newNode.metadata,
      ...newNode
    }
  }])

  // Cast to any to avoid type issues with emit
  emit('update:nodes', updatedNodes as any)
}
</script>

<style>
.workflow-canvas {
  width: 100%;
  height: 100%;
  background: #f5f5f5;
  transition: background-color 0.2s;
}

.workflow-canvas.drag-over {
  background-color: rgba(25, 118, 210, 0.05);
  box-shadow: inset 0 0 0 2px #1976d2;
}

/* Status Styles */
:deep(.vue-flow__node) {
  transition: box-shadow 0.3s ease;
  cursor: grab;
}

:deep(.vue-flow__node.selected) {
  box-shadow: 0 0 0 2px #1976d2;
}

:deep(.vue-flow__node.status-completed) {
  border-color: #4caf50 !important;
  box-shadow: 0 0 8px rgba(76, 175, 80, 0.6);
}

:deep(.vue-flow__node.status-failed) {
  border-color: #f44336 !important;
  box-shadow: 0 0 8px rgba(244, 67, 54, 0.6);
}

:deep(.vue-flow__node.status-running) {
  border-color: #2196f3 !important;
  box-shadow: 0 0 12px rgba(33, 150, 243, 0.6);
  animation: pulse 2s infinite;
}

:deep(.vue-flow__node.status-skipped) {
  opacity: 0.6;
  border-style: dashed;
}

/* Connection handles */
:deep(.vue-flow__handle) {
  width: 12px;
  height: 12px;
  border: 2px solid #1976d2;
  background: white;
  transition: all 0.2s ease;
}

:deep(.vue-flow__handle:hover) {
  width: 16px;
  height: 16px;
  background: #1976d2;
  box-shadow: 0 0 8px rgba(25, 118, 210, 0.5);
}

:deep(.vue-flow__handle-connecting) {
  background: #4caf50;
  border-color: #4caf50;
}

:deep(.vue-flow__handle-valid) {
  background: #4caf50;
  border-color: #4caf50;
}

/* Edges */
:deep(.vue-flow__edge) {
  cursor: pointer;
}

:deep(.vue-flow__edge:hover .vue-flow__edge-path) {
  stroke-width: 3;
}

:deep(.vue-flow__edge.selected .vue-flow__edge-path) {
  stroke: #1976d2;
  stroke-width: 3;
}

/* Connection line */
:deep(.vue-flow__connection-path) {
  stroke: #1976d2;
  stroke-width: 2;
  stroke-dasharray: 5, 5;
  animation: dash 0.5s linear infinite;
}

@keyframes dash {
  to {
    stroke-dashoffset: -10;
  }
}

@keyframes pulse {
  0% {
    box-shadow: 0 0 0 0 rgba(33, 150, 243, 0.7);
  }
  70% {
    box-shadow: 0 0 0 10px rgba(33, 150, 243, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(33, 150, 243, 0);
  }
}
</style>
