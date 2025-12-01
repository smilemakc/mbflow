<script setup lang="ts">
// @ts-nocheck
import { ref, onMounted, computed, markRaw } from "vue";
import { useRoute, useRouter } from "vue-router";
import { VueFlow } from "@vue-flow/core";
import { useWorkflowStore } from "@/stores/workflow";
import {
  getWorkflow,
  updateWorkflow,
  validateWorkflow,
  executeWorkflow,
} from "@/api/workflows";
import WorkflowToolbar from "@/components/workflow/WorkflowToolbar.vue";
import WorkflowCanvas from "@/components/workflow/WorkflowCanvas.vue";
import NodePalette from "@/components/workflow/NodePalette.vue";
import NodeConfigPanel from "@/components/workflow/NodeConfigPanel.vue";
import HTTPNode from "@/components/workflow/nodes/HTTPNode.vue";
import LLMNode from "@/components/workflow/nodes/LLMNode.vue";
import TransformNode from "@/components/workflow/nodes/TransformNode.vue";
import ConditionalNode from "@/components/workflow/nodes/ConditionalNode.vue";
import MergeNode from "@/components/workflow/nodes/MergeNode.vue";

const route = useRoute();
const router = useRouter();
const workflowStore = useWorkflowStore();

const workflowId = computed(() => route.params.id as string);
const canvasRef = ref<InstanceType<typeof WorkflowCanvas> | null>(null);

const isLoading = ref(true);
const isSaving = ref(false);
const isExecuting = ref(false);
const error = ref<string | null>(null);

// Register custom node types with markRaw to avoid reactivity overhead
const nodeTypes = {
  http: markRaw(HTTPNode),
  llm: markRaw(LLMNode),
  transform: markRaw(TransformNode),
  conditional: markRaw(ConditionalNode),
  merge: markRaw(MergeNode),
};

// Load workflow on mount
onMounted(async () => {
  await loadWorkflow();
});

async function loadWorkflow() {
  isLoading.value = true;
  error.value = null;

  try {
    const response = await getWorkflow(workflowId.value);
    workflowStore.loadWorkflow(response);
  } catch (err: any) {
    console.error("Failed to load workflow:", err);
    error.value = err.message || "Failed to load workflow";
  } finally {
    isLoading.value = false;
  }
}

async function handleSave() {
  isSaving.value = true;

  try {
    const workflowData = workflowStore.toBackendFormat();
    console.log("Saving workflow:", workflowData);
    await updateWorkflow(workflowId.value, workflowData);
    workflowStore.isDirty = false;
    console.log("Workflow saved successfully");
  } catch (err: any) {
    console.error("Failed to save workflow:", err);
    alert("Failed to save workflow: " + (err.message || "Unknown error"));
  } finally {
    isSaving.value = false;
  }
}

async function handleExecute() {
  isExecuting.value = true;

  try {
    const result = await executeWorkflow(workflowId.value);
    alert(`Workflow execution started! Execution ID: ${result.execution_id}`);
    router.push(`/executions/${result.execution_id}`);
  } catch (err: any) {
    console.error("Failed to execute workflow:", err);
    alert("Failed to execute workflow: " + (err.message || "Unknown error"));
  } finally {
    isExecuting.value = false;
  }
}

async function handleValidate() {
  try {
    const result = await validateWorkflow(workflowId.value);
    if (result.valid) {
      alert("Workflow is valid!");
    } else {
      alert("Workflow validation failed:\n" + result.errors?.join("\n"));
    }
  } catch (err: any) {
    console.error("Failed to validate workflow:", err);
    alert("Failed to validate workflow: " + (err.message || "Unknown error"));
  }
}

function handleAutoLayout() {
  canvasRef.value?.triggerAutoLayout();
}

function handleBack() {
  if (workflowStore.isDirty) {
    if (confirm("You have unsaved changes. Are you sure you want to leave?")) {
      router.push("/workflows");
    }
  } else {
    router.push("/workflows");
  }
}

function handleNodeClick(nodeId: string) {
  console.log("Node clicked:", nodeId);
}

function handleEdgeClick(edgeId: string) {
  console.log("Edge clicked:", edgeId);
}

// Handle drop from palette
function onDrop(event: DragEvent) {
  event.preventDefault();

  if (!event.dataTransfer) return;

  const nodeType = event.dataTransfer.getData("application/reactflow");
  if (!nodeType) return;

  // Get Vue Flow instance to convert screen coordinates to flow coordinates
  const vueFlowInstance = canvasRef.value?.vueFlowInstance;
  if (!vueFlowInstance) return;

  // Convert screen coordinates to flow coordinates
  const position = vueFlowInstance.project({
    x: event.clientX,
    y: event.clientY,
  });

  // Create new node
  const newNode = {
    id: `node_${Date.now()}`,
    type: nodeType,
    position,
    data: {
      label: `${nodeType.toUpperCase()} Node`,
      config: {},
    },
  };

  workflowStore.addNode(newNode);
}

function onDragOver(event: DragEvent) {
  event.preventDefault();
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = "copy";
  }
}
</script>

<template>
  <div class="workflow-editor flex h-screen flex-col bg-gray-50">
    <!-- Toolbar -->
    <WorkflowToolbar
      :loading="isExecuting"
      :saving="isSaving"
      @save="handleSave"
      @execute="handleExecute"
      @auto-layout="handleAutoLayout"
      @validate="handleValidate"
      @back="handleBack"
    />

    <!-- Main content -->
    <div class="flex flex-1 overflow-hidden">
      <!-- Loading state -->
      <div v-if="isLoading" class="flex flex-1 items-center justify-center">
        <div class="text-center">
          <div
            class="mx-auto size-12 animate-spin rounded-full border-b-2 border-blue-600"
          />
          <p class="mt-4 text-gray-600">Loading workflow...</p>
        </div>
      </div>

      <!-- Error state -->
      <div v-else-if="error" class="flex flex-1 items-center justify-center">
        <div class="text-center">
          <p class="mb-4 text-red-600">{{ error }}</p>
          <button
            class="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
            @click="loadWorkflow"
          >
            Retry
          </button>
        </div>
      </div>

      <!-- Canvas and palette -->
      <template v-else>
        <!-- Canvas -->
        <div class="relative flex-1" @drop="onDrop" @dragover="onDragOver">
          <VueFlow :node-types="nodeTypes">
            <WorkflowCanvas
              ref="canvasRef"
              @node-click="handleNodeClick"
              @edge-click="handleEdgeClick"
            />
          </VueFlow>
        </div>

        <!-- Node Palette -->
        <NodePalette />

        <!-- Node Config Panel -->
        <NodeConfigPanel />
      </template>
    </div>
  </div>
</template>

<style>
.workflow-editor {
  --vf-node-bg: #fff;
  --vf-node-text: #222;
  --vf-connection-path: #b1b1b7;
  --vf-handle: #555;
}
</style>
