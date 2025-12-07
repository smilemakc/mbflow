<script setup lang="ts">
import { computed, onMounted, onUnmounted } from "vue";
import { Icon } from "@iconify/vue";
import { executionObserver } from "@/services/executionObserver";
import { getExecution } from "@/api/executions";
import type { ExecutionStatus } from "@/types/execution";
import type { Execution } from "@/types/execution";

interface Props {
  executionId: string;
  autoConnect?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  autoConnect: true,
});

const emit = defineEmits<{
  complete: [];
  error: [error: string];
  statusChange: [status: ExecutionStatus];
}>();

// Use global execution observer service
const execution = computed<Execution | undefined>(() =>
  executionObserver.executions.get(props.executionId),
);

const events = computed(
  () => executionObserver.events.get(props.executionId) || [],
);

const isConnected = computed(() => executionObserver.isConnected.value);

// Register callbacks for this execution
onMounted(async () => {
  if (props.autoConnect) {
    try {
      // Fetch initial execution data
      const exec = await getExecution(props.executionId);

      // Observe this execution
      executionObserver.observe(exec);

      // Register callbacks
      executionObserver.onStatusChange(props.executionId, (status) => {
        emit("statusChange", status as ExecutionStatus);
      });

      executionObserver.onComplete(props.executionId, () => {
        emit("complete");
      });

      executionObserver.onError(props.executionId, (error) => {
        emit("error", error);
      });
    } catch (error) {
      console.error("[ExecutionStatusPanel] Failed to load execution:", error);
    }
  }
});

// Cleanup on unmount
onUnmounted(() => {
  executionObserver.unobserve(props.executionId);
});

// Computed properties
const statusColor = computed(() => {
  if (!execution.value) return "gray";

  const colors: Record<ExecutionStatus, string> = {
    pending: "yellow",
    running: "blue",
    completed: "green",
    failed: "red",
    cancelled: "gray",
    timeout: "orange",
  };

  return colors[execution.value.status] || "gray";
});

const statusIcon = computed(() => {
  if (!execution.value) return "heroicons:clock";

  const icons: Record<ExecutionStatus, string> = {
    pending: "heroicons:clock",
    running: "heroicons:arrow-path",
    completed: "heroicons:check-circle",
    failed: "heroicons:x-circle",
    cancelled: "heroicons:stop-circle",
    timeout: "heroicons:exclamation-triangle",
  };

  return icons[execution.value.status] || "heroicons:clock";
});

const isRunning = computed(() => {
  return (
    execution.value?.status === "running" ||
    execution.value?.status === "pending"
  );
});

const progress = computed(() => {
  if (!execution.value || events.value.length === 0) return 0;

  // Calculate progress based on completed nodes
  const completedNodes = events.value.filter(
    (e) => e.event?.event_type === "node.completed",
  ).length;

  const totalNodes = events.value.filter((e) =>
    e.event?.event_type?.includes("node."),
  ).length;

  if (totalNodes === 0) return 0;

  return Math.round((completedNodes / totalNodes) * 100);
});

// Format duration
function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

// Get recent events (last 5)
const recentEvents = computed(() => {
  return events.value
    .filter((e) => e.type === "event" && e.event)
    .slice(-5)
    .reverse();
});
</script>

<template>
  <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
    <!-- Header -->
    <div class="mb-4 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Icon
          :icon="statusIcon"
          :class="[
            'size-5',
            statusColor === 'green' && 'text-green-600',
            statusColor === 'blue' && 'animate-spin text-blue-600',
            statusColor === 'red' && 'text-red-600',
            statusColor === 'yellow' && 'text-yellow-600',
            statusColor === 'gray' && 'text-gray-600',
            statusColor === 'orange' && 'text-orange-600',
          ]"
        />
        <h3 class="font-semibold text-gray-900">Execution Status</h3>
      </div>

      <!-- Connection status -->
      <div class="flex items-center gap-2">
        <div
          :class="[
            'size-2 rounded-full',
            isConnected ? 'bg-green-500' : 'bg-red-500',
          ]"
        />
        <span class="text-xs text-gray-500">
          {{ isConnected ? "Connected" : "Disconnected" }}
        </span>
      </div>
    </div>

    <!-- Status and Progress -->
    <div v-if="execution" class="space-y-3">
      <!-- Status badge -->
      <div class="flex items-center gap-2">
        <span class="text-sm text-gray-600">Status:</span>
        <span
          :class="[
            'rounded-full px-3 py-1 text-sm font-medium',
            statusColor === 'green' && 'bg-green-100 text-green-700',
            statusColor === 'blue' && 'bg-blue-100 text-blue-700',
            statusColor === 'red' && 'bg-red-100 text-red-700',
            statusColor === 'yellow' && 'bg-yellow-100 text-yellow-700',
            statusColor === 'gray' && 'bg-gray-100 text-gray-700',
            statusColor === 'orange' && 'bg-orange-100 text-orange-700',
          ]"
        >
          {{ execution.status.toUpperCase() }}
        </span>
      </div>

      <!-- Progress bar -->
      <div v-if="isRunning && progress > 0" class="space-y-1">
        <div class="flex items-center justify-between text-sm">
          <span class="text-gray-600">Progress</span>
          <span class="font-medium text-gray-900">{{ progress }}%</span>
        </div>
        <div class="h-2 overflow-hidden rounded-full bg-gray-200">
          <div
            class="h-full bg-blue-600 transition-all duration-300"
            :style="{ width: `${progress}%` }"
          />
        </div>
      </div>

      <!-- Duration -->
      <div v-if="execution.duration" class="flex items-center gap-2">
        <Icon icon="heroicons:clock" class="size-4 text-gray-400" />
        <span class="text-sm text-gray-600">
          Duration: {{ formatDuration(execution.duration) }}
        </span>
      </div>

      <!-- Error message -->
      <div
        v-if="execution.status === 'failed' && execution.error"
        class="rounded-md border border-red-200 bg-red-50 p-3"
      >
        <div class="flex gap-2">
          <Icon
            icon="heroicons:exclamation-circle"
            class="size-5 shrink-0 text-red-600"
          />
          <div class="flex-1">
            <p class="text-sm font-medium text-red-800">Error</p>
            <p class="mt-1 text-sm text-red-700">{{ execution.error }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Recent Events -->
    <div
      v-if="recentEvents.length > 0"
      class="mt-4 border-t border-gray-200 pt-4"
    >
      <h4 class="mb-2 text-sm font-medium text-gray-700">Recent Events</h4>
      <div class="space-y-2">
        <div
          v-for="(event, index) in recentEvents"
          :key="index"
          class="flex items-start gap-2 text-xs"
        >
          <Icon
            :icon="
              event.event?.event_type?.includes('completed')
                ? 'heroicons:check-circle'
                : event.event?.event_type?.includes('failed')
                  ? 'heroicons:x-circle'
                  : 'heroicons:arrow-path'
            "
            :class="[
              'mt-0.5 size-4 shrink-0',
              event.event?.event_type?.includes('completed') &&
                'text-green-600',
              event.event?.event_type?.includes('failed') && 'text-red-600',
              event.event?.event_type?.includes('started') && 'text-blue-600',
            ]"
          />
          <div class="flex-1">
            <p class="text-gray-900">
              {{ event.event?.event_type }}
              <span v-if="event.event?.node_name" class="text-gray-600">
                - {{ event.event.node_name }}
              </span>
            </p>
            <p class="text-gray-500">
              {{ new Date(event.event?.timestamp || "").toLocaleTimeString() }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="!execution" class="flex items-center justify-center py-8">
      <div class="text-center">
        <div
          class="mx-auto size-8 animate-spin rounded-full border-b-2 border-blue-600"
        />
        <p class="mt-2 text-sm text-gray-600">Loading execution data...</p>
      </div>
    </div>
  </div>
</template>
