<script setup lang="ts">
import { computed } from 'vue'
import Tag from 'primevue/tag'
import { statusSeverity } from '@/utils/runStatus'
import { formatDuration } from '@/utils/formatDuration'
import type { RunStep } from './composables/useRunDetail'

const STATUS_ICONS: Record<string, string> = {
  pending: 'pi pi-clock',
  running: 'pi pi-spin pi-spinner',
  completed: 'pi pi-check-circle',
  failed: 'pi pi-times-circle',
  cancelled: 'pi pi-minus-circle',
  skipped: 'pi pi-forward',
  waiting_approval: 'pi pi-pause-circle',
}

const props = defineProps<{
  step: RunStep
  selected: boolean
}>()

const emit = defineEmits<{
  click: [step: RunStep]
}>()

const icon = computed(() => STATUS_ICONS[props.step.status] ?? 'pi pi-question-circle')
const isRunning = computed(() => props.step.status === 'running')
const duration = computed(() =>
  formatDuration(props.step.started_at, props.step.completed_at),
)
</script>

<template>
  <div
    class="flex items-center gap-2 px-3 py-2 rounded cursor-pointer transition-colors hover:bg-surface-100 dark:hover:bg-surface-700"
    :class="{
      'bg-primary-50 dark:bg-primary-900/20 border border-primary-200 dark:border-primary-700': selected,
      'border border-transparent': !selected,
    }"
    data-testid="job-row"
    @click="emit('click', step)"
  >
    <i
      :class="[icon, { 'running-indicator': isRunning }]"
      data-testid="status-icon"
    />
    <span class="flex-1 text-sm truncate" data-testid="step-name">{{ step.step_name }}</span>
    <Tag
      :severity="statusSeverity(step.status)"
      :value="step.status"
      class="text-xs"
      data-testid="status-tag"
    />
    <span class="text-xs text-surface-400 tabular-nums min-w-[3rem] text-right" data-testid="duration">
      {{ duration }}
    </span>
  </div>
</template>
