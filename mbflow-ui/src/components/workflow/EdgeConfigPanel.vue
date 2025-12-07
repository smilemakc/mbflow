<template>
  <div
    :class="[
      'edge-config-panel',
      'fixed right-0 top-0 h-full border-l border-gray-200 bg-white shadow-xl',
      'z-50 transition-transform duration-300 ease-in-out',
      isOpen ? 'translate-x-0' : 'translate-x-full',
    ]"
    style="width: 400px"
  >
    <div v-if="selectedEdge" class="flex h-full flex-col">
      <!-- Header -->
      <div
        class="flex items-center justify-between border-b border-gray-200 p-4"
      >
        <div class="flex items-center gap-2">
          <Icon icon="heroicons:arrow-long-right" class="size-5 text-gray-700" />
          <h3 class="text-lg font-semibold text-gray-900">
            Edge Configuration
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
        <!-- Edge info -->
        <div class="rounded-lg bg-gray-50 p-3">
          <div class="mb-2 text-sm font-medium text-gray-700">Connection</div>
          <div class="flex items-center gap-2 text-sm">
            <span class="rounded bg-blue-100 px-2 py-1 font-mono text-blue-700">
              {{ sourceNodeLabel }}
            </span>
            <Icon icon="heroicons:arrow-right" class="size-4 text-gray-400" />
            <span class="rounded bg-green-100 px-2 py-1 font-mono text-green-700">
              {{ targetNodeLabel }}
            </span>
          </div>
        </div>

        <!-- Source Handle (for conditional nodes) -->
        <div v-if="isFromConditionalNode" class="space-y-2">
          <label class="text-sm font-medium text-gray-700">Branch</label>
          <div class="flex gap-2">
            <button
              :class="[
                'flex-1 rounded-lg border-2 px-4 py-2 text-sm font-medium transition-colors',
                sourceHandle === 'true'
                  ? 'border-green-500 bg-green-50 text-green-700'
                  : 'border-gray-200 bg-white text-gray-600 hover:border-green-300',
              ]"
              @click="sourceHandle = 'true'"
            >
              ✓ True Branch
            </button>
            <button
              :class="[
                'flex-1 rounded-lg border-2 px-4 py-2 text-sm font-medium transition-colors',
                sourceHandle === 'false'
                  ? 'border-red-500 bg-red-50 text-red-700'
                  : 'border-gray-200 bg-white text-gray-600 hover:border-red-300',
              ]"
              @click="sourceHandle = 'false'"
            >
              ✗ False Branch
            </button>
          </div>
          <p class="text-xs text-gray-500">
            Select which branch of the conditional node this edge follows.
          </p>
        </div>

        <!-- Custom Condition Expression -->
        <div class="space-y-2">
          <label class="text-sm font-medium text-gray-700">
            Condition Expression
            <span class="text-gray-400">(optional)</span>
          </label>
          <TemplateInput
            v-model="condition"
            height="80px"
            multiline
            placeholder="output.status == 'success'"
          />
          <p class="text-xs text-gray-500">
            Use expr-lang syntax. Edge will only be followed if condition evaluates to true.
          </p>
        </div>

        <!-- Info box -->
        <div class="rounded-lg border border-blue-200 bg-blue-50 p-3">
          <h4 class="mb-2 text-sm font-semibold text-blue-800">
            💡 Condition Examples
          </h4>
          <ul class="space-y-1 text-xs text-blue-700">
            <li><code class="rounded bg-white px-1">output.success == true</code></li>
            <li><code class="rounded bg-white px-1">output.count > 0</code></li>
            <li><code class="rounded bg-white px-1">output.status == "completed"</code></li>
          </ul>
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
        <Button variant="danger" class="w-full" @click="deleteEdge">
          Delete Edge
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
import { computed, watch, ref } from "vue";
import { Icon } from "@iconify/vue";
import { toast } from "vue3-toastify";
import { useWorkflowStore } from "@/stores/workflow";
import Button from "@/components/ui/Button.vue";
import TemplateInput from "@/components/common/TemplateInput.vue";

const workflowStore = useWorkflowStore();
const emit = defineEmits(["save"]);

const selectedEdge = computed(() => workflowStore.selectedEdge);
const isOpen = computed(() => !!selectedEdge.value);

// Local form state
const sourceHandle = ref<string>("");
const condition = ref<string>("");

// Get source and target node labels
const sourceNodeLabel = computed(() => {
  if (!selectedEdge.value) return "";
  const node = workflowStore.nodes.find((n) => n.id === selectedEdge.value?.source);
  return node?.data?.label || selectedEdge.value.source;
});

const targetNodeLabel = computed(() => {
  if (!selectedEdge.value) return "";
  const node = workflowStore.nodes.find((n) => n.id === selectedEdge.value?.target);
  return node?.data?.label || selectedEdge.value.target;
});

// Check if source node is a conditional node
const isFromConditionalNode = computed(() => {
  if (!selectedEdge.value) return false;
  const node = workflowStore.nodes.find((n) => n.id === selectedEdge.value?.source);
  return node?.type === "conditional";
});

// Watch for edge selection changes
watch(
  selectedEdge,
  (edge) => {
    if (edge) {
      sourceHandle.value = edge.sourceHandle || "";
      condition.value = edge.data?.condition || "";
    } else {
      sourceHandle.value = "";
      condition.value = "";
    }
  },
  { immediate: true },
);

function closePanel() {
  workflowStore.selectEdge(null);
}

function saveConfig() {
  if (!selectedEdge.value) return;

  workflowStore.updateEdge(selectedEdge.value.id, {
    sourceHandle: sourceHandle.value || undefined,
    data: {
      ...selectedEdge.value.data,
      condition: condition.value || undefined,
    },
  });

  // Emit save event to trigger workflow persistence
  emit("save");

  toast.success("Edge configuration saved");
  closePanel();
}

function deleteEdge() {
  if (!selectedEdge.value) return;

  if (confirm("Are you sure you want to delete this edge?")) {
    workflowStore.removeEdge(selectedEdge.value.id);
    closePanel();
  }
}
</script>

<style scoped>
.edge-config-panel {
  max-width: 100vw;
}

code {
  font-family: "Monaco", "Menlo", monospace;
  font-size: 11px;
}
</style>
