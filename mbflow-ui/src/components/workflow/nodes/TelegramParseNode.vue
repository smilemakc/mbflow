<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      extract_files?: boolean;
      extract_commands?: boolean;
      extract_entities?: boolean;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const features = computed(() => {
  const f: string[] = [];
  if (props.data.config?.extract_files !== false) f.push("files");
  if (props.data.config?.extract_commands !== false) f.push("commands");
  if (props.data.config?.extract_entities) f.push("entities");
  return f;
});
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="telegram_parse">
    <div class="mt-2 space-y-1">
      <!-- Features -->
      <div class="flex flex-wrap gap-1">
        <span
          v-for="feature in features"
          :key="feature"
          class="rounded bg-sky-100 px-1.5 py-0.5 text-xs text-sky-700"
        >
          {{ feature }}
        </span>
      </div>
    </div>
  </BaseNode>
</template>
