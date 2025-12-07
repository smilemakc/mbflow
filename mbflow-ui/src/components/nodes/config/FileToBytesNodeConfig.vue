<template>
  <div class="file-to-bytes-config">
    <div class="form-group">
      <label class="label">Storage ID</label>
      <input
        v-model="localConfig.storage_id"
        type="text"
        class="input-field"
        placeholder="default"
      />
      <p class="mt-1 text-xs text-gray-500">
        Storage identifier. Leave empty or use "default" for default storage.
      </p>
    </div>

    <div class="form-group">
      <label class="label">File ID <span class="text-red-500">*</span></label>
      <input
        v-model="localConfig.file_id"
        type="text"
        class="input-field"
        placeholder="{{input.file_id}}"
      />
      <p class="mt-1 text-xs text-gray-500">
        File ID to read from storage. Supports templates like <code>{{placeholderTemplate}}</code>.
      </p>
    </div>

    <div class="form-group">
      <label class="label">Output Format</label>
      <select v-model="localConfig.output_format" class="select-field">
        <option value="base64">Base64 String</option>
        <option value="raw">Raw Bytes</option>
      </select>
      <p class="mt-1 text-xs text-gray-500">
        <strong>base64:</strong> Returns file content as base64-encoded string (recommended for text/JSON workflows).<br />
        <strong>raw:</strong> Returns raw bytes (use for binary processing).
      </p>
    </div>

    <div class="info-box">
      <h4 class="info-title">📋 Input/Output</h4>
      <div class="example-section">
        <p class="example-label">Input:</p>
        <ul class="examples-list">
          <li><code>string</code> - File ID directly</li>
          <li><code>map</code> - Object with "file_id" field</li>
        </ul>
      </div>
      <div class="example-section mt-2">
        <p class="example-label">Output:</p>
        <ul class="examples-list">
          <li><code>result</code> - File content (base64 or raw bytes)</li>
          <li><code>file_id</code> - File ID</li>
          <li><code>file_name</code> - Original file name</li>
          <li><code>mime_type</code> - File MIME type</li>
          <li><code>size</code> - File size in bytes</li>
          <li><code>format</code> - Output format used</li>
        </ul>
      </div>
    </div>

    <div class="info-box info-box-example">
      <h4 class="info-title">💡 Common Use Cases</h4>
      <ul class="examples-list">
        <li><strong>Base64:</strong> Load files for JSON APIs, LLM processing, or web responses</li>
        <li><strong>Raw:</strong> Binary processing, encryption, or further byte transformations</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { FileToBytesNodeConfig } from "@/types/nodes";

interface Props {
  config: FileToBytesNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: FileToBytesNodeConfig): void;
}>();

const placeholderTemplate = "{{input.file_id}}";
const localConfig = ref<FileToBytesNodeConfig>({
  storage_id: "default",
  file_id: "",
  output_format: "base64",
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
.file-to-bytes-config {
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
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  color: #374151;
  background-color: white;
  transition: border-color 0.2s;
  font-family: "Monaco", "Menlo", monospace;
}

.input-field:hover {
  border-color: #9ca3af;
}

.input-field:focus {
  outline: none;
  border-color: #3b82f6;
  ring: 2px;
  ring-color: #3b82f620;
}

.input-field::placeholder {
  color: #9ca3af;
  font-style: italic;
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
