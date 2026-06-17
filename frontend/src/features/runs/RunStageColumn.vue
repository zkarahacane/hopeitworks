<script setup lang="ts">
import RunJobRow from './RunJobRow.vue'
import type { RunStep } from './composables/useRunDetail'

defineProps<{
  stageName: string
  steps: RunStep[]
  selectedStepId: string | null
  isLast: boolean
}>()

const emit = defineEmits<{
  'step-selected': [step: RunStep]
}>()
</script>

<template>
  <div
    class="flex flex-col min-w-52 relative"
    :class="{ 'pr-2 mr-2': !isLast }"
    :style="!isLast ? { borderRight: '1px solid var(--p-surface-200)' } : undefined"
    data-testid="stage-column"
  >
    <!-- Stage header -->
    <div
      class="px-3 py-2 uppercase"
      :style="{
        fontSize: '0.72rem',
        fontWeight: 600,
        letterSpacing: '0.06em',
        color: 'var(--p-text-muted-color)',
      }"
      data-testid="stage-header"
    >
      {{ stageName }}
    </div>

    <!-- Connector arrow on right edge (except last column) -->
    <div
      v-if="!isLast"
      class="absolute right-0 top-1/2 -translate-y-1/2 translate-x-1/2 z-10"
      data-testid="stage-connector"
    >
      <i
        class="pi pi-chevron-right"
        :style="{ fontSize: '0.7rem', color: 'var(--p-surface-400)' }"
        aria-hidden="true"
      />
    </div>

    <!-- Step rows -->
    <div class="flex flex-col gap-1">
      <RunJobRow
        v-for="step in steps"
        :key="step.id"
        :step="step"
        :selected="step.id === selectedStepId"
        @click="emit('step-selected', $event)"
      />
      <div
        v-if="steps.length === 0"
        class="px-3 py-2"
        :style="{ fontSize: '0.72rem', color: 'var(--p-surface-400)', fontStyle: 'italic' }"
        data-testid="empty-stage"
      >
        No steps
      </div>
    </div>
  </div>
</template>
