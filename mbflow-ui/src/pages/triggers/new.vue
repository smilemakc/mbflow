<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useForm } from "vee-validate";
import { toTypedSchema } from "@vee-validate/yup";
import { toast } from "vue3-toastify";
import AppShell from "@/components/layout/AppShell.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import { createTrigger } from "@/api/triggers";
import { getWorkflows } from "@/api/workflows";
import {
  triggerBaseSchema,
  scheduleConfigSchema,
  webhookConfigSchema,
  eventConfigSchema,
} from "@/validation";
import type { Workflow } from "@/types/workflow";
import type { TriggerType, TriggerStatus } from "@/types/trigger";

const router = useRouter();

const workflows = ref<Workflow[]>([]);
const isLoadingWorkflows = ref(true);

// Initialize base form
const { handleSubmit, errors, defineField, isSubmitting, setFieldError } =
  useForm({
    validationSchema: toTypedSchema(triggerBaseSchema),
    initialValues: {
      workflow_id: "",
      name: "",
      description: "",
      type: "manual" as TriggerType,
      status: "enabled" as TriggerStatus,
    },
  });

// Define base fields
const [workflowId] = defineField("workflow_id");
const [name] = defineField("name");
const [description] = defineField("description");
const [type] = defineField("type");
const [status] = defineField("status");

// Type-specific config forms
const scheduleForm = useForm({
  validationSchema: toTypedSchema(scheduleConfigSchema),
  initialValues: {
    cron: "0 0 * * *",
    timezone: "UTC",
  },
});
const [scheduleCron] = scheduleForm.defineField("cron");
const [scheduleTimezone] = scheduleForm.defineField("timezone");

const webhookForm = useForm({
  validationSchema: toTypedSchema(webhookConfigSchema),
  initialValues: {
    webhook_path: "",
    http_method: "POST",
    auth_type: "none",
    auth_config: {},
  },
});
const [webhookPath] = webhookForm.defineField("webhook_path");
const [webhookMethod] = webhookForm.defineField("http_method");
const [webhookAuthType] = webhookForm.defineField("auth_type");

const eventForm = useForm({
  validationSchema: toTypedSchema(eventConfigSchema),
  initialValues: {
    event_type: "",
    event_filter: {},
  },
});
const [eventType] = eventForm.defineField("event_type");

onMounted(async () => {
  await loadWorkflows();
});

async function loadWorkflows() {
  isLoadingWorkflows.value = true;
  try {
    const response = await getWorkflows({ limit: 100 });
    workflows.value = response.workflows || [];
  } catch (err: any) {
    console.error("Failed to load workflows:", err);
    toast.error("Failed to load workflows");
  } finally {
    isLoadingWorkflows.value = false;
  }
}

// Submit handler
const onSubmit = handleSubmit(async (baseValues) => {
  try {
    let config: Record<string, any> = {};

    // Validate and get type-specific config
    if (baseValues.type === "schedule") {
      const scheduleValues = await scheduleForm.validate();
      if (!scheduleValues.valid) {
        toast.error("Please fix schedule configuration errors");
        return;
      }
      config = scheduleValues.values || {};
    } else if (baseValues.type === "webhook") {
      const webhookValues = await webhookForm.validate();
      if (!webhookValues.valid) {
        toast.error("Please fix webhook configuration errors");
        return;
      }
      config = webhookValues.values || {};
    } else if (baseValues.type === "event") {
      const eventValues = await eventForm.validate();
      if (!eventValues.valid) {
        toast.error("Please fix event configuration errors");
        return;
      }
      config = eventValues.values || {};
    }

    // Create trigger
    await createTrigger({
      workflow_id: baseValues.workflow_id,
      name: baseValues.name,
      description: baseValues.description,
      type: baseValues.type,
      status: baseValues.status as TriggerStatus,
      config,
    });

    toast.success("Trigger created successfully!");
    router.push("/triggers");
  } catch (error: any) {
    console.error("Failed to create trigger:", error);
    const errorMessage = error.message || "Failed to create trigger";
    setFieldError("name", errorMessage);
    toast.error(errorMessage);
  }
});

function handleCancel() {
  router.push("/triggers");
}

function getTriggerTypeDescription(triggerType: TriggerType): string {
  const descriptions: Record<TriggerType, string> = {
    manual: "Trigger workflow manually from the UI or API",
    schedule: "Trigger workflow on a schedule using cron expression",
    webhook: "Trigger workflow via HTTP webhook",
    event: "Trigger workflow when a specific event occurs",
  };
  return descriptions[triggerType] || "";
}
</script>

<template>
  <AppShell>
    <div class="mx-auto max-w-3xl">
      <div class="mb-6">
        <h1 class="text-3xl font-bold text-gray-900">Create New Trigger</h1>
        <p class="mt-2 text-gray-600">
          Create a trigger to automatically execute workflows.
        </p>
      </div>

      <div class="card">
        <form @submit="onSubmit" class="space-y-6">
          <!-- Workflow selection -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Workflow <span class="text-red-500">*</span>
            </label>
            <select
              v-model="workflowId"
              :disabled="isLoadingWorkflows"
              class="w-full rounded-md border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              :class="{
                'border-gray-300': !errors.workflow_id,
                'border-red-300': errors.workflow_id,
              }"
            >
              <option value="">Select a workflow...</option>
              <option
                v-for="workflow in workflows"
                :key="workflow.id"
                :value="workflow.id"
              >
                {{ workflow.name }}
              </option>
            </select>
            <p v-if="errors.workflow_id" class="mt-1 text-sm text-red-600">
              {{ errors.workflow_id }}
            </p>
          </div>

          <!-- Trigger name -->
          <Input
            v-model="name"
            label="Trigger Name"
            type="text"
            required
            :error="errors.name"
            placeholder="e.g., Daily Report"
          />

          <!-- Description -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Description
            </label>
            <textarea
              v-model="description"
              rows="3"
              class="w-full rounded-md border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              :class="{
                'border-gray-300': !errors.description,
                'border-red-300': errors.description,
              }"
              placeholder="Describe what this trigger does..."
            />
            <p v-if="errors.description" class="mt-1 text-sm text-red-600">
              {{ errors.description }}
            </p>
          </div>

          <!-- Trigger type -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Trigger Type <span class="text-red-500">*</span>
            </label>
            <select
              v-model="type"
              class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="manual">Manual</option>
              <option value="schedule">Schedule (Cron)</option>
              <option value="webhook">Webhook</option>
              <option value="event">Event</option>
            </select>
            <p class="mt-1 text-xs text-gray-500">
              {{ getTriggerTypeDescription(type as TriggerType) }}
            </p>
          </div>

          <!-- Schedule config -->
          <div
            v-if="type === 'schedule'"
            class="space-y-4 rounded-md border border-gray-200 bg-gray-50 p-4"
          >
            <h3 class="text-sm font-semibold text-gray-900">
              Schedule Configuration
            </h3>

            <Input
              v-model="scheduleCron"
              label="Cron Expression"
              type="text"
              required
              :error="scheduleForm.errors.value.cron"
              placeholder="0 0 * * *"
            />
            <p class="mt-1 text-xs text-gray-500">
              Format: minute hour day month weekday (e.g., "0 0 * * *" = daily
              at midnight)
            </p>

            <Input
              v-model="scheduleTimezone"
              label="Timezone"
              type="text"
              :error="scheduleForm.errors.value.timezone"
              placeholder="UTC"
            />
          </div>

          <!-- Webhook config -->
          <div
            v-if="type === 'webhook'"
            class="space-y-4 rounded-md border border-gray-200 bg-gray-50 p-4"
          >
            <h3 class="text-sm font-semibold text-gray-900">
              Webhook Configuration
            </h3>

            <Input
              v-model="webhookPath"
              label="Webhook Path"
              type="text"
              required
              :error="webhookForm.errors.value.webhook_path"
              placeholder="/webhooks/my-trigger"
            />
            <p class="mt-1 text-xs text-gray-500">
              The path that will be used for the webhook endpoint
            </p>

            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700">
                HTTP Method
              </label>
              <select
                v-model="webhookMethod"
                class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="POST">POST</option>
                <option value="GET">GET</option>
                <option value="PUT">PUT</option>
                <option value="PATCH">PATCH</option>
              </select>
            </div>

            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700">
                Authentication Type
              </label>
              <select
                v-model="webhookAuthType"
                class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="none">None</option>
                <option value="api_key">API Key</option>
                <option value="bearer">Bearer Token</option>
              </select>
            </div>
          </div>

          <!-- Event config -->
          <div
            v-if="type === 'event'"
            class="space-y-4 rounded-md border border-gray-200 bg-gray-50 p-4"
          >
            <h3 class="text-sm font-semibold text-gray-900">
              Event Configuration
            </h3>

            <Input
              v-model="eventType"
              label="Event Type"
              type="text"
              required
              :error="eventForm.errors.value.event_type"
              placeholder="user.created"
            />
            <p class="mt-1 text-xs text-gray-500">
              The type of event that will trigger this workflow
            </p>
          </div>

          <!-- Status -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Status
            </label>
            <select
              v-model="status"
              class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="enabled">Enabled</option>
              <option value="disabled">Disabled</option>
            </select>
          </div>

          <!-- Actions -->
          <div class="flex gap-3 pt-4">
            <Button
              type="submit"
              variant="primary"
              :loading="isSubmitting"
              :disabled="isSubmitting"
            >
              Create Trigger
            </Button>
            <Button
              type="button"
              variant="secondary"
              :disabled="isSubmitting"
              @click="handleCancel"
            >
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </div>
  </AppShell>
</template>
