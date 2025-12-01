<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { Icon } from "@iconify/vue";
import AppShell from "@/components/layout/AppShell.vue";
import Button from "@/components/ui/Button.vue";
import { getExecutions } from "@/api/executions";
import type { Execution, ExecutionStatus } from "@/types/execution";

const router = useRouter();

const executions = ref<Execution[]>([]);
const isLoading = ref(true);
const error = ref<string | null>(null);
const statusFilter = ref<ExecutionStatus | "all">("all");

onMounted(async () => {
  await loadExecutions();
});

async function loadExecutions() {
  isLoading.value = true;
  error.value = null;

  try {
    const params =
      statusFilter.value !== "all" ? { status: statusFilter.value } : {};
    const response = await getExecutions(params);
    executions.value = response.executions || [];
  } catch (err: any) {
    console.error("Failed to load executions:", err);
    error.value = err.message || "Failed to load executions";
  } finally {
    isLoading.value = false;
  }
}

function getStatusColor(status: ExecutionStatus) {
  const colors: Record<ExecutionStatus, string> = {
    pending: "bg-gray-100 text-gray-700",
    running: "bg-blue-100 text-blue-700",
    completed: "bg-green-100 text-green-700",
    failed: "bg-red-100 text-red-700",
    cancelled: "bg-orange-100 text-orange-700",
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
    return `${hours}h ${minutes % 60}m`;
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  } else {
    return `${seconds}s`;
  }
}

async function handleFilterChange() {
  await loadExecutions();
}
</script>

<template>
  <AppShell>
    <div class="mx-auto max-w-7xl">
      <div class="mb-6 flex items-center justify-between">
        <h1 class="text-3xl font-bold text-gray-900">Executions</h1>

        <!-- Filter -->
        <div class="flex items-center gap-2">
          <label class="text-sm text-gray-600">Status:</label>
          <select
            v-model="statusFilter"
            class="rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            @change="handleFilterChange"
          >
            <option value="all">All</option>
            <option value="pending">Pending</option>
            <option value="running">Running</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
            <option value="cancelled">Cancelled</option>
          </select>
          <Button variant="secondary" size="sm" @click="loadExecutions">
            <Icon icon="heroicons:arrow-path" class="size-4" />
          </Button>
        </div>
      </div>

      <!-- Loading state -->
      <div v-if="isLoading" class="card">
        <div class="py-12 text-center">
          <div
            class="mx-auto size-12 animate-spin rounded-full border-b-2 border-blue-600"
          />
          <p class="mt-4 text-gray-600">Loading executions...</p>
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
            <Button @click="loadExecutions"> Retry </Button>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div v-else-if="executions.length === 0" class="card">
        <div class="py-12 text-center">
          <Icon
            icon="heroicons:queue-list"
            class="mx-auto size-12 text-gray-400"
          />
          <h3 class="mt-2 text-sm font-medium text-gray-900">No executions</h3>
          <p class="mt-1 text-sm text-gray-500">
            No workflow executions found{{
              statusFilter !== "all" ? ` with status "${statusFilter}"` : ""
            }}.
          </p>
        </div>
      </div>

      <!-- Executions table -->
      <div v-else class="card overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Workflow
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Status
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Started
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Duration
              </th>
              <th
                class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 bg-white">
            <tr
              v-for="execution in executions"
              :key="execution.id"
              class="cursor-pointer transition-colors hover:bg-gray-50"
              @click="router.push(`/executions/${execution.id}`)"
            >
              <td class="px-6 py-4">
                <div class="text-sm font-medium text-gray-900">
                  {{ execution.workflow_name || execution.workflow_id }}
                </div>
                <div class="font-mono text-sm text-gray-500">
                  {{ execution.id.substring(0, 8) }}
                </div>
              </td>
              <td class="whitespace-nowrap px-6 py-4">
                <div class="flex items-center gap-2">
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
              </td>
              <td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
                {{ formatDate(execution.started_at) }}
              </td>
              <td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
                {{
                  formatDuration(execution.started_at, execution.completed_at)
                }}
              </td>
              <td
                class="whitespace-nowrap px-6 py-4 text-right text-sm font-medium"
                @click.stop
              >
                <button
                  class="text-blue-600 transition-colors hover:text-blue-900"
                  @click="router.push(`/executions/${execution.id}`)"
                >
                  <Icon icon="heroicons:eye" class="size-5" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </AppShell>
</template>
