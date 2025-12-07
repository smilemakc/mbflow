<template>
  <div class="function-call-node-config">
    <div class="form-group">
      <label class="label">Function Name</label>
      <TemplateInput
        v-model="localConfig.function_name"
        placeholder="my_function"
        :node-id="nodeId"
      />
    </div>

    <div class="form-group">
      <label class="label">Arguments (JSON)</label>
      <TemplateInput
        v-model="argumentsStr"
        :multiline="true"
        :rows="6"
        placeholder='{"key": "{{input.value}}"}'
        :node-id="nodeId"
      />
      <div class="help-text">
        Enter function arguments as JSON object. You can use template variables
        like <code v-pre>{{env.api_key}}</code> or
        <code v-pre>{{input.user_id}}</code>
      </div>
    </div>

    <div class="form-group">
      <label class="label">Timeout (seconds)</label>
      <input
        v-model.number="localConfig.timeout_seconds"
        type="number"
        min="1"
        max="300"
        class="input-field"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { FunctionCallNodeConfig } from "@/types/nodes";
import TemplateInput from "@/components/common/TemplateInput.vue";

interface Props {
  config: FunctionCallNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: FunctionCallNodeConfig): void;
}>();

const localConfig = ref<FunctionCallNodeConfig>({
  function_name: "",
  arguments: {},
  timeout_seconds: 30,
  ...(props.config || {}),
});

// Store arguments as string to support template variables like {{input.x}}
const argumentsStr = ref<string>(
  typeof props.config?.arguments === "string"
    ? props.config.arguments
    : JSON.stringify(props.config?.arguments || {}, null, 2)
);

// Sync argumentsStr to localConfig.arguments
watch(argumentsStr, (newValue) => {
  // Store as string - backend will parse after template resolution
  (localConfig.value as any).arguments_template = newValue;
  // Try to parse for validation preview, but don't reject invalid JSON
  try {
    localConfig.value.arguments = JSON.parse(newValue);
  } catch {
    // Keep as template string - will be resolved at runtime
    localConfig.value.arguments = newValue as any;
  }
});

// Watch for external config changes
// Watch for external config changes
watch(
  () => props.config,
  (newConfig) => {
    if (JSON.stringify(newConfig) !== JSON.stringify(localConfig.value)) {
      localConfig.value = { ...newConfig };
    }
  },
  { deep: true },
);

// Emit changes
watch(
  localConfig,
  (newConfig) => {
    emit("update:config", newConfig);
  },
  { deep: true },
);
</script>

<style scoped>
.function-call-node-config {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.label {
  font-size: 13px;
  font-weight: 600;
  color: #374151;
}

.input-field {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.2s;
}

.input-field:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.help-text {
  font-size: 12px;
  color: #6b7280;
  background-color: #f9fafb;
  padding: 12px;
  border-radius: 6px;
  line-height: 1.6;
}

.help-text code {
  background-color: #e5e7eb;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 11px;
}
</style>
