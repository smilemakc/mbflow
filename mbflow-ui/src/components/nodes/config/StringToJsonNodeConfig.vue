<template>
  <div class="string-to-json-config">
    <div class="form-group">
      <label class="checkbox-label">
        <input v-model="localConfig.strict_mode" type="checkbox" class="checkbox-field" />
        <span>Strict Mode</span>
      </label>
      <p class="mt-1 text-xs text-gray-500">
        When enabled, fails on invalid JSON. When disabled, returns null on parse errors.
      </p>
    </div>

    <div class="form-group">
      <label class="checkbox-label">
        <input v-model="localConfig.trim_whitespace" type="checkbox" class="checkbox-field" />
        <span>Trim Whitespace</span>
      </label>
      <p class="mt-1 text-xs text-gray-500">
        Remove leading and trailing whitespace before parsing
      </p>
    </div>

    <div class="info-box">
      <h4 class="info-title">💡 Usage Tips</h4>
      <ul class="examples-list">
        <li><strong>Strict Mode ON:</strong> Best for validating API responses or user input</li>
        <li><strong>Strict Mode OFF:</strong> Useful for optional/nullable JSON fields</li>
        <li><strong>Trim Whitespace:</strong> Recommended when parsing from text files or user input</li>
      </ul>
    </div>

    <div class="info-box info-box-example">
      <h4 class="info-title">📝 Example Input</h4>
      <pre class="code-block">{"name": "John", "age": 30, "active": true}</pre>
      <p class="text-xs text-gray-600 mt-2">
        <strong>Output:</strong> JavaScript object with parsed values
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { StringToJsonNodeConfig } from "@/types/nodes";

interface Props {
  config: StringToJsonNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: StringToJsonNodeConfig): void;
}>();

const localConfig = ref<StringToJsonNodeConfig>({
  strict_mode: true,
  trim_whitespace: true,
  ...(props.config || {}),
});

// Watch for external config changes
watch(
  () => props.config,
  (newConfig) => {
    if (JSON.stringify(newConfig) !== JSON.stringify(localConfig.value)) {
      localConfig.value = { ...newConfig };
    }
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
.string-to-json-config {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  color: #374151;
}

.checkbox-field {
  width: 16px;
  height: 16px;
  cursor: pointer;
  accent-color: #3b82f6;
}

.info-box {
  padding: 16px;
  background-color: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 8px;
}

.info-box-example {
  background-color: #fef3c7;
  border-color: #fde68a;
}

.info-title {
  font-size: 13px;
  font-weight: 700;
  color: #0c4a6e;
  margin: 0 0 12px 0;
}

.info-box-example .info-title {
  color: #92400e;
}

.examples-list {
  margin: 0;
  padding-left: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.examples-list li {
  font-size: 12px;
  color: #374151;
  line-height: 1.5;
}

.code-block {
  background-color: #ffffff;
  padding: 12px;
  border-radius: 4px;
  font-family: "Monaco", "Menlo", monospace;
  font-size: 12px;
  color: #1f2937;
  border: 1px solid #e5e7eb;
  overflow-x: auto;
  margin: 0;
}
</style>
