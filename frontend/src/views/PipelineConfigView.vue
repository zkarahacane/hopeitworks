<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Button from 'primevue/button'
import ConfirmDialog from 'primevue/confirmdialog'
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
  groups,
  steps,
  isLoading,
  isSaving,
  error,
  isDirty,
  retry,
  saveConfig,
  addGroup,
  removeGroup,
  renameGroup,
  addStepToGroup,
  removeStepFromGroup,
  updateStepInGroup,
  reorderStepsInGroup,
  reorderGroups,
} = usePipelineConfig(projectId)

const isAdmin = computed(() => user.value?.role === 'admin')
const showAddDialog = ref(false)
const addStepTargetGroupId = ref<string | null>(null)

function handleAddGroup() {
  addGroup()
}

function handleRenameGroup(groupId: string, name: string) {
  renameGroup(groupId, name)
}

function handleRemoveGroup(groupId: string) {
  removeGroup(groupId)
}

function handleOpenAddStep(groupId: string) {
  addStepTargetGroupId.value = groupId
  showAddDialog.value = true
}

function handleAddStep(step: PipelineStep) {
  if (addStepTargetGroupId.value) {
    addStepToGroup(addStepTargetGroupId.value, step)
  }
  addStepTargetGroupId.value = null
}

function handleUpdateStep(groupId: string, stepId: string, step: PipelineStep) {
  updateStepInGroup(groupId, stepId, step)
}

function handleRemoveStep(groupId: string, stepId: string) {
  removeStepFromGroup(groupId, stepId)
}

function handleReorderGroups(fromIndex: number, toIndex: number) {
  reorderGroups(fromIndex, toIndex)
}

function handleReorderStep(groupId: string, fromIndex: number, toIndex: number) {
  reorderStepsInGroup(groupId, fromIndex, toIndex)
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
          label="Add Group"
          icon="pi pi-folder-plus"
          severity="secondary"
          data-testid="add-group-btn"
          @click="handleAddGroup"
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
      v-else-if="groups.length === 0 || steps.length === 0"
      severity="info"
      :closable="false"
      data-testid="empty-message"
    >
      <span>No pipeline steps configured.</span>
      <Button
        v-if="isAdmin"
        label="Add your first group"
        text
        size="small"
        class="ml-2"
        @click="handleAddGroup"
      />
    </Message>

    <!-- Group list -->
    <PipelineStepList
      v-else
      :groups="groups"
      :is-admin="isAdmin"
      @rename-group="handleRenameGroup"
      @remove-group="handleRemoveGroup"
      @add-step="handleOpenAddStep"
      @update-step="handleUpdateStep"
      @remove-step="handleRemoveStep"
      @reorder-groups="handleReorderGroups"
      @reorder-step="handleReorderStep"
    />

    <!-- Add step dialog (admin only, scoped to target group) -->
    <AddStepDialog
      v-if="isAdmin"
      v-model:visible="showAddDialog"
      @add="handleAddStep"
    />

    <ConfirmDialog />
    <Toast />
  </div>
</template>
