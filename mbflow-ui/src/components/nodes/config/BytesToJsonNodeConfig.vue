<template>
  <div class="bytes-to-json-config">
    <div class="form-group">
      <label class="label">Encoding</label>
      <select v-model="localConfig.encoding" class="select-field">
        <option value="utf-8">UTF-8 (with BOM detection)</option>
        <option value="utf-16">UTF-16</option>
        <option value="latin1">Latin-1 (ISO-8859-1)</option>
      </select>
      <p class="mt-1 text-xs text-gray-500">
        Character encoding of the input bytes. UTF-8 automatically detects BOM markers.
      </p>
    </div>

    <div class="form-group">
      <label class="checkbox-label">
        <input v-model="localConfig.validate_json" type="checkbox" class="checkbox-field" />
        <span>Validate JSON</span>
      </label>
      <p class="mt-1 text-xs text-gray-500">
        If enabled, returns error on invalid JSON. If disabled, returns null for invalid JSON.
      </p>
    </div>

    <div class="info-box">
      <h4 class="info-title">📋 Input/Output</h4>
      <div class="example-section">
        <p class="example-label">Input:</p>
        <ul class="examples-list">
          <li><code>[]byte</code> - Raw bytes</li>
          <li><code>string</code> - Base64 encoded string or raw string</li>
          <li><code>map</code> - Object with "data" field</li>
        </ul>
      </div>
      <div class="example-section mt-2">
        <p class="example-label">Output:</p>
        <ul class="examples-list">
          <li><code>result</code> - Parsed JSON object/array</li>
          <li><code>encoding_used</code> - Detected encoding</li>
          <li><code>byte_size</code> - Original byte size</li>
        </ul>
      </div>
    </div>

    <div class="info-box info-box-example">
      <h4 class="info-title">💡 Common Use Cases</h4>
      <ul class="examples-list">
        <li><strong>UTF-8 with BOM:</strong> Telegram downloads, web responses</li>
        <li><strong>UTF-16:</strong> Windows text files, .NET exports</li>
        <li><strong>Latin-1:</strong> Legacy systems, some European text files</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { BytesToJsonNodeConfig } from "@/types/nodes";

interface Props {
  config: BytesToJsonNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: BytesToJsonNodeConfig): void;
}>();

const localConfig = ref<BytesToJsonNodeConfig>({
  encoding: "utf-8",
  validate_json: true,
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
.bytes-to-json-config {
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

.select-field {
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  color: #374151;
  background-color: white;
  transition: border-color 0.2s;
}

.select-field:hover {
  border-color: #9ca3af;
}

.select-field:focus {
  outline: none;
  border-color: #3b82f6;
  ring: 2px;
  ring-color: #3b82f620;
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
  gap: 6px;
}

.examples-list li {
  font-size: 12px;
  color: #374151;
  line-height: 1.5;
}

.example-section {
  margin-top: 8px;
}

.example-label {
  font-size: 11px;
  font-weight: 600;
  color: #0c4a6e;
  margin-bottom: 4px;
}

.info-box-example .example-label {
  color: #92400e;
}

code {
  background-color: #ffffff;
  padding: 2px 4px;
  border-radius: 3px;
  font-family: "Monaco", "Menlo", monospace;
  font-size: 11px;
  color: #0369a1;
  border: 1px solid #e0f2fe;
}
</style>
