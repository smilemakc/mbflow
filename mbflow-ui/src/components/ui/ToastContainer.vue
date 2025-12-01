<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { useToast } from "@/composables/useToast";
import type { Toast } from "@/types/toast";

const { toasts, removeToast } = useToast();

function getIcon(type: Toast["type"]) {
  const icons = {
    success: "heroicons:check-circle",
    error: "heroicons:x-circle",
    warning: "heroicons:exclamation-triangle",
    info: "heroicons:information-circle",
  };
  return icons[type];
}

function getColorClasses(type: Toast["type"]) {
  const colors = {
    success: "bg-green-50 border-green-200 text-green-800",
    error: "bg-red-50 border-red-200 text-red-800",
    warning: "bg-yellow-50 border-yellow-200 text-yellow-800",
    info: "bg-blue-50 border-blue-200 text-blue-800",
  };
  return colors[type];
}

function getIconColorClasses(type: Toast["type"]) {
  const colors = {
    success: "text-green-600",
    error: "text-red-600",
    warning: "text-yellow-600",
    info: "text-blue-600",
  };
  return colors[type];
}
</script>

<template>
  <div class="pointer-events-none fixed right-4 top-4 z-50 space-y-2">
    <TransitionGroup name="toast" tag="div" class="space-y-2">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        :class="[
          'pointer-events-auto',
          'w-full max-w-sm',
          'rounded-lg border shadow-lg',
          'p-4',
          getColorClasses(toast.type),
        ]"
      >
        <div class="flex items-start gap-3">
          <!-- Icon -->
          <Icon
            :icon="getIcon(toast.type)"
            :class="['mt-0.5 size-5 shrink-0', getIconColorClasses(toast.type)]"
          />

          <!-- Content -->
          <div class="min-w-0 flex-1">
            <p class="text-sm font-semibold">{{ toast.title }}</p>
            <p v-if="toast.message" class="mt-1 text-sm opacity-90">
              {{ toast.message }}
            </p>

            <!-- Action button -->
            <button
              v-if="toast.action"
              class="mt-2 text-sm font-medium underline hover:no-underline"
              @click="toast.action.onClick"
            >
              {{ toast.action.label }}
            </button>
          </div>

          <!-- Close button -->
          <button
            type="button"
            class="ml-auto shrink-0 opacity-70 transition-opacity hover:opacity-100"
            @click="removeToast(toast.id)"
          >
            <Icon icon="heroicons:x-mark" class="size-5" />
          </button>
        </div>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.toast-move {
  transition: transform 0.3s ease;
}
</style>
