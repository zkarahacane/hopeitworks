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
import { useRunsStore } from '@/stores/runs'
import RunLogViewer from '@/features/runs/RunLogViewer.vue'
import RunTimeline from '@/features/runs/RunTimeline.vue'
import RunCancelConfirmDialog from '@/features/runs/RunCancelConfirmDialog.vue'
import RunStepLogPanel from '@/features/runs/RunStepLogPanel.vue'
import { runStatusSeverity } from '@/utils/runStatus'
import { formatCostUSD, formatTokenCount } from '@/utils/formatCost'
import type { components } from '@/api/schema'

type RunStep = components['schemas']['RunStep']

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

/** Total run cost formatted for display. */
const totalCostDisplay = computed(() => {
  if (!costDetail.value) return null
  return formatCostUSD(costDetail.value.total_cost)
})


const canPause = computed(() => run.value?.status === 'running')
const canResume = computed(() => run.value?.status === 'paused')
const canCancel = computed(() => {
  const status = run.value?.status
  return status === 'pending' || status === 'running' || status === 'paused'
})

const pauseError = ref<string | null>(null)
const cancelDialogVisible = ref(false)
const selectedStep = ref<RunStep | null>(null)
const stepPanelVisible = ref(false)

function handleSelectStep(step: RunStep) {
  selectedStep.value = step
  stepPanelVisible.value = true
}

function handleCloseStepPanel() {
  stepPanelVisible.value = false
}

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
          <template v-if="costDetail.value?.total_tokens_input !== undefined || costDetail.value?.total_tokens_output !== undefined">
            <span class="mx-1 text-surface-400">|</span>
            <span class="text-surface-500">In: {{ formatTokenCount(costDetail.value?.total_tokens_input ?? 0) }}</span>
            <span class="ml-1 text-surface-500">Out: {{ formatTokenCount(costDetail.value?.total_tokens_output ?? 0) }}</span>
          </template>
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
          @select-step="handleSelectStep"
        />
      </div>

      <!-- Step Cost Breakdown -->
      <div v-if="costDetail.value?.steps && costDetail.value.steps.length > 0">
        <h2 class="text-lg font-semibold mb-3">Step Costs</h2>
        <div class="rounded-lg border border-surface-200 bg-surface-0 overflow-hidden">
          <table class="w-full text-sm">
            <thead class="bg-surface-50 text-surface-600">
              <tr>
                <th class="px-4 py-2 text-left font-medium">Step</th>
                <th class="px-4 py-2 text-left font-medium">Model</th>
                <th class="px-4 py-2 text-right font-medium">Tokens In</th>
                <th class="px-4 py-2 text-right font-medium">Tokens Out</th>
                <th class="px-4 py-2 text-right font-medium">Cost</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-surface-100">
              <tr v-for="step in costDetail.value.steps" :key="step.step_id">
                <td class="px-4 py-2 font-medium">{{ step.step_name }}</td>
                <td class="px-4 py-2 text-surface-500">{{ step.model }}</td>
                <td class="px-4 py-2 text-right">{{ formatTokenCount(step.tokens_input) }}</td>
                <td class="px-4 py-2 text-right">{{ formatTokenCount(step.tokens_output) }}</td>
                <td class="px-4 py-2 text-right">{{ formatCostUSD(step.cost_usd) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
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

    <!-- Step Log Panel -->
    <RunStepLogPanel
      v-if="run"
      :step="selectedStep"
      :run-id="run.id"
      :project-id="run.project_id"
      :visible="stepPanelVisible"
      @close="handleCloseStepPanel"
      @update:visible="stepPanelVisible = $event"
    />
  </div>
</template>
