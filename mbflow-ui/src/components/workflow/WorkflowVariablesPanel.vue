<template>
  <div
    :class="[
      'workflow-variables-panel',
      'fixed left-0 top-0 h-full border-r border-gray-200 bg-white shadow-xl',
      'z-50 transition-transform duration-300 ease-in-out',
      isOpen ? 'translate-x-0' : '-translate-x-full',
    ]"
    style="width: 350px"
  >
    <div class="flex h-full flex-col">
      <!-- Header -->
      <div
        class="flex items-center justify-between border-b border-gray-200 p-4"
      >
        <div class="flex items-center gap-2">
          <Icon icon="heroicons:variable" class="size-5 text-gray-700" />
          <h3 class="text-lg font-semibold text-gray-900">
            Workflow Variables
          </h3>
        </div>
        <button
          class="rounded p-1 transition-colors hover:bg-gray-100"
          @click="closePanel"
        >
          <Icon icon="heroicons:x-mark" class="size-5 text-gray-500" />
        </button>
      </div>

      <!-- Content -->
      <div class="flex-1 space-y-4 overflow-y-auto p-4">
        <!-- Info Box -->
        <div class="rounded-lg bg-blue-50 p-4">
          <div class="flex items-start gap-3">
            <Icon
              icon="heroicons:information-circle"
              class="mt-0.5 size-5 shrink-0 text-blue-600"
            />
            <div class="text-sm text-blue-900">
              <p class="font-medium">Workflow Variables</p>
              <p class="mt-1 text-blue-800">
                Define default values for variables. Use
                <code class="rounded bg-blue-100 px-1 font-mono text-xs">{{
                  variablePlaceholderExample
                }}</code>
                in any node configuration field.
              </p>
              <p class="mt-2 text-xs text-blue-700">
                These values can be overridden at execution time.
              </p>
            </div>
          </div>
        </div>

        <!-- Example Box -->
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-3">
          <p class="mb-2 text-xs font-semibold text-gray-700">Example Usage:</p>
          <div class="space-y-1.5 text-xs">
            <div class="flex items-start gap-2">
              <span class="font-semibold text-gray-600">1.</span>
              <div>
                <p class="text-gray-700">
                  Define:
                  <code class="rounded bg-gray-200 px-1 font-mono"
                    >api_key = sk_123</code
                  >
                </p>
              </div>
            </div>
            <div class="flex items-start gap-2">
              <span class="font-semibold text-gray-600">2.</span>
              <div>
                <p class="text-gray-700">
                  Use:
                  <code class="rounded bg-gray-200 px-1 font-mono">{{
                    variablePlaceholderExample
                  }}</code>
                </p>
              </div>
            </div>
            <div class="flex items-start gap-2">
              <span class="font-semibold text-gray-600">3.</span>
              <div>
                <p class="text-gray-700">
                  Result:
                  <code class="rounded bg-gray-200 px-1 font-mono">sk_123</code>
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- Variables List -->
        <div class="space-y-3">
          <div
            v-for="(item, index) in variablesList"
            :key="index"
            class="variable-item"
          >
            <div class="form-group">
              <label class="label">Variable Name</label>
              <input
                v-model="item.key"
                @input="updateVariables"
                type="text"
                placeholder="e.g., api_key"
                class="input-field"
              />
            </div>

            <div class="form-group">
              <label class="label">Value</label>
              <input
                v-model="item.value"
                @input="updateVariables"
                type="text"
                placeholder="Variable value"
                class="input-field"
              />
            </div>

            <button
              @click="removeVariable(index)"
              class="remove-button"
              title="Remove variable"
            >
              <Icon icon="heroicons:trash" class="size-4" />
            </button>
          </div>

          <button @click="addVariable" class="add-button">
            <Icon icon="heroicons:plus" class="size-4" />
            Add Variable
          </button>
        </div>
      </div>

      <!-- Footer -->
      <div class="border-t border-gray-200 p-4">
        <Button variant="primary" class="w-full" @click="saveVariables">
          Save Variables
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

<script setup lang="ts">
import { ref, watch } from "vue";
import { Icon } from "@iconify/vue";
import { useWorkflowStore } from "@/stores/workflow";
import Button from "@/components/ui/Button.vue";

interface Props {
  isOpen: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "close"): void;
}>();

const variablePlaceholderExample = "{{env.api_key}}";
const workflowStore = useWorkflowStore();

interface VariableItem {
  key: string;
  value: string;
}

const variablesList = ref<VariableItem[]>([]);

// Load variables from store
watch(
  () => props.isOpen,
  (open) => {
    if (open) {
      loadVariables();
    }
  },
  { immediate: true },
);

function loadVariables() {
  const vars = workflowStore.workflowVariables || {};
  variablesList.value = Object.entries(vars).map(([key, value]) => ({
    key,
    value: String(value),
  }));

  // Ensure at least one empty row
  if (variablesList.value.length === 0) {
    variablesList.value.push({ key: "", value: "" });
  }
}

function addVariable() {
  variablesList.value.push({ key: "", value: "" });
}

function removeVariable(index: number) {
  variablesList.value.splice(index, 1);
  // Ensure at least one empty row
  if (variablesList.value.length === 0) {
    variablesList.value.push({ key: "", value: "" });
  }
  updateVariables();
}

function updateVariables() {
  // Convert to object, filtering out empty keys
  const vars: Record<string, string> = {};
  variablesList.value.forEach(({ key, value }) => {
    if (key.trim()) {
      vars[key.trim()] = value;
    }
  });
}

function saveVariables() {
  const vars: Record<string, string> = {};
  variablesList.value.forEach(({ key, value }) => {
    if (key.trim()) {
      vars[key.trim()] = value;
    }
  });

  workflowStore.updateWorkflowVariables(vars);
  closePanel();
}

function closePanel() {
  emit("close");
}
</script>

<style scoped>
.workflow-variables-panel {
  max-width: 100vw;
}

.variable-item {
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background-color: #f9fafb;
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px;
  align-items: start;
}

.variable-item > div:first-child {
  grid-column: 1 / 3;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.label {
  font-size: 12px;
  font-weight: 600;
  color: #374151;
}

.input-field {
  width: 100%;
  padding: 6px 10px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 13px;
  transition: border-color 0.2s;
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
}

.input-field:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.remove-button {
  padding: 6px;
  background-color: #fee;
  color: #dc2626;
  border: 1px solid #fca5a5;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 20px;
}

.remove-button:hover {
  background-color: #fecaca;
  border-color: #f87171;
}

.add-button {
  width: 100%;
  padding: 8px 12px;
  background-color: #eff6ff;
  color: #1e40af;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.add-button:hover {
  background-color: #dbeafe;
  border-color: #93c5fd;
}

code {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 12px;
}
</style>
