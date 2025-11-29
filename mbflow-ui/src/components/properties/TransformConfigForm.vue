<template>
  <div class="transform-config-form">
    <div class="d-flex justify-space-between align-center mb-2">
      <h4 class="text-subtitle-2">Transformations</h4>
      <v-btn
        size="x-small"
        color="primary"
        prepend-icon="mdi-plus"
        variant="text"
        @click="addTransformation"
      >
        Add
      </v-btn>
    </div>

    <div v-if="transformations.length === 0" class="text-caption text-grey mb-4">
      No transformations defined. Add one to transform data.
    </div>

    <div v-else class="transformations-list">
      <div
        v-for="(item, index) in transformations"
        :key="index"
        class="transformation-item mb-3 pa-2 border rounded"
      >
        <div class="d-flex align-center mb-2">
          <v-text-field
            v-model="item.key"
            label="Output Key"
            density="compact"
            variant="outlined"
            hide-details
            class="mr-2"
            @update:model-value="updateConfig"
          ></v-text-field>
          <v-btn
            icon="mdi-delete"
            size="x-small"
            color="error"
            variant="text"
            @click="removeTransformation(index)"
          ></v-btn>
        </div>
        
        <v-textarea
          v-model="item.expression"
          label="Expression (Expr)"
          density="compact"
          variant="outlined"
          rows="2"
          hide-details
          auto-grow
          @update:model-value="updateConfig"
        ></v-textarea>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { TransformConfig } from '@/types'

interface Props {
  modelValue: TransformConfig
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [config: TransformConfig]
}>()

interface TransformItem {
  key: string
  expression: string
}

const transformations = ref<TransformItem[]>([])

// Initialize from props
watch(
  () => props.modelValue,
  (newConfig) => {
    if (newConfig?.transformations) {
      transformations.value = Object.entries(newConfig.transformations).map(
        ([key, expression]) => ({ key, expression: expression as string })
      )
    } else {
      transformations.value = []
    }
  },
  { immediate: true, deep: true }
)

function addTransformation() {
  transformations.value.push({ key: '', expression: '' })
  updateConfig()
}

function removeTransformation(index: number) {
  transformations.value.splice(index, 1)
  updateConfig()
}

function updateConfig() {
  const newTransformations: Record<string, string> = {}
  
  transformations.value.forEach((item) => {
    if (item.key) {
      newTransformations[item.key] = item.expression
    }
  })
  
  emit('update:modelValue', {
    ...props.modelValue,
    transformations: newTransformations,
  })
}
</script>

<style scoped>
.transformation-item {
  background-color: #f9f9f9;
}
</style>
