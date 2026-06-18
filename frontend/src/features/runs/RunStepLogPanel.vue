<script setup lang="ts">
import { computed } from 'vue'
import Drawer from 'primevue/drawer'
import Button from 'primevue/button'
import Message from 'primevue/message'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import AgentChip from '@/ui/primitives/AgentChip.vue'
import ContainerChip from '@/ui/primitives/ContainerChip.vue'
import LogStreamPanel from '@/ui/composed/LogStreamPanel.vue'
import { useStepLogs } from './composables/useStepLogs'
import { stepTypeMeta } from '@/utils/stepType'
import { format, parseISO, differenceInSeconds } from 'date-fns'
import type { components } from '@/api/schema'

type RunStep = components['schemas']['RunStep']

const props = defineProps<{
  step: RunStep | null
  runId: string
  projectId: string
  visible: boolean
  retryLoading?: boolean
}>()

const emit = defineEmits<{
  close: []
  'update:visible': [value: boolean]
  retry: [stepId: string]
}>()

const stepId = computed(() => props.step?.id ?? null)
const { lines, sseStatus, clearLogs } = useStepLogs(props.projectId, props.runId, stepId)

/** Typed-step metadata (type chip + agent flag) — single derived identity. */
const typeMeta = computed(() => stepTypeMeta(props.step?.action))

/**
 * Persisted log lines from `step.log_tail` (set by backend for completed steps).
 * Split on newlines, drop trailing empty line.
 * Timestamp is approximated from step completion time (no per-line timestamps in tail).
 */
const persistedLines = computed<import('@/ui/composed/LogViewer.vue').LogLine[]>(() => {
  const tail = props.step?.log_tail
  if (!tail) return []
  const ts = props.step?.completed_at ?? props.step?.started_at
  const timestamp = ts ? parseISO(ts) : new Date(0)
  return tail
    .split('\n')
    .filter((_, i, arr) => i < arr.length - 1 || arr[i] !== '')
    .map((text) => ({ text, timestamp }))
})

/**
 * Lines fed to LogStreamPanel: prefer live SSE lines once any have arrived,
 * fall back to persisted log_tail for completed/historical steps.
 */
const displayLines = computed(() =>
  lines.value.length > 0 ? lines.value : persistedLines.value,
)

/**
 * LogStreamPanel `active` — a stream is only expected for a selected, running
 * (or recently active) step. Completed/pending steps show "idle" instead of a
 * misleading "no output" (the U1 / #4 fix is carried by LogStreamPanel; we feed
 * it an honest `active` flag).
 */
const streamActive = computed(() => {
  if (!props.step) return false
  return props.step.status === 'running' || displayLines.value.length > 0
})

/** Format an ISO timestamp to HH:mm:ss. */
function formatTime(iso?: string | null): string {
  if (!iso) return ''
  try {
    return format(parseISO(iso), 'HH:mm:ss')
  } catch {
    return ''
  }
}

/** Compute human-readable duration between two ISO timestamps. */
const duration = computed(() => {
  if (!props.step?.started_at) return null
  const start = parseISO(props.step.started_at)
  const end = props.step.completed_at ? parseISO(props.step.completed_at) : new Date()
  const totalSeconds = differenceInSeconds(end, start)
  if (totalSeconds < 60) return `${totalSeconds}s`
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes < 60) return `${minutes}m ${seconds}s`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}h ${remainingMinutes}m`
})

const isRunning = computed(() => props.step?.status === 'running')
const canRetry = computed(() => props.step?.status === 'failed')

function handleVisibleUpdate(value: boolean) {
  emit('update:visible', value)
  if (!value) {
    emit('close')
  }
}
</script>

<template>
  <Drawer
    :visible="visible"
    position="right"
    :modal="true"
    :dismissable="true"
    :close-on-escape="true"
    :pt="{
      root: 'w-full md:w-1/2',
      content: 'flex flex-col flex-1 overflow-hidden p-0',
    }"
    data-testid="step-log-panel"
    @update:visible="handleVisibleUpdate"
  >
    <template #header>
      <div v-if="step" class="flex flex-col gap-2 w-full">
        <div class="flex items-center gap-3 flex-wrap">
          <h3 class="text-lg font-semibold m-0" data-testid="step-name">
            {{ step.step_name }}
          </h3>
          <!-- Single derived status — no badge+spinner contradiction (#2). -->
          <StatusBadge :status="step.status" data-testid="step-status" />
          <Button
            v-if="canRetry"
            label="Retry"
            icon="pi pi-refresh"
            severity="warn"
            size="small"
            :loading="retryLoading"
            data-testid="retry-step-btn"
            @click="emit('retry', step.id)"
          />
        </div>
        <!-- Typed metadata: type chip + agent/container chips. -->
        <div class="flex items-center gap-2 flex-wrap">
          <span
            class="font-mono inline-flex items-center gap-1 px-2 py-0.5 rounded-md"
            :style="{
              fontSize: '0.72rem',
              backgroundColor: 'var(--surface-overlay)',
              border: '1px solid var(--surface-border)',
              color: 'var(--p-text-muted-color)',
            }"
            data-testid="step-type-chip"
          >
            <i :class="typeMeta.icon" :style="{ fontSize: '0.7rem' }" aria-hidden="true" />
            {{ typeMeta.typeLabel }}
          </span>
          <AgentChip v-if="typeMeta.isAgent" :role="step.step_name" data-testid="step-agent-chip" />
          <ContainerChip
            v-if="step.container_id"
            :container-id="step.container_id"
            data-testid="step-container-chip"
          />
        </div>
        <div class="flex items-center gap-4 text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
          <span v-if="step.started_at" data-testid="step-started-at">
            Started: {{ formatTime(step.started_at) }}
          </span>
          <span v-if="step.completed_at" data-testid="step-completed-at">
            Completed: {{ formatTime(step.completed_at) }}
          </span>
          <span v-else-if="isRunning" data-testid="step-running-indicator"> Running... </span>
          <span v-if="duration" class="font-mono" data-testid="step-duration">
            Duration: {{ duration }}
          </span>
        </div>
      </div>
    </template>

    <div v-if="step" class="flex flex-col flex-1 overflow-hidden p-4 gap-4">
      <!-- Error message -->
      <Message
        v-if="step.error_message"
        severity="error"
        :closable="false"
        data-testid="step-error-message"
      >
        {{ step.error_message }}
      </Message>

      <!-- Live log stream (LogStreamPanel — fixes the U1/#4 lifecycle). -->
      <div class="flex-1 overflow-hidden">
        <LogStreamPanel
          :lines="displayLines"
          :status="sseStatus"
          :active="streamActive"
          @clear="clearLogs"
        />
      </div>
    </div>
  </Drawer>
</template>
