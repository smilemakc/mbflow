<script setup lang="ts">
// @ts-nocheck
import { ref } from "vue";
import { useRouter } from "vue-router";
import AppShell from "@/components/layout/AppShell.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import { createWorkflow } from "@/api/workflows";
import type { WorkflowCreateRequest } from "@/types/workflow";

const router = useRouter();

const form = ref<WorkflowCreateRequest>({
  name: "",
  description: "",
});

const errors = ref<Record<string, string>>({});
const isSubmitting = ref(false);

function validateForm(): boolean {
  errors.value = {};

  if (!form.value.name.trim()) {
    errors.value.name = "Workflow name is required";
  }

  return Object.keys(errors.value).length === 0;
}

async function handleSubmit() {
  if (!validateForm()) return;

  isSubmitting.value = true;

  try {
    const response = await createWorkflow(form.value);
    router.push(`/workflows/${response.workflow.id}`);
  } catch (error: any) {
    console.error("Failed to create workflow:", error);
    errors.value.general = error.message || "Failed to create workflow";
  } finally {
    isSubmitting.value = false;
  }
}

function handleCancel() {
  router.push("/workflows");
}
</script>

<template>
  <AppShell>
    <div class="mx-auto max-w-3xl">
      <div class="mb-6">
        <h1 class="text-3xl font-bold text-gray-900">Create New Workflow</h1>
        <p class="mt-2 text-gray-600">
          Define a new workflow to orchestrate your automation tasks.
        </p>
      </div>

      <div class="card">
        <form @submit.prevent="handleSubmit" class="space-y-6">
          <!-- General error -->
          <div
            v-if="errors.general"
            class="rounded-md border border-red-200 bg-red-50 p-4"
          >
            <p class="text-sm text-red-800">{{ errors.general }}</p>
          </div>

          <!-- Workflow name -->
          <Input
            v-model="form.name"
            label="Workflow Name"
            type="text"
            required
            :error="errors.name"
            placeholder="e.g., User Onboarding"
          />

          <!-- Description -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Description
            </label>
            <textarea
              v-model="form.description"
              rows="4"
              class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Describe what this workflow does..."
            />
          </div>

          <!-- Status -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Status
            </label>
            <select
              v-model="form.status"
              class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="draft">Draft</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
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
              Create Workflow
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
