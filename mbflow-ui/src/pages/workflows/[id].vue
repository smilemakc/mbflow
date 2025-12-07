```
<script setup lang="ts">
// @ts-nocheck
import { ref, onMounted, computed, markRaw, provide } from "vue";
import { useRoute, useRouter } from "vue-router";
import { VueFlow } from "@vue-flow/core";
import { Icon } from "@iconify/vue";
import { toast } from "vue3-toastify";
import { useWorkflowStore } from "@/stores/workflow";
import {
  getWorkflow,
  updateWorkflow,
  validateWorkflow,
  executeWorkflow,
  type ExecuteWorkflowOptions,
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
import TelegramNode from "@/components/workflow/nodes/TelegramNode.vue";
import ExecuteWorkflowDialog from "@/components/workflow/ExecuteWorkflowDialog.vue";
import WorkflowVariablesPanel from "@/components/workflow/WorkflowVariablesPanel.vue";
import ExecutionStatusPanel from "@/components/workflow/ExecutionStatusPanel.vue";
import { generateNodeId } from "@/utils/nodeId";
import { getTemplateById } from "@/data/templates";
import { executionObserver } from "@/services/executionObserver";
import { useWorkflowExecutionAnimation } from "@/composables/useWorkflowExecutionAnimation";

const route = useRoute();
const router = useRouter();
const workflowStore = useWorkflowStore();

const workflowId = computed(() => route.params.id as string);
const canvasRef = ref<InstanceType<typeof WorkflowCanvas> | null>(null);
const executeDialogRef = ref<InstanceType<typeof ExecuteWorkflowDialog> | null>(
  null,
);

const isLoading = ref(true);
const isSaving = ref(false);
const isExecuting = ref(false);
const error = ref<string | null>(null);
const showExecuteDialog = ref(false);
const showVariablesPanel = ref(false);
const showExecutionStatus = ref(false);
const currentExecutionId = ref<string | null>(null);

// Execution animation
const { handleExecutionEvent, resetAnimations } =
  useWorkflowExecutionAnimation();

// Provide function to open node config from nodes
function openNodeConfig(nodeId: string) {
  workflowStore.selectNode(nodeId);
}

provide("openNodeConfig", openNodeConfig);

// Register custom node types with markRaw to avoid reactivity overhead
const nodeTypes = {
  http: markRaw(HTTPNode),
  llm: markRaw(LLMNode),
  transform: markRaw(TransformNode),
  conditional: markRaw(ConditionalNode),
  merge: markRaw(MergeNode),
  telegram: markRaw(TelegramNode),
};

// Load workflow on mount
onMounted(async () => {
  await loadWorkflow();

  // Subscribe to execution observer events for animation
  watchExecutionEvents();
});

async function loadWorkflow() {
  isLoading.value = true;
  error.value = null;

  try {
    const response = await getWorkflow(workflowId.value);
    workflowStore.loadWorkflow(response);

    // Check if we should load a template
    const templateId = route.query.template as string;
    if (templateId && (!response.nodes || response.nodes.length === 0)) {
      const template = getTemplateById(templateId);
      if (template) {
        console.log("Loading template:", template.name);
        // Load template nodes and edges into the workflow
        workflowStore.nodes = template.nodes;
        workflowStore.edges = template.edges;
        workflowStore.isDirty = true; // Mark as dirty so user can save
        toast.success(
          `Template "${template.name}" loaded! Click Save to persist changes.`,
        );
      }
    }
  } catch (err: any) {
    console.error("Failed to load workflow:", err);
    error.value = err.message || "Failed to load workflow";
  } finally {
    isLoading.value = false;
  }
}

/**
 * Watch execution events from observer and animate
 */
let lastProcessedEventIndex = -1;

function watchExecutionEvents() {
  // Check for new events periodically
  setInterval(() => {
    if (!currentExecutionId.value) return;

    const events = executionObserver.events.get(currentExecutionId.value);
    if (!events || events.length === 0) return;

    // Process all new events since last check
    for (let i = lastProcessedEventIndex + 1; i < events.length; i++) {
      const event = events[i];
      console.log(
        "[WorkflowEditor] Processing event:",
        event.event?.event_type,
        event.event?.node_id,
      );
      handleExecutionEvent(event);
    }

    // Update last processed index
    lastProcessedEventIndex = events.length - 1;
  }, 100); // Check every 100ms
}

// Reset event tracking when starting new execution
function resetEventTracking() {
  lastProcessedEventIndex = -1;
  resetAnimations();
}

async function handleSave() {
  isSaving.value = true;

  try {
    const workflowData = workflowStore.toBackendFormat();
    console.log("Saving workflow:", workflowData);
    await updateWorkflow(workflowId.value, workflowData);
    workflowStore.isDirty = false;
    console.log("Workflow saved successfully");
    toast.success("Workflow saved successfully");
  } catch (err: any) {
    console.error("Failed to save workflow:", err);
    toast.error("Failed to save workflow: " + (err.message || "Unknown error"));
  } finally {
    isSaving.value = false;
  }
}

function handleExecute() {
  // Show execute dialog instead of executing directly
  showExecuteDialog.value = true;
}

async function handleExecuteWithOptions(options: ExecuteWorkflowOptions) {
  try {
    const result = await executeWorkflow(workflowId.value, options);
    showExecuteDialog.value = false;

    // Reset event tracking and animations for new execution
    resetEventTracking();

    // Store current execution ID for status tracking
    currentExecutionId.value = result.id;
    showExecutionStatus.value = true;

    toast.success(`Workflow execution started! Execution ID: ${result.id}`);

    // Don't navigate immediately - let user watch the execution
    // They can click on the execution ID to go to details page
  } catch (err: any) {
    console.error("Failed to execute workflow:", err);
    toast.error(
      "Failed to execute workflow: " + (err.message || "Unknown error"),
    );
    // Reset executing state in dialog so user can retry
    executeDialogRef.value?.resetExecuting();
  }
}

function handleOpenVariablesPanel() {
  showExecuteDialog.value = false;
  showVariablesPanel.value = true;
}

function handleExecutionComplete() {
  toast.success("Workflow execution completed successfully!");
}

function handleExecutionError(error: string) {
  toast.error(`Workflow execution failed: ${error}`);
}

function handleExecutionStatusChange(status: string) {
  console.log("Execution status changed:", status);
}

function handleViewExecutionDetails() {
  if (currentExecutionId.value) {
    router.push(`/executions/${currentExecutionId.value}`);
  }
}

async function handleValidate() {
  try {
    const result = await validateWorkflow(workflowId.value);
    if (result.valid) {
      toast.success("Workflow is valid!");
    } else {
      toast.error("Workflow validation failed:\n" + result.errors?.join("\n"));
    }
  } catch (err: any) {
    console.error("Failed to validate workflow:", err);
    toast.error(
      "Failed to validate workflow: " + (err.message || "Unknown error"),
    );
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

  // Generate unique node ID
  const existingIds = workflowStore.nodes.map((n) => n.id);
  const nodeId = generateNodeId(nodeType, existingIds);

  // Create new node
  const newNode = {
    id: nodeId,
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
      @open-variables="showVariablesPanel = true"
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
        <NodeConfigPanel @save="handleSave" />

        <!-- Execution Status Panel (floating) -->
        <div
          v-if="showExecutionStatus && currentExecutionId"
          class="absolute bottom-4 right-4 z-50 w-96"
        >
          <div class="relative">
            <button
              class="absolute -right-2 -top-2 rounded-full bg-white p-1 shadow-md hover:bg-gray-100"
              @click="showExecutionStatus = false"
            >
              <Icon icon="heroicons:x-mark" class="size-4 text-gray-600" />
            </button>
            <ExecutionStatusPanel
              :execution-id="currentExecutionId"
              @complete="handleExecutionComplete"
              @error="handleExecutionError"
              @status-change="handleExecutionStatusChange"
            />
            <button
              class="mt-2 w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
              @click="handleViewExecutionDetails"
            >
              View Full Details
            </button>
          </div>
        </div>
      </template>
    </div>

    <!-- Execute Workflow Dialog -->
    <ExecuteWorkflowDialog
      ref="executeDialogRef"
      :is-open="showExecuteDialog"
      @close="showExecuteDialog = false"
      @execute="handleExecuteWithOptions"
      @open-variables-panel="handleOpenVariablesPanel"
    />

    <!-- Workflow Variables Panel -->
    <WorkflowVariablesPanel
      :is-open="showVariablesPanel"
      @close="showVariablesPanel = false"
    />
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
