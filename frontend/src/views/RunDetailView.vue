<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Tag from 'primevue/tag'
import ProgressBar from 'primevue/progressbar'
import Timeline from 'primevue/timeline'
import Button from 'primevue/button'
import { differenceInSeconds } from 'date-fns'
import { useRunDetail } from '@/features/runs/composables/useRunDetail'
import RunLogViewer from '@/features/runs/RunLogViewer.vue'
import type { RunStep } from '@/features/runs/composables/useRunDetail'

const route = useRoute()
const runId = computed(() => route.params.id as string)

const { run: runRef, isLoading, error, retry } = useRunDetail(runId.value)
const run = computed(() => runRef.value)

const runStatusSeverity: Record<string, 'info' | 'success' | 'warn' | 'danger' | 'secondary'> = {
  pending: 'secondary',
  running: 'info',
  completed: 'success',
  failed: 'danger',
  cancelled: 'warn',
}

const stepSeverity: Record<string, string> = {
  completed: 'success',
  running: 'info',
  failed: 'danger',
  pending: 'secondary',
  cancelled: 'warn',
}

/** Compute progress from completed steps ratio (0-100). */
const progress = computed(() => {
  if (!run.value || run.value.steps.length === 0) return 0
  if (run.value.progress !== undefined) return run.value.progress
  const completed = run.value.steps.filter(
    (s) => s.status === 'completed' || s.status === 'failed' || s.status === 'cancelled',
  ).length
  return Math.round((completed / run.value.steps.length) * 100)
})

/** Format step duration from started_at to completed_at. */
function formatDuration(step: RunStep): string | null {
  if (!step.started_at) return null
  const end = step.completed_at ? new Date(step.completed_at) : new Date()
  const seconds = differenceInSeconds(end, new Date(step.started_at))
  if (seconds < 60) return `${seconds}s`
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}m ${secs}s`
}
</script>

<template>
  <div class="flex flex-col h-full p-6 gap-6">
    <!-- Page Header — always visible -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <h1 class="text-xl font-bold">Run Detail</h1>
        <code
          v-if="run"
          class="text-sm bg-surface-100 dark:bg-surface-800 px-2 py-1 rounded font-mono"
        >
          {{ run.id }}
        </code>
      </div>
      <Tag v-if="run" :value="run.status" :severity="runStatusSeverity[run.status]" />
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex flex-col gap-4">
      <Skeleton width="20rem" height="2rem" />
      <Skeleton width="100%" height="1rem" />
      <Skeleton width="100%" height="12rem" />
    </div>

    <!-- Error state -->
    <Message v-else-if="error" severity="error" :closable="false">
      {{ error.message }}
      <Button label="Retry" icon="pi pi-refresh" text size="small" class="ml-2" @click="retry" />
    </Message>

    <!-- Run data -->
    <template v-else-if="run">

      <!-- Progress Bar -->
      <ProgressBar :value="progress" :show-value="true" />

      <!-- Step Timeline -->
      <div v-if="run.steps.length > 0">
        <h2 class="text-lg font-semibold mb-3">Steps</h2>
        <Timeline :value="run.steps" align="left" class="w-full">
          <template #marker="{ item }">
            <Tag
              :severity="stepSeverity[(item as RunStep).status] ?? 'secondary'"
              class="w-8 h-8 flex items-center justify-center rounded-full"
            >
              <i
                :class="{
                  'pi pi-check': (item as RunStep).status === 'completed',
                  'pi pi-spin pi-spinner': (item as RunStep).status === 'running',
                  'pi pi-times': (item as RunStep).status === 'failed',
                  'pi pi-clock': (item as RunStep).status === 'pending',
                  'pi pi-ban': (item as RunStep).status === 'cancelled',
                }"
                class="text-xs"
              />
            </Tag>
          </template>
          <template #content="{ item }">
            <div class="flex flex-col gap-1 mb-4">
              <div class="flex items-center gap-2">
                <span class="font-medium">{{ (item as RunStep).step_name }}</span>
                <Tag
                  :value="(item as RunStep).status"
                  :severity="stepSeverity[(item as RunStep).status] ?? 'secondary'"
                  class="text-xs"
                />
              </div>
              <div class="text-sm text-surface-500 flex items-center gap-3">
                <span>{{ (item as RunStep).action }}</span>
                <span v-if="formatDuration(item as RunStep)" class="text-surface-400">
                  {{ formatDuration(item as RunStep) }}
                </span>
              </div>
              <div
                v-if="(item as RunStep).error_message"
                class="text-sm text-red-500 mt-1"
              >
                {{ (item as RunStep).error_message }}
              </div>
            </div>
          </template>
        </Timeline>
      </div>

      <!-- Live Log Viewer -->
      <div>
        <h2 class="text-lg font-semibold mb-3">Live Logs</h2>
        <RunLogViewer :project-id="run.project_id" :run-id="run.id" />
      </div>
    </template>
  </div>
</template>
