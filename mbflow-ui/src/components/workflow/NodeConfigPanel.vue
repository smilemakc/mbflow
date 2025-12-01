<script setup lang="ts">
import { computed, watch, ref } from "vue";
import { Icon } from "@iconify/vue";
import { useWorkflowStore } from "@/stores/workflow";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";

const workflowStore = useWorkflowStore();

const selectedNode = computed(() => workflowStore.selectedNode);
const isOpen = computed(() => !!selectedNode.value);

// Local form state
const formData = ref<Record<string, any>>({});

// Watch for node selection changes
watch(
  selectedNode,
  (node) => {
    if (node) {
      formData.value = {
        label: node.data?.label || "",
        ...node.data?.config,
      };
    } else {
      formData.value = {};
    }
  },
  { immediate: true },
);

function closePanel() {
  workflowStore.selectNode(null);
}

function saveConfig() {
  if (!selectedNode.value) return;

  const { label, ...config } = formData.value;

  workflowStore.updateNode(selectedNode.value.id, {
    data: {
      ...selectedNode.value.data,
      label,
      config,
    },
  });

  closePanel();
}

function deleteNode() {
  if (!selectedNode.value) return;

  if (confirm("Are you sure you want to delete this node?")) {
    workflowStore.removeNode(selectedNode.value.id);
  }
}

// Get configuration fields based on node type
const configFields = computed(() => {
  if (!selectedNode.value) return [];

  const nodeType = selectedNode.value.type;

  switch (nodeType) {
    case "http":
      return [
        {
          key: "method",
          label: "Method",
          type: "select",
          options: ["GET", "POST", "PUT", "DELETE", "PATCH"],
        },
        { key: "url", label: "URL", type: "text", required: true },
        { key: "headers", label: "Headers (JSON)", type: "textarea" },
        { key: "body", label: "Body (JSON)", type: "textarea" },
      ];
    case "llm":
      return [
        {
          key: "provider",
          label: "Provider",
          type: "select",
          options: ["openai", "anthropic"],
        },
        { key: "model", label: "Model", type: "text", required: true },
        {
          key: "temperature",
          label: "Temperature",
          type: "number",
          min: 0,
          max: 2,
          step: 0.1,
        },
        {
          key: "max_tokens",
          label: "Max Tokens",
          type: "number",
          min: 1,
          max: 100000,
        },
        { key: "system_prompt", label: "System Prompt", type: "textarea" },
      ];
    case "transform":
      return [
        {
          key: "expression",
          label: "Expression",
          type: "textarea",
          required: true,
        },
        { key: "variables", label: "Variables (JSON)", type: "textarea" },
      ];
    case "conditional":
      return [
        {
          key: "condition",
          label: "Condition",
          type: "textarea",
          required: true,
        },
        { key: "true_branch", label: "True Branch", type: "text" },
        { key: "false_branch", label: "False Branch", type: "text" },
      ];
    case "merge":
      return [
        {
          key: "strategy",
          label: "Strategy",
          type: "select",
          options: ["array", "object", "first", "last"],
        },
        { key: "merge_key", label: "Merge Key", type: "text" },
      ];
    default:
      return [];
  }
});
</script>

<template>
  <div
    :class="[
      'node-config-panel',
      'fixed right-0 top-0 h-full border-l border-gray-200 bg-white shadow-xl',
      'z-50 transition-transform duration-300 ease-in-out',
      isOpen ? 'translate-x-0' : 'translate-x-full',
    ]"
    style="width: 400px"
  >
    <div v-if="selectedNode" class="flex h-full flex-col">
      <!-- Header -->
      <div
        class="flex items-center justify-between border-b border-gray-200 p-4"
      >
        <div class="flex items-center gap-2">
          <Icon icon="heroicons:cog-6-tooth" class="size-5 text-gray-700" />
          <h3 class="text-lg font-semibold text-gray-900">
            Node Configuration
          </h3>
        </div>
        <button
          class="rounded p-1 transition-colors hover:bg-gray-100"
          @click="closePanel"
        >
          <Icon icon="heroicons:x-mark" class="size-5 text-gray-500" />
        </button>
      </div>

      <!-- Form -->
      <div class="flex-1 space-y-4 overflow-y-auto p-4">
        <!-- Node type badge -->
        <div class="flex items-center gap-2">
          <span
            class="rounded bg-blue-100 px-2 py-1 text-xs font-semibold text-blue-700"
          >
            {{ selectedNode.type?.toUpperCase() }}
          </span>
        </div>

        <!-- Node name -->
        <Input
          v-model="formData.label"
          label="Node Name"
          type="text"
          required
          placeholder="Enter node name"
        />

        <!-- Dynamic fields based on node type -->
        <div v-for="field in configFields" :key="field.key" class="space-y-1">
          <!-- Text input -->
          <Input
            v-if="field.type === 'text'"
            v-model="formData[field.key]"
            :label="field.label"
            :required="field.required"
            type="text"
          />

          <!-- Number input -->
          <Input
            v-else-if="field.type === 'number'"
            v-model.number="formData[field.key]"
            :label="field.label"
            :required="field.required"
            type="number"
            :min="field.min"
            :max="field.max"
            :step="field.step"
          />

          <!-- Select -->
          <div v-else-if="field.type === 'select'">
            <label class="mb-1 block text-sm font-medium text-gray-700">
              {{ field.label }}
              <span v-if="field.required" class="text-red-500">*</span>
            </label>
            <select
              v-model="formData[field.key]"
              class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option
                v-for="option in field.options"
                :key="option"
                :value="option"
              >
                {{ option }}
              </option>
            </select>
          </div>

          <!-- Textarea -->
          <div v-else-if="field.type === 'textarea'">
            <label class="mb-1 block text-sm font-medium text-gray-700">
              {{ field.label }}
              <span v-if="field.required" class="text-red-500">*</span>
            </label>
            <textarea
              v-model="formData[field.key]"
              rows="4"
              class="w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              :placeholder="field.label"
            />
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="space-y-2 border-t border-gray-200 p-4">
        <div class="flex gap-2">
          <Button variant="primary" class="flex-1" @click="saveConfig">
            Save Changes
          </Button>
          <Button variant="secondary" @click="closePanel"> Cancel </Button>
        </div>
        <Button variant="danger" class="w-full" @click="deleteNode">
          Delete Node
        </Button>
      </div>
    </div>
  </div>

  <!-- Backdrop -->
  <div
    v-if="isOpen"
    class="fixed inset-0 z-40 bg-black/20 transition-opacity"
    @click="closePanel"
  />
</template>

<style scoped>
.node-config-panel {
  max-width: 100vw;
}
</style>
