<template>
  <div class="conditional-node-config">
    <div class="form-group">
      <label class="label">Condition Expression</label>
      <TemplateInput
        v-model="localConfig.condition"
        height="100px"
        :node-id="nodeId"
        multiline
        placeholder="{{input.value}} > 0"
      />
      <p class="mt-1 text-xs text-gray-500">
        Use expr-lang syntax to evaluate conditions. Returns true/false.
      </p>
    </div>

    <div class="info-box">
      <h4 class="info-title">💡 Expression Examples</h4>
      <ul class="examples-list">
        <li><code v-pre>{{input.value}} > 100</code> - Numeric comparison</li>
        <li><code v-pre>{{input.status}} == "active"</code> - String equality</li>
        <li>
          <code v-pre>{{input.count}} > 0 && {{input.enabled}}</code> - Multiple
          conditions
        </li>
        <li>
          <code v-pre>len({{input.items}}) > 5</code> - Array length check
        </li>
      </ul>
    </div>

    <div class="info-box info-box-warning">
      <h4 class="info-title">⚠️ Important Notes</h4>
      <ul class="notes-list">
        <li>
          <strong>True branch:</strong> Nodes connected via edge labeled "true"
          or default first edge
        </li>
        <li>
          <strong>False branch:</strong> Nodes connected via edge labeled
          "false" or second edge
        </li>
        <li>
          If condition evaluates to true, only the true branch will execute
        </li>
        <li>Templates like <code v-pre>{{input.X}}</code> are resolved before evaluation</li>
      </ul>
    </div>

    <div class="reference-box">
      <h4 class="reference-title">📚 Supported Operators</h4>
      <div class="operators-grid">
        <div class="operator-category">
          <h5>Comparison</h5>
          <code>==, !=, >, <, >=, <=</code>
        </div>
        <div class="operator-category">
          <h5>Logical</h5>
          <code>&&, ||, !</code>
        </div>
        <div class="operator-category">
          <h5>Arithmetic</h5>
          <code>+, -, *, /, %</code>
        </div>
        <div class="operator-category">
          <h5>Functions</h5>
          <code>len(), contains(), matches()</code>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { ConditionalNodeConfig } from "@/types/nodes";
import TemplateInput from "@/components/common/TemplateInput.vue";

interface Props {
  config: ConditionalNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: ConditionalNodeConfig): void;
}>();

const localConfig = ref<ConditionalNodeConfig>({
  condition: "{{input.value}} > 0",
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
.conditional-node-config {
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

.info-box {
  padding: 16px;
  background-color: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 8px;
}

.info-box-warning {
  background-color: #fef3c7;
  border-color: #fde68a;
}

.info-title {
  font-size: 13px;
  font-weight: 700;
  color: #0c4a6e;
  margin: 0 0 12px 0;
}

.info-box-warning .info-title {
  color: #92400e;
}

.examples-list,
.notes-list {
  margin: 0;
  padding-left: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.examples-list li,
.notes-list li {
  font-size: 12px;
  color: #374151;
  line-height: 1.5;
}

code {
  background-color: #ffffff;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: "Monaco", "Menlo", monospace;
  font-size: 11px;
  color: #0369a1;
  border: 1px solid #e0f2fe;
}

.reference-box {
  padding: 16px;
  background-color: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.reference-title {
  font-size: 13px;
  font-weight: 700;
  color: #374151;
  margin: 0 0 12px 0;
}

.operators-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.operator-category h5 {
  font-size: 11px;
  font-weight: 600;
  color: #6b7280;
  margin: 0 0 4px 0;
  text-transform: uppercase;
}

.operator-category code {
  display: block;
  font-size: 12px;
}
</style>
