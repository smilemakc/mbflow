<template>
  <div class="merge-node-config">
    <div class="form-group">
      <label class="label">Merge Strategy</label>
      <select v-model="localConfig.merge_strategy" class="select-field">
        <option value="first">First (Use first available result)</option>
        <option value="last">Last (Use last available result)</option>
        <option value="all">All (Combine all results into array)</option>
        <option value="custom">Custom (Use expression)</option>
      </select>
      <p class="mt-1 text-xs text-gray-500">
        Choose how to combine results from multiple parent nodes.
      </p>
    </div>

    <!-- Custom Expression (only when strategy is 'custom') -->
    <div
      v-if="localConfig.merge_strategy === 'custom'"
      class="form-group"
    >
      <label class="label">Custom Merge Expression</label>
      <TemplateInput
        v-model="localConfig.custom_expression"
        height="100px"
        :node-id="nodeId"
        multiline
        placeholder="[{{input.parent1}}, {{input.parent2}}]"
      />
      <p class="mt-1 text-xs text-gray-500">
        Use expr-lang to define custom merge logic.
      </p>
    </div>

    <div class="info-box">
      <h4 class="info-title">💡 Merge Strategy Examples</h4>
      <div class="strategy-examples">
        <div class="strategy-example">
          <div class="strategy-name">📥 First</div>
          <div class="strategy-desc">
            Returns the output of the first parent node that completes.
          </div>
          <code class="strategy-result">Result: parent1_output</code>
        </div>

        <div class="strategy-example">
          <div class="strategy-name">📤 Last</div>
          <div class="strategy-desc">
            Returns the output of the last parent node that completes.
          </div>
          <code class="strategy-result">Result: parent3_output</code>
        </div>

        <div class="strategy-example">
          <div class="strategy-name">📦 All</div>
          <div class="strategy-desc">
            Combines all parent outputs into an array.
          </div>
          <code class="strategy-result"
            >Result: [parent1, parent2, parent3]</code
          >
        </div>

        <div class="strategy-example">
          <div class="strategy-name">⚙️ Custom</div>
          <div class="strategy-desc">
            Use custom expression to merge outputs.
          </div>
          <code class="strategy-result">Result: your_expression</code>
        </div>
      </div>
    </div>

    <div class="info-box info-box-info">
      <h4 class="info-title">ℹ️ How It Works</h4>
      <ul class="notes-list">
        <li>
          Merge node waits for <strong>all parent nodes</strong> to complete
        </li>
        <li>Parent outputs are accessible via <code v-pre>{{input.parentNodeId}}</code></li>
        <li>
          For "all" strategy, outputs are combined into a single array
        </li>
        <li>
          Use "custom" strategy for complex merging logic (filtering,
          transforming, etc.)
        </li>
      </ul>
    </div>

    <div class="reference-box">
      <h4 class="reference-title">📚 Custom Expression Examples</h4>
      <div class="custom-examples">
        <div class="custom-example">
          <div class="example-title">Merge objects:</div>
          <code v-pre>{"a": {{input.node1}}.value, "b": {{input.node2}}.value}</code>
        </div>
        <div class="custom-example">
          <div class="example-title">Filter and combine:</div>
          <code v-pre>filter({{input.results}}, {.status == "success"})</code>
        </div>
        <div class="custom-example">
          <div class="example-title">Sum values:</div>
          <code v-pre>{{input.node1}}.count + {{input.node2}}.count</code>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { MergeNodeConfig } from "@/types/nodes";
import TemplateInput from "@/components/common/TemplateInput.vue";

interface Props {
  config: MergeNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: MergeNodeConfig): void;
}>();

const localConfig = ref<MergeNodeConfig>({
  merge_strategy: "all",
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
.merge-node-config {
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
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.2s;
  font-family: inherit;
}

.select-field:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.info-box {
  padding: 16px;
  background-color: #f5f3ff;
  border: 1px solid #ddd6fe;
  border-radius: 8px;
}

.info-box-info {
  background-color: #eff6ff;
  border-color: #bfdbfe;
}

.info-title {
  font-size: 13px;
  font-weight: 700;
  color: #5b21b6;
  margin: 0 0 12px 0;
}

.info-box-info .info-title {
  color: #1e40af;
}

.strategy-examples {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.strategy-example {
  padding: 12px;
  background-color: #ffffff;
  border: 1px solid #e9d5ff;
  border-radius: 6px;
}

.strategy-name {
  font-size: 12px;
  font-weight: 600;
  color: #6b21a8;
  margin-bottom: 4px;
}

.strategy-desc {
  font-size: 12px;
  color: #374151;
  margin-bottom: 8px;
  line-height: 1.5;
}

.strategy-result {
  display: block;
  background-color: #faf5ff;
  padding: 6px 10px;
  border-radius: 4px;
  font-family: "Monaco", "Menlo", monospace;
  font-size: 11px;
  color: #7c3aed;
  border: 1px solid #e9d5ff;
}

.notes-list {
  margin: 0;
  padding-left: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

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
  color: #7c3aed;
  border: 1px solid #e9d5ff;
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

.custom-examples {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.custom-example {
  padding: 12px;
  background-color: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
}

.example-title {
  font-size: 11px;
  font-weight: 600;
  color: #6b7280;
  margin-bottom: 6px;
}

.custom-example code {
  display: block;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
