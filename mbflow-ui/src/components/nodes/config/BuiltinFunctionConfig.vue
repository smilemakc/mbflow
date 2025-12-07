<template>
  <div class="builtin-function-config">
    <div class="form-group">
      <label class="label">
        Built-in Function
        <span class="hint">(Select from available functions)</span>
      </label>
      <select v-model="localFunction.builtin_name" class="select-field">
        <option value="">-- Select a function --</option>
        <option
          v-for="func in builtinFunctions"
          :key="func.name"
          :value="func.name"
        >
          {{ func.label }}
        </option>
      </select>
    </div>

    <!-- Function Info -->
    <div v-if="selectedFunction" class="function-info">
      <div class="info-section">
        <h5 class="info-title">📋 Function Details</h5>
        <p class="info-description">{{ selectedFunction.description }}</p>
      </div>

      <!-- Parameters Info -->
      <div v-if="selectedFunction.parameters" class="info-section">
        <h5 class="info-title">⚙️ Parameters</h5>
        <div class="parameters-list">
          <div
            v-for="(param, key) in selectedFunction.parameters"
            :key="key"
            class="parameter-item"
          >
            <span class="param-name">{{ key }}</span>
            <span class="param-type">{{ param.type }}</span>
            <span v-if="param.required" class="param-required">required</span>
          </div>
        </div>
      </div>

      <!-- Return Type Info -->
      <div v-if="selectedFunction.returns" class="info-section">
        <h5 class="info-title">↩️ Returns</h5>
        <p class="return-type">{{ selectedFunction.returns }}</p>
      </div>

      <!-- Example -->
      <div v-if="selectedFunction.example" class="info-section">
        <h5 class="info-title">💡 Example</h5>
        <pre class="example-code">{{ selectedFunction.example }}</pre>
      </div>
    </div>

    <!-- JSON Schema Auto-fill -->
    <div v-if="selectedFunction" class="auto-fill-note">
      <p class="text-xs text-gray-600">
        ℹ️ JSON Schema parameters will be automatically generated based on the
        selected function.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';
import type { FunctionDefinition } from '@/types/nodes';
import { BUILTIN_FUNCTIONS } from '@/types/nodes';

interface BuiltinFunctionInfo {
  name: string;
  label: string;
  description: string;
  parameters?: Record<string, {
    type: string;
    description: string;
    required?: boolean;
  }>;
  returns?: string;
  example?: string;
}

interface Props {
  function: FunctionDefinition;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: 'update:function', func: FunctionDefinition): void;
}>();

const localFunction = computed({
  get: () => props.function,
  set: (value: FunctionDefinition) => emit('update:function', value),
});

// Built-in function definitions (Phase 1)
const builtinFunctions: BuiltinFunctionInfo[] = [
  {
    name: 'get_current_time',
    label: '🕐 Get Current Time',
    description: 'Returns the current date and time in various formats',
    parameters: {
      format: {
        type: 'string',
        description: 'Time format: RFC3339, Unix, ISO8601',
        required: false,
      },
    },
    returns: 'Current timestamp as string',
    example: `{
  "format": "RFC3339"
}
→ "2025-12-07T14:30:00Z"`,
  },
  {
    name: 'get_weather',
    label: '🌤️ Get Weather',
    description: 'Get weather information for a specific location (mock implementation)',
    parameters: {
      location: {
        type: 'string',
        description: 'City name or location',
        required: true,
      },
      unit: {
        type: 'string',
        description: 'Temperature unit: celsius or fahrenheit',
        required: false,
      },
    },
    returns: 'Weather information object',
    example: `{
  "location": "London",
  "unit": "celsius"
}
→ {
  "temperature": 22,
  "condition": "sunny",
  "humidity": 65
}`,
  },
  {
    name: 'calculate',
    label: '🧮 Calculate',
    description: 'Perform mathematical calculations using basic arithmetic',
    parameters: {
      expression: {
        type: 'string',
        description: 'Mathematical expression to evaluate',
        required: true,
      },
    },
    returns: 'Calculation result as number',
    example: `{
  "expression": "2 + 2 * 5"
}
→ 12`,
  },
];

const selectedFunction = computed(() => {
  if (!localFunction.value.builtin_name) return null;
  return builtinFunctions.find(
    (f) => f.name === localFunction.value.builtin_name
  );
});

// Auto-update function name and description when builtin is selected
watch(
  () => localFunction.value.builtin_name,
  (builtinName) => {
    if (builtinName) {
      const func = builtinFunctions.find((f) => f.name === builtinName);
      if (func) {
        // Update function name and description
        const updated = { ...localFunction.value };
        updated.name = func.name;
        updated.description = func.description;

        // Auto-generate JSON Schema parameters
        if (func.parameters) {
          updated.parameters = {
            type: 'object',
            properties: Object.entries(func.parameters).reduce(
              (acc, [key, param]) => {
                acc[key] = {
                  type: param.type,
                  description: param.description,
                };
                return acc;
              },
              {} as Record<string, any>
            ),
            required: Object.entries(func.parameters)
              .filter(([_, param]) => param.required)
              .map(([key, _]) => key),
          };
        }

        emit('update:function', updated);
      }
    }
  }
);
</script>

<style scoped>
.builtin-function-config {
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

.function-info {
  padding: 16px;
  background-color: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.info-title {
  font-size: 12px;
  font-weight: 700;
  color: #0c4a6e;
  margin: 0;
}

.info-description {
  font-size: 13px;
  color: #374151;
  margin: 0;
  line-height: 1.5;
}

.parameters-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.parameter-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background-color: #ffffff;
  border: 1px solid #e0f2fe;
  border-radius: 4px;
  font-size: 12px;
}

.param-name {
  font-weight: 600;
  color: #0369a1;
  font-family: 'Monaco', 'Menlo', monospace;
}

.param-type {
  color: #6b7280;
  font-style: italic;
}

.param-required {
  margin-left: auto;
  padding: 2px 8px;
  background-color: #fef2f2;
  color: #dc2626;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
}

.return-type {
  font-size: 13px;
  color: #374151;
  font-family: 'Monaco', 'Menlo', monospace;
  margin: 0;
}

.example-code {
  font-size: 12px;
  color: #1f2937;
  background-color: #ffffff;
  padding: 12px;
  border-radius: 4px;
  border: 1px solid #e0f2fe;
  margin: 0;
  overflow-x: auto;
  font-family: 'Monaco', 'Menlo', monospace;
  line-height: 1.5;
  white-space: pre-wrap;
}

.auto-fill-note {
  padding: 12px;
  background-color: #fef3c7;
  border: 1px solid #fde68a;
  border-radius: 6px;
}
</style>
