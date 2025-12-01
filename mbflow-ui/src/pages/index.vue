<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import AppShell from "@/components/layout/AppShell.vue";
import Button from "@/components/ui/Button.vue";
import { Icon } from "@iconify/vue";
import { getWorkflows } from "@/api/workflows";
import { getExecutions } from "@/api/executions";

const router = useRouter();

const stats = ref({
  workflows: 0,
  active: 0,
  executions: 0,
  successRate: "-",
});

const recentExecutions = ref<any[]>([]);
const isLoading = ref(true);

onMounted(async () => {
  await loadData();
});

async function loadData() {
  isLoading.value = true;

  try {
    // Load workflows
    const workflowsResponse = await getWorkflows({ limit: 100 });
    stats.value.workflows =
      workflowsResponse.total || workflowsResponse.workflows.length;
    stats.value.active = workflowsResponse.workflows.filter(
      (w) => w.status === "active",
    ).length;

    // Load executions
    const executionsResponse = await getExecutions({ limit: 10 });
    stats.value.executions =
      executionsResponse.total || executionsResponse.executions.length;
    recentExecutions.value = executionsResponse.executions.slice(0, 5);

    // Calculate success rate
    if (executionsResponse.executions.length > 0) {
      const completed = executionsResponse.executions.filter(
        (e) => e.status === "completed",
      ).length;
      const total = executionsResponse.executions.length;
      stats.value.successRate = `${Math.round((completed / total) * 100)}%`;
    }
  } catch (err) {
    console.error("Failed to load dashboard data:", err);
  } finally {
    isLoading.value = false;
  }
}

function getStatusColor(status: string) {
  const colors: Record<string, string> = {
    pending: "bg-gray-100 text-gray-700",
    running: "bg-blue-100 text-blue-700",
    completed: "bg-green-100 text-green-700",
    failed: "bg-red-100 text-red-700",
    cancelled: "bg-orange-100 text-orange-700",
  };
  return colors[status] || colors.pending;
}

function formatDate(dateString: string) {
  const date = new Date(dateString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `${days}d ago`;
  if (hours > 0) return `${hours}h ago`;
  if (minutes > 0) return `${minutes}m ago`;
  return "just now";
}
</script>

<template>
  <AppShell>
    <div class="mx-auto max-w-7xl">
      <h1 class="mb-6 text-3xl font-bold text-gray-900">Dashboard</h1>

      <!-- Stats -->
      <div class="mb-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <div
          class="card cursor-pointer transition-shadow hover:shadow-md"
          @click="router.push('/workflows')"
        >
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm font-medium text-gray-500">Workflows</div>
              <div class="mt-2 text-3xl font-semibold text-gray-900">
                {{ isLoading ? "-" : stats.workflows }}
              </div>
            </div>
            <Icon icon="heroicons:squares-2x2" class="size-8 text-blue-500" />
          </div>
        </div>

        <div
          class="card cursor-pointer transition-shadow hover:shadow-md"
          @click="router.push('/workflows')"
        >
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm font-medium text-gray-500">Active</div>
              <div class="mt-2 text-3xl font-semibold text-gray-900">
                {{ isLoading ? "-" : stats.active }}
              </div>
            </div>
            <Icon icon="heroicons:check-circle" class="size-8 text-green-500" />
          </div>
        </div>

        <div
          class="card cursor-pointer transition-shadow hover:shadow-md"
          @click="router.push('/executions')"
        >
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm font-medium text-gray-500">Executions</div>
              <div class="mt-2 text-3xl font-semibold text-gray-900">
                {{ isLoading ? "-" : stats.executions }}
              </div>
            </div>
            <Icon icon="heroicons:play-circle" class="size-8 text-purple-500" />
          </div>
        </div>

        <div class="card">
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm font-medium text-gray-500">Success Rate</div>
              <div class="mt-2 text-3xl font-semibold text-gray-900">
                {{ isLoading ? "-" : stats.successRate }}
              </div>
            </div>
            <Icon icon="heroicons:chart-bar" class="size-8 text-orange-500" />
          </div>
        </div>
      </div>

      <!-- Recent Executions -->
      <div class="card">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900">Recent Executions</h2>
          <Button
            variant="secondary"
            size="sm"
            @click="router.push('/executions')"
          >
            View All
          </Button>
        </div>

        <div v-if="isLoading" class="py-8 text-center">
          <div
            class="mx-auto size-8 animate-spin rounded-full border-b-2 border-blue-600"
          />
        </div>

        <div
          v-else-if="recentExecutions.length === 0"
          class="py-8 text-center text-gray-500"
        >
          No executions yet. Execute a workflow to see results here.
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="execution in recentExecutions"
            :key="execution.id"
            class="flex cursor-pointer items-center justify-between rounded-lg border border-gray-200 p-3 transition-colors hover:border-blue-300"
            @click="router.push(`/executions/${execution.id}`)"
          >
            <div class="flex flex-1 items-center gap-3">
              <span
                :class="[
                  getStatusColor(execution.status),
                  'rounded px-2 py-1 text-xs font-semibold',
                ]"
              >
                {{ execution.status }}
              </span>
              <div class="min-w-0 flex-1">
                <div class="truncate text-sm font-medium text-gray-900">
                  {{ execution.workflow_name || execution.workflow_id }}
                </div>
                <div class="text-xs text-gray-500">
                  {{ formatDate(execution.started_at) }}
                </div>
              </div>
            </div>
            <Icon icon="heroicons:chevron-right" class="size-5 text-gray-400" />
          </div>
        </div>
      </div>

      <!-- Welcome card -->
      <div class="card mt-6">
        <h2 class="mb-4 text-lg font-semibold text-gray-900">
          Welcome to MBFlow
        </h2>
        <p class="mb-4 text-gray-600">
          MBFlow is a sophisticated workflow orchestration engine with DAG-based
          workflow automation.
        </p>
        <div class="flex gap-3">
          <Button @click="router.push('/workflows/new')">
            Create Workflow
          </Button>
          <Button variant="secondary" @click="router.push('/workflows')">
            Browse Workflows
          </Button>
        </div>
      </div>
    </div>
  </AppShell>
</template>
