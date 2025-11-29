<template>
  <div class="transform-node" :class="{ selected: selected }">
    <div class="node-header">
      <v-icon icon="mdi-function-variant" size="20" color="white"></v-icon>
      <span class="node-title">{{ toTitleCase(data.label || 'Transform') }}</span>
    </div>
    <div class="node-body">
      <div v-if="transformations && Object.keys(transformations).length > 0" class="transformations">
        <div
          v-for="(expr, key) in transformations"
          :key="key"
          class="transformation-item"
        >
          <div class="transformation-key">{{ key }}</div>
          <div class="transformation-expr">{{ truncateExpr(expr) }}</div>
        </div>
      </div>
      <div v-else class="empty-state">
        <v-icon icon="mdi-function" size="20" color="#94a3b8"></v-icon>
        <p class="empty-text">No transformations</p>
      </div>
    </div>

    <Handle type="target" :position="Position.Left" class="node-handle node-handle-target" />
    <Handle type="source" :position="Position.Right" class="node-handle node-handle-source" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { TransformConfig } from '@/types'
import { toTitleCase } from '@/utils/formatting'

interface Props {
  id: string
  data: {
    label: string
    config?: TransformConfig
    [key: string]: any
  }
  selected?: boolean
}

const props = defineProps<Props>()

const transformations = computed(() => props.data.config?.transformations || {})

function truncateExpr(expr: string): string {
  return expr.length > 40 ? expr.substring(0, 40) + '...' : expr
}
</script>

<style scoped>
.transform-node {
  min-width: 240px;
  max-width: 320px;
  background: white;
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.transform-node.selected {
  border-color: #3b82f6;
  box-shadow: 0 8px 24px rgba(59, 130, 246, 0.25), 0 0 0 3px rgba(59, 130, 246, 0.15);
  transform: translateY(-2px);
}

.transform-node:hover {
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12);
  transform: translateY(-1px);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
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
}

.transformations {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.transformation-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  transition: all 0.2s ease;
}

.transformation-item:hover {
  border-color: #3b82f6;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.1);
}

.transformation-key {
  font-size: 11px;
  font-weight: 700;
  color: #3b82f6;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.transformation-expr {
  font-size: 12px;
  color: #475569;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 12px;
  opacity: 0.6;
}

.empty-text {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
}

.node-handle {
  width: 14px;
  height: 14px;
  border: 3px solid white;
  transition: all 0.2s ease;
}

.node-handle-target {
  background: #3b82f6;
}

.node-handle-source {
  background: #3b82f6;
}

.node-handle:hover {
  width: 18px;
  height: 18px;
  box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.2);
}
</style>

