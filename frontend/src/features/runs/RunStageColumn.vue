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
    :class="{ 'border-r border-surface-200 dark:border-surface-600 pr-2 mr-2': !isLast }"
    data-testid="stage-column"
  >
    <!-- Stage header -->
    <div class="px-3 py-2 text-sm font-semibold text-surface-600 dark:text-surface-300 uppercase tracking-wider" data-testid="stage-header">
      {{ stageName }}
    </div>

    <!-- Connector arrow on right edge (except last column) -->
    <div
      v-if="!isLast"
      class="absolute right-0 top-1/2 -translate-y-1/2 translate-x-1/2 z-10"
      data-testid="stage-connector"
    >
      <i class="pi pi-chevron-right text-surface-300 dark:text-surface-500 text-xs" />
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
        class="px-3 py-2 text-xs text-surface-400 italic"
        data-testid="empty-stage"
      >
        No steps
      </div>
    </div>
  </div>
</template>
