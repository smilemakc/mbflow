<template>
  <v-card class="variable-explorer" elevation="0" border>
    <v-card-title class="text-subtitle-2 py-2 px-3 bg-grey-lighten-4">
      <v-icon icon="mdi-variable" size="small" class="mr-2"></v-icon>
      Available Variables
    </v-card-title>
    
    <v-card-text class="pa-0">
      <div v-if="variables.length === 0" class="text-caption text-grey pa-3 text-center">
        Select a node to see available variables from its predecessors.
      </div>
      
      <v-list v-else density="compact" class="variable-list">
        <v-list-item
          v-for="variable in variables"
          :key="variable.name"
          :value="variable"
          @click="copyToClipboard(variable.name)"
        >
          <template #prepend>
            <v-icon :icon="getTypeIcon(variable.type)" size="x-small" color="primary"></v-icon>
          </template>
          
          <v-list-item-title class="text-caption font-weight-medium">
            {{ variable.name }}
          </v-list-item-title>
          
          <v-list-item-subtitle class="text-caption" style="font-size: 10px !important">
            {{ variable.description }}
          </v-list-item-subtitle>
          
          <template #append>
            <v-btn
              icon="mdi-content-copy"
              variant="text"
              size="x-small"
              color="grey"
            ></v-btn>
          </template>
        </v-list-item>
      </v-list>
    </v-card-text>
    
    <v-snackbar
      v-model="snackbar"
      :timeout="2000"
      color="success"
      location="bottom right"
      min-width="auto"
    >
      Copied to clipboard
    </v-snackbar>
  </v-card>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Node, Edge, VariableDefinition } from '@/types'
import { resolveAvailableVariables } from '@/utils/variable-resolver'

interface Props {
  selectedNode: Node | null
  nodes: Node[]
  edges: Edge[]
}

const props = defineProps<Props>()
const snackbar = ref(false)

const variables = computed(() => {
  if (!props.selectedNode) return []
  return resolveAvailableVariables(props.selectedNode.id, props.nodes, props.edges)
})

function getTypeIcon(type: string): string {
  const icons: Record<string, string> = {
    string: 'mdi-format-text',
    number: 'mdi-numeric',
    boolean: 'mdi-toggle-switch-outline',
    object: 'mdi-code-json',
    array: 'mdi-code-brackets',
    any: 'mdi-help-circle-outline',
  }
  return icons[type] || icons.any
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(`{{${text}}}`)
  snackbar.value = true
}
</script>

<style scoped>
.variable-explorer {
  max-height: 200px;
  display: flex;
  flex-direction: column;
}

.variable-list {
  overflow-y: auto;
  max-height: 160px;
}
</style>
