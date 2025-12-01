<script setup lang="ts">
import { ref, watch } from "vue";
import { Icon } from "@iconify/vue";
import { cn } from "@/utils/cn";

export interface Tab {
  key: string;
  label: string;
  icon?: string;
  disabled?: boolean;
}

interface Props {
  tabs: Tab[];
  modelValue?: string;
  variant?: "default" | "pills";
}

const props = withDefaults(defineProps<Props>(), {
  variant: "default",
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  change: [value: string];
}>();

const activeTab = ref(props.modelValue || props.tabs[0]?.key || "");

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      activeTab.value = newValue;
    }
  },
);

function selectTab(tab: Tab) {
  if (tab.disabled) return;

  activeTab.value = tab.key;
  emit("update:modelValue", tab.key);
  emit("change", tab.key);
}

function getTabClasses(tab: Tab) {
  const isActive = activeTab.value === tab.key;

  if (props.variant === "pills") {
    return cn("px-4 py-2 rounded-lg font-medium transition-all", {
      "bg-blue-600 text-white": isActive && !tab.disabled,
      "text-gray-600 hover:bg-gray-100": !isActive && !tab.disabled,
      "opacity-50 cursor-not-allowed": tab.disabled,
      "cursor-pointer": !tab.disabled,
    });
  }

  return cn("px-4 py-2 font-medium transition-all border-b-2", {
    "border-blue-600 text-blue-600": isActive && !tab.disabled,
    "border-transparent text-gray-600 hover:text-gray-900 hover:border-gray-300":
      !isActive && !tab.disabled,
    "opacity-50 cursor-not-allowed": tab.disabled,
    "cursor-pointer": !tab.disabled,
  });
}
</script>

<template>
  <div>
    <!-- Tab buttons -->
    <div
      :class="[
        'flex gap-1',
        variant === 'default' ? 'border-b border-gray-200' : '',
      ]"
    >
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        :class="getTabClasses(tab)"
        :disabled="tab.disabled"
        @click="selectTab(tab)"
      >
        <div class="flex items-center gap-2">
          <Icon v-if="tab.icon" :icon="tab.icon" class="size-4" />
          <span>{{ tab.label }}</span>
        </div>
      </button>
    </div>

    <!-- Tab panels -->
    <div class="mt-4">
      <slot :active-tab="activeTab" />
    </div>
  </div>
</template>
