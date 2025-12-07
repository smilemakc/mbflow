<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      text?: string;
      show_alert?: boolean;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const displayText = computed(() => {
  const text = props.data.config?.text || "";
  return text.length > 30 ? `${text.slice(0, 30)}...` : text;
});

const isAlert = computed(() => props.data.config?.show_alert === true);
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="telegram_callback">
    <div class="mt-2 space-y-1">
      <!-- Alert Badge -->
      <div class="flex items-center gap-2">
        <span
          :class="[
            isAlert
              ? 'bg-amber-100 text-amber-700'
              : 'bg-sky-100 text-sky-700',
            'rounded px-2 py-0.5 text-xs font-semibold',
          ]"
        >
          {{ isAlert ? "Alert" : "Toast" }}
        </span>
      </div>

      <!-- Text Preview -->
      <div
        v-if="displayText"
        class="line-clamp-2 text-xs text-gray-600"
        :title="props.data.config?.text"
      >
        {{ displayText }}
      </div>
    </div>
  </BaseNode>
</template>
