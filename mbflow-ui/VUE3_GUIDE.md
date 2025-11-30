# Vue 3 Frontend Development Guide

Complete guide for building the mbflow frontend with Vue 3, TypeScript, and Vue Flow.

## Tech Stack

- **Framework:** Vue 3.4+ (Composition API)
- **Language:** TypeScript 5.3+
- **DAG Visualization:** Vue Flow 1.33+
- **State Management:** Pinia 2.1+
- **API Client:** TanStack Query (Vue) 5.0+
- **Routing:** Vue Router 4.2+
- **Styling:** TailwindCSS 3.4+
- **Icons:** lucide-vue-next
- **Charts:** vue-chartjs 5.3+
- **Forms:** VeeValidate 4.12+

## Project Setup

```bash
# Install dependencies
npm install

# Install Vue Flow
npm install @vue-flow/core @vue-flow/background @vue-flow/controls @vue-flow/minimap

# Install TanStack Query
npm install @tanstack/vue-query

# Install additional dependencies
npm install axios lucide-vue-next
npm install vue-chartjs chart.js
npm install vee-validate yup

# Install dev dependencies
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

## Project Structure

```
src/
├── components/
│   ├── WorkflowEditor/
│   │   ├── WorkflowCanvas.vue
│   │   ├── NodePanel.vue
│   │   ├── NodeConfigPanel.vue
│   │   ├── Toolbar.vue
│   │   └── nodes/
│   │       ├── HTTPNode.vue
│   │       ├── LLMNode.vue
│   │       ├── DataAdapterNode.vue
│   │       ├── ConditionalNode.vue
│   │       └── MergeNode.vue
│   ├── ExecutionMonitor/
│   │   ├── ExecutionStatus.vue
│   │   ├── ExecutionTimeline.vue
│   │   ├── NodeLogs.vue
│   │   └── ProgressBar.vue
│   ├── Dashboard/
│   │   ├── DashboardStats.vue
│   │   ├── ExecutionChart.vue
│   │   ├── ExecutorUsage.vue
│   │   └── RecentExecutions.vue
│   ├── TriggerManager/
│   │   ├── TriggerList.vue
│   │   ├── CronTriggerForm.vue
│   │   └── WebhookTrigger.vue
│   └── common/
│       ├── Button.vue
│       ├── Input.vue
│       ├── Select.vue
│       └── Modal.vue
├── composables/
│   ├── useWorkflow.ts
│   ├── useExecution.ts
│   ├── useTrigger.ts
│   └── useWebSocket.ts
├── stores/
│   ├── workflow.ts
│   ├── execution.ts
│   └── ui.ts
├── api/
│   ├── client.ts
│   ├── workflow.ts
│   ├── execution.ts
│   └── trigger.ts
├── types/
│   ├── workflow.ts
│   ├── execution.ts
│   ├── node.ts
│   └── api.ts
├── router/
│   └── index.ts
├── views/
│   ├── WorkflowEditor.vue
│   ├── ExecutionMonitor.vue
│   ├── Dashboard.vue
│   └── TriggerManager.vue
├── App.vue
└── main.ts
```

## Core Components

### 1. Workflow Canvas (Vue Flow)

```vue
<script setup lang="ts">
import { ref, watch } from 'vue'
import { VueFlow, useVueFlow, type Node, type Edge } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { useWorkflowStore } from '@/stores/workflow'
import HTTPNode from './nodes/HTTPNode.vue'
import LLMNode from './nodes/LLMNode.vue'
import DataAdapterNode from './nodes/DataAdapterNode.vue'

const workflowStore = useWorkflowStore()

const nodes = ref<Node[]>([])
const edges = ref<Edge[]>([])

const nodeTypes = {
  HTTP: HTTPNode,
  LLM: LLMNode,
  DataAdapter: DataAdapterNode,
}

const { onConnect, addEdges, addNodes, project } = useVueFlow()

// Handle edge connections
onConnect((params) => {
  addEdges([params])
  workflowStore.addEdge(params)
})

// Handle node drop from panel
const onDrop = (event: DragEvent) => {
  event.preventDefault()
  
  const type = event.dataTransfer?.getData('application/node-type')
  if (!type) return
  
  const position = project({
    x: event.clientX,
    y: event.clientY,
  })
  
  const newNode = {
    id: `${type}-${Date.now()}`,
    type,
    position,
    data: {
      label: type,
      config: {},
    },
  }
  
  addNodes([newNode])
  workflowStore.addNode(newNode)
}

// Sync with store
watch(() => workflowStore.currentWorkflow, (workflow) => {
  if (workflow) {
    nodes.value = workflow.nodes
    edges.value = workflow.edges
  }
}, { deep: true })
</script>

<template>
  <div class="workflow-canvas h-screen">
    <VueFlow
      v-model:nodes="nodes"
      v-model:edges="edges"
      :node-types="nodeTypes"
      @drop="onDrop"
      @dragover.prevent
      fit-view-on-init
    >
      <Background pattern-color="#aaa" :gap="16" />
      <Controls />
      <MiniMap />
    </VueFlow>
  </div>
</template>

<style>
@import '@vue-flow/core/dist/style.css';
@import '@vue-flow/core/dist/theme-default.css';
</style>
```

### 2. Custom Node Component

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { Settings } from 'lucide-vue-next'

interface Props {
  id: string
  data: {
    label: string
    config: Record<string, any>
  }
}

const props = defineProps<Props>()
const emit = defineEmits(['configure'])

const hasConfig = computed(() => {
  return Object.keys(props.data.config).length > 0
})
</script>

<template>
  <div class="custom-node bg-white border-2 border-gray-300 rounded-lg p-4 shadow-lg">
    <Handle type="target" :position="Position.Top" />
    
    <div class="flex items-center justify-between mb-2">
      <h3 class="font-semibold text-sm">{{ data.label }}</h3>
      <button
        @click="emit('configure', id)"
        class="p-1 hover:bg-gray-100 rounded"
      >
        <Settings :size="16" />
      </button>
    </div>
    
    <div v-if="hasConfig" class="text-xs text-gray-600">
      <div v-if="data.config.url">
        URL: {{ data.config.url }}
      </div>
      <div v-if="data.config.prompt">
        Prompt configured
      </div>
    </div>
    
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<style scoped>
.custom-node {
  min-width: 150px;
}
</style>
```

### 3. Composable: useWorkflow

```typescript
// composables/useWorkflow.ts
import { ref } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import * as workflowApi from '@/api/workflow'
import type { Workflow, CreateWorkflowRequest } from '@/types/workflow'

export function useWorkflow(workflowId?: string) {
  const queryClient = useQueryClient()

  // Fetch workflow
  const { data: workflow, isLoading, error } = useQuery({
    queryKey: ['workflow', workflowId],
    queryFn: () => workflowApi.getWorkflow(workflowId!),
    enabled: !!workflowId,
  })

  // Create workflow
  const createMutation = useMutation({
    mutationFn: (data: CreateWorkflowRequest) => workflowApi.createWorkflow(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workflows'] })
    },
  })

  // Update workflow
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Workflow> }) =>
      workflowApi.updateWorkflow(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workflow', workflowId] })
    },
  })

  // Delete workflow
  const deleteMutation = useMutation({
    mutationFn: (id: string) => workflowApi.deleteWorkflow(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workflows'] })
    },
  })

  return {
    workflow,
    isLoading,
    error,
    createWorkflow: createMutation.mutate,
    updateWorkflow: updateMutation.mutate,
    deleteWorkflow: deleteMutation.mutate,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
  }
}
```

### 4. Pinia Store

```typescript
// stores/workflow.ts
import { defineStore } from 'pinia'
import type { Node, Edge } from '@vue-flow/core'
import type { Workflow } from '@/types/workflow'

export const useWorkflowStore = defineStore('workflow', {
  state: () => ({
    currentWorkflow: null as Workflow | null,
    selectedNodeId: null as string | null,
  }),

  getters: {
    nodes: (state) => state.currentWorkflow?.nodes || [],
    edges: (state) => state.currentWorkflow?.edges || [],
    selectedNode: (state) => {
      if (!state.selectedNodeId || !state.currentWorkflow) return null
      return state.currentWorkflow.nodes.find(n => n.id === state.selectedNodeId)
    },
  },

  actions: {
    setWorkflow(workflow: Workflow) {
      this.currentWorkflow = workflow
    },

    addNode(node: Node) {
      if (!this.currentWorkflow) return
      this.currentWorkflow.nodes.push(node)
    },

    updateNode(nodeId: string, updates: Partial<Node>) {
      if (!this.currentWorkflow) return
      const node = this.currentWorkflow.nodes.find(n => n.id === nodeId)
      if (node) {
        Object.assign(node, updates)
      }
    },

    removeNode(nodeId: string) {
      if (!this.currentWorkflow) return
      this.currentWorkflow.nodes = this.currentWorkflow.nodes.filter(n => n.id !== nodeId)
      this.currentWorkflow.edges = this.currentWorkflow.edges.filter(
        e => e.source !== nodeId && e.target !== nodeId
      )
    },

    addEdge(edge: Edge) {
      if (!this.currentWorkflow) return
      this.currentWorkflow.edges.push(edge)
    },

    selectNode(nodeId: string | null) {
      this.selectedNodeId = nodeId
    },
  },
})
```

### 5. API Client

```typescript
// api/client.ts
import axios from 'axios'

const client = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor
client.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('api_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor
client.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      // Handle unauthorized
      localStorage.removeItem('api_token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default client
```

### 6. WebSocket Composable

```typescript
// composables/useWebSocket.ts
import { ref, onMounted, onUnmounted } from 'vue'

export function useWebSocket<T = any>(url: string) {
  const data = ref<T | null>(null)
  const error = ref<Error | null>(null)
  const isConnected = ref(false)
  
  let ws: WebSocket | null = null
  let reconnectTimeout: NodeJS.Timeout | null = null

  const connect = () => {
    ws = new WebSocket(url)

    ws.onopen = () => {
      isConnected.value = true
      error.value = null
    }

    ws.onmessage = (event) => {
      try {
        data.value = JSON.parse(event.data)
      } catch (e) {
        error.value = new Error('Failed to parse WebSocket message')
      }
    }

    ws.onerror = (event) => {
      error.value = new Error('WebSocket error')
      isConnected.value = false
    }

    ws.onclose = () => {
      isConnected.value = false
      // Reconnect after 3 seconds
      reconnectTimeout = setTimeout(connect, 3000)
    }
  }

  const disconnect = () => {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout)
    }
    ws?.close()
  }

  const send = (message: any) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message))
    }
  }

  onMounted(connect)
  onUnmounted(disconnect)

  return {
    data,
    error,
    isConnected,
    send,
    disconnect,
  }
}
```

## Styling with TailwindCSS

```typescript
// tailwind.config.js
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          100: '#dbeafe',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
        },
      },
    },
  },
  plugins: [],
}
```

## Testing

### Component Tests (Vitest)

```typescript
// components/__tests__/WorkflowCanvas.spec.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import WorkflowCanvas from '../WorkflowEditor/WorkflowCanvas.vue'

describe('WorkflowCanvas', () => {
  it('renders canvas', () => {
    const wrapper = mount(WorkflowCanvas)
    expect(wrapper.find('.workflow-canvas').exists()).toBe(true)
  })

  it('adds node on drop', async () => {
    const wrapper = mount(WorkflowCanvas)
    const dropEvent = new DragEvent('drop', {
      dataTransfer: new DataTransfer()
    })
    dropEvent.dataTransfer?.setData('application/node-type', 'HTTP')
    
    await wrapper.trigger('drop', dropEvent)
    
    // Assert node was added
  })
})
```

## Best Practices

### 1. Use Composition API

✅ **Good:**
```vue
<script setup lang="ts">
import { ref, computed } from 'vue'

const count = ref(0)
const doubled = computed(() => count.value * 2)
</script>
```

❌ **Bad:**
```vue
<script>
export default {
  data() {
    return { count: 0 }
  },
  computed: {
    doubled() { return this.count * 2 }
  }
}
</script>
```

### 2. Type Everything

```typescript
// Define types
interface SomeType {
  id: string
  type: string
  position: { x: number; y: number }
  data: Record<string, any>
}

// Use in components
const someTypes = ref<SomeType[]>([])
```

### 3. Extract Composables

```typescript
// Reusable logic
export function useExecutionMonitor(executionId: string) {
  const { data, refetch } = useQuery(...)
  const { data: logs } = useQuery(...)
  
  return { execution: data, logs, refetch }
}
```

### 4. Proper Cleanup

```vue
<script setup lang="ts">
import { onUnmounted } from 'vue'

const ws = new WebSocket('...')

onUnmounted(() => {
  ws.close()
})
</script>
```

## Performance Optimization

### 1. Lazy Loading

```typescript
// router/index.ts
const routes = [
  {
    path: '/workflows/:id',
    component: () => import('@/views/WorkflowEditor.vue')
  }
]
```

### 2. Virtual Scrolling

For large node lists, use virtual scrolling:

```vue
<script setup lang="ts">
import { useVirtualList } from '@vueuse/core'

const { list, containerProps, wrapperProps } = useVirtualList(
  nodes,
  { itemHeight: 50 }
)
</script>
```

### 3. Computed Caching

```typescript
const expensiveOperation = computed(() => {
  // Heavy computation cached until dependencies change
  return heavyProcess(data.value)
})
```

## Deployment

```bash
# Build for production
npm run build

# Preview production build
npm run preview

# Type check
npm run type-check

# Lint
npm run lint
```

---

This guide provides everything needed to build a production-ready Vue 3 frontend for mbflow!
