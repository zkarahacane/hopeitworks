<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Tag from 'primevue/tag'
import ProgressBar from 'primevue/progressbar'
import Button from 'primevue/button'
import { useToast } from 'primevue/usetoast'
import { useRunDetail } from '@/features/runs/composables/useRunDetail'
import { useRunCosts } from '@/features/runs/composables/useRunCosts'
import type { StepCostBreakdown } from '@/features/runs/composables/useRunCosts'
import { useRunsStore } from '@/stores/runs'
import RunLogViewer from '@/features/runs/RunLogViewer.vue'
import RunTimeline from '@/features/runs/RunTimeline.vue'
import RunCancelConfirmDialog from '@/features/runs/RunCancelConfirmDialog.vue'
import { runStatusSeverity } from '@/utils/runStatus'
import { formatCostUSD, formatTokenCount } from '@/utils/formatCost'
import type { RunStep } from '@/features/runs/composables/useRunDetail'

const route = useRoute()
const runId = computed(() => route.params.id as string)
const projectId = computed(() => (route.query.projectId as string) ?? '')

const runsStore = useRunsStore()
const toast = useToast()

const { run: runRef, isLoading, error, retry, fetchRun } = useRunDetail(runId.value, projectId.value)
const run = computed(() => runRef.value)

const runStatus = computed(() => run.value?.status)
const { costDetail, isLoading: isCostLoading } = useRunCosts(
  projectId.value,
  runId.value,
  runStatus,
)

/** Map of step_id → StepCostBreakdown for quick lookups in the timeline. */
const stepCostMap = computed(() => {
  const map = new Map<string, StepCostBreakdown>()
  if (!costDetail.value?.steps) return map
  for (const step of costDetail.value.steps) {
    map.set(step.step_id, step)
  }
  return map
})

/** Total run cost formatted for display. */
const totalCostDisplay = computed(() => {
  if (!costDetail.value) return null
  return formatCostUSD(costDetail.value.total_cost)
})

/** Helper to get step cost breakdown by step ID. */
function getStepCost(stepId: string): StepCostBreakdown | undefined {
  return stepCostMap.value.get(stepId)
}

const stepSeverity: Record<string, string> = {
  completed: 'success',
  running: 'info',
  failed: 'danger',
  pending: 'secondary',
  cancelled: 'warn',
}


const canPause = computed(() => run.value?.status === 'running')
const canResume = computed(() => run.value?.status === 'paused')
const canCancel = computed(() => {
  const status = run.value?.status
  return status === 'pending' || status === 'running' || status === 'paused'
})

const pauseError = ref<string | null>(null)
const cancelDialogVisible = ref(false)

async function handlePause() {
  if (!projectId.value || !runId.value) return
  pauseError.value = null
  try {
    await runsStore.pauseRun(projectId.value, runId.value)
    toast.add({ severity: 'success', summary: 'Run paused', life: 3000 })
  } catch (err) {
    pauseError.value = err instanceof Error ? err.message : 'Failed to pause run'
    toast.add({
      severity: 'error',
      summary: 'Failed to pause run',
      detail: pauseError.value,
      life: 5000,
    })
  }
}

async function handleResume() {
  if (!projectId.value || !runId.value) return
  pauseError.value = null
  try {
    await runsStore.resumeRun(projectId.value, runId.value)
    toast.add({ severity: 'success', summary: 'Run resumed', life: 3000 })
  } catch (err) {
    pauseError.value = err instanceof Error ? err.message : 'Failed to resume run'
    toast.add({
      severity: 'error',
      summary: 'Failed to resume run',
      detail: pauseError.value,
      life: 5000,
    })
  }
}

async function handleRetryStep(stepId: string) {
  if (!runId.value) return
  try {
    await runsStore.retryStep(runId.value, stepId)
    toast.add({ severity: 'success', summary: 'Retry initiated', life: 3000 })
    await fetchRun()
  } catch (err) {
    const detail = err instanceof Error ? err.message : 'Failed to retry step'
    toast.add({ severity: 'error', summary: 'Failed to retry step', detail, life: 5000 })
  }
}

async function handleCancelConfirm() {
  if (!projectId.value || !runId.value) return
  try {
    await runsStore.cancelRun(projectId.value, runId.value)
    cancelDialogVisible.value = false
    toast.add({ severity: 'success', summary: 'Run cancelled', life: 3000 })
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Failed to cancel run'
    toast.add({ severity: 'error', summary: 'Failed to cancel run', detail: msg, life: 5000 })
  }
}

/** Compute progress from completed steps ratio (0-100). */
const progress = computed(() => {
  if (!run.value || run.value.steps.length === 0) return 0
  if (run.value.progress !== undefined) return run.value.progress
  const completed = run.value.steps.filter(
    (s) => s.status === 'completed' || s.status === 'failed' || s.status === 'cancelled',
  ).length
  return Math.round((completed / run.value.steps.length) * 100)
})
</script>

<template>
  <div class="flex flex-col h-full p-6 gap-6">
    <!-- Page Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <h1 class="text-xl font-bold">Run Detail</h1>
        <code
          v-if="run"
          class="text-sm bg-surface-100 dark:bg-surface-800 px-2 py-1 rounded font-mono"
        >
          {{ run.id }}
        </code>
      </div>
      <div class="flex items-center gap-3">
        <Tag v-if="run" :value="run.status" :severity="runStatusSeverity[run.status]" />
        <span
          v-if="totalCostDisplay && !isCostLoading"
          class="text-sm font-medium text-surface-600 bg-surface-100 dark:bg-surface-800 px-2 py-1 rounded"
          data-testid="run-total-cost"
        >
          {{ totalCostDisplay }}
        </span>
        <Button
          v-if="canPause"
          label="Pause"
          icon="pi pi-pause"
          severity="warn"
          :loading="runsStore.isPausing"
          data-testid="pause-run-btn"
          @click="handlePause"
        />
        <Button
          v-if="canResume"
          label="Resume"
          icon="pi pi-play"
          severity="success"
          :loading="runsStore.isResuming"
          data-testid="resume-run-btn"
          @click="handleResume"
        />
        <Button
          v-if="canCancel"
          label="Cancel"
          icon="pi pi-times"
          severity="danger"
          outlined
          data-testid="cancel-run-btn"
          @click="cancelDialogVisible = true"
        />
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex flex-col gap-4">
      <Skeleton width="20rem" height="2rem" />
      <Skeleton width="100%" height="1rem" />
      <Skeleton width="100%" height="12rem" />
    </div>

    <!-- Error state -->
    <Message v-else-if="error" severity="error" :closable="false">
      {{ error.message }}
      <Button label="Retry" icon="pi pi-refresh" text size="small" class="ml-2" @click="retry" />
    </Message>

    <!-- Run data -->
    <template v-else-if="run">
      <!-- Progress Bar -->
      <ProgressBar :value="progress" :show-value="true" />

      <!-- Step Timeline with retry support -->
      <div v-if="run.steps.length > 0">
        <h2 class="text-lg font-semibold mb-3">Steps</h2>
        <RunTimeline
          :steps="run.steps"
          :retry-loading="runsStore.isRetrying"
          @retry-step="handleRetryStep"
        />
      </div>

      <!-- Live Log Viewer -->
      <div>
        <h2 class="text-lg font-semibold mb-3">Live Logs</h2>
        <RunLogViewer :project-id="run.project_id" :run-id="run.id" />
      </div>
    </template>

    <!-- Cancel Confirmation Dialog -->
    <RunCancelConfirmDialog
      :visible="cancelDialogVisible"
      :loading="runsStore.isCancelling"
      @confirm="handleCancelConfirm"
      @cancel="cancelDialogVisible = false"
      @update:visible="cancelDialogVisible = $event"
    />
  </div>
</template>
