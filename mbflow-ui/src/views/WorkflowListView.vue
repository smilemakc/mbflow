<template>
  <v-container fluid class="pa-6">
    <v-row>
      <v-col cols="12">
        <div class="d-flex justify-space-between align-center mb-6">
          <h1 class="text-h3 font-weight-bold">Workflows</h1>
          <v-btn
            color="primary"
            size="large"
            prepend-icon="mdi-plus"
            @click="createNewWorkflow"
          >
            New Workflow
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-row v-if="loading">
      <v-col cols="12" class="text-center">
        <v-progress-circular indeterminate color="primary" size="64"></v-progress-circular>
      </v-col>
    </v-row>

    <v-row v-else-if="error">
      <v-col cols="12">
        <v-alert type="error" variant="tonal">
          {{ error }}
        </v-alert>
      </v-col>
    </v-row>

    <v-row v-else>
      <v-col
        v-for="workflow in workflows"
        :key="workflow.id"
        cols="12"
        md="6"
        lg="4"
      >
        <v-card hover @click="openWorkflow(workflow.id)">
          <v-card-title class="d-flex align-center">
            <v-icon icon="mdi-sitemap" class="mr-2"></v-icon>
            {{ workflow.name }}
          </v-card-title>

          <v-card-subtitle>
            Version {{ workflow.version }}
          </v-card-subtitle>

          <v-card-text>
            <p class="text-body-2 mb-3">
              {{ workflow.description || 'No description' }}
            </p>

            <v-chip size="small" class="mr-2">
              <v-icon start icon="mdi-circle-outline"></v-icon>
              {{ workflow.nodes.length }} nodes
            </v-chip>

            <v-chip size="small" class="mr-2">
              <v-icon start icon="mdi-arrow-right"></v-icon>
              {{ workflow.edges.length }} edges
            </v-chip>

            <v-chip size="small">
              <v-icon start icon="mdi-lightning-bolt"></v-icon>
              {{ workflow.triggers.length }} triggers
            </v-chip>
          </v-card-text>

          <v-card-actions>
            <v-btn
              variant="text"
              prepend-icon="mdi-pencil"
              @click.stop="openWorkflow(workflow.id)"
            >
              Edit
            </v-btn>

            <v-btn
              variant="text"
              prepend-icon="mdi-play"
              @click.stop="executeWorkflow(workflow.id)"
            >
              Run
            </v-btn>

            <v-spacer></v-spacer>

            <v-btn
              icon="mdi-delete"
              variant="text"
              color="error"
              @click.stop="deleteWorkflow(workflow.id)"
            ></v-btn>
          </v-card-actions>
        </v-card>
      </v-col>

      <v-col v-if="workflows.length === 0" cols="12">
        <v-card class="text-center pa-12">
          <v-icon icon="mdi-sitemap" size="64" color="grey-lighten-1"></v-icon>
          <h2 class="text-h5 mt-4 mb-2">No workflows yet</h2>
          <p class="text-body-1 text-grey mb-4">
            Create your first workflow to get started
          </p>
          <v-btn color="primary" size="large" @click="createNewWorkflow">
            Create Workflow
          </v-btn>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useWorkflowStore } from '@/stores/workflow.store'
import { useExecutionStore } from '@/stores/execution.store'

const router = useRouter()
const workflowStore = useWorkflowStore()
const executionStore = useExecutionStore()

const { workflows, loading, error } = storeToRefs(workflowStore)

onMounted(async () => {
  await workflowStore.fetchWorkflows()
})

function createNewWorkflow() {
  router.push('/workflows/new')
}

function openWorkflow(id: string) {
  router.push(`/workflows/${id}`)
}

async function executeWorkflow(id: string) {
  try {
    await executionStore.executeWorkflow(id)
    // Show success message
    console.log('Workflow execution started')
  } catch (e) {
    console.error('Failed to execute workflow:', e)
  }
}

async function deleteWorkflow(id: string) {
  if (confirm('Are you sure you want to delete this workflow?')) {
    try {
      await workflowStore.deleteWorkflow(id)
    } catch (e) {
      console.error('Failed to delete workflow:', e)
    }
  }
}
</script>
