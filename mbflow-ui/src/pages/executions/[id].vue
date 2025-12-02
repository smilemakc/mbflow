<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Icon } from "@iconify/vue";
import { toast } from "vue3-toastify";
import AppShell from "@/components/layout/AppShell.vue";
import Button from "@/components/ui/Button.vue";
import { getExecution, cancelExecution } from "@/api/executions";
import type { Execution, ExecutionStatus } from "@/types/execution";

const route = useRoute();
const router = useRouter();

const executionId = computed(() => route.params.id as string);
const execution = ref<Execution | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);
const isCancelling = ref(false);

onMounted(async () => {
  await loadExecution();
});

async function loadExecution() {
  isLoading.value = true;
  error.value = null;

  try {
    const response = await getExecution(executionId.value);
    execution.value = response;
  } catch (err: any) {
    console.error("Failed to load execution:", err);
    error.value = err.message || "Failed to load execution";
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
    toast.error("Failed to cancel execution: " + (err.message || "Unknown error"));
  } finally {
    isCancelling.value = false;
  }
}

function getStatusColor(status: ExecutionStatus) {
  const colors: Record<ExecutionStatus, string> = {
    pending: "bg-gray-100 text-gray-700",
    running: "bg-blue-100 text-blue-700",
    completed: "bg-green-100 text-green-700",
    failed: "bg-red-100 text-red-700",
    cancelled: "bg-orange-100 text-orange-700",
    timeout: "bg-yellow-100 text-yellow-700",
  };
  return colors[status] || colors.pending;
}

function getStatusIcon(status: ExecutionStatus) {
  const icons: Record<ExecutionStatus, string> = {
    pending: "heroicons:clock",
    running: "heroicons:arrow-path",
    completed: "heroicons:check-circle",
    failed: "heroicons:x-circle",
    cancelled: "heroicons:stop-circle",
    timeout: "heroicons:clock-circle",
  };
  return icons[status] || icons.pending;
}

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleString();
}

function formatDuration(startedAt: string, completedAt?: string) {
  const start = new Date(startedAt);
  const end = completedAt ? new Date(completedAt) : new Date();
  const diff = end.getTime() - start.getTime();

  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${(seconds % 3600) % 60}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  } else {
    return `${seconds}s`;
  }
}

function formatJSON(obj: any) {
  return JSON.stringify(obj, null, 2);
}
</script>

<template>
  <AppShell>
    <div class="mx-auto max-w-7xl">
      <!-- Header -->
      <div class="mb-6">
        <div class="mb-2 flex items-center gap-4">
          <Button variant="ghost" size="sm" @click="router.push('/executions')">
            <Icon icon="heroicons:arrow-left" class="mr-1 size-4" />
            Back
          </Button>
        </div>
        <div class="flex items-center justify-between">
          <h1 class="text-3xl font-bold text-gray-900">Execution Details</h1>
          <Button
            v-if="execution?.status === 'running'"
            variant="danger"
            size="sm"
            :loading="isCancelling"
            @click="handleCancel"
          >
            <Icon
              v-if="!isCancelling"
              icon="heroicons:stop"
              class="mr-1 size-4"
            />
            Cancel Execution
          </Button>
        </div>
      </div>

      <!-- Loading state -->
      <div v-if="isLoading" class="card">
        <div class="py-12 text-center">
          <div
            class="mx-auto size-12 animate-spin rounded-full border-b-2 border-blue-600"
          />
          <p class="mt-4 text-gray-600">Loading execution...</p>
        </div>
      </div>

      <!-- Error state -->
      <div v-else-if="error" class="card">
        <div class="py-12 text-center">
          <Icon
            icon="heroicons:exclamation-triangle"
            class="mx-auto size-12 text-red-400"
          />
          <h3 class="mt-2 text-sm font-medium text-gray-900">Error</h3>
          <p class="mt-1 text-sm text-gray-500">{{ error }}</p>
          <div class="mt-6">
            <Button @click="loadExecution"> Retry </Button>
          </div>
        </div>
      </div>

      <!-- Execution details -->
      <div v-else-if="execution" class="space-y-6">
        <!-- Overview -->
        <div class="card">
          <h2 class="mb-4 text-lg font-semibold text-gray-900">Overview</h2>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label class="text-sm text-gray-500">Execution ID</label>
              <div class="font-mono text-sm text-gray-900">
                {{ execution.id }}
              </div>
            </div>
            <div>
              <label class="text-sm text-gray-500">Workflow</label>
              <div class="text-sm text-gray-900">
                {{ execution.workflow_name || execution.workflow_id }}
              </div>
            </div>
            <div>
              <label class="text-sm text-gray-500">Status</label>
              <div class="mt-1 flex items-center gap-2">
                <Icon
                  :icon="getStatusIcon(execution.status)"
                  :class="[
                    'size-4',
                    execution.status === 'running' ? 'animate-spin' : '',
                  ]"
                />
                <span
                  :class="[
                    getStatusColor(execution.status),
                    'rounded px-2 py-1 text-xs font-semibold',
                  ]"
                >
                  {{ execution.status }}
                </span>
              </div>
            </div>
            <div>
              <label class="text-sm text-gray-500">Duration</label>
              <div class="text-sm text-gray-900">
                {{
                  formatDuration(execution.started_at, execution.completed_at)
                }}
              </div>
            </div>
            <div>
              <label class="text-sm text-gray-500">Started At</label>
              <div class="text-sm text-gray-900">
                {{ formatDate(execution.started_at) }}
              </div>
            </div>
            <div v-if="execution.completed_at">
              <label class="text-sm text-gray-500">Completed At</label>
              <div class="text-sm text-gray-900">
                {{ formatDate(execution.completed_at) }}
              </div>
            </div>
            <div v-if="execution.triggered_by">
              <label class="text-sm text-gray-500">Triggered By</label>
              <div class="text-sm text-gray-900">
                {{ execution.triggered_by }}
              </div>
            </div>
            <div v-if="execution.strict_mode !== undefined">
              <label class="text-sm text-gray-500">Strict Mode</label>
              <div class="text-sm text-gray-900">
                {{ execution.strict_mode ? "Enabled" : "Disabled" }}
              </div>
            </div>
          </div>

          <!-- Error message -->
          <div
            v-if="execution.error"
            class="mt-4 rounded-md border border-red-200 bg-red-50 p-4"
          >
            <div class="flex items-start gap-2">
              <Icon
                icon="heroicons:exclamation-circle"
                class="mt-0.5 size-5 shrink-0 text-red-600"
              />
              <div class="flex-1">
                <h3 class="text-sm font-medium text-red-800">Error</h3>
                <p class="mt-1 text-sm text-red-700">{{ execution.error }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Runtime Variables -->
        <div
          v-if="execution.variables && Object.keys(execution.variables).length > 0"
          class="card"
        >
          <h2 class="mb-4 text-lg font-semibold text-gray-900">
            Runtime Variables
          </h2>
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200">
              <thead class="bg-gray-50">
                <tr>
                  <th
                    class="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
                  >
                    Name
                  </th>
                  <th
                    class="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
                  >
                    Value
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 bg-white">
                <tr
                  v-for="(value, key) in execution.variables"
                  :key="key as string"
                >
                  <td class="px-4 py-2 font-mono text-sm text-gray-900">
                    {{ key }}
                  </td>
                  <td class="px-4 py-2 text-sm text-gray-700">
                    {{ typeof value === 'object' ? JSON.stringify(value) : value }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Node Executions -->
        <div
          v-if="
            execution.node_executions && execution.node_executions.length > 0
          "
          class="card"
        >
          <h2 class="mb-4 text-lg font-semibold text-gray-900">
            Node Executions
          </h2>
          <div class="space-y-4">
            <div
              v-for="nodeExec in execution.node_executions"
              :key="nodeExec.id"
              class="rounded-lg border border-gray-200 p-4"
            >
              <!-- Node Header -->
              <div class="mb-3 flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <div>
                    <div class="flex items-center gap-2">
                      <span class="font-medium text-gray-900">
                        {{ nodeExec.node_name || nodeExec.node_id }}
                      </span>
                      <span
                        v-if="nodeExec.node_type"
                        class="rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600"
                      >
                        {{ nodeExec.node_type }}
                      </span>
                    </div>
                    <div class="mt-1 font-mono text-xs text-gray-500">
                      {{ nodeExec.node_id }}
                    </div>
                  </div>
                  <span
                    :class="[
                      getStatusColor(nodeExec.status),
                      'rounded px-2 py-1 text-xs font-semibold',
                    ]"
                  >
                    {{ nodeExec.status }}
                  </span>
                  <span
                    v-if="nodeExec.retry_count && nodeExec.retry_count > 0"
                    class="text-xs text-gray-500"
                  >
                    ({{ nodeExec.retry_count }} {{ nodeExec.retry_count === 1 ? 'retry' : 'retries' }})
                  </span>
                </div>
                <div class="text-xs text-gray-500">
                  {{
                    formatDuration(nodeExec.started_at, nodeExec.completed_at)
                  }}
                </div>
              </div>

              <!-- Error -->
              <div
                v-if="nodeExec.error"
                class="mb-3 rounded-md border border-red-200 bg-red-50 p-3"
              >
                <div class="flex items-start gap-2">
                  <Icon
                    icon="heroicons:exclamation-circle"
                    class="mt-0.5 size-4 shrink-0 text-red-600"
                  />
                  <div class="flex-1">
                    <p class="text-sm text-red-700">{{ nodeExec.error }}</p>
                  </div>
                </div>
              </div>

              <!-- Input/Output -->
              <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                <div v-if="nodeExec.input">
                  <label class="mb-2 block text-xs font-semibold text-gray-700">
                    Input
                  </label>
                  <pre
                    class="max-h-48 overflow-auto rounded-md bg-gray-50 p-3 text-xs"
                    >{{ formatJSON(nodeExec.input) }}</pre
                  >
                </div>
                <div v-if="nodeExec.output">
                  <label class="mb-2 block text-xs font-semibold text-gray-700">
                    Output
                  </label>
                  <pre
                    class="max-h-48 overflow-auto rounded-md bg-gray-50 p-3 text-xs"
                    >{{ formatJSON(nodeExec.output) }}</pre
                  >
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Input -->
        <div v-if="execution.input" class="card">
          <h2 class="mb-4 text-lg font-semibold text-gray-900">Input</h2>
          <pre class="overflow-x-auto rounded-md bg-gray-50 p-4 text-xs">{{
            formatJSON(execution.input)
          }}</pre>
        </div>

        <!-- Output -->
        <div v-if="execution.output" class="card">
          <h2 class="mb-4 text-lg font-semibold text-gray-900">Output</h2>
          <pre class="overflow-x-auto rounded-md bg-gray-50 p-4 text-xs">{{
            formatJSON(execution.output)
          }}</pre>
        </div>

        <!-- Metadata -->
        <div
          v-if="execution.metadata && Object.keys(execution.metadata).length > 0"
          class="card"
        >
          <h2 class="mb-4 text-lg font-semibold text-gray-900">Metadata</h2>
          <pre class="overflow-x-auto rounded-md bg-gray-50 p-4 text-xs">{{
            formatJSON(execution.metadata)
          }}</pre>
        </div>
      </div>
    </div>
  </AppShell>
</template>
