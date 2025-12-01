<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/utils/cn";

interface Props {
  modelValue?: string | number;
  type?: string;
  placeholder?: string;
  disabled?: boolean;
  error?: string;
  label?: string;
  required?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  type: "text",
  disabled: false,
  required: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string | number];
}>();

const inputClasses = computed(() =>
  cn(
    "block w-full rounded-md shadow-sm sm:text-sm transition-colors",
    props.error
      ? "border-red-300 text-red-900 placeholder-red-300 focus:border-red-500 focus:ring-red-500"
      : "border-gray-300 focus:border-blue-500 focus:ring-blue-500",
  ),
);
</script>

<template>
  <div class="w-full">
    <label v-if="label" class="mb-1 block text-sm font-medium text-gray-700">
      {{ label }}
      <span v-if="required" class="text-red-500">*</span>
    </label>

    <input
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :class="inputClasses"
      @input="
        emit('update:modelValue', ($event.target as HTMLInputElement).value)
      "
    />

    <p v-if="error" class="mt-1 text-sm text-red-600">{{ error }}</p>
  </div>
</template>
