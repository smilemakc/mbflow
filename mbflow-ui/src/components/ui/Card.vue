<script setup lang="ts">
import { cn } from "@/utils/cn";

interface Props {
  variant?: "default" | "bordered" | "elevated";
  padding?: "none" | "sm" | "md" | "lg";
  hover?: boolean;
  clickable?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  variant: "default",
  padding: "md",
  hover: false,
  clickable: false,
});

const cardClasses = cn("bg-white rounded-lg transition-all", {
  // Variants
  "border border-gray-200": props.variant === "bordered",
  "shadow-md": props.variant === "elevated",
  "shadow-sm": props.variant === "default",

  // Padding
  "p-0": props.padding === "none",
  "p-3": props.padding === "sm",
  "p-6": props.padding === "md",
  "p-8": props.padding === "lg",

  // Interactions
  "hover:shadow-md": props.hover,
  "cursor-pointer": props.clickable,
});
</script>

<template>
  <div :class="cardClasses">
    <slot />
  </div>
</template>
