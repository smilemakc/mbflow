<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Icon } from "@iconify/vue";
import { toast } from "vue3-toastify";
import Card from "@/components/ui/Card.vue";
import Badge from "@/components/ui/Badge.vue";
import Button from "@/components/ui/Button.vue";
import {
  getExecution,
  cancelExecution,
  getExecutionLogs,
} from "@/api/executions";
import type {
  ExecutionStatus,
} from "@/types/execution";
import { useExecutionObserver } from "@/composables/useExecutionObserver";

const route = useRoute();
const router = useRouter();

const executionId = computed(() => {
  const params: any = route.params;
  const id = params.id;
  return (Array.isArray(id) ? id[0] : id) || "";
});
const isLoading = ref(true);
const error = ref<string | null>(null);
const isCancelling = ref(false);
const selectedNodeExecution = ref<string | null>(null);
const showTimeline = ref(true);
const logs = ref<any[]>([]);
const isLoadingLogs = ref(false);

// Use execution observer for real-time updates and events
const {
  execution,
  isConnected: wsConnected,
  setExecution,
  events,
} = useExecutionObserver({
  executionId,
  autoConnect: true,
  onStatusChange: (status) => {
    console.log(`[ExecutionDetail] Status changed to: ${status}`);
  },
  onComplete: () => {
    toast.success("Execution completed successfully!");
  },
  onError: (errorMsg) => {
    toast.error(`Execution failed: ${errorMsg}`);
  },
});

onMounted(async () => {
  await loadExecution();
  await loadLogs();
});

async function loadExecution() {
  isLoading.value = true;
  error.value = null;

  try {
    const response = await getExecution(executionId.value);
    setExecution(response);
  } catch (err: any) {
    console.error("Failed to load execution:", err);
    error.value = err.message || "Failed to load execution";
    toast.error(error.value);
  } finally {
    isLoading.value = false;
  }
}

async function handleCancel() {
  if (!confirm("Are you sure you want to cancel this execution?")) {
    return;
  }

  isCancelling.value = true;

  try {
    await cancelExecution(executionId.value);
    await loadExecution();
    toast.success("Execution cancelled successfully");
  } catch (err: any) {
    console.error("Failed to cancel execution:", err);
    toast.error(
      `Failed to cancel execution: ${err.message || "Unknown error"}`,
    );
  } finally {
    isCancelling.value = false;
  }
}

async function loadLogs() {
  isLoadingLogs.value = true;

  try {
    const response = await getExecutionLogs(executionId.value);
    logs.value = response.logs || [];
  } catch (err: any) {
    console.error("Failed to load logs:", err);
    // Don't show error toast for logs, just log it
  } finally {
    isLoadingLogs.value = false;
  }
}

// Combined events and logs for display
const allEvents = computed(() => {
  const wsEvents = events.value.map((e) => {
    const evt: any = e.event || {};
    return {
      timestamp: e.timestamp,
      event_type: evt.event_type || "unknown",
      level: getLogLevel(evt.event_type),
      message: formatLogMessage(evt.event_type, evt),
      data: evt,
      source: "websocket",
    };
  });

  const apiLogs = logs.value.map((log: any) => ({
    timestamp: log.timestamp,
    event_type: log.event_type,
    level: log.level || "info",
    message: log.message,
    data: log.data || {},
    source: "api",
  }));

  // Combine and deduplicate
  // robust deduplication: if we have an API log with same timestamp and type as WS event, prefer API log
  // actually, API log is better formatted.

  const distinctEvents = new Map();

  // Add API logs first
  apiLogs.forEach((log) => {
    const key = `${log.timestamp}-${log.event_type}`;
    distinctEvents.set(key, log);
  });

  // Add WS events if not present (or if timestamp is later than latest API log?)
  // Simple dedup by key
  wsEvents.forEach((evt) => {
    const key = `${evt.timestamp}-${evt.event_type}`;
    if (!distinctEvents.has(key)) {
      distinctEvents.set(key, evt);
    }
  });

  return Array.from(distinctEvents.values()).sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
  );
});

function getLogLevel(
  eventType: string = "",
): "info" | "success" | "warning" | "error" {
  if (eventType.includes("failed")) return "error";
  if (eventType.includes("completed")) return "success";
  if (eventType.includes("retrying")) return "warning";
  return "info";
}

function formatLogMessage(eventType: string = "", payload: any = {}): string {
  switch (eventType) {
    case "execution.started":
      return "Execution started";
    case "execution.completed":
      return payload.duration_ms
        ? `Execution completed in ${payload.duration_ms}ms`
        : "Execution completed";
    case "execution.failed":
      return payload.error
        ? `Execution failed: ${payload.error}`
        : "Execution failed";
    case "wave.started":
      return payload.wave_index !== undefined &&
        payload.node_count !== undefined
        ? `Wave ${payload.wave_index} started with ${payload.node_count} nodes`
        : "Wave started";
    case "wave.completed":
      return payload.wave_index !== undefined
        ? `Wave ${payload.wave_index} completed`
        : "Wave completed";
    case "node.started":
      return payload.node_name
        ? `Node '${payload.node_name}' started`
        : "Node started";
    case "node.completed":
      return payload.node_name && payload.duration_ms
        ? `Node '${payload.node_name}' completed in ${payload.duration_ms}ms`
        : `Node '${payload.node_name || ""}' completed`;
    case "node.failed":
      return payload.node_name && payload.error
        ? `Node '${payload.node_name}' failed: ${payload.error}`
        : `Node '${payload.node_name || ""}' failed`;
    case "node.retrying":
      return payload.node_name
        ? `Node '${payload.node_name}' retrying`
        : "Node retrying";
    default:
      return eventType;
  }
}

// Status helpers
const statusVariant = computed(
  (): "default" | "success" | "warning" | "danger" | "info" | "gray" => {
    if (!execution.value) return "default";

    const variantMap: Record<
      ExecutionStatus,
      "default" | "success" | "warning" | "danger" | "info" | "gray"
    > = {
      pending: "info",
      running: "info",
      completed: "success",
      failed: "danger",
      cancelled: "gray",
      timeout: "warning",
    };

    return variantMap[execution.value.status] || "default";
  },
);

const statusIcon = computed(() => {
  if (!execution.value) return "heroicons:question-mark-circle";

  const iconMap: Record<ExecutionStatus, string> = {
    pending: "heroicons:clock",
    running: "heroicons:arrow-path",
    completed: "heroicons:check-circle",
    failed: "heroicons:x-circle",
    cancelled: "heroicons:stop-circle",
    timeout: "heroicons:exclamation-triangle",
  };

  return iconMap[execution.value.status] || "heroicons:question-mark-circle";
});

// Node execution helpers
const sortedNodeExecutions = computed(() => {
  if (!execution.value?.node_executions) return [];
  return [...execution.value.node_executions].sort(
    (a, b) =>
      new Date(a.started_at).getTime() - new Date(b.started_at).getTime(),
  );
});

function toggleNodeExecution(nodeExecId: string) {
  selectedNodeExecution.value =
    selectedNodeExecution.value === nodeExecId ? null : nodeExecId;
}

function getNodeStatusVariant(
  status: ExecutionStatus,
): "default" | "success" | "warning" | "danger" | "info" | "gray" {
  const variantMap: Record<
    ExecutionStatus,
    "default" | "success" | "warning" | "danger" | "info" | "gray"
  > = {
    pending: "info",
    running: "info",
    completed: "success",
    failed: "danger",
    cancelled: "gray",
    timeout: "warning",
  };
  return variantMap[status] || "default";
}

function getNodeStatusIcon(status: ExecutionStatus): string {
  const iconMap: Record<ExecutionStatus, string> = {
    pending: "heroicons:clock",
    running: "heroicons:arrow-path",
    completed: "heroicons:check-circle",
    failed: "heroicons:x-circle",
    cancelled: "heroicons:stop-circle",
    timeout: "heroicons:exclamation-triangle",
  };
  return iconMap[status] || "heroicons:question-mark-circle";
}

// Formatting helpers
function formatDuration(ms: number | undefined): string {
  if (!ms) return "N/A";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = ((ms % 60000) / 1000).toFixed(0);
  return `${minutes}m ${seconds}s`;
}

function formatTimestamp(timestamp: string): string {
  return new Date(timestamp).toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function formatRelativeTime(timestamp: string): string {
  const now = new Date().getTime();
  const then = new Date(timestamp).getTime();
  const diff = now - then;

  if (diff < 60000) return "just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.success("Copied to clipboard!");
}

// Event details expansion
const selectedEventIndex = ref<number | null>(null);

function toggleEventDetails(index: number) {
  selectedEventIndex.value = selectedEventIndex.value === index ? null : index;
}

function getEventIcon(eventType: string): string {
  const iconMap: Record<string, string> = {
    "execution.started": "heroicons:play",
    "execution.completed": "heroicons:check-circle",
    "execution.failed": "heroicons:x-circle",
    "wave.started": "heroicons:queue-list",
    "wave.completed": "heroicons:check-badge",
    "node.started": "heroicons:arrow-right-circle",
    "node.completed": "heroicons:check",
    "node.failed": "heroicons:exclamation-triangle",
    "node.retrying": "heroicons:arrow-path",
  };
  return iconMap[eventType] || "heroicons:information-circle";
}
</script>

<template>
  <div class="execution-details-page">
    <!-- Header -->
    <div class="mb-6">
      <div class="flex items-center justify-between">
        <div class="flex-1">
          <div class="flex items-center gap-3">
            <Button variant="ghost" size="sm" @click="router.back()">
              <Icon icon="heroicons:arrow-left" class="size-4" />
            </Button>
            <div>
              <h1 class="text-2xl font-bold text-gray-900">
                Execution Details
              </h1>
              <div class="mt-1 flex items-center gap-2 text-sm text-gray-500">
                <span class="font-mono">{{ executionId }}</span>
                <button
                  @click="copyToClipboard(executionId)"
                  class="text-gray-400 hover:text-gray-600"
                  title="Copy ID"
                >
                  <Icon icon="heroicons:clipboard-document" class="size-4" />
                </button>
                <span v-if="execution" class="text-gray-400">•</span>
                <span v-if="execution">{{
                  formatRelativeTime(execution.started_at)
                }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex items-center gap-2">
          <Button
            v-if="execution?.status === 'running'"
            variant="danger"
            :loading="isCancelling"
            @click="handleCancel"
          >
            <Icon icon="heroicons:stop" class="mr-2 size-4" />
            Cancel
          </Button>
          <Button variant="secondary" @click="loadExecution">
            <Icon icon="heroicons:arrow-path" class="mr-2 size-4" />
            Refresh
          </Button>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <div class="text-center">
        <Icon
          icon="heroicons:arrow-path"
          class="mx-auto size-12 animate-spin text-blue-500"
        />
        <p class="mt-4 text-sm text-gray-600">Loading execution details...</p>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="rounded-lg bg-red-50 p-6">
      <div class="flex">
        <Icon icon="heroicons:x-circle" class="size-6 text-red-400" />
        <div class="ml-3">
          <h3 class="text-sm font-medium text-red-800">
            Error loading execution
          </h3>
          <p class="mt-2 text-sm text-red-700">{{ error }}</p>
          <Button
            variant="secondary"
            size="sm"
            class="mt-4"
            @click="loadExecution"
          >
            Try Again
          </Button>
        </div>
      </div>
    </div>

    <!-- Execution Content -->
    <div v-else-if="execution" class="space-y-6">
      <!-- Status Overview -->
      <Card>
        <div class="p-6">
          <div class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
            <!-- Status -->
            <div>
              <div class="text-sm font-medium text-gray-500">Status</div>
              <div class="mt-2 flex items-center gap-2">
                <Icon
                  :icon="statusIcon"
                  class="size-5"
                  :class="{
                    'text-gray-500': execution.status === 'pending',
                    'animate-spin text-blue-500':
                      execution.status === 'running',
                    'text-green-500': execution.status === 'completed',
                    'text-red-500': execution.status === 'failed',
                    'text-orange-500': execution.status === 'cancelled',
                    'text-yellow-500': execution.status === 'timeout',
                  }"
                />
                <Badge :variant="statusVariant">
                  {{ execution.status.toUpperCase() }}
                </Badge>
                <span
                  v-if="wsConnected"
                  class="flex items-center gap-1 text-xs text-green-600"
                >
                  <span
                    class="size-2 animate-pulse rounded-full bg-green-500"
                  ></span>
                  Live
                </span>
              </div>
            </div>

            <!-- Duration -->
            <div>
              <div class="text-sm font-medium text-gray-500">Duration</div>
              <div class="mt-2 text-lg font-semibold text-gray-900">
                {{ formatDuration(execution.duration) }}
              </div>
            </div>

            <!-- Started At -->
            <div>
              <div class="text-sm font-medium text-gray-500">Started</div>
              <div class="mt-2 text-sm text-gray-900">
                {{ formatTimestamp(execution.started_at) }}
              </div>
            </div>

            <!-- Completed At -->
            <div>
              <div class="text-sm font-medium text-gray-500">Completed</div>
              <div class="mt-2 text-sm text-gray-900">
                {{
                  execution.completed_at
                    ? formatTimestamp(execution.completed_at)
                    : "Running..."
                }}
              </div>
            </div>
          </div>
        </div>
      </Card>

      <!-- Workflow Info -->
      <Card>
        <div class="border-b border-gray-200 px-6 py-4">
          <h2 class="text-lg font-semibold text-gray-900">
            Workflow Information
          </h2>
        </div>
        <div class="p-6">
          <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div>
              <div class="text-sm font-medium text-gray-500">Workflow Name</div>
              <div class="mt-1 text-sm text-gray-900">
                {{ execution.workflow_name || "N/A" }}
              </div>
            </div>
            <div>
              <div class="text-sm font-medium text-gray-500">Workflow ID</div>
              <div class="mt-1 flex items-center gap-2">
                <span class="font-mono text-sm text-gray-900">{{
                  execution.workflow_id
                }}</span>
                <button
                  @click="copyToClipboard(execution.workflow_id)"
                  class="text-gray-400 hover:text-gray-600"
                  title="Copy ID"
                >
                  <Icon icon="heroicons:clipboard-document" class="size-4" />
                </button>
              </div>
            </div>
            <div v-if="execution.triggered_by">
              <div class="text-sm font-medium text-gray-500">Triggered By</div>
              <div class="mt-1 text-sm text-gray-900">
                {{ execution.triggered_by }}
              </div>
            </div>
            <div v-if="execution.strict_mode !== undefined">
              <div class="text-sm font-medium text-gray-500">Strict Mode</div>
              <div class="mt-1">
                <Badge :variant="execution.strict_mode ? 'info' : 'gray'">
                  {{ execution.strict_mode ? "Enabled" : "Disabled" }}
                </Badge>
              </div>
            </div>
          </div>
        </div>
      </Card>

      <!-- Input/Output -->
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- Input -->
        <Card>
          <div class="border-b border-gray-200 px-6 py-4">
            <div class="flex items-center justify-between">
              <h2 class="text-lg font-semibold text-gray-900">Input</h2>
              <button
                v-if="execution.input"
                @click="
                  copyToClipboard(JSON.stringify(execution.input, null, 2))
                "
                class="text-gray-400 hover:text-gray-600"
                title="Copy JSON"
              >
                <Icon icon="heroicons:clipboard-document" class="size-4" />
              </button>
            </div>
          </div>
          <div class="p-6">
            <pre v-if="execution.input" class="code-block">{{
              JSON.stringify(execution.input, null, 2)
            }}</pre>
            <div v-else class="text-sm text-gray-500">No input provided</div>
          </div>
        </Card>

        <!-- Output -->
        <Card>
          <div class="border-b border-gray-200 px-6 py-4">
            <div class="flex items-center justify-between">
              <h2 class="text-lg font-semibold text-gray-900">Output</h2>
              <button
                v-if="execution.output"
                @click="
                  copyToClipboard(JSON.stringify(execution.output, null, 2))
                "
                class="text-gray-400 hover:text-gray-600"
                title="Copy JSON"
              >
                <Icon icon="heroicons:clipboard-document" class="size-4" />
              </button>
            </div>
          </div>
          <div class="p-6">
            <pre v-if="execution.output" class="code-block">{{
              JSON.stringify(execution.output, null, 2)
            }}</pre>
            <div v-else class="text-sm text-gray-500">
              {{
                execution.status === "running"
                  ? "Execution in progress..."
                  : "No output available"
              }}
            </div>
          </div>
        </Card>
      </div>

      <!-- Error (if failed) -->
      <Card v-if="execution.error">
        <div class="border-b border-red-200 bg-red-50 px-6 py-4">
          <div class="flex items-center gap-2">
            <Icon
              icon="heroicons:exclamation-circle"
              class="size-5 text-red-600"
            />
            <h2 class="text-lg font-semibold text-red-900">Error</h2>
          </div>
        </div>
        <div class="p-6">
          <pre class="code-block bg-red-50 text-red-900">{{
            execution.error
          }}</pre>
        </div>
      </Card>

      <!-- Variables (if any) -->
      <Card
        v-if="
          execution.variables && Object.keys(execution.variables).length > 0
        "
      >
        <div class="border-b border-gray-200 px-6 py-4">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900">
              Runtime Variables
            </h2>
            <button
              @click="
                copyToClipboard(JSON.stringify(execution.variables, null, 2))
              "
              class="text-gray-400 hover:text-gray-600"
              title="Copy JSON"
            >
              <Icon icon="heroicons:clipboard-document" class="size-4" />
            </button>
          </div>
        </div>
        <div class="p-6">
          <pre class="code-block">{{
            JSON.stringify(execution.variables, null, 2)
          }}</pre>
        </div>
      </Card>

      <!-- Node Executions -->
      <Card
        v-if="execution.node_executions && execution.node_executions.length > 0"
      >
        <div class="border-b border-gray-200 px-6 py-4">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900">
              Node Executions
              <span class="ml-2 text-sm font-normal text-gray-500"
                >({{ execution.node_executions.length }})</span
              >
            </h2>
            <button
              @click="showTimeline = !showTimeline"
              class="flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900"
            >
              <Icon
                :icon="
                  showTimeline ? 'heroicons:list-bullet' : 'heroicons:chart-bar'
                "
                class="size-4"
              />
              {{ showTimeline ? "List View" : "Timeline View" }}
            </button>
          </div>
        </div>

        <!-- Timeline View -->
        <div v-if="showTimeline" class="p-6">
          <div class="relative space-y-4">
            <div
              v-for="(nodeExec, index) in sortedNodeExecutions"
              :key="nodeExec.id"
              class="relative flex gap-4"
            >
              <!-- Timeline line -->
              <div class="flex flex-col items-center">
                <div
                  class="flex size-10 items-center justify-center rounded-full border-2"
                  :class="{
                    'border-gray-300 bg-gray-100':
                      nodeExec.status === 'pending',
                    'border-blue-500 bg-blue-100':
                      nodeExec.status === 'running',
                    'border-green-500 bg-green-100':
                      nodeExec.status === 'completed',
                    'border-red-500 bg-red-100': nodeExec.status === 'failed',
                  }"
                >
                  <Icon
                    :icon="getNodeStatusIcon(nodeExec.status)"
                    class="size-5"
                    :class="{
                      'text-gray-500': nodeExec.status === 'pending',
                      'animate-spin text-blue-600':
                        nodeExec.status === 'running',
                      'text-green-600': nodeExec.status === 'completed',
                      'text-red-600': nodeExec.status === 'failed',
                    }"
                  />
                </div>
                <div
                  v-if="index < sortedNodeExecutions.length - 1"
                  class="w-0.5 flex-1 bg-gray-200"
                  style="min-height: 40px"
                ></div>
              </div>

              <!-- Node content -->
              <div class="flex-1 pb-8">
                <div
                  class="cursor-pointer rounded-lg border border-gray-200 bg-white p-4 transition-all hover:border-gray-300 hover:shadow-md"
                  @click="toggleNodeExecution(nodeExec.id)"
                >
                  <div class="flex items-start justify-between">
                    <div class="flex-1">
                      <div class="flex items-center gap-2">
                        <h3 class="font-semibold text-gray-900">
                          {{ nodeExec.node_name || nodeExec.node_id }}
                        </h3>
                        <Badge
                          :variant="getNodeStatusVariant(nodeExec.status)"
                          size="sm"
                        >
                          {{ nodeExec.status }}
                        </Badge>
                      </div>
                      <div class="mt-1 text-sm text-gray-500">
                        <span class="font-mono">{{ nodeExec.node_type }}</span>
                        <span class="mx-2">•</span>
                        <span>{{ formatDuration(nodeExec.duration) }}</span>
                        <span v-if="nodeExec.retry_count" class="mx-2">•</span>
                        <span
                          v-if="nodeExec.retry_count"
                          class="text-orange-600"
                        >
                          {{ nodeExec.retry_count }}
                          {{ nodeExec.retry_count === 1 ? "retry" : "retries" }}
                        </span>
                      </div>
                      <div class="mt-1 text-xs text-gray-400">
                        {{ formatTimestamp(nodeExec.started_at) }}
                      </div>
                    </div>
                    <Icon
                      :icon="
                        selectedNodeExecution === nodeExec.id
                          ? 'heroicons:chevron-up'
                          : 'heroicons:chevron-down'
                      "
                      class="size-5 text-gray-400"
                    />
                  </div>

                  <!-- Expanded Details -->
                  <div
                    v-if="selectedNodeExecution === nodeExec.id"
                    class="mt-4 space-y-4 border-t border-gray-100 pt-4"
                  >
                    <!-- Input -->
                    <div v-if="nodeExec.input">
                      <div class="mb-2 flex items-center justify-between">
                        <div class="text-sm font-medium text-gray-700">
                          Input
                        </div>
                        <button
                          @click.stop="
                            copyToClipboard(
                              JSON.stringify(nodeExec.input, null, 2),
                            )
                          "
                          class="text-gray-400 hover:text-gray-600"
                          title="Copy JSON"
                        >
                          <Icon
                            icon="heroicons:clipboard-document"
                            class="size-4"
                          />
                        </button>
                      </div>
                      <pre class="code-block-sm bg-blue-50">{{
                        JSON.stringify(nodeExec.input, null, 2)
                      }}</pre>
                    </div>

                    <!-- Output -->
                    <div v-if="nodeExec.output">
                      <div class="mb-2 flex items-center justify-between">
                        <div class="text-sm font-medium text-gray-700">
                          Output
                        </div>
                        <button
                          @click.stop="
                            copyToClipboard(
                              JSON.stringify(nodeExec.output, null, 2),
                            )
                          "
                          class="text-gray-400 hover:text-gray-600"
                          title="Copy JSON"
                        >
                          <Icon
                            icon="heroicons:clipboard-document"
                            class="size-4"
                          />
                        </button>
                      </div>
                      <pre class="code-block-sm bg-green-50">{{
                        JSON.stringify(nodeExec.output, null, 2)
                      }}</pre>
                    </div>

                    <!-- Error -->
                    <div v-if="nodeExec.error">
                      <div class="mb-2 text-sm font-medium text-red-700">
                        Error
                      </div>
                      <pre class="code-block-sm bg-red-50 text-red-900">{{
                        nodeExec.error
                      }}</pre>
                    </div>

                    <!-- Metadata -->
                    <div
                      v-if="
                        nodeExec.metadata &&
                        Object.keys(nodeExec.metadata).length > 0
                      "
                    >
                      <div class="mb-2 flex items-center justify-between">
                        <div class="text-sm font-medium text-gray-700">
                          Metadata
                        </div>
                        <button
                          @click.stop="
                            copyToClipboard(
                              JSON.stringify(nodeExec.metadata, null, 2),
                            )
                          "
                          class="text-gray-400 hover:text-gray-600"
                          title="Copy JSON"
                        >
                          <Icon
                            icon="heroicons:clipboard-document"
                            class="size-4"
                          />
                        </button>
                      </div>
                      <pre class="code-block-sm bg-purple-50">{{
                        JSON.stringify(nodeExec.metadata, null, 2)
                      }}</pre>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Card>

      <!-- Metadata -->
      <Card
        v-if="execution.metadata && Object.keys(execution.metadata).length > 0"
      >
        <div class="border-b border-gray-200 px-6 py-4">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900">Metadata</h2>
            <button
              @click="
                copyToClipboard(JSON.stringify(execution.metadata, null, 2))
              "
              class="text-gray-400 hover:text-gray-600"
              title="Copy JSON"
            >
              <Icon icon="heroicons:clipboard-document" class="size-4" />
            </button>
          </div>
        </div>
        <div class="p-6">
          <pre class="code-block">{{
            JSON.stringify(execution.metadata, null, 2)
          }}</pre>
        </div>
      </Card>

      <!-- Events & Logs -->
      <Card>
        <div class="border-b border-gray-200 px-6 py-4">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900">
              Events & Logs
              <span
                v-if="!isLoadingLogs"
                class="ml-2 text-sm font-normal text-gray-500"
                >({{ allEvents.length }})</span
              >
            </h2>
            <Button
              variant="secondary"
              size="sm"
              @click="loadLogs"
              :loading="isLoadingLogs"
            >
              <Icon icon="heroicons:arrow-path" class="mr-2 size-4" />
              Refresh
            </Button>
          </div>
        </div>

        <!-- Loading State -->
        <div
          v-if="isLoadingLogs"
          class="flex items-center justify-center py-12"
        >
          <Icon
            icon="heroicons:arrow-path"
            class="size-8 animate-spin text-blue-500"
          />
        </div>

        <!-- Events Timeline -->
        <div v-else-if="allEvents.length > 0" class="divide-y divide-gray-100">
          <div
            v-for="(event, index) in allEvents"
            :key="index"
            class="group relative px-6 py-4 transition-colors hover:bg-gray-50"
          >
            <div class="flex items-start gap-4">
              <!-- Timeline indicator -->
              <div class="flex flex-col items-center">
                <div
                  class="flex size-8 shrink-0 items-center justify-center rounded-full"
                  :class="{
                    'bg-blue-100': event.level === 'info',
                    'bg-green-100': event.level === 'success',
                    'bg-yellow-100': event.level === 'warning',
                    'bg-red-100': event.level === 'error',
                  }"
                >
                  <Icon
                    :icon="getEventIcon(event.event_type)"
                    class="size-4"
                    :class="{
                      'text-blue-600': event.level === 'info',
                      'text-green-600': event.level === 'success',
                      'text-yellow-600': event.level === 'warning',
                      'text-red-600': event.level === 'error',
                    }"
                  />
                </div>
                <div
                  v-if="index < allEvents.length - 1"
                  class="w-0.5 flex-1 bg-gray-200"
                  style="min-height: 20px"
                ></div>
              </div>

              <!-- Event content -->
              <div class="min-w-0 flex-1">
                <div class="flex items-start justify-between gap-4">
                  <div class="flex-1">
                    <div class="flex items-center gap-2">
                      <p class="font-medium text-gray-900">
                        {{ event.message }}
                      </p>
                      <Badge
                        :variant="
                          event.level === 'error'
                            ? 'danger'
                            : event.level === 'success'
                              ? 'success'
                              : event.level === 'warning'
                                ? 'warning'
                                : 'info'
                        "
                        size="sm"
                      >
                        {{ event.event_type }}
                      </Badge>
                    </div>
                    <p class="mt-1 text-xs text-gray-500">
                      {{ formatTimestamp(event.timestamp) }}
                      <span class="mx-2">•</span>
                      {{ formatRelativeTime(event.timestamp) }}
                    </p>
                  </div>

                  <!-- Expand button -->
                  <button
                    v-if="Object.keys(event.data).length > 0"
                    @click="toggleEventDetails(index)"
                    class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                  >
                    <Icon
                      :icon="
                        selectedEventIndex === index
                          ? 'heroicons:chevron-up'
                          : 'heroicons:chevron-down'
                      "
                      class="size-5"
                    />
                  </button>
                </div>

                <!-- Expanded event details -->
                <div
                  v-if="
                    selectedEventIndex === index &&
                    Object.keys(event.data).length > 0
                  "
                  class="mt-4"
                >
                  <div class="rounded-lg bg-gray-50 p-4">
                    <div class="mb-2 flex items-center justify-between">
                      <span class="text-sm font-medium text-gray-700"
                        >Event Data</span
                      >
                      <button
                        @click="
                          copyToClipboard(JSON.stringify(event.data, null, 2))
                        "
                        class="text-gray-400 hover:text-gray-600"
                        title="Copy JSON"
                      >
                        <Icon
                          icon="heroicons:clipboard-document"
                          class="size-4"
                        />
                      </button>
                    </div>
                    <pre class="code-block-sm">{{
                      JSON.stringify(event.data, null, 2)
                    }}</pre>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="px-6 py-12 text-center">
          <Icon
            icon="heroicons:document-text"
            class="mx-auto size-12 text-gray-400"
          />
          <p class="mt-4 text-sm text-gray-600">No events or logs available</p>
        </div>
      </Card>

      <!-- Timestamps -->
      <Card>
        <div class="border-b border-gray-200 px-6 py-4">
          <h2 class="text-lg font-semibold text-gray-900">Timestamps</h2>
        </div>
        <div class="p-6">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div>
              <div class="text-sm font-medium text-gray-500">Created</div>
              <div class="mt-1 text-sm text-gray-900">
                {{ formatTimestamp(execution.created_at) }}
              </div>
            </div>
            <div>
              <div class="text-sm font-medium text-gray-500">Updated</div>
              <div class="mt-1 text-sm text-gray-900">
                {{ formatTimestamp(execution.updated_at) }}
              </div>
            </div>
            <div>
              <div class="text-sm font-medium text-gray-500">Last Modified</div>
              <div class="mt-1 text-sm text-gray-900">
                {{ formatRelativeTime(execution.updated_at) }}
              </div>
            </div>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>

<style scoped>
.execution-details-page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 2rem;
}

.code-block {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 0.75rem;
  line-height: 1.5;
  padding: 1rem;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  overflow: auto;
  max-height: 400px;
  white-space: pre-wrap;
  word-break: break-word;
}

.code-block-sm {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 0.7rem;
  line-height: 1.4;
  padding: 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  overflow: auto;
  max-height: 300px;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
