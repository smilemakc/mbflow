<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      function_name?: string;
      timeout_seconds?: number;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const functionName = computed(() => props.data.config?.function_name || "");
const timeout = computed(() => props.data.config?.timeout_seconds || 30);
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="function_call">
    <div class="mt-2 space-y-1">
      <!-- Function Name Badge -->
      <div class="flex items-center gap-2">
        <span
          class="rounded bg-blue-100 px-2 py-0.5 text-xs font-semibold text-blue-700"
        >
          🔧 {{ functionName || "No function" }}
        </span>
      </div>

      <!-- Timeout -->
      <div
        v-if="timeout"
        class="flex items-center gap-1 text-xs text-gray-500"
      >
        <span class="font-medium">Timeout:</span>
        <span>{{ timeout }}s</span>
      </div>
    </div>
  </BaseNode>
</template>
