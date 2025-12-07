<template>
  <div class="key-value-editor">
    <div v-for="(item, index) in items" :key="index" class="key-value-row">
      <TemplateInput
        :model-value="item.key"
        @update:model-value="updateKey(index, $event)"
        :placeholder="placeholderKey"
        :show-variable-hint="false"
        :node-id="nodeId"
        class="key-input"
      />
      <TemplateInput
        :model-value="item.value"
        @update:model-value="updateValue(index, $event)"
        :placeholder="placeholderValue"
        :node-id="nodeId"
        class="value-input"
      />
      <button @click="removeItem(index)" class="remove-button" title="Remove">
        ✕
      </button>
    </div>

    <button @click="addItem" class="add-button">+ Add {{ itemLabel }}</button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import TemplateInput from "./TemplateInput.vue";

interface Props {
  modelValue: Record<string, string>;
  nodeId?: string;
  placeholderKey?: string;
  placeholderValue?: string;
  itemLabel?: string;
}

const props = withDefaults(defineProps<Props>(), {
  placeholderKey: "Key",
  placeholderValue: "Value",
  itemLabel: "Item",
});

const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, string>): void;
}>();

interface KeyValueItem {
  key: string;
  value: string;
}

const items = ref<KeyValueItem[]>([]);

// Convert object to array format
function objectToItems(obj: Record<string, string>): KeyValueItem[] {
  return Object.entries(obj).map(([key, value]) => ({ key, value }));
}

// Convert array to object format
function itemsToObject(items: KeyValueItem[]): Record<string, string> {
  const result: Record<string, string> = {};
  items.forEach(({ key, value }) => {
    if (key.trim()) {
      result[key] = value;
    }
  });
  return result;
}

// Initialize items
watch(
  () => props.modelValue,
  (newValue) => {
    items.value = objectToItems(newValue);
    if (items.value.length === 0) {
      items.value.push({ key: "", value: "" });
    }
  },
  { immediate: true },
);

function updateKey(index: number, newKey: string) {
  const item = items.value[index];
  if (item) {
    item.key = newKey;
    emitChanges();
  }
}

function updateValue(index: number, newValue: string) {
  const item = items.value[index];
  if (item) {
    item.value = newValue;
    emitChanges();
  }
}

function addItem() {
  items.value.push({ key: "", value: "" });
}

function removeItem(index: number) {
  items.value.splice(index, 1);
  if (items.value.length === 0) {
    items.value.push({ key: "", value: "" });
  }
  emitChanges();
}

function emitChanges() {
  emit("update:modelValue", itemsToObject(items.value));
}
</script>

<style scoped>
.key-value-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.key-value-row {
  display: grid;
  grid-template-columns: 1fr 2fr auto;
  gap: 8px;
  align-items: start;
}

.key-input,
.value-input {
  min-width: 0;
}

.remove-button {
  padding: 8px 12px;
  background-color: #fee;
  color: #dc2626;
  border: 1px solid #fca5a5;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  height: 38px;
}

.remove-button:hover {
  background-color: #fecaca;
  border-color: #f87171;
}

.add-button {
  padding: 8px 12px;
  background-color: #eff6ff;
  color: #1e40af;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  align-self: flex-start;
}

.add-button:hover {
  background-color: #dbeafe;
  border-color: #93c5fd;
}
</style>
