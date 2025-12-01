<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      expression?: string;
      variables?: Record<string, any>;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const expression = computed(() => props.data.config?.expression || "");
const hasExpression = computed(() => expression.value.length > 0);
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="transform">
    <div class="mt-2 space-y-1">
      <div
        v-if="hasExpression"
        class="truncate rounded bg-gray-50 px-2 py-1 font-mono text-xs text-gray-500"
        :title="expression"
      >
        {{ expression }}
      </div>
      <div v-else class="text-xs italic text-gray-400">No expression set</div>
    </div>
  </BaseNode>
</template>
