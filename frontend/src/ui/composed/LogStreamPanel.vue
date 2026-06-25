<script setup lang="ts">
import { ref, computed, watch, nextTick, useTemplateRef } from 'vue'
import Button from 'primevue/button'
import { formatLogLine } from '@/utils/formatLogLine'
import type { SSEStatus } from '@/composables/useSSE'
import type { LogLine } from '@/ui/composed/LogViewer.vue'

/**
 * LogStreamPanel — the redesigned, robust log stream.
 *
 * Replaces RunStepLogPanel's inner LogViewer usage and fixes the U1 lifecycle
 * bug ("Connecting…/No output" confusion): the displayed lifecycle is derived
 * from BOTH the connection status AND whether a stream is even expected
 * (`active`) and whether any lines have arrived. Clear states:
 *
 *   idle      — nothing selected at all → "No step selected" (RG3)
 *   connecting— stream active, opening, nothing yet
 *   streaming — open + receiving (blinking caret)
 *   waiting   — open but no lines yet (agent quiet) — NOT "no output"
 *   empty     — a non-terminal step IS selected but has no lines → "No logs
 *               available" (RG1, #297) — distinct from idle/no-selection
 *   closed    — terminal step ended with no captured output → "No output was
 *               captured", OR a finished SSE stream (shows captured lines)
 *   error     — connection error (RG4)
 *
 * `active` means "a step is targeted" (selection), NOT "the stream is live".
 * When the optional `stepStatus` is provided, the empty/closed distinction is
 * driven by the STEP status (terminal vs non-terminal) rather than the SSE
 * status, so a non-running selected step never shows a perpetual "Connecting…".
 *
 * Fully prop-driven (lines + status in) so Run Detail and the DAG inspector can
 * both mount it. ANSI rendered via the shared formatLogLine (ansi-to-html).
 */
/** Step status the panel may be targeting (subset relevant to the stream). */
type StepStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

const props = withDefaults(
  defineProps<{
    /** Captured log lines (ANSI allowed). */
    lines: LogLine[]
    /** SSE connection status from the host composable. */
    status: SSEStatus
    /**
     * Whether a step is targeted at all. False when nothing is selected — so
     * we show "idle" ("No step selected") instead of a misleading "no output".
     * Default true.
     */
    active?: boolean
    /**
     * Status of the targeted step, when known. Drives the empty/closed split:
     * a terminal step (completed/failed/cancelled) with no lines shows "No
     * output was captured" (closed); a non-terminal selected step with no lines
     * shows "No logs available" (empty, RG1). When omitted, the panel falls
     * back to deriving everything from the SSE status (legacy callers).
     */
    stepStatus?: StepStatus | null
  }>(),
  { active: true, stepStatus: null },
)

const emit = defineEmits<{
  clear: []
}>()

const scrollContainer = useTemplateRef<HTMLElement>('scrollContainer')
const autoScroll = ref(true)

const hasLines = computed(() => props.lines.length > 0)

const isTerminalStep = computed(
  () =>
    props.stepStatus === 'completed' ||
    props.stepStatus === 'failed' ||
    props.stepStatus === 'cancelled',
)
const isRunningStep = computed(() => props.stepStatus === 'running')

/** Derived lifecycle — the single source of truth for what the panel shows. */
const lifecycle = computed(() => {
  // RG3: nothing is targeted → neutral "No step selected".
  if (!props.active) return 'idle'

  // RG2: lines have arrived → live stream (or terminal recap when SSE closed).
  if (hasLines.value) {
    if (props.status === 'error') return 'error'
    if (props.status === 'closed') return 'closed'
    return 'streaming'
  }

  // No lines yet. A connection error always wins (RG4).
  if (props.status === 'error') return 'error'

  // When the step status is known, it (not the SSE phase) decides the empty
  // distinction — a non-running step must not show a perpetual "Connecting…".
  if (props.stepStatus !== null) {
    if (isTerminalStep.value) return 'closed' // Q1: terminal, no output captured
    if (isRunningStep.value) {
      return props.status === 'connecting' ? 'connecting' : 'waiting'
    }
    return 'empty' // RG1: selected (pending/seed) non-terminal step, no logs
  }

  // Legacy callers (no stepStatus): derive purely from the SSE status.
  switch (props.status) {
    case 'connecting':
      return 'connecting'
    case 'open':
      return 'waiting'
    case 'closed':
      return 'closed'
    default:
      return 'idle'
  }
})

const STATE_META: Record<
  string,
  { label: string; tone: 'running' | 'queued' | 'failed' | 'done' }
> = {
  idle: { label: 'No step selected', tone: 'queued' },
  connecting: { label: 'Connecting…', tone: 'queued' },
  waiting: { label: 'Connected — waiting for output…', tone: 'running' },
  streaming: { label: 'Live', tone: 'running' },
  empty: { label: 'No logs', tone: 'queued' },
  closed: { label: 'Stream ended', tone: 'done' },
  error: { label: 'Connection error', tone: 'failed' },
}

const meta = computed(() => STATE_META[lifecycle.value] ?? STATE_META.idle!)
const indicatorColor = computed(() => `var(--status-${meta.value.tone}-color)`)
const showCaret = computed(() => lifecycle.value === 'streaming')
const isLive = computed(() => lifecycle.value === 'streaming' || lifecycle.value === 'waiting')

/** Empty-body message — distinguishes "idle" / "waiting" / genuinely empty. */
const emptyMessage = computed(() => {
  switch (lifecycle.value) {
    case 'idle':
      return 'Select a step to stream its logs'
    case 'connecting':
      return 'Connecting to the log stream…'
    case 'waiting':
      return 'Connected. Waiting for the agent to emit output…'
    case 'empty':
      return 'No logs available'
    case 'error':
      return 'Could not connect to the log stream'
    case 'closed':
      return 'No output was captured'
    default:
      return ''
  }
})

watch(
  () => props.lines.length,
  async () => {
    if (!autoScroll.value) return
    await nextTick()
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
    }
  },
)

function onScroll() {
  if (!scrollContainer.value) return
  const { scrollTop, scrollHeight, clientHeight } = scrollContainer.value
  autoScroll.value = scrollHeight - scrollTop - clientHeight < 50
}
</script>

<template>
  <div
    class="flex flex-col rounded-lg overflow-hidden"
    :style="{ border: '1px solid var(--surface-border)' }"
    data-testid="log-stream-panel"
    :data-lifecycle="lifecycle"
  >
    <!-- Status bar -->
    <div
      class="flex items-center justify-between px-3 py-2"
      :style="{ backgroundColor: 'var(--surface-overlay)', borderBottom: '1px solid var(--surface-border)' }"
    >
      <div class="flex items-center gap-2">
        <span
          class="inline-block rounded-full"
          :class="{ 'live-pulse': isLive }"
          :style="{ width: '0.55rem', height: '0.55rem', backgroundColor: indicatorColor }"
          aria-hidden="true"
        />
        <span
          :style="{ fontSize: '0.78rem', color: 'var(--p-text-color)', fontWeight: 500 }"
          data-testid="log-stream-status"
        >
          {{ meta.label }}
        </span>
      </div>
      <Button
        v-if="hasLines"
        label="Clear"
        icon="pi pi-trash"
        severity="secondary"
        text
        size="small"
        @click="emit('clear')"
      />
    </div>

    <!-- Log body -->
    <div
      ref="scrollContainer"
      class="overflow-y-auto font-mono p-3 min-h-48 max-h-96"
      :style="{ fontSize: '0.78rem', backgroundColor: 'var(--surface-base)', color: 'var(--p-text-color)' }"
      @scroll="onScroll"
    >
      <div
        v-for="(line, index) in lines"
        :key="index"
        class="log-line whitespace-pre-wrap leading-relaxed"
        v-html="formatLogLine(line.text, line.timestamp)"
      />
      <!-- Blinking caret while actively streaming -->
      <span
        v-if="showCaret"
        class="blink-caret"
        :style="{ color: 'var(--status-running-color)' }"
        data-testid="log-stream-caret"
        aria-hidden="true"
        >▋</span
      >
      <!-- Empty-body message (only when there are no lines) -->
      <div
        v-if="!hasLines && emptyMessage"
        class="text-center py-8"
        :style="{ color: 'var(--p-text-muted-color)' }"
        data-testid="log-stream-empty"
      >
        {{ emptyMessage }}
      </div>
    </div>
  </div>
</template>

<style scoped>
:deep(.log-ts) {
  color: var(--p-text-muted-color);
  opacity: 0.7;
  font-size: 0.8em;
}
</style>
