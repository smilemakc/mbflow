<template>
  <div class="http-node" :class="{ selected: selected }">
    <div class="node-header">
      <v-icon icon="mdi-web" size="20" color="white"></v-icon>
      <span class="node-title">{{ toTitleCase(data.label || 'HTTP Request') }}</span>
    </div>
    <div class="node-body">
      <div v-if="httpConfig" class="http-config">
        <div class="config-method">
          <v-chip size="small" :color="methodColor" variant="flat">
            {{ httpConfig.method || 'GET' }}
          </v-chip>
        </div>
        <div v-if="httpConfig.url" class="config-url">
          <v-icon icon="mdi-link-variant" size="14" class="url-icon"></v-icon>
          <span class="url-text">{{ truncateUrl(httpConfig.url) }}</span>
        </div>
      </div>
      <div v-else class="empty-state">
        <v-icon icon="mdi-cog-outline" size="20" color="#94a3b8"></v-icon>
        <p class="empty-text">Not configured</p>
      </div>
    </div>

    <Handle type="target" :position="Position.Left" class="node-handle node-handle-target" />
    <Handle type="source" :position="Position.Right" class="node-handle node-handle-source" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { HttpConfig } from '@/types'
import { toTitleCase } from '@/utils/formatting'

interface Props {
  id: string
  data: {
    label: string
    config?: HttpConfig
    [key: string]: any
  }
  selected?: boolean
}

const props = defineProps<Props>()

const httpConfig = computed(() => props.data.config as HttpConfig | undefined)

const methodColor = computed(() => {
  const colors: Record<string, string> = {
    GET: 'success',
    POST: 'primary',
    PUT: 'warning',
    DELETE: 'error',
    PATCH: 'info',
  }
  return colors[httpConfig.value?.method || 'GET'] || 'grey'
})

function truncateUrl(url: string): string {
  return url.length > 35 ? url.substring(0, 35) + '...' : url
}
</script>

<style scoped>
.http-node {
  min-width: 240px;
  max-width: 320px;
  background: white;
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.http-node.selected {
  border-color: #f97316;
  box-shadow: 0 8px 24px rgba(249, 115, 22, 0.25), 0 0 0 3px rgba(249, 115, 22, 0.15);
  transform: translateY(-2px);
}

.http-node:hover {
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12);
  transform: translateY(-1px);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: linear-gradient(135deg, #f97316 0%, #ea580c 100%);
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

.http-config {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.config-method {
  display: flex;
}

.config-url {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.url-icon {
  color: #64748b;
  flex-shrink: 0;
}

.url-text {
  color: #475569;
  font-size: 12px;
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
  background: #f97316;
}

.node-handle-source {
  background: #f97316;
}

.node-handle:hover {
  width: 18px;
  height: 18px;
  box-shadow: 0 0 0 4px rgba(249, 115, 22, 0.2);
}
</style>

