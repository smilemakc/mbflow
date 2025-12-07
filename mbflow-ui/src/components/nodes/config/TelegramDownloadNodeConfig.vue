<template>
  <div class="telegram-download-config space-y-4">
    <!-- Credentials -->
    <div class="space-y-4 rounded-md border border-gray-200 bg-gray-50 p-3">
      <h4 class="text-xs font-semibold uppercase text-gray-500">Credentials</h4>

      <div class="space-y-1">
        <label class="text-sm font-medium text-gray-700">Bot Token</label>
        <TemplateInput
          v-model="localConfig.bot_token"
          placeholder="{{env.TELEGRAM_BOT_TOKEN}}"
          :node-id="nodeId"
        />
      </div>
    </div>

    <!-- File Settings -->
    <div class="space-y-4">
      <h4 class="text-xs font-semibold uppercase text-gray-500">File</h4>

      <div class="space-y-1">
        <label class="text-sm font-medium text-gray-700">File ID</label>
        <TemplateInput
          v-model="localConfig.file_id"
          placeholder="{{input.message.document.file_id}}"
          :node-id="nodeId"
        />
        <p class="text-xs text-gray-500">
          File ID from Telegram message (photo, document, audio, video, etc.)
        </p>
      </div>

      <Select
        v-model="localConfig.output_format"
        label="Output Format"
        :options="outputFormatOptions"
      />

      <div class="space-y-1">
        <label class="text-sm font-medium text-gray-700">Timeout (seconds)</label>
        <input
          v-model.number="localConfig.timeout"
          type="number"
          min="1"
          max="300"
          class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from "vue";
import TemplateInput from "@/components/common/TemplateInput.vue";
import Select from "@/components/ui/Select.vue";
import type { TelegramDownloadNodeConfig } from "@/types/nodes";

const props = defineProps<{
  config: TelegramDownloadNodeConfig;
  nodeId: string;
}>();

const emit = defineEmits<{
  (e: "update:config", config: TelegramDownloadNodeConfig): void;
}>();

const localConfig = ref<TelegramDownloadNodeConfig>({
  ...props.config,
  bot_token: props.config.bot_token || "",
  file_id: props.config.file_id || "",
  output_format: props.config.output_format || "base64",
  timeout: props.config.timeout || 60,
});

const outputFormatOptions = computed(() => [
  { label: "Base64 (download content)", value: "base64" },
  { label: "URL (link only)", value: "url" },
]);

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
