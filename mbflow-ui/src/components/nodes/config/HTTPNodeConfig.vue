<template>
  <div class="http-node-config">
    <div class="form-group">
      <label class="label">HTTP Method</label>
      <select v-model="localConfig.method" class="select-field">
        <option v-for="method in HTTP_METHODS" :key="method" :value="method">
          {{ method }}
        </option>
      </select>
    </div>

    <div class="form-group">
      <label class="label">URL</label>
      <TemplateInput
        v-model="localConfig.url"
        placeholder="https://api.example.com/endpoint"
        :node-id="nodeId"
      />
    </div>

    <div class="form-group">
      <label class="label">Headers</label>
      <KeyValueEditor
        v-model="localConfig.headers"
        :node-id="nodeId"
        placeholder-key="Content-Type"
        placeholder-value="application/json"
      />
    </div>

    <div
      v-if="['POST', 'PUT', 'PATCH'].includes(localConfig.method)"
      class="form-group"
    >
      <label class="label">Body</label>
      <MonacoEditor
        v-model="localConfig.body"
        language="json"
        height="150px"
        :node-id="nodeId"
      />
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

    <div class="form-group">
      <label class="label">Retry Count</label>
      <input
        v-model.number="localConfig.retry_count"
        type="number"
        min="0"
        max="10"
        class="input-field"
      />
    </div>

    <div class="form-group">
      <label class="checkbox-label">
        <input
          v-model="localConfig.follow_redirects"
          type="checkbox"
          class="checkbox-field"
        />
        Follow Redirects
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { HTTPNodeConfig } from "@/types/nodes";
import { HTTP_METHODS } from "@/types/nodes";
import TemplateInput from "@/components/common/TemplateInput.vue";
import MonacoEditor from "@/components/common/MonacoEditor.vue";
import KeyValueEditor from "@/components/common/KeyValueEditor.vue";

interface Props {
  config: HTTPNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: HTTPNodeConfig): void;
}>();

const localConfig = ref<HTTPNodeConfig>({ ...props.config });

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
.http-node-config {
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

.input-field,
.select-field {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.2s;
}

.input-field:focus,
.select-field:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #374151;
  cursor: pointer;
}

.checkbox-field {
  width: 18px;
  height: 18px;
  cursor: pointer;
}
</style>
