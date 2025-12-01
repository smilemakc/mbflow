<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      strategy?: string;
      merge_key?: string;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const strategy = computed(() => props.data.config?.strategy || "array");

const strategyLabel = computed(() => {
  const labels: Record<string, string> = {
    array: "Array",
    object: "Object",
    first: "First",
    last: "Last",
  };
  return labels[strategy.value] || strategy.value;
});
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="merge">
    <div class="mt-2 space-y-1">
      <div class="flex items-center gap-2">
        <span
          class="rounded bg-pink-100 px-2 py-0.5 text-xs font-semibold text-pink-700"
        >
          {{ strategyLabel }}
        </span>
      </div>
    </div>
  </BaseNode>
</template>
