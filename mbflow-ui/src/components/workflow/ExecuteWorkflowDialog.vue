<template>
  <div
    v-if="isOpen"
    class="fixed inset-0 z-50 flex items-center justify-center"
    @click.self="close"
  >
    <!-- Backdrop -->
    <div class="absolute inset-0 bg-black/50 transition-opacity" />

    <!-- Dialog -->
    <div
      class="relative z-10 w-full max-w-2xl rounded-lg bg-white shadow-xl"
      @click.stop
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-gray-200 p-6">
        <div>
          <h2 class="text-xl font-semibold text-gray-900">Execute Workflow</h2>
          <p class="mt-1 text-sm text-gray-500">
            Configure execution options and runtime variables
          </p>
        </div>
        <button
          class="rounded p-1 transition-colors hover:bg-gray-100"
          @click="close"
        >
          <Icon icon="heroicons:x-mark" class="size-5 text-gray-500" />
        </button>
      </div>

      <!-- Content -->
      <div class="max-h-[60vh] space-y-6 overflow-y-auto p-6">
        <!-- Variable Precedence Info -->
        <div class="rounded-lg bg-blue-50 p-4">
          <div class="flex items-start gap-3">
            <Icon
              icon="heroicons:information-circle"
              class="mt-0.5 size-5 shrink-0 text-blue-600"
            />
            <div class="text-sm text-blue-900">
              <p class="font-medium">Variable Resolution Order</p>
              <p class="mt-1 text-blue-800">
                Execution Variables (below) override Workflow Variables. Use
                <code class="rounded bg-blue-100 px-1 font-mono text-xs"
                  >{{templatePlaceholderExample}}</code
                >
                in node configurations.
              </p>
            </div>
          </div>
        </div>

        <!-- Workflow Variables Display -->
        <div v-if="hasWorkflowVariables" class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="text-sm font-semibold text-gray-700">
              Workflow Variables (Default)
            </label>
            <button
              class="text-xs text-blue-600 hover:text-blue-700"
              @click="openWorkflowVariablesPanel"
            >
              Manage
            </button>
          </div>
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3">
            <div class="space-y-1">
              <div
                v-for="(value, key) in workflowVariables"
                :key="key"
                class="flex items-center gap-2 text-sm"
              >
                <code class="font-mono text-gray-600">{{ key }}</code>
                <span class="text-gray-400">=</span>
                <span class="truncate text-gray-700">{{
                  formatValue(value)
                }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Execution Variables -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="text-sm font-semibold text-gray-700">
              Execution Variables (Runtime Override)
            </label>
            <button
              class="flex items-center gap-1 text-xs text-blue-600 hover:text-blue-700"
              @click="addExecutionVariable"
            >
              <Icon icon="heroicons:plus" class="size-3" />
              Add Variable
            </button>
          </div>

          <div v-if="executionVariablesList.length > 0" class="space-y-2">
            <div
              v-for="(item, index) in executionVariablesList"
              :key="index"
              class="flex items-start gap-2 rounded-lg border border-gray-200 bg-white p-3"
            >
              <div class="grid flex-1 grid-cols-2 gap-2">
                <div>
                  <input
                    v-model="item.key"
                    type="text"
                    placeholder="variable_name"
                    class="w-full rounded border border-gray-300 px-2 py-1.5 text-sm font-mono focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <input
                    v-model="item.value"
                    type="text"
                    placeholder="value"
                    class="w-full rounded border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
              </div>
              <button
                class="shrink-0 rounded p-1.5 text-red-600 hover:bg-red-50"
                @click="removeExecutionVariable(index)"
                title="Remove variable"
              >
                <Icon icon="heroicons:trash" class="size-4" />
              </button>
            </div>
          </div>

          <div v-else class="rounded-lg border border-dashed border-gray-300 p-4 text-center">
            <p class="text-sm text-gray-500">
              No execution variables defined. Execution will use workflow variables.
            </p>
          </div>
        </div>

        <!-- Input Data (Optional) -->
        <div class="space-y-2">
          <label class="text-sm font-semibold text-gray-700">
            Input Data (Optional)
          </label>
          <p class="text-xs text-gray-500">
            Provide initial input data for the workflow execution (JSON format)
          </p>
          <textarea
            v-model="inputJson"
            rows="4"
            placeholder='{\n  "user_id": "123",\n  "action": "signup"\n}'
            class="w-full rounded-lg border border-gray-300 p-3 font-mono text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
          <p v-if="inputError" class="text-xs text-red-600">
            {{ inputError }}
          </p>
        </div>

        <!-- Strict Mode -->
        <div class="flex items-start gap-3">
          <input
            id="strictMode"
            v-model="strictMode"
            type="checkbox"
            class="mt-0.5 size-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          <div class="flex-1">
            <label for="strictMode" class="text-sm font-medium text-gray-700 cursor-pointer">
              Strict Mode
            </label>
            <p class="mt-0.5 text-xs text-gray-500">
              Fail execution if any template variable is undefined. Recommended for production.
            </p>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex justify-end gap-3 border-t border-gray-200 p-6">
        <button
          class="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
          @click="close"
        >
          Cancel
        </button>
        <button
          :disabled="isExecuting"
          class="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          @click="handleExecute"
        >
          <Icon
            v-if="isExecuting"
            icon="heroicons:arrow-path"
            class="size-4 animate-spin"
          />
          <Icon v-else icon="heroicons:play" class="size-4" />
          {{ isExecuting ? "Executing..." : "Execute Workflow" }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { Icon } from "@iconify/vue";
import { useWorkflowStore } from "@/stores/workflow";
import type { ExecuteWorkflowOptions } from "@/api/workflows";

interface Props {
  isOpen: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "execute", options: ExecuteWorkflowOptions): void;
  (e: "open-variables-panel"): void;
}>();

const workflowStore = useWorkflowStore();
const templatePlaceholderExample = "{{env.variable_name}}"
interface VariableItem {
  key: string;
  value: string;
}

const executionVariablesList = ref<VariableItem[]>([]);
const inputJson = ref("");
const inputError = ref("");
const strictMode = ref(false);
const isExecuting = ref(false);

const workflowVariables = computed(() => workflowStore.workflowVariables || {});

const hasWorkflowVariables = computed(() => {
  return Object.keys(workflowVariables.value).length > 0;
});

// Reset form when dialog opens
watch(
  () => props.isOpen,
  (open) => {
    if (open) {
      executionVariablesList.value = [];
      inputJson.value = "";
      inputError.value = "";
      strictMode.value = false;
      isExecuting.value = false;
    }
  }
);

function formatValue(value: any): string {
  if (typeof value === "string") return value;
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function addExecutionVariable() {
  executionVariablesList.value.push({ key: "", value: "" });
}

function removeExecutionVariable(index: number) {
  executionVariablesList.value.splice(index, 1);
}

function openWorkflowVariablesPanel() {
  emit("open-variables-panel");
}

function validateInput(): boolean {
  inputError.value = "";

  if (inputJson.value.trim()) {
    try {
      JSON.parse(inputJson.value);
    } catch (e) {
      inputError.value = "Invalid JSON format";
      return false;
    }
  }

  return true;
}

function handleExecute() {
  if (!validateInput()) {
    return;
  }

  isExecuting.value = true;

  // Build execution variables object
  const variables: Record<string, string> = {};
  executionVariablesList.value.forEach(({ key, value }) => {
    if (key.trim()) {
      variables[key.trim()] = value;
    }
  });

  // Parse input JSON if provided
  let input: Record<string, any> | undefined;
  if (inputJson.value.trim()) {
    try {
      input = JSON.parse(inputJson.value);
    } catch (e) {
      // Already validated, should not happen
    }
  }

  const options: ExecuteWorkflowOptions = {
    variables: Object.keys(variables).length > 0 ? variables : undefined,
    input,
    strict_mode: strictMode.value,
  };

  emit("execute", options);
}

// Public method to reset executing state (called from parent on error)
function resetExecuting() {
  isExecuting.value = false;
}

// Expose method to parent
defineExpose({
  resetExecuting,
});

function close() {
  if (!isExecuting.value) {
    emit("close");
  }
}
</script>

<style scoped>
code {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
}
</style>
