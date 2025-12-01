<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/utils/cn";

interface SelectOption {
  value: string | number;
  label: string;
  disabled?: boolean;
}

interface Props {
  modelValue?: string | number;
  options: SelectOption[];
  label?: string;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  placeholder?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  "update:modelValue": [value: string | number];
}>();

const selectClasses = computed(() =>
  cn(
    "w-full px-3 py-2 border rounded-md transition-colors",
    "focus:outline-none focus:ring-2",
    {
      "border-gray-300 focus:border-blue-500 focus:ring-blue-500":
        !props.error && !props.disabled,
      "border-red-500 focus:border-red-500 focus:ring-red-500": props.error,
      "bg-gray-100 cursor-not-allowed opacity-60": props.disabled,
    },
  ),
);

function handleChange(event: Event) {
  const target = event.target as HTMLSelectElement;
  emit("update:modelValue", target.value);
}
</script>

<template>
  <div class="space-y-1">
    <label v-if="label" class="block text-sm font-medium text-gray-700">
      {{ label }}
      <span v-if="required" class="text-red-500">*</span>
    </label>

    <select
      :value="modelValue"
      :disabled="disabled"
      :class="selectClasses"
      @change="handleChange"
    >
      <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
      <option
        v-for="option in options"
        :key="option.value"
        :value="option.value"
        :disabled="option.disabled"
      >
        {{ option.label }}
      </option>
    </select>

    <p v-if="error" class="text-sm text-red-600">
      {{ error }}
    </p>
  </div>
</template>
