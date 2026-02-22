<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Tag from 'primevue/tag'
import ProgressBar from 'primevue/progressbar'
import Timeline from 'primevue/timeline'
import Button from 'primevue/button'
import { useToast } from 'primevue/usetoast'
import { differenceInSeconds } from 'date-fns'
import { useRunDetail } from '@/features/runs/composables/useRunDetail'
import { useRunCosts } from '@/features/runs/composables/useRunCosts'
import type { StepCostBreakdown } from '@/features/runs/composables/useRunCosts'
import { useRunsStore } from '@/stores/runs'
import RunLogViewer from '@/features/runs/RunLogViewer.vue'
import { runStatusSeverity } from '@/utils/runStatus'
import { formatCostUSD, formatTokenCount } from '@/utils/formatCost'
import type { RunStep } from '@/features/runs/composables/useRunDetail'

const route = useRoute()
const runId = computed(() => route.params.id as string)
const projectId = computed(() => route.query.projectId as string ?? '')

const runsStore = useRunsStore()
const toast = useToast()

const { run: runRef, isLoading, error, retry } = useRunDetail(runId.value, projectId.value)
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

const stepSeverity: Record<string, string> = {
  completed: 'success',
  running: 'info',
  failed: 'danger',
  pending: 'secondary',
  cancelled: 'warn',
}

const canPause = computed(() => run.value?.status === 'running')
const canResume = computed(() => run.value?.status === 'paused')

const pauseError = ref<string | null>(null)

async function handlePause() {
  if (!projectId.value || !runId.value) return
  pauseError.value = null
  try {
    await runsStore.pauseRun(projectId.value, runId.value)
    toast.add({ severity: 'success', summary: 'Run paused', life: 3000 })
  } catch (err) {
    pauseError.value = err instanceof Error ? err.message : 'Failed to pause run'
    toast.add({ severity: 'error', summary: 'Failed to pause run', detail: pauseError.value, life: 5000 })
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
    toast.add({ severity: 'error', summary: 'Failed to resume run', detail: pauseError.value, life: 5000 })
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

/** Format step duration from started_at to completed_at. */
function formatDuration(step: RunStep): string | null {
  if (!step.started_at) return null
  const end = step.completed_at ? new Date(step.completed_at) : new Date()
  const seconds = differenceInSeconds(end, new Date(step.started_at))
  if (seconds < 60) return `${seconds}s`
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}m ${secs}s`
}
</script>

<template>
  <div class="flex flex-col h-full p-6 gap-6">
    <!-- Page Header — always visible -->
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

      <!-- Step Timeline -->
      <div v-if="run.steps.length > 0">
        <h2 class="text-lg font-semibold mb-3">Steps</h2>
        <Timeline :value="run.steps" align="left" class="w-full">
          <template #marker="{ item }">
            <Tag
              :severity="stepSeverity[(item as RunStep).status] ?? 'secondary'"
              class="w-8 h-8 flex items-center justify-center rounded-full"
            >
              <i
                :class="{
                  'pi pi-check': (item as RunStep).status === 'completed',
                  'pi pi-spin pi-spinner': (item as RunStep).status === 'running',
                  'pi pi-times': (item as RunStep).status === 'failed',
                  'pi pi-clock': (item as RunStep).status === 'pending',
                  'pi pi-ban': (item as RunStep).status === 'cancelled',
                }"
                class="text-xs"
              />
            </Tag>
          </template>
          <template #content="{ item }">
            <div class="flex flex-col gap-1 mb-4">
              <div class="flex items-center gap-2">
                <span class="font-medium">{{ (item as RunStep).step_name }}</span>
                <Tag
                  :value="(item as RunStep).status"
                  :severity="stepSeverity[(item as RunStep).status] ?? 'secondary'"
                  class="text-xs"
                />
              </div>
              <div class="text-sm text-surface-500 flex items-center gap-3">
                <span>{{ (item as RunStep).action }}</span>
                <span v-if="formatDuration(item as RunStep)" class="text-surface-400">
                  {{ formatDuration(item as RunStep) }}
                </span>
              </div>
              <!-- Per-step cost breakdown -->
              <div
                v-if="stepCostMap.get((item as RunStep).id)"
                class="flex items-center gap-3 text-xs text-surface-500 mt-1"
                data-testid="step-cost"
              >
                <span class="font-mono bg-surface-100 dark:bg-surface-800 px-1.5 py-0.5 rounded">
                  {{ stepCostMap.get((item as RunStep).id)!.model }}
                </span>
                <span>
                  {{ formatTokenCount(stepCostMap.get((item as RunStep).id)!.tokens_input) }} in
                  /
                  {{ formatTokenCount(stepCostMap.get((item as RunStep).id)!.tokens_output) }} out
                </span>
                <span class="font-medium text-surface-700 dark:text-surface-300">
                  {{ formatCostUSD(stepCostMap.get((item as RunStep).id)!.cost_usd) }}
                </span>
              </div>
              <div
                v-if="(item as RunStep).error_message"
                class="text-sm text-red-500 mt-1"
              >
                {{ (item as RunStep).error_message }}
              </div>
            </div>
          </template>
        </Timeline>
      </div>

      <!-- Live Log Viewer -->
      <div>
        <h2 class="text-lg font-semibold mb-3">Live Logs</h2>
        <RunLogViewer :project-id="run.project_id" :run-id="run.id" />
      </div>
    </template>
  </div>
</template>
