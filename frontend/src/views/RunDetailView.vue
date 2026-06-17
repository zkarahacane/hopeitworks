<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import { useToast } from 'primevue/usetoast'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import LiveProgress from '@/ui/primitives/LiveProgress.vue'
import StepTimeline from '@/ui/composed/StepTimeline.vue'
import HitlGateCard from '@/ui/composed/HitlGateCard.vue'
import LogStreamPanel from '@/ui/composed/LogStreamPanel.vue'
import { useRunDetail } from '@/features/runs/composables/useRunDetail'
import { useRunCosts } from '@/features/runs/composables/useRunCosts'
import { useRunCostByRole } from '@/features/runs/composables/useRunCostByRole'
import { useRunHitl } from '@/features/runs/composables/useRunHitl'
import { useStepLogs } from '@/features/runs/composables/useStepLogs'
import { useRunsStore } from '@/stores/runs'
import { useRuntimeStream } from '@/stores/runtimeStream'
import { useSSE } from '@/composables/useSSE'
import { useCountUp } from '@/composables/useCountUp'
import RunPipelineView from '@/features/runs/RunPipelineView.vue'
import RunCancelConfirmDialog from '@/features/runs/RunCancelConfirmDialog.vue'
import RunStepLogPanel from '@/features/runs/RunStepLogPanel.vue'
import RunCostByRole from '@/features/runs/RunCostByRole.vue'
import { formatDuration } from '@/utils/formatDuration'
import type { RunStep, RunWithSteps } from '@/features/runs/composables/useRunDetail'
import type { TimelineStep } from '@/ui/composed/StepTimeline.vue'

const route = useRoute()
const runId = computed(() => route.params.id as string)
const projectId = computed(() => (route.query.projectId as string) ?? '')

const runsStore = useRunsStore()
const toast = useToast()
const stream = useRuntimeStream()

const { run: runRef, isLoading, error, retry, fetchRun } = useRunDetail(runId.value, projectId.value)
const run = computed<RunWithSteps | null>(() => runRef.value)
const runStatus = computed(() => run.value?.status)

// ── Live runtime: wire SSE → runtimeStream + advance the elapsed clock ──────────
let clockTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  if (projectId.value) {
    useSSE(projectId.value, (name, data) => stream.ingest(name, data))
  }
  stream.tick()
  clockTimer = setInterval(() => stream.tick(), 1000)
})
onBeforeUnmount(() => {
  if (clockTimer) clearInterval(clockTimer)
})

// ── ELAPSED (mono, live count-up) ───────────────────────────────────────────────
// Prefer the live runtime signal; fall back to REST started/completed timestamps
// so a freshly-loaded page (no SSE event yet) still shows a real elapsed time.
const elapsedSeconds = computed(() => {
  const live = stream.runElapsedSeconds(runId.value)
  if (live > 0) return live
  const startedAt = run.value?.started_at
  if (!startedAt) return 0
  const start = new Date(startedAt).getTime()
  const end = run.value?.completed_at
    ? new Date(run.value.completed_at).getTime()
    : stream.clock
  return Math.max(0, Math.floor((end - start) / 1000))
})
const { current: elapsedAnimated } = useCountUp(elapsedSeconds, { durationMs: 400 })
const elapsedLabel = computed(() => {
  const total = Math.floor(elapsedAnimated.value)
  const m = Math.floor(total / 60)
    .toString()
    .padStart(2, '0')
  const s = (total % 60).toString().padStart(2, '0')
  return `${m}:${s}`
})

// ── Cost (run-level rollup #3 + by-role #6 best-effort) ─────────────────────────
const { costDetail, isLoading: isCostLoading } = useRunCosts(
  projectId.value,
  runId.value,
  runStatus,
)
// `costDetail` is a plain `{ value }` ref from useAsyncAction — wrap in a real
// computed so the by-role composable receives a proper reactive Ref.
const costDetailRef = computed(() => costDetail.value)
const { breakdown } = useRunCostByRole(costDetailRef)

// ── HITL gate wiring (HitlGateCard emits → real approve/reject flow) ────────────
const { hitlRequest, gateStep, isAtGate, busy, pendingAction, actionError, approve, reject, requestChanges } =
  useRunHitl(run, () => {
    // After a decision resolves, refetch the run so its status reflects the gate.
    void fetchRun()
  })

const showGate = computed(() => isAtGate.value && !!gateStep.value)

async function handleApprove() {
  try {
    await approve()
    toast.add({ severity: 'success', summary: 'Approved — merging', life: 3000 })
  } catch {
    toast.add({ severity: 'error', summary: 'Approve failed', detail: actionError.value ?? '', life: 5000 })
  }
}
async function handleRequestChanges() {
  try {
    await requestChanges('Changes requested from Run Detail gate')
    toast.add({ severity: 'success', summary: 'Changes requested', life: 3000 })
  } catch {
    toast.add({ severity: 'error', summary: 'Request changes failed', detail: actionError.value ?? '', life: 5000 })
  }
}
async function handleReject() {
  try {
    await reject('Rejected from Run Detail gate')
    toast.add({ severity: 'success', summary: 'Rejected', life: 3000 })
  } catch {
    toast.add({ severity: 'error', summary: 'Reject failed', detail: actionError.value ?? '', life: 5000 })
  }
}

// ── Phase timeline steps (StepTimeline) ─────────────────────────────────────────
const timelineSteps = computed<TimelineStep[]>(() =>
  (run.value?.steps ?? []).map((s) => ({
    id: s.id,
    name: s.step_name,
    status: s.status,
    actionType: s.action,
    duration: formatDuration(s.started_at, s.completed_at),
  })),
)

// ── STREAM panel: stream logs for the active (running) step, else selected ──────
const selectedStep = ref<RunStep | null>(null)
const stepPanelVisible = ref(false)

const activeStep = computed<RunStep | null>(() => {
  const steps = run.value?.steps ?? []
  return steps.find((s) => s.status === 'running') ?? selectedStep.value ?? null
})
const streamStepId = computed(() => activeStep.value?.id ?? null)
const { lines: streamLines, sseStatus: streamStatus } = useStepLogs(
  projectId.value,
  runId.value,
  streamStepId,
)
const streamActive = computed(
  () => activeStep.value?.status === 'running' || streamLines.value.length > 0,
)
const streamContainerLabel = computed(() => {
  const id = activeStep.value?.container_id
  if (!id) return null
  return `ctr·${id.slice(-4)}`
})

function handleTimelineSelect(stepId: string) {
  const step = run.value?.steps.find((s) => s.id === stepId) ?? null
  if (step) handleStepSelected(step)
}

function handleStepSelected(step: RunStep) {
  selectedStep.value = step
  stepPanelVisible.value = true
}
function handleCloseStepPanel() {
  stepPanelVisible.value = false
}

// ── Run lifecycle actions ───────────────────────────────────────────────────────
const canPause = computed(() => run.value?.status === 'running')
const canResume = computed(() => run.value?.status === 'paused')
const canCancel = computed(() => {
  const status = run.value?.status
  return status === 'pending' || status === 'running' || status === 'paused'
})
const cancelDialogVisible = ref(false)

async function handlePause() {
  if (!projectId.value || !runId.value) return
  try {
    await runsStore.pauseRun(projectId.value, runId.value)
    toast.add({ severity: 'success', summary: 'Run paused', life: 3000 })
  } catch (err) {
    const detail = err instanceof Error ? err.message : 'Failed to pause run'
    toast.add({ severity: 'error', summary: 'Failed to pause run', detail, life: 5000 })
  }
}
async function handleResume() {
  if (!projectId.value || !runId.value) return
  try {
    await runsStore.resumeRun(projectId.value, runId.value)
    toast.add({ severity: 'success', summary: 'Run resumed', life: 3000 })
  } catch (err) {
    const detail = err instanceof Error ? err.message : 'Failed to resume run'
    toast.add({ severity: 'error', summary: 'Failed to resume run', detail, life: 5000 })
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

/** Progress (0-100) from completed-steps ratio (determinate bar). */
const progress = computed(() => {
  if (!run.value || run.value.steps.length === 0) return 0
  if (run.value.progress !== undefined) return run.value.progress
  const done = run.value.steps.filter(
    (s) => s.status === 'completed' || s.status === 'failed' || s.status === 'cancelled',
  ).length
  return Math.round((done / run.value.steps.length) * 100)
})

/** Short run id `run·<8 hex>` (machine voice). */
const shortRunId = computed(() => {
  const id = run.value?.id ?? runId.value
  return `run·${id.slice(0, 8)}`
})
const storyKey = computed(() => hitlRequest.value?.story_key ?? null)
const runTitle = computed(() => hitlRequest.value?.story_title ?? null)

// Reset the selected step if the run reloads to a different run.
watch(runId, () => {
  selectedStep.value = null
  stepPanelVisible.value = false
})
</script>

<template>
  <div class="flex flex-col h-full p-6 gap-5">
    <!-- ── Header ───────────────────────────────────────────────────────────── -->
    <div class="flex flex-col gap-2">
      <!-- Breadcrumb (mono machine voice for ids). -->
      <nav
        class="flex items-center gap-2 flex-wrap"
        :style="{ fontSize: '0.78rem', color: 'var(--p-text-muted-color)' }"
        data-testid="run-breadcrumb"
      >
        <span>Runs</span>
        <i class="pi pi-angle-right" :style="{ fontSize: '0.65rem' }" aria-hidden="true" />
        <span class="font-mono">{{ shortRunId }}</span>
      </nav>

      <div class="flex items-start justify-between gap-4 flex-wrap">
        <div class="flex flex-col gap-1">
          <div class="flex items-center gap-3 flex-wrap">
            <span
              v-if="storyKey"
              class="font-mono"
              :style="{ fontSize: '0.82rem', fontWeight: 600 }"
              data-testid="run-story-key"
            >
              {{ storyKey }}
            </span>
            <span
              class="font-mono"
              :style="{ fontSize: '0.82rem', color: 'var(--p-text-muted-color)' }"
              data-testid="run-id-mono"
            >
              {{ shortRunId }}
            </span>
            <!-- Single derived status — no badge+spinner contradiction (#2). -->
            <StatusBadge v-if="run" :status="run.status" data-testid="run-status-badge" />
          </div>
          <h1 v-if="runTitle" class="text-xl font-bold m-0" data-testid="run-title">
            {{ runTitle }}
          </h1>
          <h1 v-else class="text-xl font-bold m-0">Run Detail</h1>
        </div>

        <div class="flex items-center gap-4">
          <!-- ELAPSED (mono, live count-up). -->
          <div class="flex flex-col items-end" data-testid="run-elapsed">
            <span
              :style="{ fontSize: '0.62rem', letterSpacing: '0.06em', color: 'var(--p-text-muted-color)' }"
            >
              ELAPSED
            </span>
            <span class="font-mono" :style="{ fontSize: '1.1rem', fontWeight: 600 }" data-testid="run-elapsed-value">
              {{ elapsedLabel }}
            </span>
          </div>

          <div class="flex items-center gap-2">
            <Button
              v-if="canPause"
              label="Pause"
              icon="pi pi-pause"
              severity="warn"
              size="small"
              :loading="runsStore.isPausing"
              data-testid="pause-run-btn"
              @click="handlePause"
            />
            <Button
              v-if="canResume"
              label="Resume"
              icon="pi pi-play"
              severity="success"
              size="small"
              :loading="runsStore.isResuming"
              data-testid="resume-run-btn"
              @click="handleResume"
            />
            <Button
              v-if="canCancel"
              label="Cancel"
              icon="pi pi-times"
              severity="danger"
              size="small"
              outlined
              data-testid="cancel-run-btn"
              @click="cancelDialogVisible = true"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- ── Loading ──────────────────────────────────────────────────────────── -->
    <div v-if="isLoading" class="flex flex-col gap-4">
      <Skeleton width="20rem" height="2rem" />
      <Skeleton width="100%" height="1rem" />
      <Skeleton width="100%" height="12rem" />
    </div>

    <!-- ── Error ────────────────────────────────────────────────────────────── -->
    <Message v-else-if="error" severity="error" :closable="false">
      {{ error.message }}
      <Button label="Retry" icon="pi pi-refresh" text size="small" class="ml-2" @click="retry" />
    </Message>

    <!-- ── Run body ─────────────────────────────────────────────────────────── -->
    <template v-else-if="run">
      <!-- Progress -->
      <LiveProgress :value="progress" :show-value="true" />

      <!-- Phase timeline (Setup ── Development ── Review & Merge). -->
      <div data-testid="run-phase-timeline">
        <StepTimeline
          :steps="timelineSteps"
          :selected-id="activeStep?.id ?? null"
          @select="handleTimelineSelect"
        />
      </div>

      <!-- Amber gate card — wired to the real HITL approve/reject flow. -->
      <HitlGateCard
        v-if="showGate"
        :story-key="storyKey"
        :step-name="gateStep?.step_name ?? null"
        :pending-since="hitlRequest?.created_at ?? gateStep?.started_at ?? null"
        :busy="busy"
        :pending-action="pendingAction"
        data-testid="run-hitl-gate"
        @approve="handleApprove"
        @request-changes="handleRequestChanges"
        @reject="handleReject"
      />

      <!-- Two-column body: steps (left) + cost-by-role & stream (right). -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-5 flex-1 min-h-0">
        <!-- STEPS (typed list) -->
        <div class="lg:col-span-2 flex flex-col gap-3 min-h-0">
          <h2
            :style="{ fontSize: '0.78rem', fontWeight: 600, letterSpacing: '0.04em', color: 'var(--p-text-muted-color)' }"
          >
            STEPS
          </h2>
          <RunPipelineView
            :run="run"
            :steps="run.steps"
            data-testid="run-pipeline-view"
            @step-selected="handleStepSelected"
          />
        </div>

        <!-- Right rail: COST BY ROLE + STREAM -->
        <div class="flex flex-col gap-5 min-h-0">
          <RunCostByRole :breakdown="breakdown" :loading="isCostLoading" />

          <div class="flex flex-col gap-2 min-h-0">
            <div class="flex items-center justify-between">
              <h2
                :style="{ fontSize: '0.78rem', fontWeight: 600, letterSpacing: '0.04em', color: 'var(--p-text-muted-color)' }"
              >
                STREAM
              </h2>
              <span
                v-if="streamContainerLabel"
                class="font-mono"
                :style="{ fontSize: '0.72rem', color: 'var(--p-text-muted-color)' }"
                data-testid="stream-container"
              >
                {{ streamContainerLabel }}
              </span>
            </div>
            <LogStreamPanel
              :lines="streamLines"
              :status="streamStatus"
              :active="streamActive"
            />
          </div>
        </div>
      </div>
    </template>

    <!-- Cancel confirmation dialog -->
    <RunCancelConfirmDialog
      :visible="cancelDialogVisible"
      :loading="runsStore.isCancelling"
      @confirm="handleCancelConfirm"
      @cancel="cancelDialogVisible = false"
      @update:visible="cancelDialogVisible = $event"
    />

    <!-- Step log drawer (deep-dive on a selected step) -->
    <RunStepLogPanel
      v-if="run"
      :step="selectedStep"
      :run-id="run.id"
      :project-id="run.project_id"
      :visible="stepPanelVisible"
      :retry-loading="runsStore.isRetrying"
      @close="handleCloseStepPanel"
      @update:visible="stepPanelVisible = $event"
      @retry="handleRetryStep"
    />
  </div>
</template>
