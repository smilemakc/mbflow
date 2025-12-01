<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/utils/cn";

interface Props {
  variant?: "primary" | "secondary" | "danger" | "ghost";
  size?: "sm" | "md" | "lg";
  disabled?: boolean;
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  variant: "primary",
  size: "md",
  disabled: false,
  loading: false,
});

const classes = computed(() =>
  cn(
    "inline-flex items-center justify-center rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none",
    {
      "bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500":
        props.variant === "primary",
      "bg-gray-200 text-gray-900 hover:bg-gray-300 focus:ring-gray-500":
        props.variant === "secondary",
      "bg-red-600 text-white hover:bg-red-700 focus:ring-red-500":
        props.variant === "danger",
      "hover:bg-gray-100 focus:ring-gray-500": props.variant === "ghost",
    },
    {
      "px-3 py-1.5 text-sm": props.size === "sm",
      "px-4 py-2 text-base": props.size === "md",
      "px-6 py-3 text-lg": props.size === "lg",
    },
  ),
);
</script>

<template>
  <button :class="classes" :disabled="disabled || loading">
    <span v-if="loading" class="mr-2">
      <svg
        class="size-4 animate-spin"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle
          class="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          stroke-width="4"
        ></circle>
        <path
          class="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
        ></path>
      </svg>
    </span>
    <slot />
  </button>
</template>
