<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { Icon } from "@iconify/vue";
import AppShell from "@/components/layout/AppShell.vue";
import Button from "@/components/ui/Button.vue";
import { deleteWorkflow, getWorkflows } from "@/api/workflows";
import type { Workflow } from "@/types/workflow";

const router = useRouter();

const workflows = ref<Workflow[]>([]);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Load workflows on mount
onMounted(async () => {
  await loadWorkflows();
});

async function loadWorkflows() {
  isLoading.value = true;
  error.value = null;

  try {
    const response = await getWorkflows();
    workflows.value = response.workflows || [];
  } catch (err: any) {
    console.error("Failed to load workflows:", err);
    error.value = err.message || "Failed to load workflows";
  } finally {
    isLoading.value = false;
  }
}

async function handleDelete(workflowId: string) {
  if (!confirm("Are you sure you want to delete this workflow?")) {
    return;
  }

  try {
    await deleteWorkflow(workflowId);
    workflows.value = workflows.value.filter((w) => w.id !== workflowId);
  } catch (err: any) {
    console.error("Failed to delete workflow:", err);
    alert("Failed to delete workflow: " + (err.message || "Unknown error"));
  }
}

function getStatusColor(status: string) {
  const colors: Record<string, string> = {
    draft: "bg-gray-100 text-gray-700",
    active: "bg-green-100 text-green-700",
    inactive: "bg-yellow-100 text-yellow-700",
    archived: "bg-red-100 text-red-700",
  };
  return colors[status] || colors.draft;
}

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleDateString();
}
</script>

<template>
  <AppShell>
    <div class="mx-auto max-w-7xl">
      <div class="mb-6 flex items-center justify-between">
        <h1 class="text-3xl font-bold text-gray-900">Workflows</h1>
        <Button @click="router.push('/workflows/new')"> New Workflow </Button>
      </div>

      <!-- Loading state -->
      <div v-if="isLoading" class="card">
        <div class="py-12 text-center">
          <div
            class="mx-auto size-12 animate-spin rounded-full border-b-2 border-blue-600"
          />
          <p class="mt-4 text-gray-600">Loading workflows...</p>
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
            <Button @click="loadWorkflows"> Retry </Button>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div v-else-if="workflows.length === 0" class="card">
        <div class="py-12 text-center">
          <svg
            class="mx-auto size-12 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
            />
          </svg>
          <h3 class="mt-2 text-sm font-medium text-gray-900">No workflows</h3>
          <p class="mt-1 text-sm text-gray-500">
            Get started by creating a new workflow.
          </p>
          <div class="mt-6">
            <Button @click="router.push('/workflows/new')">
              Create Workflow
            </Button>
          </div>
        </div>
      </div>

      <!-- Workflows table -->
      <div v-else class="card overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Name
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Status
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Nodes
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Updated
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
              v-for="workflow in workflows"
              :key="workflow.id"
              class="transition-colors hover:bg-gray-50"
            >
              <td class="whitespace-nowrap px-6 py-4">
                <div class="text-sm font-medium text-gray-900">
                  {{ workflow.name }}
                </div>
                <div
                  v-if="workflow.description"
                  class="max-w-md truncate text-sm text-gray-500"
                >
                  {{ workflow.description }}
                </div>
              </td>
              <td class="whitespace-nowrap px-6 py-4">
                <span
                  :class="[
                    getStatusColor(workflow.status),
                    'rounded px-2 py-1 text-xs font-semibold',
                  ]"
                >
                  {{ workflow.status }}
                </span>
              </td>
              <td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
                {{ workflow.nodes?.length || 0 }}
              </td>
              <td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
                {{ formatDate(workflow.updated_at) }}
              </td>
              <td
                class="whitespace-nowrap px-6 py-4 text-right text-sm font-medium"
              >
                <div class="flex items-center justify-end gap-2">
                  <button
                    class="text-blue-600 transition-colors hover:text-blue-900"
                    @click="router.push(`/workflows/${workflow.id}`)"
                  >
                    <Icon icon="heroicons:pencil" class="size-5" />
                  </button>
                  <button
                    class="text-red-600 transition-colors hover:text-red-900"
                    @click="handleDelete(workflow.id)"
                  >
                    <Icon icon="heroicons:trash" class="size-5" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </AppShell>
</template>
