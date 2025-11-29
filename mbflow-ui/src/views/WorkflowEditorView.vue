<template>
  <v-container fluid class="fill-height pa-0">
    <v-row no-gutters class="fill-height">
      <!-- Toolbar -->
      <v-col cols="12">
        <v-toolbar color="surface" elevation="1">
          <v-btn icon="mdi-arrow-left" @click="goBack"></v-btn>

          <v-toolbar-title v-if="currentWorkflow">
            {{ currentWorkflow.name }}
          </v-toolbar-title>

          <v-spacer></v-spacer>

          <v-btn
            variant="text"
            prepend-icon="mdi-undo"
            :disabled="!canUndo"
            @click="undo"
          >
            Undo
          </v-btn>

          <v-btn
            variant="text"
            prepend-icon="mdi-redo"
            :disabled="!canRedo"
            @click="redo"
          >
            Redo
          </v-btn>

          <v-divider vertical class="mx-2"></v-divider>

          <v-btn
            variant="text"
            prepend-icon="mdi-fit-to-screen"
            @click="fitView"
          >
            Fit View
          </v-btn>

          <v-btn
            variant="text"
            prepend-icon="mdi-graph"
            @click="autoLayout"
          >
            Auto Layout
          </v-btn>

          <v-btn
            color="primary"
            prepend-icon="mdi-content-save"
            @click="saveWorkflow"
          >
            Save
          </v-btn>

          <v-btn
            color="success"
            prepend-icon="mdi-play"
            class="ml-2"
            @click="runWorkflow"
          >
            Run
          </v-btn>
        </v-toolbar>
      </v-col>

      <!-- Main Content -->
      <v-col cols="12" style="height: calc(100vh - 128px)">
        <div class="editor-container">
          <!-- Left Panel: Palette & Variables -->
          <div class="left-panel">
            <v-tabs v-model="leftTab" density="compact" grow>
              <v-tab value="palette">Palette</v-tab>
              <v-tab value="variables">Variables</v-tab>
            </v-tabs>
            
            <v-window v-model="leftTab" class="fill-height">
              <v-window-item value="palette" class="fill-height">
                <NodePalette />
              </v-window-item>
              <v-window-item value="variables" class="fill-height pa-2">
                <VariableExplorer
                  :selected-node="selectedNode"
                  :nodes="currentWorkflow?.nodes || []"
                  :edges="currentWorkflow?.edges || []"
                />
              </v-window-item>
            </v-window>
          </div>

          <!-- Workflow Canvas (Center) -->
          <div class="canvas-container">
            <VueFlowProvider>
              <WorkflowCanvas
                v-if="currentWorkflow"
                :workflow-nodes="currentWorkflow.nodes"
                :workflow-edges="currentWorkflow.edges"
                :fit-view-trigger="fitViewTrigger"
                @update:nodes="onNodesUpdate"
                @update:edges="onEdgesUpdate"
                @node-selected="onNodeSelected"
                @edge-selected="onEdgeSelected"
              />
            </VueFlowProvider>
            <div v-if="!currentWorkflow" class="empty-state">
              <v-icon icon="mdi-sitemap" size="64" color="grey-lighten-1"></v-icon>
              <h3 class="text-h6 mt-4">No workflow loaded</h3>
            </div>
          </div>

          <!-- Property Panel (Right) -->
          <div class="properties-container">
            <v-card v-if="selectedNode" elevation="2">
              <v-card-title class="text-h6">
                <v-icon icon="mdi-cog" class="mr-2"></v-icon>
                Node Properties
              </v-card-title>
              <v-card-text>
                <v-text-field
                  :model-value="selectedNode.name"
                  label="Name"
                  variant="outlined"
                  density="compact"
                  @update:model-value="onNodeNameChange"
                ></v-text-field>

                <v-select
                  :model-value="selectedNode.type"
                  label="Type"
                  :items="nodeTypes"
                  variant="outlined"
                  density="compact"
                  disabled
                ></v-select>

                <v-divider class="my-4"></v-divider>

                <h4 class="text-subtitle-2 mb-2">Configuration</h4>
                
                <!-- Specialized Config Forms -->
                <TransformConfigForm
                  v-if="selectedNode.type === NodeTypes.TRANSFORM"
                  :model-value="selectedNode.config || { transformations: {} }"
                  @update:model-value="updateNodeConfigObject"
                />
                
                <HttpConfigForm
                  v-else-if="selectedNode.type === NodeTypes.HTTP"
                  :model-value="selectedNode.config || { url: '', method: 'GET' }"
                  @update:model-value="updateNodeConfigObject"
                />
                
                <!-- Generic JSON Editor for other nodes -->
                <v-textarea
                  v-else
                  :model-value="JSON.stringify(selectedNode.config || {}, null, 2)"
                  label="Config (JSON)"
                  variant="outlined"
                  rows="10"
                  @update:model-value="updateNodeConfigJson"
                ></v-textarea>
              </v-card-text>
              <v-card-actions>
                <v-btn
                  color="error"
                  variant="text"
                  prepend-icon="mdi-delete"
                  @click="deleteSelectedNode"
                >
                  Delete Node
                </v-btn>
              </v-card-actions>
            </v-card>

            <v-card v-else-if="selectedEdge" elevation="2">
              <v-card-title class="text-h6">
                <v-icon icon="mdi-arrow-right" class="mr-2"></v-icon>
                Edge Properties
              </v-card-title>
              <v-card-text>
                <v-select
                  v-model="selectedEdge.type"
                  label="Type"
                  :items="edgeTypes"
                  variant="outlined"
                  density="compact"
                  @update:model-value="updateSelectedEdge"
                ></v-select>

                <v-text-field
                  v-if="selectedEdge.type === 'conditional'"
                  v-model="selectedEdge.condition.expression"
                  label="Condition Expression"
                  variant="outlined"
                  density="compact"
                  @update:model-value="updateSelectedEdge"
                ></v-text-field>
              </v-card-text>
              <v-card-actions>
                <v-btn
                  color="error"
                  variant="text"
                  prepend-icon="mdi-delete"
                  @click="deleteSelectedEdge"
                >
                  Delete Edge
                </v-btn>
              </v-card-actions>
            </v-card>

            <v-card v-else elevation="2">
              <v-card-text class="text-center pa-8">
                <v-icon icon="mdi-cursor-default-click" size="48" color="grey-lighten-1"></v-icon>
                <p class="text-body-2 text-grey mt-4">
                  Select a node or edge to view properties
                </p>
              </v-card-text>
            </v-card>
          </div>
        </div>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useWorkflowStore } from '@/stores/workflow.store'
import { useExecutionStore } from '@/stores/execution.store'
import WorkflowCanvas from '@/components/workflow/WorkflowCanvas.vue'
import NodePalette from '@/components/workflow/NodePalette.vue'
import VariableExplorer from '@/components/workflow/VariableExplorer.vue'
import TransformConfigForm from '@/components/properties/TransformConfigForm.vue'
import HttpConfigForm from '@/components/properties/HttpConfigForm.vue'
import { getLayoutedElements } from '@/utils/layout'
import type { Node, Edge } from '@/types'
import { NodeTypes, EdgeTypes } from '@/types'

const route = useRoute()
const router = useRouter()
const workflowStore = useWorkflowStore()
const executionStore = useExecutionStore()

const { currentWorkflow, canUndo, canRedo } = storeToRefs(workflowStore)

const selectedNode = ref<Node | null>(null)
const selectedEdge = ref<Edge | null>(null)
const fitViewTrigger = ref(0)
const leftTab = ref('palette')

const nodeTypes = Object.values(NodeTypes)
const edgeTypes = Object.values(EdgeTypes)

onMounted(async () => {
  const workflowId = route.params.id as string
  
  if (workflowId === 'new') {
    // Create a new empty workflow
    currentWorkflow.value = {
      id: `workflow-${Date.now()}`,
      name: 'New Workflow',
      version: '1.0.0',
      description: '',
      nodes: [],
      edges: [],
      triggers: [
        {
          id: `trigger-${Date.now()}`,
          type: 'manual',
          config: {}
        }
      ],
      metadata: {}
    }
    
    // Fit view after creating
    setTimeout(() => {
      fitViewTrigger.value++
    }, 100)
  } else if (workflowId) {
    // Load existing workflow
    await workflowStore.fetchWorkflow(workflowId)
    
    // Fit view after loading
    setTimeout(() => {
      fitViewTrigger.value++
    }, 100)
  }
})

function goBack() {
  router.push('/workflows')
}

function undo() {
  workflowStore.undo()
}

function redo() {
  workflowStore.redo()
}

function fitView() {
  fitViewTrigger.value++
}

async function autoLayout() {
  if (!currentWorkflow.value) return
  
  const layoutedNodes = await getLayoutedElements(
    currentWorkflow.value.nodes,
    currentWorkflow.value.edges
  )
  
  currentWorkflow.value.nodes = layoutedNodes
  
  // Fit view after layout
  setTimeout(() => {
    fitView()
  }, 100)
}

function onNodesUpdate(nodes: any[]) {
  if (!currentWorkflow.value) return
  currentWorkflow.value.nodes = nodes as Node[]
}

function onEdgesUpdate(edges: any[]) {
  if (!currentWorkflow.value) return
  currentWorkflow.value.edges = edges as Edge[]
}

function onNodeSelected(node: Node | null) {
  selectedNode.value = node
  selectedEdge.value = null
  if (node) {
    // Switch to variables tab to show context
    leftTab.value = 'variables'
  }
}

function onEdgeSelected(edge: Edge | null) {
  if (edge && edge.type === 'conditional' && !edge.condition) {
    edge.condition = { expression: '' }
  }
  selectedEdge.value = edge
  selectedNode.value = null
}

import { toSnakeCase } from '@/utils/formatting'

function onNodeNameChange(newName: string) {
  if (!selectedNode.value) return
  // Format to snake_case immediately
  selectedNode.value.name = toSnakeCase(newName)
  updateSelectedNode()
}

function updateSelectedNode() {
  if (!selectedNode.value) return
  workflowStore.updateNode(selectedNode.value.id, selectedNode.value)
}

function updateNodeConfigObject(config: any) {
  if (!selectedNode.value) return
  selectedNode.value.config = config
  updateSelectedNode()
}

function updateNodeConfigJson(configJson: string) {
  if (!selectedNode.value) return
  try {
    selectedNode.value.config = JSON.parse(configJson)
    updateSelectedNode()
  } catch (e) {
    console.error('Invalid JSON:', e)
  }
}

function deleteSelectedNode() {
  if (!selectedNode.value) return
  workflowStore.removeNode(selectedNode.value.id)
  selectedNode.value = null
}

function updateSelectedEdge() {
  if (!selectedEdge.value) return
  workflowStore.updateEdge(selectedEdge.value.id, selectedEdge.value)
}

function deleteSelectedEdge() {
  if (!selectedEdge.value) return
  workflowStore.removeEdge(selectedEdge.value.id)
  selectedEdge.value = null
}

async function saveWorkflow() {
  if (!currentWorkflow.value) return

  try {
    const isNew = route.params.id === 'new'
    
    if (isNew) {
      // Create new workflow
      // Remove temporary ID and let backend assign one (or use it if backend supports it, 
      // but store.createWorkflow expects Omit<Workflow, 'id'>)
      const { id, ...workflowData } = currentWorkflow.value
      const createdWorkflow = await workflowStore.createWorkflow(workflowData)
      
      console.log('Workflow created:', createdWorkflow)
      
      // Update route to new ID
      await router.replace({ name: 'workflow-editor', params: { id: createdWorkflow.id } })
    } else {
      // Update existing workflow
      await workflowStore.updateWorkflow(currentWorkflow.value.id, currentWorkflow.value)
      console.log('Workflow saved')
    }
  } catch (e) {
    console.error('Failed to save workflow:', e)
  }
}

async function runWorkflow() {
  if (!currentWorkflow.value) return

  try {
    const execution = await executionStore.executeWorkflow(currentWorkflow.value.id)
    console.log('Workflow execution started', execution)
    router.push(`/executions/${execution.id}`)
  } catch (e) {
    console.error('Failed to execute workflow:', e)
  }
}
</script>

<style scoped>
.editor-container {
  display: flex;
  height: 100%;
  background: #fafafa;
}

.left-panel {
  width: 280px;
  border-right: 1px solid #e0e0e0;
  display: flex;
  flex-direction: column;
  background: white;
}

.palette-container {
  flex: 1;
  overflow-y: auto;
}

.canvas-container {
  flex: 1;
  position: relative;
}

.properties-container {
  width: 320px;
  border-left: 1px solid #e0e0e0;
  overflow-y: auto;
  background: white;
  padding: 16px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #999;
}
</style>
