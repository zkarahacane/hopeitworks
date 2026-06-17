<script setup lang="ts">
import { computed } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Skeleton from 'primevue/skeleton'
import { formatCostUSD, formatTokenCount } from '@/utils/formatCost'
import type { components } from '@/api/schema'

type AgentCostBreakdown = components['schemas']['AgentCostBreakdown']

const props = defineProps<{
  data: AgentCostBreakdown[]
  isLoading: boolean
}>()

const totalCost = computed(() => props.data.reduce((sum, r) => sum + r.cost_usd, 0))

function percentOf(cost: number): string {
  if (totalCost.value === 0) return '0.0%'
  return ((cost / totalCost.value) * 100).toFixed(1) + '%'
}

const skeletonRows = [1, 2, 3]
</script>

<template>
  <div
    class="rounded-lg"
    :style="{ background: 'var(--surface-raised)', border: '1px solid var(--surface-border)' }"
  >
    <div class="px-4 py-3" :style="{ borderBottom: '1px solid var(--surface-border)' }">
      <h3 class="font-semibold">Cost by Agent</h3>
    </div>

    <!-- Skeleton loading state -->
    <div v-if="isLoading" class="divide-y" :style="{ '--tw-divide-color': 'var(--surface-border)' }">
      <div v-for="n in skeletonRows" :key="n" class="flex items-center gap-4 px-4 py-3">
        <Skeleton width="6rem" height="1.25rem" />
        <Skeleton width="3rem" height="1.25rem" />
        <Skeleton width="5rem" height="1.25rem" />
        <Skeleton width="5rem" height="1.25rem" />
        <Skeleton width="5rem" height="1.25rem" class="ml-auto" />
        <Skeleton width="4rem" height="1.25rem" />
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="data.length === 0"
      class="flex flex-col items-center justify-center py-10"
      data-testid="agent-cost-empty"
    >
      <i class="pi pi-users mb-3 text-3xl" :style="{ color: 'var(--p-text-muted-color)' }" />
      <p :style="{ color: 'var(--p-text-muted-color)' }">No agent cost data available</p>
    </div>

    <!-- Data table -->
    <DataTable
      v-else
      :value="data"
      :default-sort-order="-1"
      sort-field="cost_usd"
      data-testid="agent-cost-table"
    >
      <Column field="agent_name" header="Agent Name" sortable />
      <Column field="runs_count" header="Runs" sortable />
      <Column field="tokens_input" header="Tokens In" sortable>
        <template #body="{ data: row }">
          {{ formatTokenCount(row.tokens_input) }}
        </template>
      </Column>
      <Column field="tokens_output" header="Tokens Out" sortable>
        <template #body="{ data: row }">
          {{ formatTokenCount(row.tokens_output) }}
        </template>
      </Column>
      <Column field="cost_usd" header="Cost (USD)" sortable>
        <template #body="{ data: row }">
          {{ formatCostUSD(row.cost_usd) }}
        </template>
      </Column>
      <Column header="% of Total">
        <template #body="{ data: row }">
          {{ percentOf(row.cost_usd) }}
        </template>
      </Column>
    </DataTable>
  </div>
</template>
