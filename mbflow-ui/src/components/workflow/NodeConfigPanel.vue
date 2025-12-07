<template>
  <div
    :class="[
      'node-config-panel',
      'fixed right-0 top-0 h-full border-l border-gray-200 bg-white shadow-xl',
      'z-50 transition-transform duration-300 ease-in-out',
      isOpen ? 'translate-x-0' : 'translate-x-full',
    ]"
    style="width: 450px"
  >
    <div v-if="selectedNode" class="flex h-full flex-col">
      <!-- Header -->
      <div
        class="flex items-center justify-between border-b border-gray-200 p-4"
      >
        <div class="flex items-center gap-2">
          <Icon icon="heroicons:cog-6-tooth" class="size-5 text-gray-700" />
          <h3 class="text-lg font-semibold text-gray-900">
            Node Configuration
          </h3>
        </div>
        <button
          class="rounded p-1 transition-colors hover:bg-gray-100"
          @click="closePanel"
        >
          <Icon icon="heroicons:x-mark" class="size-5 text-gray-500" />
        </button>
      </div>

      <!-- Form -->
      <div class="flex-1 space-y-4 overflow-y-auto p-4">
        <!-- Node type badge -->
        <div class="flex items-center gap-2">
          <span
            :class="[
              'rounded px-2 py-1 text-xs font-semibold',
              nodeTypeBadgeClass,
            ]"
          >
            {{ nodeTypeLabel }}
          </span>
        </div>

        <!-- Node ID -->
        <div class="space-y-1">
          <Input
            v-model="nodeId"
            label="Node ID"
            type="text"
            required
            placeholder="e.g., http, llm_2"
          />
          <p v-if="nodeIdError" class="text-xs text-red-600">
            {{ nodeIdError }}
          </p>
          <p v-else class="text-xs text-gray-500">
            Only letters (a-Z) and underscores (_) allowed
          </p>
        </div>

        <!-- Node name -->
        <Input
          v-model="nodeName"
          label="Node Name"
          type="text"
          required
          placeholder="Enter node name"
        />

        <!-- Node-specific configuration -->
        <div class="config-section">
          <component
            :is="currentConfigComponent"
            v-if="currentConfigComponent"
            :config="nodeConfig"
            :node-id="selectedNode.id"
            @update:config="updateNodeConfig"
          />
          <div v-else class="text-sm text-gray-500">
            No configuration available for this node type
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="space-y-2 border-t border-gray-200 p-4">
        <div class="flex gap-2">
          <Button variant="primary" class="flex-1" @click="saveConfig">
            Save Changes
          </Button>
          <Button variant="secondary" @click="closePanel"> Cancel </Button>
        </div>
        <Button variant="danger" class="w-full" @click="deleteNode">
          Delete Node
        </Button>
      </div>
    </div>
  </div>

  <!-- Backdrop -->
  <div
    v-if="isOpen"
    class="fixed inset-0 z-40 bg-black/20 transition-opacity"
    @click="closePanel"
  />
</template>

<script setup lang="ts">
import { computed, watch, ref, markRaw, type Component } from "vue";
import { Icon } from "@iconify/vue";
import { toast } from "vue3-toastify";
import { useWorkflowStore } from "@/stores/workflow";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import { validateNodeId, isNodeIdUnique } from "@/utils/nodeId";
import {
  NodeType,
  NODE_TYPE_METADATA,
  DEFAULT_NODE_CONFIGS,
} from "@/types/nodes";
import type { NodeConfig } from "@/types/nodes";

// Import node config components
import HTTPNodeConfig from "@/components/nodes/config/HTTPNodeConfig.vue";
import LLMNodeConfig from "@/components/nodes/config/LLMNodeConfig.vue";
import TransformNodeConfig from "@/components/nodes/config/TransformNodeConfig.vue";
import FunctionCallNodeConfig from "@/components/nodes/config/FunctionCallNodeConfig.vue";
import TelegramNodeConfig from "@/components/nodes/config/TelegramNodeConfig.vue";
import FileStorageNodeConfig from "@/components/nodes/config/FileStorageNodeConfig.vue";
import ConditionalNodeConfig from "@/components/nodes/config/ConditionalNodeConfig.vue";
import MergeNodeConfig from "@/components/nodes/config/MergeNodeConfig.vue";

const workflowStore = useWorkflowStore();
const emit = defineEmits(["save"]);

const selectedNode = computed(() => workflowStore.selectedNode);
const isOpen = computed(() => !!selectedNode.value);

// Local form state
const nodeId = ref<string>("");
const nodeName = ref<string>("");
const nodeConfig = ref<NodeConfig>({} as NodeConfig);
const nodeIdError = ref<string>("");

// Node type metadata
const nodeTypeLabel = computed(() => {
  const type = selectedNode.value?.type as NodeType;
  return NODE_TYPE_METADATA[type]?.label || type?.toUpperCase() || "UNKNOWN";
});

const nodeTypeBadgeClass = computed(() => {
  const type = selectedNode.value?.type as NodeType;
  const color = NODE_TYPE_METADATA[type]?.color || "#6B7280";

  // Map hex colors to Tailwind classes
  const colorMap: Record<string, string> = {
    "#10B981": "bg-green-100 text-green-700",
    "#8B5CF6": "bg-purple-100 text-purple-700",
    "#F59E0B": "bg-amber-100 text-amber-700",
    "#3B82F6": "bg-blue-100 text-blue-700",
    "#0EA5E9": "bg-sky-100 text-sky-700",
    "#14B8A6": "bg-teal-100 text-teal-700",
    "#EC4899": "bg-pink-100 text-pink-700",
    "#A855F7": "bg-violet-100 text-violet-700",
  };

  return colorMap[color] || "bg-gray-100 text-gray-700";
});

// Map node types to config components
const configComponentMap: Record<NodeType, Component> = {
  [NodeType.HTTP]: markRaw(HTTPNodeConfig),
  [NodeType.LLM]: markRaw(LLMNodeConfig),
  [NodeType.TRANSFORM]: markRaw(TransformNodeConfig),
  [NodeType.FUNCTION_CALL]: markRaw(FunctionCallNodeConfig),
  [NodeType.TELEGRAM]: markRaw(TelegramNodeConfig),
  [NodeType.FILE_STORAGE]: markRaw(FileStorageNodeConfig),
  [NodeType.CONDITIONAL]: markRaw(ConditionalNodeConfig),
  [NodeType.MERGE]: markRaw(MergeNodeConfig),
};

const currentConfigComponent = computed(() => {
  const type = selectedNode.value?.type as NodeType;
  return configComponentMap[type] || null;
});

// Watch for node selection changes
watch(
  selectedNode,
  (node) => {
    if (node) {
      nodeId.value = node.id;
      nodeName.value = node.data?.label || "";

      // Load config with defaults
      const type = node.type as NodeType;
      const defaults = DEFAULT_NODE_CONFIGS[type] || {};
      nodeConfig.value = { ...defaults, ...(node.data?.config || {}) };

      nodeIdError.value = "";
    } else {
      nodeId.value = "";
      nodeName.value = "";
      nodeConfig.value = {} as NodeConfig;
      nodeIdError.value = "";
    }
  },
  { immediate: true },
);

// Validate node ID on change
watch(nodeId, (newId) => {
  if (!newId) {
    nodeIdError.value = "Node ID is required";
    return;
  }

  if (!validateNodeId(newId)) {
    nodeIdError.value = "Only letters (a-Z) and underscores (_) are allowed";
    return;
  }

  const existingIds = workflowStore.nodes.map((n) => n.id);
  if (!isNodeIdUnique(newId, existingIds, selectedNode.value?.id)) {
    nodeIdError.value = "This ID is already in use";
    return;
  }

  nodeIdError.value = "";
});

function updateNodeConfig(updatedConfig: NodeConfig) {
  nodeConfig.value = updatedConfig;
}

function closePanel() {
  workflowStore.selectNode(null);
}

function saveConfig() {
  if (!selectedNode.value) return;

  // Check if node ID has validation errors
  if (nodeIdError.value) {
    toast.error("Please fix the node ID error before saving");
    return;
  }

  const oldNodeId = selectedNode.value.id;
  const newNodeId = nodeId.value;

  // Update node with new ID if changed
  if (oldNodeId !== newNodeId) {
    workflowStore.updateNodeId(oldNodeId, newNodeId);
  }

  // Update node data
  workflowStore.updateNode(newNodeId, {
    data: {
      ...selectedNode.value.data,
      label: nodeName.value,
      config: nodeConfig.value,
    },
  });

  // Emit save event to trigger workflow persistence
  emit("save");

  toast.success("Node configuration saved successfully");
  closePanel();
}

function deleteNode() {
  if (!selectedNode.value) return;

  if (confirm("Are you sure you want to delete this node?")) {
    workflowStore.removeNode(selectedNode.value.id);
  }
}
</script>

<style scoped>
.node-config-panel {
  max-width: 100vw;
}

.config-section {
  padding-top: 8px;
}
</style>
