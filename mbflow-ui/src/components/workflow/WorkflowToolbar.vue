<script setup lang="ts">
import { computed } from "vue";
import { Icon } from "@iconify/vue";
import { useWorkflowStore } from "@/stores/workflow";
import Button from "@/components/ui/Button.vue";

interface Props {
  loading?: boolean;
  saving?: boolean;
  readonly?: boolean;
}

withDefaults(defineProps<Props>(), {
  loading: false,
  saving: false,
  readonly: false,
});

const emit = defineEmits<{
  save: [];
  execute: [];
  autoLayout: [];
  validate: [];
  back: [];
}>();

const workflowStore = useWorkflowStore();

const hasChanges = computed(() => workflowStore.isDirty);
const nodeCount = computed(() => workflowStore.nodeCount);
const edgeCount = computed(() => workflowStore.edgeCount);
</script>

<template>
  <div class="workflow-toolbar border-b border-gray-200 bg-white px-4 py-3">
    <div class="flex items-center justify-between">
      <!-- Left side - Navigation and info -->
      <div class="flex items-center gap-4">
        <Button variant="ghost" size="sm" @click="emit('back')">
          <Icon icon="heroicons:arrow-left" class="mr-1 size-4" />
          Back
        </Button>

        <div class="h-6 w-px bg-gray-300" />

        <div class="text-sm text-gray-600">
          <span class="font-medium">{{ nodeCount }}</span> nodes,
          <span class="font-medium">{{ edgeCount }}</span> edges
        </div>

        <div
          v-if="hasChanges"
          class="flex items-center gap-1 text-xs text-orange-600"
        >
          <Icon icon="heroicons:exclamation-circle" class="size-4" />
          Unsaved changes
        </div>
      </div>

      <!-- Right side - Actions -->
      <div class="flex items-center gap-2">
        <Button
          v-if="!readonly"
          variant="secondary"
          size="sm"
          @click="emit('validate')"
        >
          <Icon icon="heroicons:check-circle" class="mr-1 size-4" />
          Validate
        </Button>

        <Button variant="secondary" size="sm" @click="emit('autoLayout')">
          <Icon icon="heroicons:arrows-pointing-out" class="mr-1 size-4" />
          Auto-layout
        </Button>

        <div class="h-6 w-px bg-gray-300" />

        <Button
          v-if="!readonly"
          variant="primary"
          size="sm"
          :loading="saving"
          :disabled="!hasChanges"
          @click="emit('save')"
        >
          <Icon v-if="!saving" icon="heroicons:check" class="mr-1 size-4" />
          Save
        </Button>

        <Button
          variant="secondary"
          size="sm"
          :loading="loading"
          @click="emit('execute')"
        >
          <Icon v-if="!loading" icon="heroicons:play" class="mr-1 size-4" />
          Execute
        </Button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.workflow-toolbar {
  position: sticky;
  top: 0;
  z-index: 10;
}
</style>
