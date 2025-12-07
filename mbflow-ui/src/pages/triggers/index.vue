<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { Icon } from "@iconify/vue";
import { toast } from "vue3-toastify";
import AppShell from "@/components/layout/AppShell.vue";
import Button from "@/components/ui/Button.vue";
import {
  getTriggers,
  deleteTrigger,
  enableTrigger,
  disableTrigger,
  executeTrigger,
} from "@/api/triggers";
import type { Trigger, TriggerType, TriggerStatus } from "@/types/trigger";

const router = useRouter();

const triggers = ref<Trigger[]>([]);
const isLoading = ref(true);
const error = ref<string | null>(null);
const typeFilter = ref<TriggerType | "all">("all");
const statusFilter = ref<TriggerStatus | "all">("all");

onMounted(async () => {
  await loadTriggers();
});

async function loadTriggers() {
  isLoading.value = true;
  error.value = null;

  try {
    const params: any = {};
    if (typeFilter.value !== "all") params.type = typeFilter.value;
    if (statusFilter.value !== "all") params.status = statusFilter.value;

    const response = await getTriggers(params);
    triggers.value = response.triggers || [];
  } catch (err: any) {
    console.error("Failed to load triggers:", err);
    error.value = err.message || "Failed to load triggers";
  } finally {
    isLoading.value = false;
  }
}

async function handleDelete(triggerId: string) {
  if (!confirm("Are you sure you want to delete this trigger?")) {
    return;
  }

  try {
    await deleteTrigger(triggerId);
    triggers.value = triggers.value.filter((t) => t.id !== triggerId);
    toast.success("Trigger deleted successfully");
  } catch (err: any) {
    console.error("Failed to delete trigger:", err);
    toast.error(
      "Failed to delete trigger: " + (err.message || "Unknown error"),
    );
  }
}

async function handleToggle(trigger: Trigger) {
  try {
    if (trigger.status === "enabled") {
      await disableTrigger(trigger.id);
      trigger.status = "disabled";
      toast.success("Trigger disabled successfully");
    } else {
      await enableTrigger(trigger.id);
      trigger.status = "enabled";
      toast.success("Trigger enabled successfully");
    }
  } catch (err: any) {
    console.error("Failed to toggle trigger:", err);
    toast.error(
      "Failed to toggle trigger: " + (err.message || "Unknown error"),
    );
  }
}

async function handleExecute(trigger: Trigger) {
  try {
    const result = await executeTrigger(trigger.id);
    toast.success(`Trigger executed! Execution ID: ${result.execution_id}`);
    router.push(`/executions/${result.execution_id}`);
  } catch (err: any) {
    console.error("Failed to execute trigger:", err);
    toast.error(
      "Failed to execute trigger: " + (err.message || "Unknown error"),
    );
  }
}

function getTriggerTypeIcon(type: TriggerType) {
  const icons: Record<TriggerType, string> = {
    manual: "heroicons:hand-raised",
    schedule: "heroicons:clock",
    webhook: "heroicons:globe-alt",
    event: "heroicons:bolt",
  };
  return icons[type] || icons.manual;
}

function getTriggerTypeColor(type: TriggerType) {
  const colors: Record<TriggerType, string> = {
    manual: "bg-gray-100 text-gray-700",
    schedule: "bg-blue-100 text-blue-700",
    webhook: "bg-green-100 text-green-700",
    event: "bg-purple-100 text-purple-700",
  };
  return colors[type] || colors.manual;
}

function getStatusColor(status: TriggerStatus) {
  return status === "enabled"
    ? "bg-green-100 text-green-700"
    : "bg-gray-100 text-gray-700";
}

function formatDate(dateString?: string) {
  if (!dateString) return "Never";
  return new Date(dateString).toLocaleString();
}

async function handleFilterChange() {
  await loadTriggers();
}
</script>

<template>
  <AppShell>
    <div class="mx-auto max-w-7xl">
      <div class="mb-6 flex items-center justify-between">
        <h1 class="text-3xl font-bold text-gray-900">Triggers</h1>

        <div class="flex items-center gap-4">
          <Button @click="router.push('/triggers/new')"> New Trigger </Button>
          <!-- Filters -->
          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600">Type:</label>
            <select
              v-model="typeFilter"
              class="rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              @change="handleFilterChange"
            >
              <option value="all">All</option>
              <option value="manual">Manual</option>
              <option value="schedule">Schedule</option>
              <option value="webhook">Webhook</option>
              <option value="event">Event</option>
            </select>
          </div>

          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600">Status:</label>
            <select
              v-model="statusFilter"
              class="rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              @change="handleFilterChange"
            >
              <option value="all">All</option>
              <option value="enabled">Enabled</option>
              <option value="disabled">Disabled</option>
            </select>
          </div>

          <Button variant="secondary" size="sm" @click="loadTriggers">
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
          <p class="mt-4 text-gray-600">Loading triggers...</p>
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
            <Button @click="loadTriggers"> Retry </Button>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div v-else-if="triggers.length === 0" class="card">
        <div class="py-12 text-center">
          <Icon
            icon="heroicons:bell-alert"
            class="mx-auto size-12 text-gray-400"
          />
          <h3 class="mt-2 text-sm font-medium text-gray-900">No triggers</h3>
          <p class="mt-1 text-sm text-gray-500">
            No triggers found{{
              typeFilter !== "all" || statusFilter !== "all"
                ? " with selected filters"
                : ""
            }}.
          </p>
        </div>
      </div>

      <!-- Triggers table -->
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
                Workflow
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Type
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Status
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                Last Triggered
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
              v-for="trigger in triggers"
              :key="trigger.id"
              class="transition-colors hover:bg-gray-50"
            >
              <td class="px-6 py-4">
                <div class="text-sm font-medium text-gray-900">
                  {{ trigger.name }}
                </div>
                <div
                  v-if="trigger.description"
                  class="max-w-md truncate text-sm text-gray-500"
                >
                  {{ trigger.description }}
                </div>
              </td>
              <td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
                {{ trigger.workflow_name || trigger.workflow_id }}
              </td>
              <td class="whitespace-nowrap px-6 py-4">
                <div class="flex items-center gap-2">
                  <Icon
                    :icon="getTriggerTypeIcon(trigger.type)"
                    class="size-4"
                  />
                  <span
                    :class="[
                      getTriggerTypeColor(trigger.type),
                      'rounded px-2 py-1 text-xs font-semibold',
                    ]"
                  >
                    {{ trigger.type }}
                  </span>
                </div>
              </td>
              <td class="whitespace-nowrap px-6 py-4">
                <button
                  :class="[
                    getStatusColor(trigger.status),
                    'cursor-pointer rounded px-2 py-1 text-xs font-semibold transition-opacity hover:opacity-80',
                  ]"
                  @click="handleToggle(trigger)"
                >
                  {{ trigger.status }}
                </button>
              </td>
              <td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
                {{ formatDate(trigger.last_triggered_at) }}
              </td>
              <td
                class="whitespace-nowrap px-6 py-4 text-right text-sm font-medium"
              >
                <div class="flex items-center justify-end gap-2">
                  <button
                    class="text-blue-600 transition-colors hover:text-blue-900"
                    title="Execute trigger"
                    @click="handleExecute(trigger)"
                  >
                    <Icon icon="heroicons:play" class="size-5" />
                  </button>
                  <button
                    class="text-red-600 transition-colors hover:text-red-900"
                    title="Delete trigger"
                    @click="handleDelete(trigger.id)"
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
