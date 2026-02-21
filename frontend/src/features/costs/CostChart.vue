<script setup lang="ts">
import { computed } from 'vue'
import { format, parseISO } from 'date-fns'
import Chart from 'primevue/chart'
import Skeleton from 'primevue/skeleton'
import type { components } from '@/api/schema'

type CostDataPoint = components['schemas']['CostDataPoint']

const props = defineProps<{
  data: CostDataPoint[]
  isLoading: boolean
}>()

const chartDataset = computed(() => ({
  labels: props.data.map((d) => format(parseISO(d.date), 'MMM d')),
  datasets: [
    {
      label: 'Daily Cost (USD)',
      data: props.data.map((d) => d.total_cost_usd),
      fill: false,
      tension: 0.3,
      borderColor: '#6366f1',
      pointBackgroundColor: '#6366f1',
    },
  ],
}))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
  },
  scales: {
    x: {
      grid: { display: false },
    },
    y: {
      beginAtZero: true,
      ticks: {
        callback: (value: number) => `$${value.toFixed(2)}`,
      },
    },
  },
}
</script>

<template>
  <div class="rounded-lg border border-surface-200 bg-surface-0 p-4">
    <h3 class="mb-4 font-semibold">Cost Over Time</h3>
    <Skeleton v-if="isLoading" width="100%" height="16rem" />
    <div v-else-if="data.length === 0" class="flex flex-col items-center justify-center py-12">
      <i class="pi pi-chart-line mb-3 text-3xl text-surface-300" />
      <p class="text-surface-500">No cost data yet</p>
    </div>
    <div v-else style="height: 16rem">
      <Chart type="line" :data="chartDataset" :options="chartOptions" style="height: 100%" />
    </div>
  </div>
</template>
