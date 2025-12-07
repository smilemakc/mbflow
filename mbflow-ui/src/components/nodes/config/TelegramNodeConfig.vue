<template>
  <div class="telegram-config space-y-4">
    <!-- API Credentials -->
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

      <div class="space-y-1">
        <label class="text-sm font-medium text-gray-700">Chat ID</label>
        <TemplateInput
          v-model="localConfig.chat_id"
          placeholder="@channel_name or {{env.CHAT_ID}}"
          :node-id="nodeId"
        />
      </div>
    </div>

    <!-- Message Settings -->
    <div class="space-y-4">
      <h4 class="text-xs font-semibold uppercase text-gray-500">Message</h4>

      <Select
        v-model="localConfig.message_type"
        label="Message Type"
        :options="messageTypeOptions"
      />

      <!-- Text Content -->
      <div v-if="localConfig.message_type === 'text'" class="space-y-1">
        <label class="text-sm font-medium text-gray-700">Message Text</label>
        <TemplateInput
          v-model="localConfig.text"
          :multiline="true"
          :rows="4"
          placeholder="Hello world! {{input.data}}"
          :node-id="nodeId"
        />
      </div>

      <!-- Media Settings -->
      <template v-else>
        <!-- Caption -->
        <div class="space-y-1">
          <label class="text-sm font-medium text-gray-700">
            Caption (Optional)
          </label>
          <TemplateInput
            v-model="localConfig.text"
            :multiline="true"
            :rows="2"
            placeholder="Media caption..."
            :node-id="nodeId"
          />
        </div>

        <div class="space-y-4 rounded-md border border-gray-200 p-3">
          <h5 class="text-xs font-medium text-gray-700">File Settings</h5>

          <Select
            v-model="localConfig.file_source"
            label="File Source"
            :options="fileSourceOptions"
          />

          <div class="space-y-1">
            <label class="text-sm font-medium text-gray-700">
              {{
                localConfig.file_source === "url"
                  ? "File URL"
                  : localConfig.file_source === "file_id"
                    ? "File ID"
                    : "Base64 Data"
              }}
            </label>
            <TemplateInput
              v-model="localConfig.file_data"
              :placeholder="
                localConfig.file_source === 'url'
                  ? 'https://example.com/image.jpg'
                  : 'File data...'
              "
              :node-id="nodeId"
            />
          </div>

          <div
            v-if="
              localConfig.file_source === 'base64' ||
              localConfig.file_source === 'url'
            "
            class="space-y-1"
          >
            <label class="text-sm font-medium text-gray-700">
              File Name (Optional)
            </label>
            <TemplateInput
              v-model="localConfig.file_name"
              placeholder="image.jpg"
              :node-id="nodeId"
            />
          </div>
        </div>
      </template>

      <!-- Common Options -->
      <div class="space-y-1">
        <Select
          v-model="localConfig.parse_mode"
          label="Parse Mode"
          :options="parseModeOptions"
        />
      </div>

      <div class="flex flex-col gap-2 pt-2">
        <Checkbox
          :model-value="localConfig.disable_web_page_preview ?? false"
          @update:model-value="localConfig.disable_web_page_preview = $event"
          label="Disable Web Page Preview"
        />
        <Checkbox
          :model-value="localConfig.disable_notification ?? false"
          @update:model-value="localConfig.disable_notification = $event"
          label="Disable Notification"
        />
        <Checkbox
          :model-value="localConfig.protect_content ?? false"
          @update:model-value="localConfig.protect_content = $event"
          label="Protect Content (Prevent Forwarding)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from "vue";
import TemplateInput from "@/components/common/TemplateInput.vue";
import Select from "@/components/ui/Select.vue";
import Checkbox from "@/components/ui/Checkbox.vue";
import {
  TELEGRAM_MESSAGE_TYPES,
  TELEGRAM_PARSE_MODES,
  TELEGRAM_FILE_SOURCES,
  type TelegramNodeConfig,
} from "@/types/nodes";

const props = defineProps<{
  config: TelegramNodeConfig;
  nodeId: string;
}>();

const emit = defineEmits<{
  (e: "update:config", config: TelegramNodeConfig): void;
}>();

// Initialize local config copy
const localConfig = ref<TelegramNodeConfig>({
  ...props.config,
  bot_token: props.config.bot_token || "",
  chat_id: props.config.chat_id || "",
  message_type: props.config.message_type || "text",
  text: props.config.text || "",
  parse_mode: props.config.parse_mode || "HTML",
  file_source: props.config.file_source || "url",
  file_data: props.config.file_data || "",
  file_name: props.config.file_name || "",
  disable_notification: props.config.disable_notification ?? false,
  protect_content: props.config.protect_content ?? false,
  disable_web_page_preview: props.config.disable_web_page_preview ?? false,
});

// Options for selects
const messageTypeOptions = computed(() =>
  TELEGRAM_MESSAGE_TYPES.map((t) => ({
    label: t.charAt(0).toUpperCase() + t.slice(1),
    value: t,
  })),
);

const parseModeOptions = computed(() =>
  TELEGRAM_PARSE_MODES.map((m) => ({
    label: m,
    value: m,
  })),
);

const fileSourceOptions = computed(() =>
  TELEGRAM_FILE_SOURCES.map((s) => ({
    label: s === "file_id" ? "File ID" : s.toUpperCase(),
    value: s,
  })),
);

// Watch for changes and emit updates
watch(
  localConfig,
  (newConfig) => {
    emit("update:config", { ...newConfig });
  },
  { deep: true },
);

// Watch for external prop changes
watch(
  () => props.config,
  (newConfig) => {
    if (JSON.stringify(newConfig) !== JSON.stringify(localConfig.value)) {
      localConfig.value = { ...localConfig.value, ...newConfig };
    }
  },
  { deep: true },
);
</script>
