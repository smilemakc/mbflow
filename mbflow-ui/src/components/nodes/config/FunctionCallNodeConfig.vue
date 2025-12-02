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
      <MonacoEditor
        v-model="argumentsJson"
        language="json"
        height="200px"
        :node-id="nodeId"
      />
      <div class="help-text">
        Enter function arguments as JSON object. You can use template variables
        like <code>{"{{"}env.api_key{"}}"}</code> or
        <code>{"{{"}input.user_id{"}}"}</code>
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
import { ref, watch, computed } from "vue";
import type { FunctionCallNodeConfig } from "@/types/nodes";
import TemplateInput from "@/components/common/TemplateInput.vue";
import MonacoEditor from "@/components/common/MonacoEditor.vue";

interface Props {
  config: FunctionCallNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: FunctionCallNodeConfig): void;
}>();

const localConfig = ref<FunctionCallNodeConfig>({ ...props.config });

// Convert arguments object to/from JSON string for editor
const argumentsJson = computed({
  get: () => {
    try {
      return JSON.stringify(localConfig.value.arguments || {}, null, 2);
    } catch {
      return "{}";
    }
  },
  set: (value: string) => {
    try {
      localConfig.value.arguments = JSON.parse(value);
    } catch {
      // Keep previous value if parsing fails
    }
  },
});

// Watch for external config changes
watch(
  () => props.config,
  (newConfig) => {
    localConfig.value = { ...newConfig };
  },
  { deep: true }
);

// Emit changes
watch(
  localConfig,
  (newConfig) => {
    emit("update:config", newConfig);
  },
  { deep: true }
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
