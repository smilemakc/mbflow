<template>
  <div class="transform-node-config">
    <div class="form-group">
      <label class="label">Language</label>
      <select v-model="localConfig.language" class="select-field">
        <option v-for="lang in TRANSFORM_LANGUAGES" :key="lang" :value="lang">
          {{ lang.toUpperCase() }}
        </option>
      </select>
    </div>

    <div class="form-group">
      <label class="label">
        Expression
        <span class="hint">
          {{
            localConfig.language === "jq" ? "jq filter" : "JavaScript function"
          }}
        </span>
      </label>
      <MonacoEditor
        v-model="localConfig.expression"
        :language="editorLanguage"
        height="250px"
        :node-id="nodeId"
      />
      <div class="help-text">
        <template v-if="localConfig.language === 'jq'">
          <strong>jq Examples:</strong><br />
          <code>.</code> - Pass through input<br />
          <code>.field</code> - Extract field<br />
          <code>{"{"} name: .user.name, count: .items | length {"}"}</code> -
          Transform structure
        </template>
        <template v-else>
          <strong>JavaScript Example:</strong><br />
          <code
            >return {"{"} name: input.user.name, count: input.items.length
            {"}"};</code
          >
        </template>
      </div>
    </div>

    <div class="form-group">
      <label class="label">Timeout (seconds)</label>
      <input
        v-model.number="localConfig.timeout_seconds"
        type="number"
        min="1"
        max="60"
        class="input-field"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from "vue";
import type { TransformNodeConfig } from "@/types/nodes";
import { TRANSFORM_LANGUAGES } from "@/types/nodes";
import MonacoEditor from "@/components/common/MonacoEditor.vue";

interface Props {
  config: TransformNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: TransformNodeConfig): void;
}>();

const localConfig = ref<TransformNodeConfig>({ ...props.config });

const editorLanguage = computed(() => {
  return localConfig.value.language === "jq" ? "jq" : "javascript";
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
.transform-node-config {
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
  display: flex;
  align-items: center;
  gap: 8px;
}

.hint {
  font-size: 11px;
  font-weight: 400;
  color: #9ca3af;
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

.help-text strong {
  color: #374151;
}
</style>
