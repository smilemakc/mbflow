<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      provider?: string;
      model?: string;
      temperature?: number;
      max_tokens?: number;
      system_prompt?: string;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const provider = computed(() => props.data.config?.provider || "OpenAI");
const model = computed(() => props.data.config?.model || "");

const providerColor = computed(() => {
  const colors: Record<string, string> = {
    openai: "bg-green-100 text-green-700",
    anthropic: "bg-purple-100 text-purple-700",
    default: "bg-gray-100 text-gray-700",
  };
  const key = provider.value.toLowerCase();
  return colors[key] || colors.default;
});
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="llm">
    <div class="mt-2 space-y-1">
      <div class="flex items-center gap-2">
        <span
          :class="[providerColor, 'rounded px-2 py-0.5 text-xs font-semibold']"
        >
          {{ provider }}
        </span>
      </div>
      <div v-if="model" class="truncate text-xs text-gray-500" :title="model">
        {{ model }}
      </div>
    </div>
  </BaseNode>
</template>
