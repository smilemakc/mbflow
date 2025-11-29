<template>
  <v-container fluid class="pa-6">
    <v-row>
      <v-col cols="12">
        <h1 class="text-h3 font-weight-bold mb-6">Execution History</h1>
      </v-col>
    </v-row>

    <v-row>
      <v-col cols="12">
        <v-card>
          <v-card-title>
            <v-text-field
              v-model="search"
              prepend-inner-icon="mdi-magnify"
              label="Search executions"
              variant="outlined"
              density="compact"
              hide-details
              clearable
            ></v-text-field>
          </v-card-title>

          <v-data-table
            :headers="headers"
            :items="executionsList"
            :loading="loading"
            :search="search"
            @click:row="openExecution"
          >
            <template #item.phase="{ item }">
              <v-chip :color="getPhaseColor(item.phase)" size="small">
                {{ item.phase }}
              </v-chip>
            </template>

            <template #item.started_at="{ item }">
              {{ formatDate(item.started_at) }}
            </template>

            <template #item.duration_ms="{ item }">
              {{ item.duration_ms ? `${item.duration_ms}ms` : 'N/A' }}
            </template>

            <template #item.actions="{ item }">
              <v-btn
                icon="mdi-eye"
                variant="text"
                size="small"
                @click.stop="openExecution(item)"
              ></v-btn>
            </template>
          </v-data-table>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useExecutionStore } from '@/stores/execution.store'
import type { Execution, ExecutionPhase } from '@/types'

const router = useRouter()
const executionStore = useExecutionStore()

const { executionsList, loading } = storeToRefs(executionStore)
const search = ref('')

const headers = [
  { title: 'ID', key: 'id', sortable: true },
  { title: 'Workflow', key: 'workflow_name', sortable: true },
  { title: 'Phase', key: 'phase', sortable: true },
  { title: 'Started', key: 'started_at', sortable: true },
  { title: 'Duration', key: 'duration_ms', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false },
]

onMounted(async () => {
  await executionStore.fetchExecutions()
})

function getPhaseColor(phase: ExecutionPhase): string {
  const colors: Record<ExecutionPhase, string> = {
    planning: 'blue',
    executing: 'orange',
    paused: 'grey',
    completed: 'success',
    failed: 'error',
    cancelled: 'warning',
  }
  return colors[phase] || 'grey'
}

function formatDate(date?: string): string {
  if (!date) return 'N/A'
  return new Date(date).toLocaleString()
}

function openExecution(item: Execution) {
  router.push(`/executions/${item.id}`)
}
</script>
