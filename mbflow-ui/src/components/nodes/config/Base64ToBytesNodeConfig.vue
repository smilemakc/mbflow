<template>
  <div class="base64-to-bytes-config">
    <div class="form-group">
      <label class="label">Encoding Format</label>
      <select v-model="localConfig.encoding" class="select-field">
        <option value="standard">Standard Base64</option>
        <option value="url">URL-safe Base64</option>
        <option value="raw_standard">Raw Standard (no padding)</option>
        <option value="raw_url">Raw URL-safe (no padding)</option>
      </select>
      <p class="mt-1 text-xs text-gray-500">
        Choose the base64 encoding variant to decode from
      </p>
    </div>

    <div class="form-group">
      <label class="label">Output Format</label>
      <select v-model="localConfig.output_format" class="select-field">
        <option value="raw">Raw Bytes</option>
        <option value="hex">Hexadecimal String</option>
      </select>
      <p class="mt-1 text-xs text-gray-500">
        Format of the decoded output
      </p>
    </div>

    <div class="info-box">
      <h4 class="info-title">💡 Usage Examples</h4>
      <ul class="examples-list">
        <li><strong>Standard:</strong> Decode regular base64 (e.g., from API responses)</li>
        <li><strong>URL-safe:</strong> For URL parameters and file names</li>
        <li><strong>Raw:</strong> Base64 without padding characters (=)</li>
        <li><strong>Hex output:</strong> For debugging or when you need hexadecimal representation</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { Base64ToBytesNodeConfig } from "@/types/nodes";

interface Props {
  config: Base64ToBytesNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: Base64ToBytesNodeConfig): void;
}>();

const localConfig = ref<Base64ToBytesNodeConfig>({
  encoding: "standard",
  output_format: "raw",
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
.base64-to-bytes-config {
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
  cursor: pointer;
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

.info-box {
  padding: 16px;
  background-color: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 8px;
}

.info-title {
  font-size: 13px;
  font-weight: 700;
  color: #0c4a6e;
  margin: 0 0 12px 0;
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
</style>
