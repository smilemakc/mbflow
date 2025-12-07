<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      file_id?: string;
      output_format?: string;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const fileId = computed(() => {
  const id = props.data.config?.file_id || "";
  return id.length > 20 ? `${id.slice(0, 20)}...` : id;
});

const outputFormat = computed(
  () => props.data.config?.output_format || "base64"
);
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="telegram_download">
    <div class="mt-2 space-y-1">
      <!-- Output Format Badge -->
      <div class="flex items-center gap-2">
        <span
          class="rounded bg-sky-100 px-2 py-0.5 text-xs font-semibold uppercase text-sky-700"
        >
          {{ outputFormat }}
        </span>
      </div>

      <!-- File ID Preview -->
      <div
        v-if="fileId"
        class="flex items-center gap-1 text-xs text-gray-500"
      >
        <span class="font-medium">File:</span>
        <span class="truncate font-mono" :title="props.data.config?.file_id">
          {{ fileId }}
        </span>
      </div>
    </div>
  </BaseNode>
</template>
