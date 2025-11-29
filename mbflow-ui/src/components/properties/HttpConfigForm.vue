<template>
  <div class="http-config-form">
    <v-select
      v-model="config.method"
      label="Method"
      :items="['GET', 'POST', 'PUT', 'DELETE', 'PATCH']"
      variant="outlined"
      density="compact"
      class="mb-3"
      @update:model-value="updateConfig"
    ></v-select>

    <v-text-field
      v-model="config.url"
      label="URL"
      variant="outlined"
      density="compact"
      class="mb-3"
      placeholder="https://api.example.com/data"
      @update:model-value="updateConfig"
    ></v-text-field>

    <v-expansion-panels variant="accordion" class="mb-3">
      <v-expansion-panel title="Headers">
        <v-expansion-panel-text class="pa-2">
          <div v-for="(value, key) in headers" :key="key" class="d-flex align-center mb-2">
            <v-text-field
              :model-value="key"
              label="Key"
              density="compact"
              variant="outlined"
              hide-details
              class="mr-2"
              readonly
            ></v-text-field>
            <v-text-field
              :model-value="value"
              label="Value"
              density="compact"
              variant="outlined"
              hide-details
              readonly
            ></v-text-field>
            <v-btn
              icon="mdi-delete"
              size="x-small"
              color="error"
              variant="text"
              @click="removeHeader(key as string)"
            ></v-btn>
          </div>
          
          <div class="d-flex align-center mt-2 pt-2 border-t">
            <v-text-field
              v-model="newHeaderKey"
              label="New Key"
              density="compact"
              variant="outlined"
              hide-details
              class="mr-2"
            ></v-text-field>
            <v-text-field
              v-model="newHeaderValue"
              label="Value"
              density="compact"
              variant="outlined"
              hide-details
              class="mr-2"
            ></v-text-field>
            <v-btn
              icon="mdi-plus"
              size="small"
              color="primary"
              variant="text"
              @click="addHeader"
            ></v-btn>
          </div>
        </v-expansion-panel-text>
      </v-expansion-panel>
    </v-expansion-panels>

    <v-textarea
      v-if="['POST', 'PUT', 'PATCH'].includes(config.method || '')"
      v-model="bodyJson"
      label="Body (JSON)"
      variant="outlined"
      rows="5"
      auto-grow
      @update:model-value="updateBody"
    ></v-textarea>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { HttpConfig } from '@/types'

interface Props {
  modelValue: HttpConfig
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [config: HttpConfig]
}>()

const config = ref<Partial<HttpConfig>>({})
const headers = ref<Record<string, string>>({})
const newHeaderKey = ref('')
const newHeaderValue = ref('')
const bodyJson = ref('')

// Initialize from props
watch(
  () => props.modelValue,
  (newConfig) => {
    config.value = { ...newConfig }
    headers.value = { ...(newConfig.headers || {}) }
    
    if (newConfig.body) {
      try {
        bodyJson.value = JSON.stringify(newConfig.body, null, 2)
      } catch {
        bodyJson.value = '{}'
      }
    } else {
      bodyJson.value = ''
    }
  },
  { immediate: true }
)

function addHeader() {
  if (newHeaderKey.value && newHeaderValue.value) {
    headers.value[newHeaderKey.value] = newHeaderValue.value
    newHeaderKey.value = ''
    newHeaderValue.value = ''
    updateConfig()
  }
}

function removeHeader(key: string) {
  delete headers.value[key]
  updateConfig()
}

function updateBody(value: string) {
  try {
    const parsed = JSON.parse(value)
    config.value.body = parsed
    updateConfig()
  } catch {
    // Invalid JSON, ignore
  }
}

function updateConfig() {
  emit('update:modelValue', {
    ...props.modelValue,
    ...config.value,
    headers: headers.value,
  })
}
</script>
