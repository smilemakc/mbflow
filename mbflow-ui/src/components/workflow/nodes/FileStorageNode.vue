<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      action?: string;
      storage_id?: string;
      file_name?: string;
      access_scope?: string;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const action = computed(() => props.data.config?.action || "store");
const storageId = computed(() => props.data.config?.storage_id || "default");
const fileName = computed(() => props.data.config?.file_name || "");
const accessScope = computed(() => props.data.config?.access_scope || "workflow");

const actionColor = computed(() => {
  const colors: Record<string, string> = {
    store: "bg-teal-100 text-teal-700",
    get: "bg-blue-100 text-blue-700",
    delete: "bg-red-100 text-red-700",
    list: "bg-purple-100 text-purple-700",
    metadata: "bg-orange-100 text-orange-700",
  };
  return colors[action.value] || "bg-gray-100 text-gray-700";
});

const actionIcon = computed(() => {
  const icons: Record<string, string> = {
    store: "↑",
    get: "↓",
    delete: "×",
    list: "☰",
    metadata: "ℹ",
  };
  return icons[action.value] || "📁";
});
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="file_storage">
    <div class="mt-2 space-y-1">
      <!-- Action Badge -->
      <div class="flex items-center gap-2">
        <span
          :class="[
            actionColor,
            'rounded px-2 py-0.5 text-xs font-semibold uppercase',
          ]"
        >
          {{ actionIcon }} {{ action }}
        </span>
      </div>

      <!-- Storage ID -->
      <div
        v-if="storageId"
        class="flex items-center gap-1 text-xs text-gray-500"
      >
        <span class="font-medium">Storage:</span>
        <span class="truncate" :title="storageId">{{ storageId }}</span>
      </div>

      <!-- File Name (for store/get actions) -->
      <div
        v-if="fileName && ['store', 'get'].includes(action)"
        class="line-clamp-1 text-xs text-gray-600"
        :title="fileName"
      >
        {{ fileName }}
      </div>

      <!-- Access Scope Badge -->
      <div
        v-if="accessScope"
        class="flex items-center gap-1 text-xs text-gray-400"
      >
        <span class="rounded bg-gray-100 px-1.5 py-0.5 text-xs">
          {{ accessScope }}
        </span>
      </div>
    </div>
  </BaseNode>
</template>
