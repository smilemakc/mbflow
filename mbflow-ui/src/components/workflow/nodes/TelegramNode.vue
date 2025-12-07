<script setup lang="ts">
import { computed } from "vue";
import BaseNode from "./BaseNode.vue";

interface Props {
  data: {
    label: string;
    config?: {
      message_type?: string;
      chat_id?: string;
      text?: string;
      file_source?: string;
    };
    metadata?: Record<string, any>;
  };
  selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const messageType = computed(() => props.data.config?.message_type || "text");
const chatId = computed(() => props.data.config?.chat_id || "");
const displayText = computed(() => {
  if (messageType.value === "text") {
    return props.data.config?.text || "";
  }
  return `[${messageType.value.toUpperCase()}]`;
});

const messageTypeColor = computed(() => {
  const colors: Record<string, string> = {
    text: "bg-blue-100 text-blue-700",
    photo: "bg-green-100 text-green-700",
    document: "bg-orange-100 text-orange-700",
    audio: "bg-purple-100 text-purple-700",
    video: "bg-red-100 text-red-700",
  };
  return colors[messageType.value] || "bg-gray-100 text-gray-700";
});
</script>

<template>
  <BaseNode :data="data" :selected="selected" type="telegram">
    <div class="mt-2 space-y-1">
      <!-- Message Type Badge -->
      <div class="flex items-center gap-2">
        <span
          :class="[
            messageTypeColor,
            'rounded px-2 py-0.5 text-xs font-semibold uppercase',
          ]"
        >
          {{ messageType }}
        </span>
      </div>

      <!-- Chat ID -->
      <div v-if="chatId" class="flex items-center gap-1 text-xs text-gray-500">
        <span class="font-medium">To:</span>
        <span class="truncate" :title="chatId">{{ chatId }}</span>
      </div>

      <!-- Content Preview -->
      <div
        v-if="displayText"
        class="line-clamp-2 text-xs text-gray-600"
        :title="displayText"
      >
        {{ displayText }}
      </div>
    </div>
  </BaseNode>
</template>
