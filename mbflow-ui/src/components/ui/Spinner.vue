<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/utils/cn";

interface Props {
  size?: "xs" | "sm" | "md" | "lg" | "xl";
  color?: "primary" | "white" | "gray";
}

const props = withDefaults(defineProps<Props>(), {
  size: "md",
  color: "primary",
});

const spinnerClasses = computed(() =>
  cn("animate-spin rounded-full border-2 border-solid border-transparent", {
    // Sizes
    "w-3 h-3": props.size === "xs",
    "w-4 h-4": props.size === "sm",
    "w-6 h-6": props.size === "md",
    "w-8 h-8": props.size === "lg",
    "w-12 h-12": props.size === "xl",

    // Colors
    "border-t-blue-600": props.color === "primary",
    "border-t-white": props.color === "white",
    "border-t-gray-600": props.color === "gray",
  }),
);
</script>

<template>
  <div :class="spinnerClasses" role="status" aria-label="Loading">
    <span class="sr-only">Loading...</span>
  </div>
</template>

<style scoped>
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
