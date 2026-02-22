<script setup lang="ts">
import { useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import ProgressBar from 'primevue/progressbar'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import { useRecentRuns } from '@/features/runs/composables/useRecentRuns'
import { runStatusSeverity } from '@/utils/runStatus'
import { formatRelativeDate } from '@/utils/formatDate'
import type { RunSummary } from '@/features/runs/composables/useRecentRuns'

const router = useRouter()
const { runs, isLoading, error, refresh } = useRecentRuns()

function onRowClick(row: RunSummary) {
  router.push({ name: 'run-detail', params: { id: row.id }, query: { projectId: row.project_id } })
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Runs</h1>
      <Button
        v-if="!isLoading && !error"
        icon="pi pi-refresh"
        text
        rounded
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
      <p class="text-sm">Pipeline runs will appear here once triggered from a project.</p>
    </div>

    <!-- Data table -->
    <DataTable
      v-else
      :value="runs"
      :rows="20"
      row-hover
      class="cursor-pointer"
      data-testid="runs-table"
      @row-click="onRowClick($event.data)"
    >
      <Column field="project_name" header="Project" />
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
    </DataTable>
  </div>
</template>
