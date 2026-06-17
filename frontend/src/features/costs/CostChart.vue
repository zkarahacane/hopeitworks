<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { format, parseISO } from 'date-fns'
import Chart from 'primevue/chart'
import Skeleton from 'primevue/skeleton'
import type { components } from '@/api/schema'
import { useTheme } from '@/composables/useTheme'

type CostDataPoint = components['schemas']['CostDataPoint']

const props = defineProps<{
  data: CostDataPoint[]
  isLoading: boolean
}>()

const { resolvedScheme } = useTheme()

function readToken(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

const chartDataset = ref<object>({})
const chartOptions = ref<object>({})

function buildChart() {
  const lineColor = readToken('--status-running-color')
  const gridColor = readToken('--surface-border')
  chartDataset.value = {
    labels: props.data.map((d) => format(parseISO(d.date), 'MMM d')),
    datasets: [
      {
        label: 'Daily Cost (USD)',
        data: props.data.map((d) => d.total_cost_usd),
        fill: false,
        tension: 0.4,
        borderColor: lineColor,
        pointBackgroundColor: lineColor,
      },
    ],
  }
  chartOptions.value = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
    },
    scales: {
      x: {
        grid: { color: gridColor },
      },
      y: {
        beginAtZero: true,
        grid: { color: gridColor },
        ticks: {
          callback: (value: number) => `$${value.toFixed(2)}`,
        },
      },
    },
    elements: {
      line: { borderColor: lineColor, tension: 0.4 },
      point: { backgroundColor: lineColor },
    },
  }
}

onMounted(buildChart)
watch(() => props.data, buildChart, { deep: true })
watch(resolvedScheme, buildChart)
</script>

<template>
  <div
    class="rounded-lg p-4"
    :style="{ background: 'var(--surface-raised)', border: '1px solid var(--surface-border)' }"
  >
    <h3 class="mb-4 font-semibold">Cost Over Time</h3>
    <Skeleton v-if="isLoading" width="100%" height="16rem" />
    <div v-else-if="data.length === 0" class="flex flex-col items-center justify-center py-12">
      <i class="pi pi-chart-line mb-3 text-3xl" :style="{ color: 'var(--p-text-muted-color)' }" />
      <p :style="{ color: 'var(--p-text-muted-color)' }">No cost data yet</p>
    </div>
    <div v-else style="height: 16rem">
      <Chart type="line" :data="chartDataset" :options="chartOptions" style="height: 100%" data-testid="cost-chart-canvas" />
    </div>
  </div>
</template>
