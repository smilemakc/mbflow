<template>
  <v-card class="node-palette" elevation="0">
    <div class="palette-header">
      <div class="header-icon">
        <v-icon icon="mdi-palette" size="24" color="white"></v-icon>
      </div>
      <h3 class="header-title">Node Palette</h3>
    </div>

    <v-card-text class="pa-3">
      <v-expansion-panels variant="accordion" class="custom-panels">
        <!-- Control Nodes -->
        <v-expansion-panel>
          <v-expansion-panel-title class="category-title">
            <div class="category-icon control-icon">
              <v-icon icon="mdi-cog" size="18" color="white"></v-icon>
            </div>
            <span>Control Flow</span>
          </v-expansion-panel-title>
          <v-expansion-panel-text>
            <div class="node-list">
              <div
                v-for="node in controlNodes"
                :key="node.type"
                :class="['node-item', `node-${node.type}`]"
                draggable="true"
                @dragstart="onDragStart($event, node)"
              >
                <div :class="['node-icon', getNodeColorClass(node.type)]">
                  <v-icon :icon="node.icon" size="20" color="white"></v-icon>
                </div>
                <div class="node-info">
                  <span class="node-label">{{ node.label }}</span>
                  <span class="node-type">{{ formatNodeType(node.type) }}</span>
                </div>
                <v-icon icon="mdi-drag-vertical" size="18" class="drag-handle"></v-icon>
              </div>
            </div>
          </v-expansion-panel-text>
        </v-expansion-panel>

        <!-- Transform Nodes -->
        <v-expansion-panel>
          <v-expansion-panel-title class="category-title">
            <div class="category-icon transform-icon">
              <v-icon icon="mdi-function-variant" size="18" color="white"></v-icon>
            </div>
            <span>Transform</span>
          </v-expansion-panel-title>
          <v-expansion-panel-text>
            <div class="node-list">
              <div
                v-for="node in transformNodes"
                :key="node.type"
                :class="['node-item', `node-${node.type}`]"
                draggable="true"
                @dragstart="onDragStart($event, node)"
              >
                <div :class="['node-icon', getNodeColorClass(node.type)]">
                  <v-icon :icon="node.icon" size="20" color="white"></v-icon>
                </div>
                <div class="node-info">
                  <span class="node-label">{{ node.label }}</span>
                  <span class="node-type">{{ formatNodeType(node.type) }}</span>
                </div>
                <v-icon icon="mdi-drag-vertical" size="18" class="drag-handle"></v-icon>
              </div>
            </div>
          </v-expansion-panel-text>
        </v-expansion-panel>

        <!-- Integration Nodes -->
        <v-expansion-panel>
          <v-expansion-panel-title class="category-title">
            <div class="category-icon integration-icon">
              <v-icon icon="mdi-connection" size="18" color="white"></v-icon>
            </div>
            <span>Integration</span>
          </v-expansion-panel-title>
          <v-expansion-panel-text>
            <div class="node-list">
              <div
                v-for="node in integrationNodes"
                :key="node.type"
                :class="['node-item', `node-${node.type}`]"
                draggable="true"
                @dragstart="onDragStart($event, node)"
              >
                <div :class="['node-icon', getNodeColorClass(node.type)]">
                  <v-icon :icon="node.icon" size="20" color="white"></v-icon>
                </div>
                <div class="node-info">
                  <span class="node-label">{{ node.label }}</span>
                  <span class="node-type">{{ formatNodeType(node.type) }}</span>
                </div>
                <v-icon icon="mdi-drag-vertical" size="18" class="drag-handle"></v-icon>
              </div>
            </div>
          </v-expansion-panel-text>
        </v-expansion-panel>

        <!-- AI Nodes -->
        <v-expansion-panel>
          <v-expansion-panel-title class="category-title">
            <div class="category-icon ai-icon">
              <v-icon icon="mdi-robot" size="18" color="white"></v-icon>
            </div>
            <span>AI & ML</span>
          </v-expansion-panel-title>
          <v-expansion-panel-text>
            <div class="node-list">
              <div
                v-for="node in aiNodes"
                :key="node.type"
                :class="['node-item', `node-${node.type}`]"
                draggable="true"
                @dragstart="onDragStart($event, node)"
              >
                <div :class="['node-icon', getNodeColorClass(node.type)]">
                  <v-icon :icon="node.icon" size="20" color="white"></v-icon>
                </div>
                <div class="node-info">
                  <span class="node-label">{{ node.label }}</span>
                  <span class="node-type">{{ formatNodeType(node.type) }}</span>
                </div>
                <v-icon icon="mdi-drag-vertical" size="18" class="drag-handle"></v-icon>
              </div>
            </div>
          </v-expansion-panel-text>
        </v-expansion-panel>

        <!-- Data Nodes -->
        <v-expansion-panel>
          <v-expansion-panel-title class="category-title">
            <div class="category-icon data-icon">
              <v-icon icon="mdi-database" size="18" color="white"></v-icon>
            </div>
            <span>Data</span>
          </v-expansion-panel-title>
          <v-expansion-panel-text>
            <div class="node-list">
              <div
                v-for="node in dataNodes"
                :key="node.type"
                :class="['node-item', `node-${node.type}`]"
                draggable="true"
                @dragstart="onDragStart($event, node)"
              >
                <div :class="['node-icon', getNodeColorClass(node.type)]">
                  <v-icon :icon="node.icon" size="20" color="white"></v-icon>
                </div>
                <div class="node-info">
                  <span class="node-label">{{ node.label }}</span>
                  <span class="node-type">{{ formatNodeType(node.type) }}</span>
                </div>
                <v-icon icon="mdi-drag-vertical" size="18" class="drag-handle"></v-icon>
              </div>
            </div>
          </v-expansion-panel-text>
        </v-expansion-panel>
      </v-expansion-panels>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { NodeTypes } from '@/types'

interface NodePaletteItem {
  type: string
  label: string
  icon: string
  category: string
}

const controlNodes: NodePaletteItem[] = [
  { type: NodeTypes.CONDITIONAL_ROUTER, label: 'Conditional Router', icon: 'mdi-source-branch', category: 'control' },
  { type: NodeTypes.PARALLEL, label: 'Parallel', icon: 'mdi-source-fork', category: 'control' },
]

const transformNodes: NodePaletteItem[] = [
  { type: NodeTypes.TRANSFORM, label: 'Transform', icon: 'mdi-function-variant', category: 'transform' },
  { type: NodeTypes.CODE, label: 'Code', icon: 'mdi-code-braces', category: 'transform' },
  { type: NodeTypes.SCRIPT_EXECUTOR, label: 'Script', icon: 'mdi-script-text', category: 'transform' },
]

const integrationNodes: NodePaletteItem[] = [
  { type: NodeTypes.HTTP, label: 'HTTP Request', icon: 'mdi-web', category: 'integration' },
  { type: NodeTypes.HTTP_REQUEST, label: 'HTTP', icon: 'mdi-api', category: 'integration' },
  { type: NodeTypes.TELEGRAM_MESSAGE, label: 'Telegram', icon: 'mdi-send', category: 'integration' },
]

const aiNodes: NodePaletteItem[] = [
  { type: NodeTypes.OPENAI_COMPLETION, label: 'OpenAI Completion', icon: 'mdi-robot', category: 'ai' },
  { type: NodeTypes.OPENAI_RESPONSES, label: 'OpenAI Responses', icon: 'mdi-chat', category: 'ai' },
  { type: NodeTypes.LLM, label: 'LLM', icon: 'mdi-brain', category: 'ai' },
  { type: NodeTypes.FUNCTION_CALL, label: 'Function Call', icon: 'mdi-function', category: 'ai' },
]

const dataNodes: NodePaletteItem[] = [
  { type: NodeTypes.JSON_PARSER, label: 'JSON Parser', icon: 'mdi-code-json', category: 'data' },
  { type: NodeTypes.DATA_MERGER, label: 'Data Merger', icon: 'mdi-merge', category: 'data' },
  { type: NodeTypes.DATA_AGGREGATOR, label: 'Data Aggregator', icon: 'mdi-sigma', category: 'data' },
]

function getNodeColorClass(nodeType: string): string {
  const colorMap: Record<string, string> = {
    [NodeTypes.TRANSFORM]: 'color-blue',
    [NodeTypes.HTTP]: 'color-orange',
    [NodeTypes.HTTP_REQUEST]: 'color-orange',
    [NodeTypes.OPENAI_COMPLETION]: 'color-purple',
    [NodeTypes.OPENAI_RESPONSES]: 'color-purple',
    [NodeTypes.LLM]: 'color-pink',
    [NodeTypes.CONDITIONAL_ROUTER]: 'color-cyan',
    [NodeTypes.PARALLEL]: 'color-indigo',
    [NodeTypes.JSON_PARSER]: 'color-slate',
    [NodeTypes.CODE]: 'color-stone',
    [NodeTypes.SCRIPT_EXECUTOR]: 'color-stone',
    [NodeTypes.TELEGRAM_MESSAGE]: 'color-sky',
    [NodeTypes.FUNCTION_CALL]: 'color-violet',
    [NodeTypes.DATA_MERGER]: 'color-teal',
    [NodeTypes.DATA_AGGREGATOR]: 'color-emerald',
  }
  return colorMap[nodeType] || 'color-gray'
}

function formatNodeType(type: string): string {
  return type.replace(/-/g, ' ').toLowerCase()
}

function onDragStart(event: DragEvent, node: NodePaletteItem) {
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    const data = JSON.stringify({ type: node.type })
    event.dataTransfer.setData('application/json', data)
    event.dataTransfer.setData('text/plain', node.type)
    
    console.log('[NodePalette] Drag started:', node.type)
  }
}
</script>

<style scoped>
.node-palette {
  width: 300px;
  height: 100%;
  overflow-y: auto;
  background: #fafafa;
  border-right: 1px solid #e2e8f0;
}

/* Header */
.palette-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  position: sticky;
  top: 0;
  z-index: 10;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.2);
}

.header-icon {
  width: 40px;
  height: 40px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(10px);
}

.header-title {
  color: white;
  font-size: 18px;
  font-weight: 700;
  margin: 0;
  letter-spacing: 0.3px;
}

/* Custom Panels */
.custom-panels {
  background: transparent;
}

.custom-panels :deep(.v-expansion-panel) {
  background: white;
  margin-bottom: 8px;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.category-title {
  font-weight: 600;
  color: #1e293b;
  padding: 14px 16px;
  min-height: 56px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.category-title:hover {
  background: #f8fafc;
}

.category-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.control-icon {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.transform-icon {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.integration-icon {
  background: linear-gradient(135deg, #f97316 0%, #ea580c 100%);
}

.ai-icon {
  background: linear-gradient(135deg, #a855f7 0%, #9333ea 100%);
}

.data-icon {
  background: linear-gradient(135deg, #64748b 0%, #475569 100%);
}

.custom-panels :deep(.v-expansion-panel-text__wrapper) {
  padding: 12px;
}

/* Node List */
.node-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

/* Node Item */
.node-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: white;
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  cursor: grab;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.node-item::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(102, 126, 234, 0.05) 0%, rgba(118, 75, 162, 0.05) 100%);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.node-item:hover::before {
  opacity: 1;
}

.node-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.node-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.node-label {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
  line-height: 1.3;
}

.node-type {
  font-size: 10px;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 500;
}

.drag-handle {
  color: #cbd5e1;
  flex-shrink: 0;
  transition: all 0.3s ease;
  opacity: 0;
}

.node-item:hover {
  border-color: #667eea;
  transform: translateX(4px);
  box-shadow: 0 8px 20px rgba(102, 126, 234, 0.15);
}

.node-item:hover .node-icon {
  transform: scale(1.1) rotate(5deg);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.15);
}

.node-item:hover .drag-handle {
  opacity: 1;
  color: #667eea;
}

.node-item:active {
  cursor: grabbing;
  transform: translateX(2px) scale(0.98);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.2);
}

/* Color Classes */
.color-green {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.color-red {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
}

.color-blue {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.color-orange {
  background: linear-gradient(135deg, #f97316 0%, #ea580c 100%);
}

.color-purple {
  background: linear-gradient(135deg, #a855f7 0%, #9333ea 100%);
}

.color-pink {
  background: linear-gradient(135deg, #ec4899 0%, #db2777 100%);
}

.color-cyan {
  background: linear-gradient(135deg, #06b6d4 0%, #0891b2 100%);
}

.color-indigo {
  background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
}

.color-slate {
  background: linear-gradient(135deg, #64748b 0%, #475569 100%);
}

.color-stone {
  background: linear-gradient(135deg, #78716c 0%, #57534e 100%);
}

.color-sky {
  background: linear-gradient(135deg, #0ea5e9 0%, #0284c7 100%);
}

.color-violet {
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
}

.color-teal {
  background: linear-gradient(135deg, #14b8a6 0%, #0d9488 100%);
}

.color-emerald {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.color-gray {
  background: linear-gradient(135deg, #94a3b8 0%, #64748b 100%);
}

/* Scrollbar */
.node-palette::-webkit-scrollbar {
  width: 8px;
}

.node-palette::-webkit-scrollbar-track {
  background: #f1f5f9;
}

.node-palette::-webkit-scrollbar-thumb {
  background: linear-gradient(180deg, #667eea 0%, #764ba2 100%);
  border-radius: 4px;
}

.node-palette::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(180deg, #764ba2 0%, #667eea 100%);
}
</style>
