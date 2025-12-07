<script setup lang="ts">
import { computed } from "vue";

interface Props {
  modelValue: boolean;
  label?: string;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: false,
  label: "",
  disabled: false,
});

const emit = defineEmits<{
  (e: "update:modelValue", value: boolean): void;
}>();

const value = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});
</script>

<template>
  <label
    :class="[
      'flex items-center gap-2',
      disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',
    ]"
  >
    <input
      v-model="value"
      type="checkbox"
      :disabled="disabled"
      class="size-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
    />
    <span v-if="label" class="select-none text-sm text-gray-700">
      {{ label }}
    </span>
  </label>
</template>
