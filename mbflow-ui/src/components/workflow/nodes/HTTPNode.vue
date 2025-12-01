<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      method?: string;
      url?: string;
      headers?: Record<string, string>;
      body?: any;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const method = computed(() => props.data.config?.method || "GET");
const url = computed(() => props.data.config?.url || "");

const methodColor = computed(() => {
  const colors: Record<string, string> = {
    GET: "bg-blue-100 text-blue-700",
    POST: "bg-green-100 text-green-700",
    PUT: "bg-orange-100 text-orange-700",
    DELETE: "bg-red-100 text-red-700",
    PATCH: "bg-purple-100 text-purple-700",
  };
  return colors[method.value] || "bg-gray-100 text-gray-700";
});
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="http">
    <div class="mt-2 space-y-1">
      <div class="flex items-center gap-2">
        <span
          :class="[methodColor, 'rounded px-2 py-0.5 text-xs font-semibold']"
        >
          {{ method }}
        </span>
      </div>
      <div v-if="url" class="truncate text-xs text-gray-500" :title="url">
        {{ url }}
      </div>
    </div>
  </BaseNode>
</template>
