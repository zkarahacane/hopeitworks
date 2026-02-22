<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import ProgressBar from 'primevue/progressbar'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import { differenceInSeconds } from 'date-fns'
import { useRecentRuns } from '@/features/runs/composables/useRecentRuns'
import { runStatusSeverity } from '@/utils/runStatus'
import { formatRelativeDate } from '@/utils/formatDate'
import type { RunSummary } from '@/features/runs/composables/useRecentRuns'

const route = useRoute()
const router = useRouter()
const projectId = computed(() => route.params.id as string)

const { runs, isLoading, error, refresh } = useRecentRuns({ projectId: projectId.value, limit: 20 })

function onRowClick(row: RunSummary) {
  router.push({ name: 'run-detail', params: { id: row.id }, query: { projectId: row.project_id } })
}

/** Format duration between started_at and completed_at (or now if still running). */
function formatDuration(run: RunSummary): string {
  if (!run.started_at) return '-'
  const end = run.completed_at ? new Date(run.completed_at) : new Date()
  const seconds = differenceInSeconds(end, new Date(run.started_at))
  if (seconds < 60) return `${seconds}s`
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}m ${secs}s`
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold">Run History</h2>
      <Button
        v-if="!isLoading && !error"
        icon="pi pi-refresh"
        text
        rounded
        size="small"
        severity="secondary"
        aria-label="Refresh runs"
        @click="refresh"
      />
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex flex-col gap-3">
      <Skeleton v-for="i in 5" :key="i" width="100%" height="2.5rem" />
    </div>

    <!-- Error state -->
    <Message v-else-if="error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error.message }}</span>
        <Button label="Retry" icon="pi pi-refresh" text size="small" @click="refresh" />
      </div>
    </Message>

    <!-- Empty state -->
    <div v-else-if="runs.length === 0" class="flex flex-col items-center justify-center py-12 text-surface-400">
      <i class="pi pi-play text-4xl mb-3" />
      <p class="text-lg font-medium">No runs yet</p>
      <p class="text-sm">Pipeline runs for this project will appear here once triggered.</p>
    </div>

    <!-- Data table -->
    <DataTable
      v-else
      :value="runs"
      :rows="20"
      :paginator="runs.length > 20"
      row-hover
      class="cursor-pointer"
      data-testid="project-runs-table"
      @row-click="onRowClick($event.data)"
    >
      <Column header="Run ID">
        <template #body="{ data }">
          <code class="text-xs bg-surface-100 px-1.5 py-0.5 rounded font-mono">
            {{ data.id.substring(0, 8) }}
          </code>
        </template>
      </Column>
      <Column field="story_key" header="Story Key" />
      <Column field="status" header="Status">
        <template #body="{ data }">
          <Tag :value="data.status" :severity="runStatusSeverity[data.status]" />
        </template>
      </Column>
      <Column field="progress" header="Progress">
        <template #body="{ data }">
          <ProgressBar :value="data.progress ?? 0" :show-value="false" style="height: 0.5rem; width: 6rem" />
        </template>
      </Column>
      <Column field="started_at" header="Started">
        <template #body="{ data }">
          {{ data.started_at ? formatRelativeDate(data.started_at) : '-' }}
        </template>
      </Column>
      <Column header="Duration">
        <template #body="{ data }">
          <span v-if="data.status === 'running'" class="text-blue-500">running...</span>
          <span v-else>{{ formatDuration(data) }}</span>
        </template>
      </Column>
    </DataTable>
  </div>
</template>
