<template>
  <div class="file-storage-config space-y-4">
    <!-- Action Selection -->
    <Select v-model="localConfig.action" label="Action" :options="actionOptions" />

    <!-- Storage ID (optional) -->
    <div class="space-y-1">
      <label class="text-sm font-medium text-gray-700">Storage ID (optional)</label>
      <TemplateInput
        v-model="localConfig.storage_id"
        placeholder="default"
        :node-id="nodeId"
      />
      <p class="text-xs text-gray-500">Leave empty for default storage</p>
    </div>

    <!-- Store Action Fields -->
    <template v-if="localConfig.action === 'store'">
      <div class="space-y-4 rounded-md border border-gray-200 p-3">
        <h5 class="text-xs font-semibold uppercase text-gray-500">File Source</h5>

        <Select
          v-model="localConfig.file_source"
          label="Source Type"
          :options="fileSourceOptions"
        />

        <div class="space-y-1">
          <label class="text-sm font-medium text-gray-700">
            {{ localConfig.file_source === 'url' ? 'File URL' : 'Base64 Data' }}
          </label>
          <TemplateInput
            v-if="localConfig.file_source === 'url'"
            v-model="localConfig.file_url"
            placeholder="https://example.com/document.pdf or {{input.url}}"
            :node-id="nodeId"
          />
          <TemplateInput
            v-else
            v-model="localConfig.file_data"
            :multiline="true"
            :rows="3"
            placeholder="{{input.base64_data}}"
            :node-id="nodeId"
          />
        </div>

        <div class="space-y-1">
          <label class="text-sm font-medium text-gray-700">File Name</label>
          <TemplateInput
            v-model="localConfig.file_name"
            placeholder="document.pdf or {{input.filename}}"
            :node-id="nodeId"
          />
        </div>

        <div class="space-y-1">
          <label class="text-sm font-medium text-gray-700">MIME Type (optional)</label>
          <TemplateInput
            v-model="localConfig.mime_type"
            placeholder="Auto-detected if empty"
            :node-id="nodeId"
          />
        </div>
      </div>

      <!-- Access Scope & Options -->
      <div class="space-y-4 rounded-md border border-gray-200 p-3">
        <h5 class="text-xs font-semibold uppercase text-gray-500">Storage Options</h5>

        <Select
          v-model="localConfig.access_scope"
          label="Access Scope"
          :options="accessScopeOptions"
        />

        <div class="space-y-1">
          <label class="text-sm font-medium text-gray-700">TTL (seconds, 0 = no expiration)</label>
          <TemplateInput
            v-model="localConfig.ttl"
            placeholder="0"
            :node-id="nodeId"
          />
        </div>

        <div class="space-y-1">
          <label class="text-sm font-medium text-gray-700">Tags (comma-separated)</label>
          <TemplateInput
            v-model="localConfig.tags_str"
            placeholder="document, important"
            :node-id="nodeId"
          />
        </div>
      </div>
    </template>

    <!-- Get/Delete/Metadata Action Fields -->
    <template v-else-if="['get', 'delete', 'metadata'].includes(localConfig.action)">
      <div class="space-y-1">
        <label class="text-sm font-medium text-gray-700">File ID</label>
        <TemplateInput
          v-model="localConfig.file_id"
          placeholder="{{input.file_id}}"
          :node-id="nodeId"
        />
      </div>
    </template>

    <!-- List Action Fields -->
    <template v-else-if="localConfig.action === 'list'">
      <div class="space-y-4 rounded-md border border-gray-200 p-3">
        <h5 class="text-xs font-semibold uppercase text-gray-500">Filters</h5>

        <Select
          v-model="localConfig.access_scope"
          label="Access Scope (optional)"
          :options="accessScopeOptionsWithEmpty"
        />

        <div class="space-y-1">
          <label class="text-sm font-medium text-gray-700">Tags Filter (comma-separated)</label>
          <TemplateInput
            v-model="localConfig.tags_str"
            placeholder="document, important"
            :node-id="nodeId"
          />
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div class="space-y-1">
            <label class="text-sm font-medium text-gray-700">Limit</label>
            <TemplateInput
              v-model="localConfig.limit"
              placeholder="100"
              :node-id="nodeId"
            />
          </div>
          <div class="space-y-1">
            <label class="text-sm font-medium text-gray-700">Offset</label>
            <TemplateInput
              v-model="localConfig.offset"
              placeholder="0"
              :node-id="nodeId"
            />
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from "vue";
import TemplateInput from "@/components/common/TemplateInput.vue";
import Select from "@/components/ui/Select.vue";

interface FileStorageNodeConfig {
  action: string;
  storage_id?: string;
  file_source?: string;
  file_data?: string;
  file_url?: string;
  file_name?: string;
  mime_type?: string;
  file_id?: string;
  access_scope?: string;
  ttl?: string;
  tags?: string[];
  tags_str?: string;
  limit?: string;
  offset?: string;
}

const props = defineProps<{
  config: FileStorageNodeConfig;
  nodeId: string;
}>();

const emit = defineEmits<{
  (e: "update:config", config: FileStorageNodeConfig): void;
}>();

// Initialize local config copy
const localConfig = ref<FileStorageNodeConfig>({
  action: props.config.action || "store",
  storage_id: props.config.storage_id || "",
  file_source: props.config.file_source || "url",
  file_data: props.config.file_data || "",
  file_url: props.config.file_url || "",
  file_name: props.config.file_name || "",
  mime_type: props.config.mime_type || "",
  file_id: props.config.file_id || "",
  access_scope: props.config.access_scope || "workflow",
  ttl: props.config.ttl || "0",
  tags_str: props.config.tags?.join(", ") || "",
  limit: props.config.limit || "100",
  offset: props.config.offset || "0",
});

// Options
const actionOptions = computed(() => [
  { label: "Store File", value: "store" },
  { label: "Get File", value: "get" },
  { label: "Delete File", value: "delete" },
  { label: "List Files", value: "list" },
  { label: "Get Metadata", value: "metadata" },
]);

const fileSourceOptions = computed(() => [
  { label: "URL", value: "url" },
  { label: "Base64 Data", value: "base64" },
]);

const accessScopeOptions = computed(() => [
  { label: "Workflow", value: "workflow" },
  { label: "Edge (Connected Nodes)", value: "edge" },
  { label: "Result (Output Storage)", value: "result" },
]);

const accessScopeOptionsWithEmpty = computed(() => [
  { label: "All Scopes", value: "" },
  ...accessScopeOptions.value,
]);

// Watch for changes and emit updates
watch(
  localConfig,
  (newConfig) => {
    // Convert tags string to array
    const configToEmit = { ...newConfig };
    if (newConfig.tags_str) {
      configToEmit.tags = newConfig.tags_str
        .split(",")
        .map((t) => t.trim())
        .filter((t) => t);
    }
    emit("update:config", configToEmit);
  },
  { deep: true },
);

// Watch for external prop changes
watch(
  () => props.config,
  (newConfig) => {
    if (JSON.stringify(newConfig) !== JSON.stringify(localConfig.value)) {
      localConfig.value = {
        ...localConfig.value,
        ...newConfig,
        tags_str: newConfig.tags?.join(", ") || localConfig.value.tags_str,
      };
    }
  },
  { deep: true },
);
</script>
