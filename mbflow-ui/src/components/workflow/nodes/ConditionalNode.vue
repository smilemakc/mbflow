<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      condition?: string;
      true_branch?: string;
      false_branch?: string;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const condition = computed(() => props.data.config?.condition || "");
const hasCondition = computed(() => condition.value.length > 0);
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="conditional">
    <div class="mt-2 space-y-1">
      <div
        v-if="hasCondition"
        class="truncate rounded bg-gray-50 px-2 py-1 font-mono text-xs text-gray-500"
        :title="condition"
      >
        if {{ condition }}
      </div>
      <div v-else class="text-xs italic text-gray-400">No condition set</div>
      <div class="flex gap-1 text-xs">
        <span class="rounded bg-green-100 px-1.5 py-0.5 text-green-700">
          True
        </span>
        <span class="rounded bg-red-100 px-1.5 py-0.5 text-red-700">
          False
        </span>
      </div>
    </div>
  </BaseNode>
</template>
