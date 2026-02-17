<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Button from 'primevue/button'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import Toast from 'primevue/toast'
import PipelineStepList from '@/features/pipeline/PipelineStepList.vue'
import AddStepDialog from '@/features/pipeline/AddStepDialog.vue'
import { usePipelineConfig } from '@/composables/usePipelineConfig'
import { useAuth } from '@/composables/useAuth'
import type { PipelineStep } from '@/stores/pipelineConfig'

const route = useRoute()
const toast = useToast()
const { user } = useAuth()

const projectId = computed(() => route.params.id as string)
const {
  steps,
  isLoading,
  isSaving,
  error,
  isDirty,
  retry,
  saveConfig,
  addStep,
  removeStep,
  reorderSteps,
  updateStep,
} = usePipelineConfig(projectId)

const isAdmin = computed(() => user.value?.role === 'admin')
const showAddDialog = ref(false)

function handleAddStep(step: PipelineStep) {
  addStep(step)
}

function handleUpdateStep(index: number, step: PipelineStep) {
  updateStep(index, step)
}

function handleRemoveStep(index: number) {
  removeStep(index)
}

function handleReorder(fromIndex: number, toIndex: number) {
  reorderSteps(fromIndex, toIndex)
}

async function handleSave() {
  const success = await saveConfig()
  if (success) {
    toast.add({
      severity: 'success',
      summary: 'Configuration saved',
      detail: 'Pipeline configuration has been updated successfully',
      life: 3000,
    })
  } else {
    toast.add({
      severity: 'error',
      summary: 'Save failed',
      detail: 'Failed to save pipeline configuration. Please try again.',
      life: 5000,
    })
  }
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Pipeline Configuration</h1>
      <div v-if="isAdmin && !isLoading && !error" class="flex gap-2">
        <Button
          label="Add Step"
          icon="pi pi-plus"
          severity="secondary"
          data-testid="add-step-btn"
          @click="showAddDialog = true"
        />
        <Button
          label="Save"
          icon="pi pi-save"
          severity="success"
          :disabled="!isDirty"
          :loading="isSaving"
          data-testid="save-config-btn"
          @click="handleSave"
        />
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex flex-col gap-3" data-testid="loading-skeleton">
      <Skeleton height="4rem" />
      <Skeleton height="4rem" />
      <Skeleton height="4rem" />
    </div>

    <!-- Error state -->
    <Message v-else-if="error" severity="error" :closable="false" data-testid="error-message">
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
      </div>
    </Message>

    <!-- Empty state -->
    <Message
      v-else-if="steps.length === 0"
      severity="info"
      :closable="false"
      data-testid="empty-message"
    >
      <span>No pipeline steps configured.</span>
      <Button
        v-if="isAdmin"
        label="Add your first step"
        text
        size="small"
        class="ml-2"
        @click="showAddDialog = true"
      />
    </Message>

    <!-- Step list -->
    <PipelineStepList
      v-else
      :steps="steps"
      :is-admin="isAdmin"
      @update="handleUpdateStep"
      @remove="handleRemoveStep"
      @reorder="handleReorder"
    />

    <!-- Add step dialog (admin only) -->
    <AddStepDialog
      v-if="isAdmin"
      v-model:visible="showAddDialog"
      @add="handleAddStep"
    />

    <Toast />
  </div>
</template>
