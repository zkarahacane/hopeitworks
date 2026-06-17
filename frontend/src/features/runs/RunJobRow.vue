<script setup lang="ts">
import { computed } from 'vue'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import AgentChip from '@/ui/primitives/AgentChip.vue'
import ContainerChip from '@/ui/primitives/ContainerChip.vue'
import { stepTypeMeta } from '@/utils/stepType'
import { formatDuration } from '@/utils/formatDuration'
import type { RunStep } from './composables/useRunDetail'

/**
 * RunJobRow — one typed step in the steps list.
 *
 * Status is routed entirely through StatusBadge → statusToken (the single
 * derived status, fix #2 — no separate spinner icon contradicting the badge).
 * Shows the step type chip and, for agent steps, an AgentChip + container chip.
 */
const props = defineProps<{
  step: RunStep
  selected: boolean
}>()

const emit = defineEmits<{
  click: [step: RunStep]
}>()

const typeMeta = computed(() => stepTypeMeta(props.step.action))
const duration = computed(() => formatDuration(props.step.started_at, props.step.completed_at))

const rowStyle = computed(() => ({
  backgroundColor: props.selected ? 'var(--p-surface-100)' : 'transparent',
  border: props.selected
    ? '1px solid var(--status-accent-color)'
    : '1px solid transparent',
}))
</script>

<template>
  <button
    type="button"
    class="flex items-center gap-2 px-3 py-2 rounded w-full text-left cursor-pointer"
    :style="rowStyle"
    data-testid="job-row"
    :data-selected="selected"
    @click="emit('click', step)"
  >
    <!-- Type chip (mono): git_branch · agent_run · human · git_pr · ci_wait · notify -->
    <span
      class="font-mono inline-flex items-center gap-1"
      :style="{ fontSize: '0.7rem', color: 'var(--p-text-muted-color)', minWidth: '1rem' }"
      data-testid="step-type-icon"
    >
      <i :class="typeMeta.icon" :style="{ fontSize: '0.72rem' }" aria-hidden="true" />
    </span>

    <span class="flex-1 text-sm truncate" data-testid="step-name">{{ step.step_name }}</span>

    <AgentChip v-if="typeMeta.isAgent" :role="typeMeta.typeLabel" data-testid="job-agent-chip" />
    <ContainerChip
      v-else-if="step.container_id"
      :container-id="step.container_id"
      data-testid="job-container-chip"
    />

    <StatusBadge :status="step.status" :icon="false" data-testid="status-tag" />

    <span
      class="font-mono text-right"
      :style="{ fontSize: '0.72rem', color: 'var(--p-text-muted-color)', minWidth: '3rem' }"
      data-testid="duration"
    >
      {{ duration }}
    </span>
  </button>
</template>
