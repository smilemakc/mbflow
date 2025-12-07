<template>
  <div class="template-input">
    <div class="input-wrapper" :class="{ 'is-multiline': multiline }">
      <textarea
        v-if="multiline"
        ref="inputRef"
        :value="modelValue"
        @input="handleInput"
        @keydown="handleKeydown"
        @blur="hideAutocomplete"
        :placeholder="placeholder"
        :rows="rows"
        class="input-field textarea-field"
      ></textarea>
      <input
        v-else
        ref="inputRef"
        :value="modelValue"
        @input="handleInput"
        @keydown="handleKeydown"
        @blur="hideAutocomplete"
        :placeholder="placeholder"
        class="input-field"
      />
      <span v-if="showVariableHint" class="hint-text">
        Use {{ "{{" }}env.variable }} or {{ "{{" }}input.field }}
      </span>
    </div>

    <!-- Autocomplete dropdown -->
    <div
      v-if="showAutocomplete && suggestions.length > 0"
      class="autocomplete-dropdown"
      :style="{ top: dropdownTop + 'px', left: dropdownLeft + 'px' }"
    >
      <div
        v-for="(suggestion, index) in suggestions"
        :key="suggestion.value"
        :class="['suggestion-item', { active: index === selectedIndex }]"
        @mousedown.prevent="selectSuggestion(suggestion)"
        @mouseenter="selectedIndex = index"
      >
        <span class="suggestion-type" :class="suggestion.type">
          {{ suggestion.type }}
        </span>
        <span class="suggestion-value">{{ suggestion.label }}</span>
        <span v-if="suggestion.description" class="suggestion-desc">
          {{ suggestion.description }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from "vue";
import { useVariableContext } from "@/composables/useVariableContext";

interface Props {
  modelValue?: string;
  placeholder?: string;
  showVariableHint?: boolean;
  nodeId?: string;
  multiline?: boolean;
  rows?: number;
}

interface Suggestion {
  type: "env" | "input";
  label: string;
  value: string;
  description?: string;
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: "",
  showVariableHint: true,
  multiline: false,
  rows: 3,
});

const emit = defineEmits<{
  (e: "update:modelValue", value: string): void;
}>();

const { getAvailableVariables } = useVariableContext();

const inputRef = ref<HTMLInputElement | HTMLTextAreaElement | null>(null);
const showAutocomplete = ref(false);
const suggestions = ref<Suggestion[]>([]);
const selectedIndex = ref(0);
const dropdownTop = ref(0);
const dropdownLeft = ref(0);
const currentPrefix = ref("");
const cursorPosition = ref(0);

// Build suggestions based on available variables
const buildSuggestions = (prefix: string): Suggestion[] => {
  const available = getAvailableVariables(props.nodeId);
  const results: Suggestion[] = [];
  
  const parts = prefix.split(".");
  const firstPart = parts[0] || "";

  // 1. Env variables type suggestion
  if ("env".startsWith(firstPart)) {
      if (prefix === "" || ["e", "en", "env"].includes(prefix)) {
         results.push({
            type: "env",
            label: "env",
            value: "{{env.",
            description: "Workflow/execution variables",
          });
      }
  }

  // 2. Input variables type suggestion
  if ("input".startsWith(firstPart)) {
       if (prefix === "" || ["i", "in", "inp", "input"].includes(prefix)) {
         results.push({
            type: "input",
            label: "input",
            value: "{{input.",
            description: "Parent node output",
          });
       }
  }

  // If prefix implies we are deeper
  if (prefix.includes(".")) {
    const searchPrefix = prefix;
    
    // Check Env
    available.workflow.forEach(({ key, value }) => {
       const fullPath = `env.${key}`;
       if (fullPath.startsWith(searchPrefix) && fullPath !== searchPrefix) {
           results.push({
            type: "env",
            label: key,
            value: `{{${fullPath}}}`,
            description: typeof value === "string" ? value : JSON.stringify(value),
          });
       }
    });

    // Check Input
    available.input.forEach(({ key, description, type }) => {
        const fullPath = `input.${key}`;
        
        if (fullPath.startsWith(searchPrefix)) {
            const remaining = fullPath.slice(searchPrefix.length);
            const nextSegment = remaining.split(".")[0];
            
            if (nextSegment) {
                // Deduplicate
                if (!results.some(r => r.label === nextSegment && r.type === "input")) {
                     const isComplete = fullPath === searchPrefix + nextSegment;
                     // We suggest if it's a direct child
                     const segmentEndParams = fullPath.indexOf(".", searchPrefix.length);
                     const isDirectChild = segmentEndParams === -1 || segmentEndParams >= (searchPrefix.length + nextSegment.length);
                     
                    if (isDirectChild) {
                         results.push({
                            type: "input",
                            label: nextSegment,
                            value: `{{${searchPrefix}${nextSegment}${type === 'object' && isComplete ? '.' : ''}}}`, 
                            description: description,
                        });
                    }
                }
            }
        }
    });
  }

  return results.slice(0, 10);
};

const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement | HTMLTextAreaElement;
  const value = target.value;
  cursorPosition.value = target.selectionStart || 0;

  emit("update:modelValue", value);

  // Check if we're inside a template {{...}}
  const beforeCursor = value.slice(0, cursorPosition.value);
  const lastOpenBrace = beforeCursor.lastIndexOf("{{");
  const lastCloseBrace = beforeCursor.lastIndexOf("}}");

  if (lastOpenBrace > lastCloseBrace && lastOpenBrace !== -1) {
    // We're inside a template
    const templateContent = beforeCursor.slice(lastOpenBrace + 2);
    currentPrefix.value = templateContent;
    suggestions.value = buildSuggestions(templateContent);

    if (suggestions.value.length > 0) {
      showAutocomplete.value = true;
      selectedIndex.value = 0;
      updateDropdownPosition();
    } else {
      showAutocomplete.value = false;
    }
  } else {
    showAutocomplete.value = false;
  }
};

const handleKeydown = (event: KeyboardEvent) => {
  if (!showAutocomplete.value) return;

  switch (event.key) {
    case "ArrowDown":
      event.preventDefault();
      selectedIndex.value = Math.min(
        selectedIndex.value + 1,
        suggestions.value.length - 1,
      );
      break;
    case "ArrowUp":
      event.preventDefault();
      selectedIndex.value = Math.max(selectedIndex.value - 1, 0);
      break;
    case "Enter":
    case "Tab":
      event.preventDefault();
      const suggestion = suggestions.value[selectedIndex.value];
      if (suggestion) {
        selectSuggestion(suggestion);
      }
      break;
    case "Escape":
      event.preventDefault();
      hideAutocomplete();
      break;
  }
};

const selectSuggestion = (suggestion: Suggestion) => {
  const value = props.modelValue || "";
  const beforeCursor = value.slice(0, cursorPosition.value);
  const afterCursor = value.slice(cursorPosition.value);

  const lastOpenBrace = beforeCursor.lastIndexOf("{{");

  // Replace from {{ to cursor with the suggestion
  const newValue =
    value.slice(0, lastOpenBrace) + suggestion.value + afterCursor;

  emit("update:modelValue", newValue);
  hideAutocomplete();

  // Set cursor position after the inserted value
  nextTick(() => {
    if (inputRef.value) {
      const newCursorPos = lastOpenBrace + suggestion.value.length;
      inputRef.value.setSelectionRange(newCursorPos, newCursorPos);
      inputRef.value.focus();
    }
  });
};

const hideAutocomplete = () => {
  // Delay to allow click events on suggestions
  setTimeout(() => {
    showAutocomplete.value = false;
  }, 200);
};

const updateDropdownPosition = () => {
  nextTick(() => {
    if (inputRef.value) {
      const rect = inputRef.value.getBoundingClientRect();
      dropdownTop.value = rect.bottom + window.scrollY;
      dropdownLeft.value = rect.left + window.scrollX;
    }
  });
};
</script>

<style scoped>
.template-input {
  position: relative;
  width: 100%;
}

.input-wrapper {
  position: relative;
}

/* Adjust hint text for multiline */
.input-wrapper.is-multiline .hint-text {
  top: auto;
  bottom: 8px;
  transform: none;
}

.input-field {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  transition: border-color 0.2s;
}

.textarea-field {
  resize: vertical;
  min-height: 80px;
}

.input-field:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.hint-text {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 11px;
  color: #9ca3af;
  pointer-events: none;
}

.autocomplete-dropdown {
  position: fixed;
  z-index: 1000;
  background: white;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  max-height: 300px;
  overflow-y: auto;
  min-width: 300px;
}

.suggestion-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  font-size: 13px;
  transition: background-color 0.1s;
}

.suggestion-item:hover,
.suggestion-item.active {
  background-color: #f3f4f6;
}

.suggestion-type {
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  flex-shrink: 0;
}

.suggestion-type.env {
  background-color: #dbeafe;
  color: #1e40af;
}

.suggestion-type.input {
  background-color: #fef3c7;
  color: #92400e;
}

.suggestion-value {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-weight: 500;
  color: #111827;
  flex-shrink: 0;
}

.suggestion-desc {
  color: #6b7280;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}
</style>
