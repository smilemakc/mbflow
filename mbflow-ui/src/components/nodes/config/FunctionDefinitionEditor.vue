<template>
  <div class="function-definition-editor">
    <div class="editor-header">
      <h4 class="editor-title">{{ title }}</h4>
      <button
        v-if="showDelete"
        @click="$emit('delete')"
        class="delete-button"
        type="button"
        title="Delete function"
      >
        🗑️
      </button>
    </div>

    <div class="editor-body">
      <!-- Function Type Selector -->
      <div class="form-group">
        <label class="label">Function Type</label>
        <select v-model="localFunction.type" class="select-field">
          <option
            v-for="type in functionTypes"
            :key="type"
            :value="type"
          >
            {{ formatFunctionType(type) }}
          </option>
        </select>
      </div>

      <!-- Common Fields -->
      <div class="form-group">
        <label class="label">Function Name</label>
        <input
          v-model="localFunction.name"
          type="text"
          class="input-field"
          placeholder="e.g., get_weather"
        />
      </div>

      <div class="form-group">
        <label class="label">Description</label>
        <textarea
          v-model="localFunction.description"
          class="textarea-field"
          rows="2"
          placeholder="Describe what this function does"
        />
      </div>

      <!-- Type-Specific Configuration -->
      <div class="type-specific-config">
        <!-- Built-in Function -->
        <div v-if="localFunction.type === 'builtin'" class="config-section">
          <BuiltinFunctionConfig
            v-if="builtinFunctionConfigExists"
            :function="localFunction"
            @update:function="updateFunction"
          />
          <div v-else class="placeholder">
            <p class="text-sm text-gray-600">
              BuiltinFunctionConfig component will be implemented next.
            </p>
          </div>
        </div>

        <!-- Sub-Workflow Function -->
        <div v-else-if="localFunction.type === 'sub_workflow'" class="config-section">
          <div class="placeholder">
            <p class="text-sm text-gray-600">
              Sub-workflow configuration (Phase 2)
            </p>
            <p class="text-xs text-gray-500 mt-1">
              Workflow selector, input mapping, output extraction
            </p>
          </div>
        </div>

        <!-- Custom Code Function -->
        <div v-else-if="localFunction.type === 'custom_code'" class="config-section">
          <div class="placeholder">
            <p class="text-sm text-gray-600">
              Custom code editor (Phase 3)
            </p>
            <p class="text-xs text-gray-500 mt-1">
              Language selector, code editor, sandboxing settings
            </p>
          </div>
        </div>

        <!-- OpenAPI Function -->
        <div v-else-if="localFunction.type === 'openapi'" class="config-section">
          <div class="placeholder">
            <p class="text-sm text-gray-600">
              OpenAPI configuration (Phase 4)
            </p>
            <p class="text-xs text-gray-500 mt-1">
              Spec URL, operation selector, authentication
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import type { FunctionDefinition, FunctionType } from '@/types/nodes';
import { FUNCTION_TYPES } from '@/types/nodes';

// Check if BuiltinFunctionConfig component exists
const builtinFunctionConfigExists = false; // Will be true after BuiltinFunctionConfig is created

interface Props {
  function: FunctionDefinition;
  title?: string;
  showDelete?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: 'Function Configuration',
  showDelete: true,
});

const emit = defineEmits<{
  (e: 'update:function', func: FunctionDefinition): void;
  (e: 'delete'): void;
}>();

const localFunction = ref<FunctionDefinition>({ ...props.function });

const functionTypes = FUNCTION_TYPES;

const formatFunctionType = (type: FunctionType): string => {
  const labels: Record<FunctionType, string> = {
    builtin: '🔧 Built-in Function',
    sub_workflow: '🔄 Sub-Workflow',
    custom_code: '💻 Custom Code',
    openapi: '🌐 OpenAPI',
  };
  return labels[type];
};

const updateFunction = (updated: FunctionDefinition) => {
  localFunction.value = updated;
};

// Watch for external changes
watch(
  () => props.function,
  (newFunc) => {
    if (JSON.stringify(newFunc) !== JSON.stringify(localFunction.value)) {
      localFunction.value = { ...newFunc };
    }
  },
  { deep: true }
);

// Emit changes
watch(
  localFunction,
  (newFunc) => {
    emit('update:function', newFunc);
  },
  { deep: true }
);
</script>

<style scoped>
.function-definition-editor {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  overflow: hidden;
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background-color: #f9fafb;
  border-bottom: 1px solid #e5e7eb;
}

.editor-title {
  font-size: 14px;
  font-weight: 600;
  color: #374151;
  margin: 0;
}

.delete-button {
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.delete-button:hover {
  background-color: #fee2e2;
}

.editor-body {
  padding: 16px;
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
  font-family: inherit;
  line-height: 1.5;
}

.input-field:focus,
.select-field:focus,
.textarea-field:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.type-specific-config {
  margin-top: 8px;
}

.config-section {
  padding: 16px;
  background-color: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
}

.placeholder {
  padding: 24px;
  text-align: center;
  background-color: #ffffff;
  border: 2px dashed #d1d5db;
  border-radius: 8px;
}
</style>
