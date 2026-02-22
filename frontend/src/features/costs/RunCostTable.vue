<script setup lang="ts">
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Skeleton from 'primevue/skeleton'
import { useRelativeTime } from '@/composables/useRelativeTime'
import { formatCostUSD } from '@/utils/formatCost'
import { statusSeverity } from '@/utils/runStatus'
import type { components } from '@/api/schema'

type RunCostRow = components['schemas']['RunCostRow']

defineProps<{
  runs: RunCostRow[]
  isLoading: boolean
}>()

const emit = defineEmits<{
  navigate: [runId: string]
}>()

const skeletonRows = [1, 2, 3]
</script>

<template>
  <div class="rounded-lg border border-surface-200 bg-surface-0">
    <div class="border-b border-surface-200 px-4 py-3">
      <h3 class="font-semibold">Recent Runs</h3>
    </div>

    <!-- Skeleton loading state -->
    <div v-if="isLoading" class="divide-y divide-surface-100">
      <div v-for="n in skeletonRows" :key="n" class="flex items-center gap-4 px-4 py-3">
        <Skeleton width="4rem" height="1.25rem" />
        <Skeleton width="5rem" height="1.5rem" class="rounded-full" />
        <Skeleton width="5rem" height="1.25rem" />
        <Skeleton width="5rem" height="1.25rem" class="ml-auto" />
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="runs.length === 0"
      class="flex flex-col items-center justify-center py-10"
    >
      <i class="pi pi-list mb-3 text-3xl text-surface-300" />
      <p class="text-surface-500">No runs in this period</p>
    </div>

    <!-- Data table -->
    <DataTable
      v-else
      :value="runs"
      row-hover
      class="cursor-pointer"
      @row-click="emit('navigate', $event.data.run_id)"
    >
      <Column field="story_key" header="Story" />
      <Column field="status" header="Status">
        <template #body="{ data: row }">
          <Tag :value="row.status" :severity="statusSeverity(row.status)" />
        </template>
      </Column>
      <Column field="started_at" header="Started">
        <template #body="{ data: row }">
          <RelativeCell :date="row.started_at" />
        </template>
      </Column>
      <Column field="total_cost_usd" header="Cost">
        <template #body="{ data: row }">
          {{ formatCostUSD(row.total_cost_usd) }}
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<!-- Inline helper to avoid passing composable into template directly -->
<script lang="ts">
import { defineComponent, h } from 'vue'

const RelativeCell = defineComponent({
  props: { date: { type: String, required: true } },
  setup(props) {
    const rel = useRelativeTime(props.date)
    return () => h('span', rel.value ?? props.date)
  },
})

export { RelativeCell }
</script>
