<template>
  <div class="monaco-editor-wrapper">
    <div ref="editorContainer" class="editor-container"></div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import * as monaco from "monaco-editor";
import { useVariableContext } from "@/composables/useVariableContext";

interface Props {
  modelValue: string;
  language?: "json" | "javascript" | "jq";
  height?: string;
  readonly?: boolean;
  nodeId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  language: "json",
  height: "200px",
  readonly: false,
});

const emit = defineEmits<{
  (e: "update:modelValue", value: string): void;
}>();

const { getAvailableVariables } = useVariableContext();

const editorContainer = ref<HTMLElement | null>(null);
let editor: monaco.editor.IStandaloneCodeEditor | null = null;

onMounted(() => {
  if (!editorContainer.value) return;

  // Create editor
  editor = monaco.editor.create(editorContainer.value, {
    value: props.modelValue,
    language: props.language === "jq" ? "plaintext" : props.language,
    theme: "vs",
    minimap: { enabled: false },
    lineNumbers: "on",
    readOnly: props.readonly,
    fontSize: 13,
    fontFamily: '"Monaco", "Menlo", "Ubuntu Mono", monospace',
    scrollBeyondLastLine: false,
    automaticLayout: true,
    wordWrap: "on",
    wrappingStrategy: "advanced",
    tabSize: 2,
    insertSpaces: true,
  });

  // Listen to content changes
  editor.onDidChangeModelContent(() => {
    if (editor) {
      const value = editor.getValue();
      emit("update:modelValue", value);
    }
  });

  // Setup autocomplete for templates
  setupTemplateAutocomplete();
});

onBeforeUnmount(() => {
  if (editor) {
    editor.dispose();
  }
});

// Watch for external value changes
watch(
  () => props.modelValue,
  (newValue) => {
    if (editor && editor.getValue() !== newValue) {
      editor.setValue(newValue);
    }
  },
);

// Watch for language changes
watch(
  () => props.language,
  (newLanguage) => {
    if (editor) {
      const model = editor.getModel();
      if (model) {
        monaco.editor.setModelLanguage(
          model,
          newLanguage === "jq" ? "plaintext" : newLanguage,
        );
      }
    }
  },
);

function setupTemplateAutocomplete() {
  if (!editor) return;

  const available = getAvailableVariables();

  // Register completion provider for template variables
  monaco.languages.registerCompletionItemProvider(
    props.language === "jq" ? "plaintext" : props.language,
    {
      triggerCharacters: ["{", "."],
      provideCompletionItems: (model, position) => {
        const textUntilPosition = model.getValueInRange({
          startLineNumber: position.lineNumber,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });

        // Check if we're inside {{...}}
        const lastOpenBrace = textUntilPosition.lastIndexOf("{{");
        const lastCloseBrace = textUntilPosition.lastIndexOf("}}");

        if (lastOpenBrace > lastCloseBrace && lastOpenBrace !== -1) {
          const templateContent = textUntilPosition.slice(lastOpenBrace + 2);
          const parts = templateContent.split(".");

          const suggestions: monaco.languages.CompletionItem[] = [];

          if (parts.length === 1) {
            // Suggest types: env, input
            suggestions.push(
              {
                label: "env",
                kind: monaco.languages.CompletionItemKind.Keyword,
                insertText: "env.",
                detail: "Workflow/execution variables",
                range: {
                  startLineNumber: position.lineNumber,
                  startColumn: position.column,
                  endLineNumber: position.lineNumber,
                  endColumn: position.column,
                },
              },
              {
                label: "input",
                kind: monaco.languages.CompletionItemKind.Keyword,
                insertText: "input.",
                detail: "Parent node output",
                range: {
                  startLineNumber: position.lineNumber,
                  startColumn: position.column,
                  endLineNumber: position.lineNumber,
                  endColumn: position.column,
                },
              },
            );
          } else if (parts.length === 2) {
            // Suggest variable keys
            const type = parts[0];

            if (type === "env") {
              available.workflow.forEach(({ key, value }) => {
                suggestions.push({
                  label: key,
                  kind: monaco.languages.CompletionItemKind.Variable,
                  insertText: key + "}}",
                  detail:
                    typeof value === "string" ? value : JSON.stringify(value),
                  range: {
                    startLineNumber: position.lineNumber,
                    startColumn: position.column,
                    endLineNumber: position.lineNumber,
                    endColumn: position.column,
                  },
                });
              });
            } else if (type === "input") {
              available.input.forEach(({ key, description }) => {
                suggestions.push({
                  label: key,
                  kind: monaco.languages.CompletionItemKind.Variable,
                  insertText: key + "}}",
                  detail: description,
                  range: {
                    startLineNumber: position.lineNumber,
                    startColumn: position.column,
                    endLineNumber: position.lineNumber,
                    endColumn: position.column,
                  },
                });
              });
            }
          }

          return { suggestions };
        }

        return { suggestions: [] };
      },
    },
  );
}
</script>

<style scoped>
.monaco-editor-wrapper {
  border: 1px solid #d1d5db;
  border-radius: 6px;
  overflow: hidden;
}

.editor-container {
  height: v-bind(height);
  width: 100%;
}
</style>
