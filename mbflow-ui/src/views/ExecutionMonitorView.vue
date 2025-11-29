<template>
  <v-container fluid class="fill-height pa-0">
    <v-row no-gutters class="fill-height">
      <!-- Toolbar -->
      <v-col cols="12">
        <v-toolbar color="surface" elevation="1">
          <v-btn icon="mdi-arrow-left" @click="goBack"></v-btn>

          <v-toolbar-title>
            Execution: {{ executionId }}
            <v-chip
              :color="getStatusColor(currentExecution?.status)"
              size="small"
              class="ml-2"
            >
              {{ currentExecution?.status }}
            </v-chip>
          </v-toolbar-title>

          <v-spacer></v-spacer>

          <v-btn
            variant="text"
            prepend-icon="mdi-refresh"
            @click="refreshExecution"
          >
            Refresh
          </v-btn>
        </v-toolbar>
      </v-col>

      <!-- Main Content -->
      <v-col cols="12" style="height: calc(100vh - 64px)">
        <div class="monitor-container">
          <!-- Canvas (Left/Center) -->
          <div class="canvas-container">
            <WorkflowCanvas
              v-if="workflow"
              :workflow-nodes="workflow.nodes"
              :workflow-edges="workflow.edges"
              :execution-status="nodeStatuses"
              :fit-view-trigger="fitViewTrigger"
              @node-selected="onNodeSelected"
            />
            <div v-else class="d-flex justify-center align-center fill-height">
              <v-progress-circular indeterminate color="primary"></v-progress-circular>
            </div>
          </div>

          <!-- Details Panel (Right) -->
          <div class="details-container">
            <v-card v-if="selectedNode" elevation="0" border class="fill-height">
              <v-card-title class="text-subtitle-1">
                Node Details: {{ selectedNode.name }}
              </v-card-title>
              
              <v-card-text>
                <div class="mb-4">
                  <div class="text-caption text-grey">Status</div>
                  <v-chip
                    :color="getStatusColor(nodeStatuses[selectedNode.id])"
                    size="small"
                  >
                    {{ nodeStatuses[selectedNode.id] || 'pending' }}
                  </v-chip>
                </div>

                <v-divider class="mb-4"></v-divider>

                <div class="text-subtitle-2 mb-2">Logs / Output</div>
                <div class="logs-viewer pa-2 bg-grey-lighten-4 rounded text-caption font-mono">
                  No logs available for this node.
                </div>
              </v-card-text>
            </v-card>
            
            <v-card v-else elevation="0" border class="fill-height d-flex align-center justify-center">
              <div class="text-center text-grey">
                <v-icon icon="mdi-information-outline" size="large" class="mb-2"></v-icon>
                <div>Select a node to view details</div>
              </div>
            </v-card>
          </div>
        </div>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useExecutionStore } from '@/stores/execution.store'
import { useWorkflowStore } from '@/stores/workflow.store'
import WorkflowCanvas from '@/components/workflow/WorkflowCanvas.vue'
import type { Node, NodeStatus } from '@/types'
import { NodeStatuses, ExecutionPhases } from '@/types'

const route = useRoute()
const router = useRouter()
const executionStore = useExecutionStore()
const workflowStore = useWorkflowStore()

const { currentExecution } = storeToRefs(executionStore)
const workflow = ref<any>(null) // Using any for now to avoid strict type checks with store
const selectedNode = ref<Node | null>(null)
const fitViewTrigger = ref(0)
const executionId = route.params.id as string

let pollInterval: any = null

const nodeStatuses = computed(() => {
  if (!currentExecution.value?.node_states) return {}
  
  const statuses: Record<string, NodeStatus> = {}
  Object.values(currentExecution.value.node_states).forEach((state) => {
    statuses[state.node_id] = state.status
  })
  return statuses
})

onMounted(async () => {
  if (executionId) {
    await loadData()
    // Start polling
    pollInterval = setInterval(loadData, 2000)
  }
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})

async function loadData() {
  await executionStore.fetchExecution(executionId)
  
  if (currentExecution.value && (!workflow.value || workflow.value.id !== currentExecution.value.workflow_id)) {
    // Load workflow definition if not loaded or different
    // Note: In a real app, we might fetch the specific version used in execution
    await workflowStore.fetchWorkflow(currentExecution.value.workflow_id)
    workflow.value = workflowStore.currentWorkflow
    
    // Fit view on first load
    if (fitViewTrigger.value === 0) {
      setTimeout(() => {
        fitViewTrigger.value++
      }, 100)
    }
  }
}

function goBack() {
  router.push('/executions')
}

function refreshExecution() {
  loadData()
}

function onNodeSelected(node: Node | null) {
  selectedNode.value = node
}

function getStatusColor(status: string | undefined): string {
  switch (status) {
    case NodeStatuses.COMPLETED:
    case ExecutionPhases.COMPLETED:
      return 'success'
    case NodeStatuses.FAILED:
    case ExecutionPhases.FAILED:
      return 'error'
    case NodeStatuses.RUNNING:
    case ExecutionPhases.EXECUTING:
      return 'info'
    case NodeStatuses.SKIPPED:
      return 'warning'
    default:
      return 'grey'
  }
}
</script>

<style scoped>
.monitor-container {
  display: flex;
  height: 100%;
}

.canvas-container {
  flex: 1;
  position: relative;
}

.details-container {
  width: 320px;
  border-left: 1px solid #e0e0e0;
  background: white;
}

.logs-viewer {
  height: 200px;
  overflow-y: auto;
  white-space: pre-wrap;
}
</style>
