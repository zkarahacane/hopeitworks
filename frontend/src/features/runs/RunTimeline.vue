<script setup lang="ts">
import { computed } from 'vue'
import Tag from 'primevue/tag'
import RetryStepEntry from './RetryStepEntry.vue'
import { useRunTimeline } from './composables/useRunTimeline'
import type { components } from '@/api/schema'

type RunStep = components['schemas']['RunStep']

const props = defineProps<{
  steps: RunStep[]
}>()

const stepsRef = computed(() => props.steps)
const { groupedSteps } = useRunTimeline(stepsRef)

/** Severity map for step status to PrimeVue Tag severity */
function statusSeverity(status: string): 'success' | 'danger' | 'warn' | 'info' | 'secondary' {
  switch (status) {
    case 'completed':
      return 'success'
    case 'failed':
      return 'danger'
    case 'running':
      return 'info'
    case 'waiting_approval':
      return 'warn'
    default:
      return 'secondary'
  }
}

function formatDate(iso?: string | null): string {
  if (!iso) return ''
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div
      v-for="group in groupedSteps"
      :key="group.root.id"
      class="rounded border border-surface-200 p-4"
      data-testid="step-group"
    >
      <!-- Root step header -->
      <div class="flex items-center gap-3" data-testid="root-step">
        <span class="font-semibold text-surface-800">{{ group.root.step_name }}</span>
        <Tag
          :severity="statusSeverity(group.root.status)"
          :value="group.root.status"
          class="text-xs"
        />
        <span v-if="group.root.started_at" class="text-xs text-surface-400">
          {{ formatDate(group.root.started_at) }}
        </span>
        <span
          v-if="group.root.completed_at"
          class="text-xs text-surface-400"
          data-testid="completed-at"
        >
          → {{ formatDate(group.root.completed_at) }}
        </span>
      </div>

      <!-- Error message on root step -->
      <div v-if="group.root.error_message" class="mt-2 text-sm text-red-600">
        {{ group.root.error_message }}
      </div>

      <!-- Retry entries -->
      <RetryStepEntry
        v-for="retry in group.retries"
        :key="retry.id"
        :step="retry"
        :parent-step="group.root"
        data-testid="retry-entry"
      />
    </div>

    <div v-if="groupedSteps.length === 0" class="text-surface-400 text-sm" data-testid="empty-state">
      No steps to display.
    </div>
  </div>
</template>
