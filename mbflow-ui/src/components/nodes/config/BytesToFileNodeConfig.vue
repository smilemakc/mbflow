<template>
  <div class="bytes-to-file-config">
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
      <label class="label">File Name <span class="text-red-500">*</span></label>
      <input
        v-model="localConfig.file_name"
        type="text"
        class="input-field"
        placeholder="output.bin"
      />
      <p class="mt-1 text-xs text-gray-500">
        Name for the stored file. Supports templates like <code v-pre>{{placeholderTemplate}}</code>.
      </p>
    </div>

    <div class="form-group">
      <label class="label">MIME Type</label>
      <input
        v-model="localConfig.mime_type"
        type="text"
        class="input-field"
        placeholder="Auto-detect from content"
      />
      <p class="mt-1 text-xs text-gray-500">
        Leave empty for auto-detection. Examples: <code>application/json</code>, <code>image/png</code>, <code>text/plain</code>.
      </p>
    </div>

    <div class="form-group">
      <label class="label">Access Scope</label>
      <select v-model="localConfig.access_scope" class="select-field">
        <option value="workflow">Workflow</option>
        <option value="edge">Edge</option>
        <option value="result">Result</option>
      </select>
      <p class="mt-1 text-xs text-gray-500">
        <strong>workflow:</strong> Accessible within the entire workflow execution.<br />
        <strong>edge:</strong> Accessible only between connected nodes.<br />
        <strong>result:</strong> Included in the final workflow result.
      </p>
    </div>

    <div class="form-group">
      <label class="label">TTL (Time to Live)</label>
      <input
        v-model.number="localConfig.ttl"
        type="number"
        min="0"
        class="input-field"
        placeholder="0 (no expiration)"
      />
      <p class="mt-1 text-xs text-gray-500">
        Time-to-live in seconds. 0 means no expiration. Example: 3600 = 1 hour.
      </p>
    </div>

    <div class="form-group">
      <label class="label">Tags</label>
      <div class="tags-input">
        <div v-for="(tag, index) in localConfig.tags" :key="index" class="tag-item">
          <span class="tag-text">{{ tag }}</span>
          <button @click="removeTag(index)" class="tag-remove" type="button">×</button>
        </div>
        <input
          v-model="newTag"
          @keydown.enter.prevent="addTag"
          type="text"
          class="tag-input-field"
          placeholder="Add tag..."
        />
      </div>
      <p class="mt-1 text-xs text-gray-500">
        Press Enter to add tags. Tags help organize and filter files.
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
          <li><code>file_id</code> - Stored file ID (UUID)</li>
          <li><code>storage_id</code> - Storage identifier</li>
          <li><code>file_name</code> - File name</li>
          <li><code>mime_type</code> - Detected/configured MIME type</li>
          <li><code>size</code> - File size in bytes</li>
          <li><code>checksum</code> - File checksum</li>
          <li><code>access_scope</code> - Access scope</li>
        </ul>
      </div>
    </div>

    <div class="info-box info-box-example">
      <h4 class="info-title">💡 Common Use Cases</h4>
      <ul class="examples-list">
        <li><strong>Workflow scope:</strong> Intermediate files used across multiple nodes</li>
        <li><strong>Edge scope:</strong> Temporary files for direct node-to-node transfer</li>
        <li><strong>Result scope:</strong> Final output files to be returned to the user</li>
        <li><strong>TTL:</strong> Cleanup temporary files automatically after processing</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { BytesToFileNodeConfig } from "@/types/nodes";

interface Props {
  config: BytesToFileNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: BytesToFileNodeConfig): void;
}>();

const placeholderTemplate = "{{input.filename}}";
const localConfig = ref<BytesToFileNodeConfig>({
  storage_id: "default",
  file_name: "",
  mime_type: "",
  access_scope: "workflow",
  ttl: 0,
  tags: [],
  ...(props.config || {}),
});

const newTag = ref("");

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

function addTag() {
  const tag = newTag.value.trim();
  if (tag && !localConfig.value.tags?.includes(tag)) {
    if (!localConfig.value.tags) {
      localConfig.value.tags = [];
    }
    localConfig.value.tags.push(tag);
    newTag.value = "";
  }
}

function removeTag(index: number) {
  if (localConfig.value.tags) {
    localConfig.value.tags.splice(index, 1);
  }
}
</script>

<style scoped>
.bytes-to-file-config {
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

.tags-input {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background-color: white;
  min-height: 42px;
  align-items: center;
}

.tags-input:focus-within {
  border-color: #3b82f6;
  ring: 2px;
  ring-color: #3b82f620;
}

.tag-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background-color: #3b82f6;
  color: white;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.tag-text {
  line-height: 1;
}

.tag-remove {
  background: none;
  border: none;
  color: white;
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  padding: 0;
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 2px;
  transition: background-color 0.2s;
}

.tag-remove:hover {
  background-color: rgba(255, 255, 255, 0.2);
}

.tag-input-field {
  flex: 1;
  min-width: 120px;
  border: none;
  outline: none;
  padding: 4px;
  font-size: 14px;
  color: #374151;
  font-family: "Monaco", "Menlo", monospace;
}

.tag-input-field::placeholder {
  color: #9ca3af;
  font-style: italic;
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
