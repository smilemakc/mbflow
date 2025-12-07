<template>
  <div class="json-to-string-config">
    <div class="form-group">
      <label class="checkbox-label">
        <input v-model="localConfig.pretty" type="checkbox" class="checkbox-field" />
        <span>Pretty Print</span>
      </label>
      <p class="mt-1 text-xs text-gray-500">
        Format JSON with indentation for better readability
      </p>
    </div>

    <div v-if="localConfig.pretty" class="form-group">
      <label class="label">Indentation</label>
      <input
        v-model="localConfig.indent"
        type="text"
        class="input-field"
        placeholder="  "
      />
      <p class="mt-1 text-xs text-gray-500">
        Characters to use for indentation (default: 2 spaces)
      </p>
    </div>

    <div class="form-group">
      <label class="checkbox-label">
        <input v-model="localConfig.escape_html" type="checkbox" class="checkbox-field" />
        <span>Escape HTML Characters</span>
      </label>
      <p class="mt-1 text-xs text-gray-500">
        Escape <code>&lt;</code>, <code>&gt;</code>, <code>&amp;</code> to prevent XSS attacks
      </p>
    </div>

    <div class="form-group">
      <label class="checkbox-label">
        <input v-model="localConfig.sort_keys" type="checkbox" class="checkbox-field" />
        <span>Sort Object Keys</span>
      </label>
      <p class="mt-1 text-xs text-gray-500">
        Sort all object keys alphabetically for consistent output
      </p>
    </div>

    <div class="info-box">
      <h4 class="info-title">💡 Common Configurations</h4>
      <ul class="examples-list">
        <li><strong>API Response:</strong> Pretty OFF, Escape HTML ON</li>
        <li><strong>Logging/Debug:</strong> Pretty ON, Sort Keys ON</li>
        <li><strong>Web Display:</strong> Pretty ON, Escape HTML ON</li>
        <li><strong>File Export:</strong> Pretty ON with custom indent</li>
      </ul>
    </div>

    <div class="info-box info-box-example">
      <h4 class="info-title">📝 Output Examples</h4>
      <div class="example-section">
        <p class="example-label">Compact (pretty: false):</p>
        <pre class="code-block">{"name":"John","age":30}</pre>
      </div>
      <div class="example-section mt-3">
        <p class="example-label">Pretty (pretty: true):</p>
        <pre class="code-block">{
  "name": "John",
  "age": 30
}</pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { JsonToStringNodeConfig } from "@/types/nodes";

interface Props {
  config: JsonToStringNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: JsonToStringNodeConfig): void;
}>();

const localConfig = ref<JsonToStringNodeConfig>({
  pretty: false,
  indent: "  ",
  escape_html: true,
  sort_keys: false,
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
.json-to-string-config {
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

.example-section {
  margin-top: 12px;
}

.example-label {
  font-size: 11px;
  font-weight: 600;
  color: #92400e;
  margin-bottom: 4px;
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
