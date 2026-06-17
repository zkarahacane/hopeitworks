<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import { differenceInSeconds } from 'date-fns'
import { useRecentRuns } from '@/features/runs/composables/useRecentRuns'
import { formatRelativeDate } from '@/utils/formatDate'
import { formatCostUSD } from '@/utils/formatCost'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import type { RunSummary } from '@/features/runs/composables/useRecentRuns'

const route = useRoute()
const router = useRouter()
const projectId = computed(() => route.params.id as string)

const { runs, isLoading, error, refresh } = useRecentRuns({ projectId: projectId.value, limit: 20 })

function onRowClick(row: RunSummary) {
  router.push({ name: 'run-detail', params: { id: row.id }, query: { projectId: row.project_id } })
}

function formatDuration(run: RunSummary): string {
  if (!run.started_at) return '—'
  const end = run.completed_at ? new Date(run.completed_at) : new Date()
  const secs = differenceInSeconds(end, new Date(run.started_at))
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex flex-col gap-1">
        <h2 class="text-xl font-semibold" :style="{ color: 'var(--p-text-color)' }">Run History</h2>
        <p class="text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
          Pipeline run history for this project
        </p>
      </div>
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
        <span :style="{ color: 'var(--p-text-color)' }">{{ error.message }}</span>
        <Button label="Retry" icon="pi pi-refresh" text size="small" @click="refresh" />
      </div>
    </Message>

    <!-- Empty state -->
    <div
      v-else-if="runs.length === 0"
      class="flex flex-col items-center justify-center py-12"
      :style="{ color: 'var(--p-text-muted-color)' }"
    >
      <i class="pi pi-activity text-4xl mb-3" />
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
      <!-- Story -->
      <Column header="Story">
        <template #body="{ data }">
          <span class="font-semibold" :style="{ color: 'var(--p-text-color)' }">
            {{ data.story_key }}
          </span>
        </template>
      </Column>

      <!-- Run ID -->
      <Column header="Run ID">
        <template #body="{ data }">
          <span class="font-mono text-xs" :style="{ color: 'var(--p-text-muted-color)' }">
            {{ data.id.substring(0, 8) }}
          </span>
        </template>
      </Column>

      <!-- Status -->
      <Column header="Status">
        <template #body="{ data }">
          <StatusBadge :status="data.status" />
        </template>
      </Column>

      <!-- Started -->
      <Column header="Started">
        <template #body="{ data }">
          <span :style="{ color: 'var(--p-text-muted-color)' }">
            {{ data.started_at ? formatRelativeDate(data.started_at) : '—' }}
          </span>
        </template>
      </Column>

      <!-- Duration -->
      <Column header="Duration">
        <template #body="{ data }">
          <span
            v-if="data.status === 'running'"
            class="inline-flex items-center gap-1.5 font-mono text-sm"
            :style="{ color: 'var(--p-text-color)' }"
          >
            <span
              class="live-pulse inline-block rounded-full"
              :style="{
                width: '0.5rem',
                height: '0.5rem',
                backgroundColor: 'var(--status-running-color)',
                flexShrink: '0',
              }"
              aria-hidden="true"
            />
            {{ formatDuration(data) }}
          </span>
          <span
            v-else
            class="font-mono text-sm"
            :style="{ color: 'var(--p-text-muted-color)' }"
          >
            {{ formatDuration(data) }}
          </span>
        </template>
      </Column>

      <!-- Cost -->
      <Column header="Cost">
        <template #body="{ data }">
          <span
            v-if="data.total_cost_usd != null"
            class="font-mono text-sm"
            :style="{ color: 'var(--p-text-muted-color)' }"
          >
            {{ formatCostUSD(data.total_cost_usd) }}
          </span>
          <span v-else :style="{ color: 'var(--p-text-muted-color)' }">—</span>
        </template>
      </Column>
    </DataTable>
  </div>
</template>
