```vue
<script setup lang="ts">
// @ts-nocheck
import { computed, ref, onUnmounted, inject } from "vue";
import { Handle, Position } from "@vue-flow/core";
import { Icon } from "@iconify/vue";

interface Props {
  id?: string; // Node ID
  data: {
    label: string;
    icon?: string;
    config?: Record<string, any>;
    metadata?: Record<string, any>;
    executionState?: "running" | "completed" | "failed";
    animated?: boolean;
    executionInput?: Record<string, any>;
    executionOutput?: Record<string, any>;
    executionError?: any;
  };
  selected?: boolean;
  type?: string;
  hideSourceHandle?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
  hideSourceHandle: false,
});

// Inject openNodeConfig function from parent
const openNodeConfig = inject<(nodeId: string) => void>("openNodeConfig");

const showTooltip = ref(false);

// Toggle tooltip on click
function toggleTooltip(event: MouseEvent) {
  event.stopPropagation();
  showTooltip.value = !showTooltip.value;
}

// Open config panel
function openConfig(event: MouseEvent) {
  event.stopPropagation();
  if (openNodeConfig && props.id) {
    openNodeConfig(props.id);
  }
}

// Close tooltip when clicking outside
function handleClickOutside(event: MouseEvent) {
  if (showTooltip.value) {
    const target = event.target as HTMLElement;
    // Check if the click is outside the tooltip and not on the toggle button
    if (
      !target.closest(".node-tooltip") &&
      !target.closest(".data-toggle-btn")
    ) {
      showTooltip.value = false;
    }
  }
}

// Add click outside listener when tooltip is shown
if (typeof window !== "undefined") {
  window.addEventListener("click", handleClickOutside);
}

// Remove listener when component is unmounted
onUnmounted(() => {
  if (typeof window !== "undefined") {
    window.removeEventListener("click", handleClickOutside);
  }
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

// Get execution state classes
const executionStateClasses = computed(() => {
  if (!props.data.executionState) return "";

  const stateMap: Record<string, string> = {
    running: "border-blue-500 shadow-lg shadow-blue-500/50 animate-pulse",
    completed: "border-green-500 shadow-lg shadow-green-500/30",
    failed: "border-red-500 shadow-lg shadow-red-500/50 animate-shake",
  };

  return stateMap[props.data.executionState] || "";
});

// Check if node has execution data
const hasExecutionData = computed(() => {
  return !!(
    props.data.executionInput ||
    props.data.executionOutput ||
    props.data.executionError
  );
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
      executionStateClasses,
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

        <!-- Execution state indicator -->
        <span
          v-if="data.executionState"
          :class="[
            'size-2 rounded-full',
            data.executionState === 'running' && 'animate-ping bg-blue-500',
            data.executionState === 'completed' && 'bg-green-500',
            data.executionState === 'failed' && 'bg-red-500',
          ]"
        />

        <!-- Action buttons -->
        <div class="ml-auto flex items-center gap-1">
          <!-- Settings button -->
          <button
            class="settings-btn rounded p-1 transition-colors hover:bg-gray-100"
            @click="openConfig"
            title="Configure node"
          >
            <Icon
              icon="heroicons:cog-6-tooth"
              class="size-4 text-gray-400 hover:text-gray-600"
            />
          </button>

          <!-- Data button (only show if has execution data) -->
          <button
            v-if="hasExecutionData"
            class="data-toggle-btn rounded p-1 transition-colors hover:bg-gray-100"
            :class="{ 'bg-blue-50': showTooltip }"
            @click="toggleTooltip"
            title="View execution data"
          >
            <Icon
              icon="heroicons:document-text"
              class="size-4"
              :class="
                showTooltip
                  ? 'text-blue-600'
                  : 'text-gray-400 hover:text-gray-600'
              "
            />
          </button>
        </div>
      </div>

      <!-- Additional content slot -->
      <slot />
    </div>

    <!-- Output handle (hidden if hideSourceHandle is true) -->
    <Handle v-if="!hideSourceHandle" :position="Position.Bottom" type="source" class="handle-source" />

    <!-- Execution Data Tooltip (Click-based, interactive) -->
    <div v-if="showTooltip" class="node-tooltip">
      <div class="tooltip-header">
        <span class="tooltip-header-title">Execution Data</span>
        <button
          class="tooltip-close-btn"
          @click.stop="showTooltip = false"
          title="Close"
        >
          <Icon icon="heroicons:x-mark" class="size-4" />
        </button>
      </div>

      <!-- Input -->
      <div v-if="data.executionInput" class="tooltip-section">
        <div class="tooltip-title">
          <Icon icon="heroicons:arrow-down-tray" class="size-3" />
          Input
        </div>
        <pre class="tooltip-content">{{
          JSON.stringify(data.executionInput, null, 2)
        }}</pre>
      </div>

      <!-- Output -->
      <div v-if="data.executionOutput" class="tooltip-section">
        <div class="tooltip-title">
          <Icon icon="heroicons:arrow-up-tray" class="size-3" />
          Output
        </div>
        <pre class="tooltip-content">{{
          JSON.stringify(data.executionOutput, null, 2)
        }}</pre>
      </div>

      <!-- Error -->
      <div v-if="data.executionError" class="tooltip-section">
        <div class="tooltip-title text-red-600">
          <Icon icon="heroicons:exclamation-circle" class="size-3" />
          Error
        </div>
        <pre class="tooltip-content bg-red-50 text-red-900">{{
          typeof data.executionError === "string"
            ? data.executionError
            : JSON.stringify(data.executionError, null, 2)
        }}</pre>
      </div>
    </div>
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

/* Shake animation for failed nodes */
@keyframes shake {
  0%,
  100% {
    transform: translateX(0);
  }
  10%,
  30%,
  50%,
  70%,
  90% {
    transform: translateX(-2px);
  }
  20%,
  40%,
  60%,
  80% {
    transform: translateX(2px);
  }
}

.animate-shake {
  animation: shake 0.5s ease-in-out;
}

/* Execution Data Tooltip */
.node-tooltip {
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  z-index: 10000; /* Very high z-index to appear above all nodes */
  min-width: 350px;
  max-width: 600px;
  margin-top: 12px;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  box-shadow:
    0 20px 25px -5px rgba(0, 0, 0, 0.1),
    0 10px 10px -5px rgba(0, 0, 0, 0.04);
  pointer-events: auto; /* Enable interaction */
  user-select: text; /* Allow text selection */
}

.tooltip-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #e5e7eb;
  background: #f9fafb;
  border-radius: 8px 8px 0 0;
}

.tooltip-header-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: #111827;
}

.tooltip-close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  border-radius: 4px;
  transition: background-color 0.2s;
  color: #6b7280;
}

.tooltip-close-btn:hover {
  background-color: #e5e7eb;
  color: #111827;
}

.tooltip-section {
  padding: 12px 16px;
  border-bottom: 1px solid #f3f4f6;
}

.tooltip-section:last-child {
  border-bottom: none;
  padding-bottom: 16px;
}

.tooltip-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  color: #374151;
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.tooltip-content {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 0.75rem;
  line-height: 1.5;
  padding: 12px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  overflow: auto;
  max-height: 300px;
  white-space: pre-wrap;
  word-break: break-word;
  cursor: text; /* Show text cursor */
}

/* Data toggle button */
.data-toggle-btn,
.settings-btn {
  cursor: pointer;
  border: none;
  background: transparent;
}

.data-toggle-btn:hover,
.settings-btn:hover {
  background-color: #f3f4f6;
}
</style>
