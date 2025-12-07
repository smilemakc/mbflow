<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useForm } from "vee-validate";
import { toTypedSchema } from "@vee-validate/yup";
import { Icon } from "@iconify/vue";
import AppShell from "@/components/layout/AppShell.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import { createWorkflow } from "@/api/workflows";
import { workflowCreateSchema } from "@/validation";
import { toast } from "vue3-toastify";
import { getTemplateById } from "@/data/templates";
import type { WorkflowTemplate } from "@/types/workflowTemplate";

const router = useRouter();
const route = useRoute();

const selectedTemplate = ref<WorkflowTemplate | null>(null);

// Initialize VeeValidate form
const {
  handleSubmit,
  errors,
  defineField,
  isSubmitting,
  setFieldError,
  setValues,
} = useForm({
  validationSchema: toTypedSchema(workflowCreateSchema),
  initialValues: {
    name: "",
    description: "",
    status: "draft",
  },
});

// Define form fields
const [name] = defineField("name");
const [description] = defineField("description");
const [status] = defineField("status");

// Check if creating from template
onMounted(() => {
  const templateId = route.query.template as string;
  if (templateId) {
    const template = getTemplateById(templateId);
    if (template) {
      selectedTemplate.value = template;
      // Pre-fill form with template data
      setValues({
        name: template.name,
        description: template.description,
        status: "draft",
      });
    }
  }
});

// Submit handler
const onSubmit = handleSubmit(async (values) => {
  try {
    const workflow = await createWorkflow(values);

    // If using template, navigate to editor with template data
    if (selectedTemplate.value) {
      toast.success("Workflow created from template!");
      router.push({
        path: `/workflows/${workflow.id}`,
        query: { template: selectedTemplate.value.id },
      });
    } else {
      toast.success("Workflow created successfully!");
      router.push(`/workflows/${workflow.id}`);
    }
  } catch (error: any) {
    console.error("Failed to create workflow:", error);
    const errorMessage = error.message || "Failed to create workflow";
    setFieldError("name", errorMessage);
    toast.error(errorMessage);
  }
});

function handleCancel() {
  router.push("/workflows");
}

function handleBrowseTemplates() {
  router.push("/templates");
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

      <!-- Template info banner -->
      <div
        v-if="selectedTemplate"
        class="mb-6 rounded-lg border border-blue-200 bg-blue-50 p-4"
      >
        <div class="flex items-start gap-3">
          <Icon
            :icon="selectedTemplate.icon || 'heroicons:document-duplicate'"
            class="size-6 text-blue-600"
          />
          <div class="flex-1">
            <h3 class="font-semibold text-blue-900">
              Creating from template: {{ selectedTemplate.name }}
            </h3>
            <p class="mt-1 text-sm text-blue-700">
              {{ selectedTemplate.description }}
            </p>
            <p class="mt-2 text-xs text-blue-600">
              The workflow will be initialized with
              {{ selectedTemplate.nodes.length }} pre-configured nodes
            </p>
          </div>
        </div>
      </div>

      <!-- Browse templates button -->
      <div v-else class="card mb-6">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="font-semibold text-gray-900">Start from a template?</h3>
            <p class="mt-1 text-sm text-gray-600">
              Browse pre-built templates to get started quickly
            </p>
          </div>
          <Button variant="secondary" @click="handleBrowseTemplates">
            <Icon icon="heroicons:document-duplicate" class="mr-2 size-4" />
            Browse Templates
          </Button>
        </div>
      </div>

      <div class="card">
        <form @submit="onSubmit" class="space-y-6">
          <!-- Workflow name -->
          <Input
            v-model="name"
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
              v-model="description"
              rows="4"
              class="w-full rounded-md border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              :class="{
                'border-gray-300': !errors.description,
                'border-red-300': errors.description,
              }"
              placeholder="Describe what this workflow does..."
            />
            <p v-if="errors.description" class="mt-1 text-sm text-red-600">
              {{ errors.description }}
            </p>
          </div>

          <!-- Status -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Status
            </label>
            <select
              v-model="status"
              class="w-full rounded-md border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              :class="{
                'border-gray-300': !errors.status,
                'border-red-300': errors.status,
              }"
            >
              <option value="draft">Draft</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
            <p v-if="errors.status" class="mt-1 text-sm text-red-600">
              {{ errors.status }}
            </p>
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
