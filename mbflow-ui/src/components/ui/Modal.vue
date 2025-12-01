<script setup lang="ts">
import {
  TransitionRoot,
  TransitionChild,
  Dialog,
  DialogPanel,
  DialogTitle,
} from "@headlessui/vue";
import { Icon } from "@iconify/vue";

interface Props {
  open: boolean;
  title?: string;
  size?: "sm" | "md" | "lg" | "xl" | "full";
  showClose?: boolean;
}

withDefaults(defineProps<Props>(), {
  size: "md",
  showClose: true,
});

const emit = defineEmits<{
  close: [];
}>();

const sizeClasses = {
  sm: "max-w-md",
  md: "max-w-lg",
  lg: "max-w-2xl",
  xl: "max-w-4xl",
  full: "max-w-7xl",
};
</script>

<template>
  <TransitionRoot appear :show="open" as="template">
    <Dialog as="div" class="relative z-50" @close="emit('close')">
      <!-- Backdrop -->
      <TransitionChild
        as="template"
        enter="duration-300 ease-out"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="duration-200 ease-in"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-black/30" />
      </TransitionChild>

      <!-- Modal container -->
      <div class="fixed inset-0 overflow-y-auto">
        <div class="flex min-h-full items-center justify-center p-4">
          <TransitionChild
            as="template"
            enter="duration-300 ease-out"
            enter-from="opacity-0 scale-95"
            enter-to="opacity-100 scale-100"
            leave="duration-200 ease-in"
            leave-from="opacity-100 scale-100"
            leave-to="opacity-0 scale-95"
          >
            <DialogPanel
              :class="[
                'w-full overflow-hidden rounded-lg bg-white shadow-xl transition-all',
                sizeClasses[size],
              ]"
            >
              <!-- Header -->
              <div
                v-if="title || showClose"
                class="flex items-center justify-between border-b border-gray-200 px-6 py-4"
              >
                <DialogTitle
                  v-if="title"
                  class="text-lg font-semibold text-gray-900"
                >
                  {{ title }}
                </DialogTitle>
                <button
                  v-if="showClose"
                  type="button"
                  class="rounded-md p-1 transition-colors hover:bg-gray-100"
                  @click="emit('close')"
                >
                  <Icon icon="heroicons:x-mark" class="size-5 text-gray-500" />
                </button>
              </div>

              <!-- Content -->
              <div class="px-6 py-4">
                <slot />
              </div>

              <!-- Footer -->
              <div
                v-if="$slots.footer"
                class="border-t border-gray-200 bg-gray-50 px-6 py-4"
              >
                <slot name="footer" />
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
