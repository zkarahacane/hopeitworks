<script setup lang="ts">
import { computed } from 'vue'
import Button from 'primevue/button'
import Message from 'primevue/message'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import { suggestedRemedy } from '@/features/approvals/composables/useProbeHaltActions'
import type { HaltReason } from '@/stores/probeHalts'

/**
 * HaltGateCard — the resolution panel for a probe_halt halt-gate.
 *
 * Presentational: renders halt reason human-readably, shows a suggested
 * remedy hint, and emits the 5 resolution actions. No API calls.
 */
const props = withDefaults(
  defineProps<{
    /** Story key, e.g. "PROJ-12". */
    storyKey?: string | null
    /** Step name that triggered the halt. */
    stepName?: string | null
    /** Stage name at time of halt. */
    stageName?: string | null
    /** Structured halt reason from the backend. */
    haltReason?: HaltReason
    /** ISO timestamp the halt has been pending since. */
    pendingSince?: string | null
    /** Disable all actions (e.g. while a parent submits). */
    busy?: boolean
    /** Which action is currently in-flight (for per-button spinners). */
    pendingAction?: 'resume' | 'override' | 'send_back' | 'skip' | 'abort' | null
  }>(),
  {
    storyKey: null,
    stepName: null,
    stageName: null,
    haltReason: undefined,
    pendingSince: null,
    busy: false,
    pendingAction: null,
  },
)

const emit = defineEmits<{
  resume: []
  override: []
  sendBack: []
  skip: []
  abort: []
}>()

/** Human-readable description of why the run was halted. */
const haltDescription = computed<string>(() => {
  const r = props.haltReason
  if (!r) return 'The run was halted by a guard probe.'

  const obs = r.observed ?? 0
  const thr = r.threshold ?? 0

  switch (r.probe) {
    case 'log_silence':
      return `No agent output for ${obs}${r.unit === 'seconds' ? 's' : (r.unit ?? '')} (limit ${thr}${r.unit === 'seconds' ? 's' : (r.unit ?? '')})`
    case 'wallclock':
      return `Step ran ${obs}${r.unit === 'seconds' ? 's' : (r.unit ?? '')} (limit ${thr}${r.unit === 'seconds' ? 's' : (r.unit ?? '')})`
    case 'cost_batch':
      return `Run cost $${obs.toFixed(2)} over budget $${thr.toFixed(2)}`
    default:
      return r.detail ?? 'The run was halted by a guard probe.'
  }
})

const remedy = computed(() => suggestedRemedy(props.haltReason?.probe ?? ''))

const cardStyle = computed(() => ({
  border: '1px solid var(--status-gate-color)',
  backgroundColor: 'var(--status-gate-surface)',
}))
</script>

<template>
  <div
    class="flex flex-col gap-3 p-4 rounded-lg amber-breathe"
    :style="cardStyle"
    data-testid="halt-gate-card"
  >
    <!-- Header -->
    <div class="flex items-center gap-2 flex-wrap">
      <i
        class="pi pi-exclamation-triangle"
        :style="{ color: 'var(--status-gate-color)' }"
        aria-hidden="true"
      />
      <span :style="{ fontWeight: 600 }">Run halted by guard</span>
      <StatusBadge status="waiting_approval" label="HALT" :animated="false" class="ml-auto" />
    </div>

    <!-- Context -->
    <div class="flex items-center gap-3 flex-wrap" :style="{ fontSize: '0.82rem' }">
      <span v-if="storyKey" class="font-mono" data-testid="halt-gate-story">{{ storyKey }}</span>
      <span
        v-if="stepName"
        :style="{ color: 'var(--p-text-muted-color)' }"
        data-testid="halt-gate-step"
      >
        {{ stepName }}
      </span>
      <span
        v-if="stageName"
        :style="{ color: 'var(--p-text-muted-color)' }"
        data-testid="halt-gate-stage"
      >
        {{ stageName }}
      </span>
    </div>

    <!-- Halt reason description -->
    <p class="m-0" :style="{ fontSize: '0.875rem' }" data-testid="halt-gate-description">
      {{ haltDescription }}
    </p>

    <!-- Suggested remedy -->
    <Message severity="info" :closable="false" data-testid="halt-gate-remedy">
      <span class="font-medium">Suggested: {{ remedy.label }}</span>
      &mdash;
      {{ remedy.hint }}
    </Message>

    <!-- Action buttons -->
    <div class="flex items-center gap-2 flex-wrap">
      <Button
        label="Resume"
        icon="pi pi-play"
        severity="success"
        size="small"
        :disabled="busy"
        :loading="pendingAction === 'resume'"
        data-testid="halt-gate-resume"
        @click="emit('resume')"
      />
      <Button
        label="Override"
        icon="pi pi-forward"
        severity="warn"
        size="small"
        outlined
        :disabled="busy"
        :loading="pendingAction === 'override'"
        data-testid="halt-gate-override"
        @click="emit('override')"
      />
      <Button
        label="Skip"
        icon="pi pi-step-forward"
        severity="secondary"
        size="small"
        outlined
        :disabled="busy"
        :loading="pendingAction === 'skip'"
        data-testid="halt-gate-skip"
        @click="emit('skip')"
      />
      <Button
        label="Send back"
        icon="pi pi-reply"
        severity="secondary"
        size="small"
        outlined
        :disabled="busy"
        :loading="pendingAction === 'send_back'"
        data-testid="halt-gate-send-back"
        @click="emit('sendBack')"
      />
      <Button
        label="Abort"
        icon="pi pi-ban"
        severity="danger"
        size="small"
        outlined
        :disabled="busy"
        :loading="pendingAction === 'abort'"
        data-testid="halt-gate-abort"
        @click="emit('abort')"
      />
    </div>
  </div>
</template>
