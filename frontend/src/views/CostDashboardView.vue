<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Message from 'primevue/message'
import ProgressBar from 'primevue/progressbar'
import { useCosts } from '@/composables/useCosts'
import { formatCostUSD } from '@/utils/formatCost'
import CostSummaryCard from '@/features/costs/CostSummaryCard.vue'
import CostChart from '@/features/costs/CostChart.vue'
import RunCostTable from '@/features/costs/RunCostTable.vue'

const route = useRoute()
const router = useRouter()

const projectId = route.params.id as string
const { period, summary, chartData, runs, isLoading, error, fetchAll, setPeriod } =
  useCosts(projectId)

function onRunNavigate(runId: string) {
  router.push({ name: 'run-detail', params: { id: runId } })
}

function budgetPercent(total: number, limit: number): number {
  if (limit <= 0) return 0
  return Math.min((total / limit) * 100, 100)
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <!-- Error state -->
    <Message
      v-if="error"
      severity="error"
      :closable="false"
      data-testid="cost-error"
    >
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="fetchAll()" />
      </div>
    </Message>

    <template v-if="!error">
      <!-- Period toggle -->
      <div class="flex items-center gap-2">
        <span class="text-sm font-medium text-surface-600">Period:</span>
        <Button
          label="7d"
          size="small"
          :outlined="period !== '7d'"
          data-testid="period-7d"
          @click="setPeriod('7d')"
        />
        <Button
          label="30d"
          size="small"
          :outlined="period !== '30d'"
          data-testid="period-30d"
          @click="setPeriod('30d')"
        />
      </div>

      <!-- Summary cards -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <CostSummaryCard
          label="Total cost this week"
          :value="summary ? formatCostUSD(summary.total_cost_week_usd ?? 0) : '$0.00'"
          :is-loading="isLoading"
          data-testid="card-week"
        />
        <CostSummaryCard
          label="Total cost this month"
          :value="summary ? formatCostUSD(summary.total_cost_month_usd ?? 0) : '$0.00'"
          :is-loading="isLoading"
          data-testid="card-month"
        />
        <CostSummaryCard
          label="Average cost per story"
          :value="summary ? formatCostUSD(summary.avg_cost_per_story_usd) : '$0.00'"
          :is-loading="isLoading"
          data-testid="card-avg"
        />
      </div>

      <!-- Budget progress bar -->
      <div
        v-if="summary && (summary.budget_limit_usd ?? 0) > 0"
        class="rounded-lg border border-surface-200 bg-surface-0 p-4"
        data-testid="budget-bar"
      >
        <div class="mb-2 flex items-center justify-between text-sm">
          <span class="font-medium">Budget usage</span>
          <span class="text-surface-500">
            {{ formatCostUSD(summary.total_cost_usd) }} /
            {{ formatCostUSD(summary.budget_limit_usd ?? 0) }} used
          </span>
        </div>
        <ProgressBar
          :value="budgetPercent(summary.total_cost_usd, summary.budget_limit_usd ?? 0)"
          :show-value="false"
        />
      </div>

      <!-- Cost over time chart -->
      <CostChart :data="chartData" :is-loading="isLoading" data-testid="cost-chart" />

      <!-- Recent runs table -->
      <RunCostTable
        :runs="runs"
        :is-loading="isLoading"
        data-testid="runs-table"
        @navigate="onRunNavigate"
      />
    </template>
  </div>
</template>
