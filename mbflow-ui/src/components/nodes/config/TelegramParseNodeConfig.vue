<template>
  <div class="telegram-parse-config space-y-4">
    <!-- Extraction Options -->
    <div class="space-y-4">
      <h4 class="text-xs font-semibold uppercase text-gray-500">
        Extraction Settings
      </h4>

      <div class="flex flex-col gap-3">
        <Checkbox
          :model-value="localConfig.extract_files ?? true"
          @update:model-value="localConfig.extract_files = $event"
          label="Extract Files"
        />
        <p class="ml-6 -mt-2 text-xs text-gray-500">
          Extract photo, document, audio, video files from messages
        </p>

        <Checkbox
          :model-value="localConfig.extract_commands ?? true"
          @update:model-value="localConfig.extract_commands = $event"
          label="Extract Commands"
        />
        <p class="ml-6 -mt-2 text-xs text-gray-500">
          Parse /commands and extract arguments
        </p>

        <Checkbox
          :model-value="localConfig.extract_entities ?? false"
          @update:model-value="localConfig.extract_entities = $event"
          label="Extract Entities"
        />
        <p class="ml-6 -mt-2 text-xs text-gray-500">
          Extract URLs, emails, and @mentions from text
        </p>
      </div>
    </div>

    <!-- Output Info -->
    <div class="rounded-md bg-blue-50 p-3">
      <h5 class="text-xs font-medium text-blue-800">Output Fields</h5>
      <ul class="mt-1 text-xs text-blue-700 list-disc list-inside space-y-0.5">
        <li><code>update_type</code> - message, callback_query, etc.</li>
        <li><code>message_type</code> - text, photo, document, etc.</li>
        <li><code>text</code> - message text or caption</li>
        <li><code>command</code> - extracted command (if enabled)</li>
        <li><code>files</code> - array of file info (if enabled)</li>
        <li><code>user</code> - sender information</li>
        <li><code>chat</code> - chat/group information</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import Checkbox from "@/components/ui/Checkbox.vue";
import type { TelegramParseNodeConfig } from "@/types/nodes";

const props = defineProps<{
  config: TelegramParseNodeConfig;
  nodeId: string;
}>();

const emit = defineEmits<{
  (e: "update:config", config: TelegramParseNodeConfig): void;
}>();

const localConfig = ref<TelegramParseNodeConfig>({
  ...props.config,
  extract_files: props.config.extract_files ?? true,
  extract_commands: props.config.extract_commands ?? true,
  extract_entities: props.config.extract_entities ?? false,
});

watch(
  localConfig,
  (newConfig) => {
    emit("update:config", { ...newConfig });
  },
  { deep: true }
);

watch(
  () => props.config,
  (newConfig) => {
    if (JSON.stringify(newConfig) !== JSON.stringify(localConfig.value)) {
      localConfig.value = { ...localConfig.value, ...newConfig };
    }
  },
  { deep: true }
);
</script>
