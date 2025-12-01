<script setup lang="ts">
import { computed } from "vue";
import { Icon } from "@iconify/vue";
import { cn } from "@/utils/cn";

interface Props {
  variant?: "default" | "success" | "warning" | "danger" | "info" | "gray";
  size?: "sm" | "md" | "lg";
  icon?: string;
  dot?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  variant: "default",
  size: "md",
});

const badgeClasses = computed(() =>
  cn(
    "inline-flex items-center gap-1 font-semibold rounded-full transition-colors",
    {
      // Variants
      "bg-blue-100 text-blue-700": props.variant === "default",
      "bg-green-100 text-green-700": props.variant === "success",
      "bg-yellow-100 text-yellow-700": props.variant === "warning",
      "bg-red-100 text-red-700": props.variant === "danger",
      "bg-purple-100 text-purple-700": props.variant === "info",
      "bg-gray-100 text-gray-700": props.variant === "gray",

      // Sizes
      "px-2 py-0.5 text-xs": props.size === "sm",
      "px-2.5 py-1 text-sm": props.size === "md",
      "px-3 py-1.5 text-base": props.size === "lg",
    },
  ),
);

const dotClasses = computed(() =>
  cn("w-2 h-2 rounded-full", {
    "bg-blue-500": props.variant === "default",
    "bg-green-500": props.variant === "success",
    "bg-yellow-500": props.variant === "warning",
    "bg-red-500": props.variant === "danger",
    "bg-purple-500": props.variant === "info",
    "bg-gray-500": props.variant === "gray",
  }),
);

const iconSize = computed(() => {
  const sizes = {
    sm: "w-3 h-3",
    md: "w-4 h-4",
    lg: "w-5 h-5",
  };
  return sizes[props.size];
});
</script>

<template>
  <span :class="badgeClasses">
    <span v-if="dot" :class="dotClasses" />
    <Icon v-if="icon" :icon="icon" :class="iconSize" />
    <slot />
  </span>
</template>
