<template>
  <div class="bytes-to-base64-config">
    <div class="form-group">
      <label class="label">Encoding Format</label>
      <select v-model="localConfig.encoding" class="select-field">
        <option value="standard">Standard Base64</option>
        <option value="url">URL-safe Base64</option>
        <option value="raw_standard">Raw Standard (no padding)</option>
        <option value="raw_url">Raw URL-safe (no padding)</option>
      </select>
      <p class="mt-1 text-xs text-gray-500">
        Choose the base64 encoding variant to use
      </p>
    </div>

    <div class="form-group">
      <label class="label">Line Length</label>
      <input
        v-model.number="localConfig.line_length"
        type="number"
        min="0"
        class="input-field"
        placeholder="0"
      />
      <p class="mt-1 text-xs text-gray-500">
        Wrap lines at specified length (0 = no wrapping, 76 = MIME format)
      </p>
    </div>

    <div class="info-box">
      <h4 class="info-title">💡 Common Use Cases</h4>
      <ul class="examples-list">
        <li><strong>Standard (no wrapping):</strong> Most common, single line output</li>
        <li><strong>MIME format (76 chars):</strong> For email attachments</li>
        <li><strong>URL-safe:</strong> Safe for URLs, filenames, and cookies</li>
        <li><strong>Raw:</strong> Compact format without padding, smaller size</li>
      </ul>
    </div>

    <div class="info-box info-box-tip">
      <h4 class="info-title">📝 Tips</h4>
      <ul class="examples-list">
        <li>Use <strong>standard</strong> for general purpose encoding</li>
        <li>Use <strong>url</strong> when encoding data for URLs or filenames</li>
        <li>Set line_length to 76 for RFC 2045 (MIME) compliance</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { BytesToBase64NodeConfig } from "@/types/nodes";

interface Props {
  config: BytesToBase64NodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: BytesToBase64NodeConfig): void;
}>();

const localConfig = ref<BytesToBase64NodeConfig>({
  encoding: "standard",
  line_length: 0,
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
.bytes-to-base64-config {
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

.select-field,
.input-field {
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  color: #374151;
  background-color: white;
  transition: border-color 0.2s;
}

.select-field {
  cursor: pointer;
}

.select-field:hover,
.input-field:hover {
  border-color: #9ca3af;
}

.select-field:focus,
.input-field:focus {
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

.info-box-tip {
  background-color: #f0fdf4;
  border-color: #bbf7d0;
}

.info-title {
  font-size: 13px;
  font-weight: 700;
  color: #0c4a6e;
  margin: 0 0 12px 0;
}

.info-box-tip .info-title {
  color: #14532d;
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
