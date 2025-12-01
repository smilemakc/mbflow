<script setup lang="ts">
// @ts-nocheck
import { computed } from "vue";
import { Handle, Position } from "@vue-flow/core";
import { Icon } from "@iconify/vue";

interface Props {
  data: {
    label: string;
    icon?: string;
    config?: Record<string, any>;
    metadata?: Record<string, any>;
  };
  selected?: boolean;
  type?: string;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

// Get node color based on type
const nodeColor = computed(() => {
  const colorMap: Record<string, string> = {
    http: "bg-node-http border-blue-300",
    llm: "bg-node-llm border-purple-300",
    transform: "bg-node-transform border-orange-300",
    conditional: "bg-node-conditional border-green-300",
    merge: "bg-node-merge border-pink-300",
    default: "bg-white border-gray-300",
  };
  return colorMap[props.type || "default"] || colorMap.default;
});

// Get icon for node type
const nodeIcon = computed(() => {
  if (props.data.icon) return props.data.icon;

  const iconMap: Record<string, string> = {
    http: "heroicons:globe-alt",
    llm: "heroicons:sparkles",
    transform: "heroicons:arrow-path",
    conditional: "heroicons:code-bracket",
    merge: "heroicons:arrows-pointing-in",
    default: "heroicons:square-3-stack-3d",
  };
  return iconMap[props.type || "default"] || iconMap.default;
});
</script>

<template>
  <div
    :class="[
      'base-node',
      nodeColor,
      'rounded-lg border-2 shadow-sm transition-all',
      { 'ring-2 ring-blue-500 ring-offset-2': selected },
    ]"
  >
    <!-- Input handle -->
    <Handle :position="Position.Top" type="target" class="handle-target" />

    <!-- Node content -->
    <div class="px-4 py-3">
      <div class="flex items-center gap-2">
        <Icon :icon="nodeIcon" class="size-5 shrink-0 text-gray-700" />
        <span class="truncate text-sm font-medium text-gray-900">
          {{ data.label }}
        </span>
      </div>

      <!-- Additional content slot -->
      <slot />
    </div>

    <!-- Output handle -->
    <Handle :position="Position.Bottom" type="source" class="handle-source" />
  </div>
</template>

<style scoped>
.base-node {
  min-width: 200px;
  max-width: 300px;
}

:deep(.handle-target),
:deep(.handle-source) {
  width: 12px;
  height: 12px;
  border: 2px solid white;
  background: #3b82f6;
  transition: all 0.2s;
}

:deep(.handle-target:hover),
:deep(.handle-source:hover) {
  width: 16px;
  height: 16px;
  background: #2563eb;
}

:deep(.handle-target) {
  top: -6px;
}

:deep(.handle-source) {
  bottom: -6px;
}
</style>
