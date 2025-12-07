<template>
  <div class="llm-node-config">
    <!-- Basic Settings -->
    <div class="form-group">
      <label class="label">Provider</label>
      <select v-model="localConfig.provider" class="select-field">
        <option value="openai">OpenAI</option>
        <option value="anthropic">Anthropic</option>
        <option value="google">Google</option>
        <option value="azure">Azure</option>
        <option value="ollama">Ollama</option>
      </select>
    </div>

    <div class="form-group">
      <label class="label">Model</label>
      <select v-model="localConfig.model" class="select-field">
        <option v-for="model in availableModels" :key="model" :value="model">
          {{ model }}
        </option>
      </select>
    </div>

    <div class="form-group">
      <label class="label">API Key</label>
      <TemplateInput
        v-model="localConfig.api_key"
        placeholder="{{env.openai_api_key}}"
        :node-id="nodeId"
      />
      <p class="mt-1 text-xs text-gray-500">
        Use templates like
        <code class="rounded bg-gray-100 px-1">{{
          variablePlaceholderExample
        }}</code>
        to reference workflow variables
      </p>
    </div>

    <div class="form-group">
      <label class="label">System Prompt (Optional)</label>
      <TemplateInput
        :model-value="localConfig.instruction || ''"
        @update:model-value="localConfig.instruction = $event"
        height="100px"
        :node-id="nodeId"
        multiline
      />
    </div>

    <div class="form-group">
      <label class="label">User Prompt</label>
      <TemplateInput
        v-model="localConfig.prompt"
        height="150px"
        :node-id="nodeId"
        multiline
      />
    </div>

    <!-- Advanced Settings (Progressive Disclosure) -->
    <button
      @click="showAdvanced = !showAdvanced"
      class="toggle-button"
      type="button"
    >
      {{ showAdvanced ? "▼" : "▶" }} Advanced Settings
    </button>

    <div v-if="showAdvanced" class="advanced-section">
      <div class="form-group">
        <label class="label">
          Temperature
          <span class="hint">(0.0 - 2.0, default: 0.7)</span>
        </label>
        <input
          v-model.number="localConfig.temperature"
          type="number"
          min="0"
          max="2"
          step="0.1"
          class="input-field"
        />
      </div>

      <div class="form-group">
        <label class="label">
          Max Tokens
          <span class="hint">(Maximum response length)</span>
        </label>
        <input
          v-model.number="localConfig.max_tokens"
          type="number"
          min="1"
          max="100000"
          class="input-field"
        />
      </div>

      <div class="form-group">
        <label class="label">
          Top P
          <span class="hint">(0.0 - 1.0, nucleus sampling)</span>
        </label>
        <input
          v-model.number="localConfig.top_p"
          type="number"
          min="0"
          max="1"
          step="0.1"
          class="input-field"
        />
      </div>

      <div class="form-group">
        <label class="label">
          Frequency Penalty
          <span class="hint">(-2.0 - 2.0)</span>
        </label>
        <input
          v-model.number="localConfig.frequency_penalty"
          type="number"
          min="-2"
          max="2"
          step="0.1"
          class="input-field"
        />
      </div>

      <div class="form-group">
        <label class="label">
          Presence Penalty
          <span class="hint">(-2.0 - 2.0)</span>
        </label>
        <input
          v-model.number="localConfig.presence_penalty"
          type="number"
          min="-2"
          max="2"
          step="0.1"
          class="input-field"
        />
      </div>

      <div class="form-group">
        <label class="label">Stop Sequences (one per line)</label>
        <textarea
          v-model="stopSequencesText"
          placeholder="Enter stop sequences, one per line"
          rows="3"
          class="textarea-field"
        />
      </div>

      <div class="form-group">
        <label class="label">Response Format</label>
        <select v-model="localConfig.response_format" class="select-field">
          <option value="text">Text</option>
          <option value="json">JSON</option>
        </select>
      </div>

      <div class="form-group">
        <label class="label">Timeout (seconds)</label>
        <input
          v-model.number="localConfig.timeout_seconds"
          type="number"
          min="1"
          max="300"
          class="input-field"
        />
      </div>

      <div class="form-group">
        <label class="label">Retry Count</label>
        <input
          v-model.number="localConfig.retry_count"
          type="number"
          min="0"
          max="5"
          class="input-field"
        />
      </div>

      <!-- Tool Calling Section -->
      <div class="tool-calling-section">
        <h3 class="section-title">🔧 Tool Calling (Phase 1)</h3>

        <div class="form-group">
          <label class="label">Enable Tool Calling</label>
          <label class="checkbox-label">
            <input
              type="checkbox"
              v-model="toolCallingEnabled"
              class="checkbox-field"
            />
            Allow LLM to call functions
          </label>
        </div>

        <div v-if="toolCallingEnabled" class="tool-config-panel">
          <div class="form-group">
            <label class="label">Tool Call Mode</label>
            <select v-model="toolCallMode" class="select-field">
              <option value="auto">
                Auto (Automatic loop until completion)
              </option>
              <option value="manual">
                Manual (Connect FunctionCall nodes via edges)
              </option>
            </select>
            <p class="mt-1 text-xs text-gray-500">
              <strong>Auto:</strong> LLM automatically calls functions in a loop.
              <strong>Manual:</strong> Use FunctionCall nodes connected via edges.
            </p>
          </div>

          <!-- Auto Mode Settings -->
          <div v-if="toolCallMode === 'auto'" class="auto-mode-settings">
            <div class="form-group">
              <label class="label">
                Max Iterations
                <span class="hint">(Prevents infinite loops)</span>
              </label>
              <input
                v-model.number="toolCallConfig.max_iterations"
                type="number"
                min="1"
                max="50"
                class="input-field"
              />
            </div>

            <div class="form-group">
              <label class="label">
                Timeout Per Tool (seconds)
                <span class="hint">(Max time for each tool call)</span>
              </label>
              <input
                v-model.number="toolCallConfig.timeout_per_tool"
                type="number"
                min="1"
                max="300"
                class="input-field"
              />
            </div>

            <div class="form-group">
              <label class="label">
                Total Timeout (seconds)
                <span class="hint">(Max time for entire loop)</span>
              </label>
              <input
                v-model.number="toolCallConfig.total_timeout"
                type="number"
                min="1"
                max="1800"
                class="input-field"
              />
            </div>

            <div class="form-group">
              <label class="checkbox-label">
                <input
                  type="checkbox"
                  v-model="toolCallConfig.stop_on_tool_failure"
                  class="checkbox-field"
                />
                Stop on tool failure
              </label>
              <p class="mt-1 text-xs text-gray-500">
                If enabled, execution stops when any tool fails. Otherwise,
                errors are added to conversation.
              </p>
            </div>
          </div>

          <!-- Functions List -->
          <div class="form-group">
            <label class="label">
              Functions ({{ functionCount }})
              <span class="hint">Phase 1: Built-in functions only</span>
            </label>
            <div class="functions-placeholder">
              <p class="text-sm text-gray-600">
                Function editor will be available in the next update.
              </p>
              <p class="text-xs text-gray-500 mt-2">
                For now, configure functions via JSON in workflow definition.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { LLMNodeConfig } from "@/types/nodes";
import { LLM_PROVIDER_MODELS, DEFAULT_TOOL_CALL_CONFIG } from "@/types/nodes";
import TemplateInput from "@/components/common/TemplateInput.vue";

interface Props {
  config: LLMNodeConfig;
  nodeId?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:config", config: LLMNodeConfig): void;
}>();

const localConfig = ref<LLMNodeConfig>({ ...props.config });
const showAdvanced = ref(false);
const variablePlaceholderExample = "{{env.openai_api_key}}";

// Tool Calling State
const toolCallingEnabled = computed({
  get: () => !!localConfig.value.tool_call_config,
  set: (enabled: boolean) => {
    if (enabled && !localConfig.value.tool_call_config) {
      localConfig.value.tool_call_config = { ...DEFAULT_TOOL_CALL_CONFIG };
      localConfig.value.functions = [];
    } else if (!enabled) {
      localConfig.value.tool_call_config = undefined;
      localConfig.value.functions = undefined;
    }
  },
});

const toolCallMode = computed({
  get: () => localConfig.value.tool_call_config?.mode || "manual",
  set: (mode: "auto" | "manual") => {
    if (localConfig.value.tool_call_config) {
      localConfig.value.tool_call_config.mode = mode;
    }
  },
});

const toolCallConfig = computed(() => {
  if (!localConfig.value.tool_call_config) {
    localConfig.value.tool_call_config = { ...DEFAULT_TOOL_CALL_CONFIG };
  }
  return localConfig.value.tool_call_config;
});

const functionCount = computed(() => {
  return localConfig.value.functions?.length || 0;
});

// Available models based on selected provider
const availableModels = computed(() => {
  return LLM_PROVIDER_MODELS[localConfig.value.provider] || [];
});

// Convert stop sequences array to/from text
const stopSequencesText = computed({
  get: () => {
    return (localConfig.value.stop_sequences || []).join("\n");
  },
  set: (value: string) => {
    localConfig.value.stop_sequences = value
      .split("\n")
      .map((s) => s.trim())
      .filter((s) => s.length > 0);
  },
});

// Watch for provider changes to update model
watch(
  () => localConfig.value.provider,
  (newProvider) => {
    const models = LLM_PROVIDER_MODELS[newProvider];
    if (
      models &&
      models.length > 0 &&
      !models.includes(localConfig.value.model)
    ) {
      const firstModel = models[0];
      if (firstModel) {
        localConfig.value.model = firstModel;
      }
    }
  },
);

// Watch for external config changes
// Watch for external config changes
watch(
  () => props.config,
  (newConfig) => {
    // Prevent infinite loop by checking if value actually changed
    // Simple JSON serialization check is sufficient for config objects
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
.llm-node-config {
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
.select-field,
.textarea-field {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.2s;
  font-family: inherit;
}

.textarea-field {
  resize: vertical;
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  line-height: 1.5;
}

.input-field:focus,
.select-field:focus,
.textarea-field:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.toggle-button {
  padding: 10px 16px;
  background-color: #f9fafb;
  color: #374151;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
  display: flex;
  align-items: center;
  gap: 8px;
}

.toggle-button:hover {
  background-color: #f3f4f6;
  border-color: #9ca3af;
}

.advanced-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
  background-color: #f9fafb;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
}

/* Tool Calling Styles */
.tool-calling-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 2px solid #e5e7eb;
}

.section-title {
  font-size: 14px;
  font-weight: 700;
  color: #374151;
  margin-bottom: 12px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #374151;
  cursor: pointer;
  user-select: none;
}

.checkbox-field {
  width: 16px;
  height: 16px;
  cursor: pointer;
  accent-color: #3b82f6;
}

.tool-config-panel {
  margin-top: 12px;
  padding: 16px;
  background-color: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.auto-mode-settings {
  padding: 12px;
  background-color: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.functions-placeholder {
  padding: 24px;
  background-color: #f9fafb;
  border: 2px dashed #d1d5db;
  border-radius: 8px;
  text-align: center;
}
</style>
