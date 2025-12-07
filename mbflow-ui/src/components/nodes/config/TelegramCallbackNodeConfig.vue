<template>
  <div class="telegram-callback-config space-y-4">
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

    <!-- Callback Settings -->
    <div class="space-y-4">
      <h4 class="text-xs font-semibold uppercase text-gray-500">
        Callback Response
      </h4>

      <div class="space-y-1">
        <label class="text-sm font-medium text-gray-700">Callback Query ID</label>
        <TemplateInput
          v-model="localConfig.callback_query_id"
          placeholder="{{input.callback_query.id}}"
          :node-id="nodeId"
        />
      </div>

      <div class="space-y-1">
        <label class="text-sm font-medium text-gray-700">
          Notification Text (Optional)
        </label>
        <TemplateInput
          v-model="localConfig.text"
          placeholder="Processing..."
          :node-id="nodeId"
        />
        <p class="text-xs text-gray-500">Max 200 characters</p>
      </div>

      <Checkbox
        :model-value="localConfig.show_alert ?? false"
        @update:model-value="localConfig.show_alert = $event"
        label="Show as Alert Dialog"
      />
      <p class="-mt-2 ml-6 text-xs text-gray-500">
        Show modal alert instead of toast notification
      </p>

      <div class="space-y-1">
        <label class="text-sm font-medium text-gray-700">
          Cache Time (seconds)
        </label>
        <input
          v-model.number="localConfig.cache_time"
          type="number"
          min="0"
          class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
          placeholder="0"
        />
        <p class="text-xs text-gray-500">
          How long to cache the answer (0 = no caching)
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import TemplateInput from "@/components/common/TemplateInput.vue";
import Checkbox from "@/components/ui/Checkbox.vue";
import type { TelegramCallbackNodeConfig } from "@/types/nodes";

const props = defineProps<{
  config: TelegramCallbackNodeConfig;
  nodeId: string;
}>();

const emit = defineEmits<{
  (e: "update:config", config: TelegramCallbackNodeConfig): void;
}>();

const localConfig = ref<TelegramCallbackNodeConfig>({
  ...props.config,
  bot_token: props.config.bot_token || "",
  callback_query_id: props.config.callback_query_id || "",
  text: props.config.text || "",
  show_alert: props.config.show_alert ?? false,
  cache_time: props.config.cache_time || 0,
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
