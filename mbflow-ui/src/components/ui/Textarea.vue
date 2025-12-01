<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/utils/cn";

interface Props {
  modelValue?: string;
  label?: string;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  placeholder?: string;
  rows?: number;
  maxlength?: number;
  resize?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  rows: 4,
  resize: true,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const textareaClasses = computed(() =>
  cn(
    "w-full px-3 py-2 border rounded-md transition-colors",
    "focus:outline-none focus:ring-2",
    {
      "border-gray-300 focus:border-blue-500 focus:ring-blue-500":
        !props.error && !props.disabled,
      "border-red-500 focus:border-red-500 focus:ring-red-500": props.error,
      "bg-gray-100 cursor-not-allowed opacity-60": props.disabled,
      "resize-none": !props.resize,
      "resize-y": props.resize,
    },
  ),
);

function handleInput(event: Event) {
  const target = event.target as HTMLTextAreaElement;
  emit("update:modelValue", target.value);
}
</script>

<template>
  <div class="space-y-1">
    <label v-if="label" class="block text-sm font-medium text-gray-700">
      {{ label }}
      <span v-if="required" class="text-red-500">*</span>
    </label>

    <textarea
      :value="modelValue"
      :disabled="disabled"
      :placeholder="placeholder"
      :rows="rows"
      :maxlength="maxlength"
      :class="textareaClasses"
      @input="handleInput"
    />

    <div class="flex items-center justify-between">
      <p v-if="error" class="text-sm text-red-600">
        {{ error }}
      </p>
      <p v-if="maxlength" class="ml-auto text-xs text-gray-500">
        {{ modelValue?.length || 0 }} / {{ maxlength }}
      </p>
    </div>
  </div>
</template>
