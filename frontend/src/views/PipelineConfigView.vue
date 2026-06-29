<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Button from 'primevue/button'
import ConfirmDialog from 'primevue/confirmdialog'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import Toast from 'primevue/toast'
import PipelineStepList from '@/features/pipeline/PipelineStepList.vue'
import AddStepDialog from '@/features/pipeline/AddStepDialog.vue'
import PipelineStepPalette from '@/features/pipeline/PipelineStepPalette.vue'
import { usePipelineConfig } from '@/composables/usePipelineConfig'
import { useAuth } from '@/composables/useAuth'
import { useAgents } from '@/composables/useAgents'
import type { PipelineStep, Guard, TransitionPolicy } from '@/stores/pipelineConfig'

const route = useRoute()
const router = useRouter()
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
  updateGroupTransition,
  addGuard,
  removeGuard,
  updateGuard,
} = usePipelineConfig(projectId)

const { agents, fetchAgents } = useAgents(projectId.value)

onMounted(() => {
  fetchAgents({ per_page: 100 })
})

const isAdmin = computed(() => user.value?.role === 'admin')
const showAddDialog = ref(false)
const addStepTargetGroupId = ref<string | null>(null)
const paletteActionType = ref<string | null>(null)

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

function handleUpdateTransition(groupId: string, transition: TransitionPolicy) {
  updateGroupTransition(groupId, transition)
}

function handleAddGuard(groupId: string) {
  addGuard(groupId)
}

function handleRemoveGuard(groupId: string, guardIndex: number) {
  removeGuard(groupId, guardIndex)
}

function handleUpdateGuard(groupId: string, guardIndex: number, guard: Guard) {
  updateGuard(groupId, guardIndex, guard)
}

function handlePaletteAddStep(actionType: string) {
  paletteActionType.value = actionType
  const firstGroupId = groups.value[0]?.id ?? null
  addStepTargetGroupId.value = firstGroupId
  showAddDialog.value = true
}

watch(showAddDialog, (visible) => {
  if (!visible) {
    paletteActionType.value = null
  }
})

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
  <div class="flex flex-col gap-4 p-6">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex flex-col gap-1">
        <div class="flex items-center gap-3">
          <h1 class="text-2xl font-bold">Pipeline</h1>
          <span
            class="text-xs px-2 py-0.5 rounded-full"
            :style="{ backgroundColor: 'var(--surface-overlay)', border: '1px solid var(--surface-border)', color: 'var(--p-text-muted-color)' }"
          >
            opinionated on runtime · free on process
          </span>
          <span
            v-if="isDirty"
            class="text-xs"
            style="color: var(--status-gate-color)"
            data-testid="unsaved-indicator"
          >
            · unsaved changes
          </span>
        </div>
        <p class="text-sm" style="color: var(--p-text-muted-color)">
          Compose roles, steps and gates. The runtime handles containers, isolation &amp; parallelism.
        </p>
      </div>
      <div class="flex items-center gap-2 flex-wrap justify-end">
        <Button
          label="Tracker & sync"
          icon="pi pi-link"
          severity="secondary"
          text
          size="small"
          data-testid="tracker-sync-link"
          @click="router.push({ name: 'project-settings', params: { id: projectId } })"
        />
        <template v-if="isAdmin && !isLoading && !error">
          <Button
            label="+ Add group"
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
        </template>
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

    <!-- Two-column layout -->
    <div v-else class="flex gap-6 items-start">
      <!-- Left: pipeline groups (~70%) -->
      <div class="flex-1 min-w-0 flex flex-col gap-3">
        <!-- Empty state inline -->
        <Message
          v-if="groups.length === 0 || steps.length === 0"
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
        <PipelineStepList
          v-else
          :groups="groups"
          :is-admin="isAdmin"
          :agents="agents"
          @rename-group="handleRenameGroup"
          @remove-group="handleRemoveGroup"
          @add-step="handleOpenAddStep"
          @update-step="handleUpdateStep"
          @remove-step="handleRemoveStep"
          @reorder-groups="handleReorderGroups"
          @reorder-step="handleReorderStep"
          @update-transition="handleUpdateTransition"
          @add-guard="handleAddGuard"
          @remove-guard="handleRemoveGuard"
          @update-guard="handleUpdateGuard"
        />
      </div>

      <!-- Right rail: palette (~30%) -->
      <div v-if="isAdmin" class="w-72 shrink-0">
        <PipelineStepPalette
          :agents="agents"
          @add-step="handlePaletteAddStep"
        />
      </div>
    </div>

    <!-- Add step dialog -->
    <AddStepDialog
      v-if="isAdmin"
      v-model:visible="showAddDialog"
      :agents="agents"
      :initial-action-type="paletteActionType ?? undefined"
      @add="handleAddStep"
    />

    <ConfirmDialog />
    <Toast />
  </div>
</template>
