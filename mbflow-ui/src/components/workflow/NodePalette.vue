<script setup lang="ts">
import { ref } from "vue";
import { Icon } from "@iconify/vue";
import { NodeType, NODE_TYPE_METADATA, DEFAULT_NODE_CONFIGS } from "@/types/nodes";

export interface NodeTemplate {
  type: string;
  label: string;
  icon: string;
  description: string;
  defaultConfig?: Record<string, any>;
  color: string;
  disabled?: boolean;
}

// Create node templates from metadata
const nodeTemplates: NodeTemplate[] = [
  {
    type: NodeType.HTTP,
    label: NODE_TYPE_METADATA[NodeType.HTTP].label,
    icon: "heroicons:globe-alt",
    description: NODE_TYPE_METADATA[NodeType.HTTP].description,
    color: NODE_TYPE_METADATA[NodeType.HTTP].color,
    defaultConfig: DEFAULT_NODE_CONFIGS[NodeType.HTTP],
  },
  {
    type: NodeType.LLM,
    label: NODE_TYPE_METADATA[NodeType.LLM].label,
    icon: "heroicons:sparkles",
    description: NODE_TYPE_METADATA[NodeType.LLM].description,
    color: NODE_TYPE_METADATA[NodeType.LLM].color,
    defaultConfig: DEFAULT_NODE_CONFIGS[NodeType.LLM],
  },
  {
    type: NodeType.TRANSFORM,
    label: NODE_TYPE_METADATA[NodeType.TRANSFORM].label,
    icon: "heroicons:arrow-path",
    description: NODE_TYPE_METADATA[NodeType.TRANSFORM].description,
    color: NODE_TYPE_METADATA[NodeType.TRANSFORM].color,
    defaultConfig: DEFAULT_NODE_CONFIGS[NodeType.TRANSFORM],
  },
  {
    type: NodeType.FUNCTION_CALL,
    label: NODE_TYPE_METADATA[NodeType.FUNCTION_CALL].label,
    icon: "heroicons:command-line",
    description: NODE_TYPE_METADATA[NodeType.FUNCTION_CALL].description,
    color: NODE_TYPE_METADATA[NodeType.FUNCTION_CALL].color,
    defaultConfig: DEFAULT_NODE_CONFIGS[NodeType.FUNCTION_CALL],
  },
  // Disabled nodes (kept for future use)
  // {
  //   type: "conditional",
  //   label: "Conditional",
  //   icon: "heroicons:code-bracket",
  //   description: "Branch workflow based on conditions (Coming soon)",
  //   color: "#6B7280",
  //   disabled: true,
  // },
  // {
  //   type: "merge",
  //   label: "Merge",
  //   icon: "heroicons:arrows-pointing-in",
  //   description: "Merge results from multiple nodes (Coming soon)",
  //   color: "#6B7280",
  //   disabled: true,
  // },
];

const draggedType = ref<string | null>(null);

function onDragStart(event: DragEvent, nodeType: string) {
  if (!event.dataTransfer) return;

  draggedType.value = nodeType;
  event.dataTransfer.effectAllowed = "copy";
  event.dataTransfer.setData("application/reactflow", nodeType);
  event.dataTransfer.setData("text/plain", nodeType);
}

function onDragEnd() {
  draggedType.value = null;
}
</script>

<template>
  <div
    class="node-palette w-64 overflow-y-auto border-l border-gray-200 bg-white"
  >
    <div class="border-b border-gray-200 p-4">
      <h3 class="text-sm font-semibold text-gray-900">Node Palette</h3>
      <p class="mt-1 text-xs text-gray-500">
        Drag and drop nodes to the canvas
      </p>
    </div>

    <div class="space-y-2 p-3">
      <div
        v-for="template in nodeTemplates"
        :key="template.type"
        :draggable="true"
        :class="[
          'node-template',
          'cursor-move rounded-lg border-2 border-gray-200 p-3',
          'hover:border-blue-400 hover:bg-blue-50',
          'transition-colors',
          { 'opacity-50': draggedType === template.type },
        ]"
        @dragstart="onDragStart($event, template.type)"
        @dragend="onDragEnd"
      >
        <div class="flex items-start gap-3">
          <Icon
            :icon="template.icon"
            class="mt-0.5 size-5 shrink-0 text-gray-700"
          />
          <div class="min-w-0 flex-1">
            <div class="text-sm font-medium text-gray-900">
              {{ template.label }}
            </div>
            <div class="mt-1 text-xs text-gray-500">
              {{ template.description }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Helper text -->
    <div class="border-t border-gray-200 bg-gray-50 p-4">
      <div class="space-y-2 text-xs text-gray-600">
        <p class="font-medium">How to use:</p>
        <ul class="list-inside list-disc space-y-1 text-gray-500">
          <li>Drag nodes onto canvas</li>
          <li>Connect nodes by dragging handles</li>
          <li>Click nodes to configure</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.node-palette {
  height: 100%;
}

.node-template {
  user-select: none;
}

.node-template:active {
  cursor: grabbing;
}
</style>
