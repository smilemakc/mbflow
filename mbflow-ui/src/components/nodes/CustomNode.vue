<template>
  <div class="custom-node" :class="{ selected: selected }">
    <div class="node-header" :style="{ background: headerGradient }">
      <v-icon :icon="icon" size="20" color="white"></v-icon>
      <span class="node-title">{{ toTitleCase(data.label || type) }}</span>
    </div>
    <div class="node-body">
      <div class="node-type">
        <v-icon :icon="icon" size="16" :color="iconColor"></v-icon>
        <span class="type-text">{{ formatType(type) }}</span>
      </div>
      <div v-if="data.config && Object.keys(data.config).length > 0" class="config-info">
        <v-icon icon="mdi-cog" size="14" color="#64748b"></v-icon>
        <span class="config-text">{{ Object.keys(data.config).length }} parameters</span>
      </div>
    </div>

    <Handle type="target" :position="Position.Left" class="node-handle node-handle-target" :style="{ background: primaryColor }" />
    <Handle type="source" :position="Position.Right" class="node-handle node-handle-source" :style="{ background: primaryColor }" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { NodeType } from '@/types'
import { toTitleCase } from '@/utils/formatting'

interface Props {
  id: string
  type: NodeType
  data: {
    label: string
    config?: Record<string, any>
    [key: string]: any
  }
  selected?: boolean
}

const props = defineProps<Props>()

const icon = computed(() => {
  const icons: Record<string, string> = {
    'openai-completion': 'mdi-robot',
    'conditional-router': 'mdi-source-branch',
    parallel: 'mdi-source-fork',
    'json-parser': 'mdi-code-json',
    llm: 'mdi-brain',
    code: 'mdi-code-braces',
    join: 'mdi-call-merge',
    filter: 'mdi-filter',
  }
  return icons[props.type] || 'mdi-circle-outline'
})

const primaryColor = computed(() => {
  const colors: Record<string, string> = {
    'openai-completion': '#a855f7',
    'conditional-router': '#06b6d4',
    parallel: '#6366f1',
    'json-parser': '#64748b',
    llm: '#ec4899',
    code: '#78716c',
    join: '#14b8a6',
    filter: '#f59e0b',
  }
  return colors[props.type] || '#94a3b8'
})

const headerGradient = computed(() => {
  const gradients: Record<string, string> = {
    'openai-completion': 'linear-gradient(135deg, #a855f7 0%, #9333ea 100%)',
    'conditional-router': 'linear-gradient(135deg, #06b6d4 0%, #0891b2 100%)',
    parallel: 'linear-gradient(135deg, #6366f1 0%, #4f46e5 100%)',
    'json-parser': 'linear-gradient(135deg, #64748b 0%, #475569 100%)',
    llm: 'linear-gradient(135deg, #ec4899 0%, #db2777 100%)',
    code: 'linear-gradient(135deg, #78716c 0%, #57534e 100%)',
    join: 'linear-gradient(135deg, #14b8a6 0%, #0d9488 100%)',
    filter: 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)',
  }
  return gradients[props.type] || 'linear-gradient(135deg, #94a3b8 0%, #64748b 100%)'
})

const iconColor = computed(() => primaryColor.value)

function formatType(type: string): string {
  return type.split('-').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
}
</script>

<style scoped>
.custom-node {
  min-width: 240px;
  max-width: 320px;
  background: white;
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.custom-node.selected {
  border-color: v-bind(primaryColor);
  box-shadow: 0 8px 24px v-bind('primaryColor + "40"'), 0 0 0 3px v-bind('primaryColor + "26"');
  transform: translateY(-2px);
}

.custom-node:hover {
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12);
  transform: translateY(-1px);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  color: white;
  font-weight: 600;
  font-size: 14px;
  position: relative;
}

.node-header::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0) 100%);
  pointer-events: none;
}

.node-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  position: relative;
  z-index: 1;
}

.node-body {
  padding: 14px 16px;
  background: #fafafa;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.node-type {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.type-text {
  font-size: 12px;
  font-weight: 600;
  color: #475569;
}

.config-info {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
}

.config-text {
  font-size: 11px;
  color: #64748b;
  font-weight: 500;
}

.node-handle {
  width: 14px;
  height: 14px;
  border: 3px solid white;
  transition: all 0.2s ease;
}

.node-handle:hover {
  width: 18px;
  height: 18px;
  box-shadow: 0 0 0 4px v-bind('primaryColor + "33"');
}
</style>

