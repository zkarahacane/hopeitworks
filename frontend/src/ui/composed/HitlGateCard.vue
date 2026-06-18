<script setup lang="ts">
import { computed } from 'vue'
import Button from 'primevue/button'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'

/**
 * HitlGateCard — the amber human-in-the-loop gate.
 *
 * Presentational: shows what's awaiting a decision and emits the three actions.
 * NO API calls — hero screens (Run Detail, Board "In Review", Approvals) wire
 * the emits to their own approval composables. Breathes amber while awaiting
 * via the gate status family.
 */
withDefaults(
  defineProps<{
    /** Story key, e.g. "PROJ-12". Machine voice. */
    storyKey?: string | null
    /** Step name that hit the gate. */
    stepName?: string | null
    /** Optional PR / change URL to open. */
    prUrl?: string | null
    /** ISO timestamp the gate has been pending since. */
    pendingSince?: string | null
    /** Disable actions (e.g. while a parent submits). */
    busy?: boolean
    /** Which action is currently in-flight (for per-button spinners). */
    pendingAction?: 'approve' | 'request_changes' | 'reject' | null
    /** Breathe the amber accent (awaiting human). */
    animated?: boolean
  }>(),
  {
    storyKey: null,
    stepName: null,
    prUrl: null,
    pendingSince: null,
    busy: false,
    pendingAction: null,
    animated: true,
  },
)

const emit = defineEmits<{
  approve: []
  requestChanges: []
  reject: []
}>()

const cardStyle = computed(() => ({
  border: '1px solid var(--status-gate-color)',
  backgroundColor: 'var(--status-gate-surface)',
}))
</script>

<template>
  <div
    class="flex flex-col gap-3 p-4 rounded-lg"
    :class="{ 'amber-breathe': animated }"
    :style="cardStyle"
    data-testid="hitl-gate-card"
  >
    <!-- Header -->
    <div class="flex items-center gap-2 flex-wrap">
      <i class="pi pi-pause-circle" :style="{ color: 'var(--status-gate-color)' }" aria-hidden="true" />
      <span :style="{ fontWeight: 600 }">Awaiting your approval</span>
      <StatusBadge status="waiting_approval" label="HITL" :animated="false" class="ml-auto" />
    </div>

    <!-- Context -->
    <div class="flex items-center gap-3 flex-wrap" :style="{ fontSize: '0.82rem' }">
      <span v-if="storyKey" class="font-mono" data-testid="hitl-gate-story">{{ storyKey }}</span>
      <span v-if="stepName" :style="{ color: 'var(--p-text-muted-color)' }" data-testid="hitl-gate-step">
        {{ stepName }}
      </span>
      <a
        v-if="prUrl"
        :href="prUrl"
        target="_blank"
        rel="noopener"
        class="inline-flex items-center gap-1"
        :style="{ color: 'var(--status-accent-color)' }"
        data-testid="hitl-gate-pr-link"
      >
        <i class="pi pi-external-link" :style="{ fontSize: '0.7rem' }" aria-hidden="true" />
        View changes
      </a>
    </div>

    <!-- Actions (emit only) -->
    <div class="flex items-center gap-2 flex-wrap">
      <Button
        label="Approve"
        icon="pi pi-check"
        severity="success"
        size="small"
        :disabled="busy"
        :loading="pendingAction === 'approve'"
        data-testid="hitl-gate-approve"
        @click="emit('approve')"
      />
      <Button
        label="Request changes"
        icon="pi pi-pencil"
        severity="warn"
        size="small"
        outlined
        :disabled="busy"
        :loading="pendingAction === 'request_changes'"
        data-testid="hitl-gate-request-changes"
        @click="emit('requestChanges')"
      />
      <Button
        label="Reject"
        icon="pi pi-times"
        severity="danger"
        size="small"
        outlined
        :disabled="busy"
        :loading="pendingAction === 'reject'"
        data-testid="hitl-gate-reject"
        @click="emit('reject')"
      />
    </div>
  </div>
</template>
