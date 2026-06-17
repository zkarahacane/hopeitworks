<script setup lang="ts">
import { computed } from 'vue'
import CostTicker from '@/ui/primitives/CostTicker.vue'
import { formatCostUSD } from '@/utils/formatCost'
import type { CostByRoleResult } from '@/utils/costByRole'

/**
 * RunCostByRole — the COST BY ROLE panel for the Run Detail hero.
 *
 * Horizontal bars per role (Dev Agent / Review Agent / Merge Agent) + a
 * "Total this run" line. Prop-driven: the breakdown is derived upstream from
 * the REST per-step cost records (the per-role endpoint, lot #6, is not ready —
 * see useRunCostByRole / costByRole). The total is the REAL rolled-up run cost
 * (fix #3: never $0.00 on failed runs).
 */
const props = withDefaults(
  defineProps<{
    breakdown: CostByRoleResult
    /** Whether costs are still loading (shows a muted placeholder). */
    loading?: boolean
  }>(),
  { loading: false },
)

const hasRoles = computed(() => props.breakdown.roles.length > 0)
</script>

<template>
  <section
    class="flex flex-col gap-3 p-4 rounded-lg"
    :style="{ border: '1px solid var(--p-surface-200)', backgroundColor: 'var(--p-surface-0)' }"
    data-testid="run-cost-by-role"
  >
    <header class="flex items-center justify-between">
      <span
        :style="{ fontWeight: 600, fontSize: '0.78rem', letterSpacing: '0.04em', color: 'var(--p-text-muted-color)' }"
      >
        COST BY ROLE
      </span>
    </header>

    <!-- Loading placeholder -->
    <div
      v-if="loading && !hasRoles"
      :style="{ color: 'var(--p-text-muted-color)', fontSize: '0.82rem' }"
      data-testid="cost-by-role-loading"
    >
      Loading costs…
    </div>

    <!-- Per-role bars -->
    <div v-else-if="hasRoles" class="flex flex-col gap-3" data-testid="cost-by-role-bars">
      <div
        v-for="role in breakdown.roles"
        :key="role.role"
        class="flex flex-col gap-1"
        :data-role="role.role"
        data-testid="cost-by-role-row"
      >
        <div class="flex items-center justify-between">
          <span :style="{ fontSize: '0.82rem', fontWeight: 500 }">{{ role.label }}</span>
          <span
            class="font-mono"
            :style="{ fontSize: '0.78rem', color: 'var(--p-text-color)' }"
            data-testid="cost-by-role-amount"
          >
            {{ formatCostUSD(role.costUsd) }}
          </span>
        </div>
        <!-- Bar track + fill (fill width = fraction of largest role). -->
        <div
          class="rounded-full overflow-hidden"
          :style="{ height: '0.4rem', backgroundColor: 'var(--p-surface-200)' }"
        >
          <div
            class="rounded-full h-full"
            :style="{
              width: `${Math.max(2, role.fraction * 100)}%`,
              backgroundColor: 'var(--status-accent-color)',
            }"
            data-testid="cost-by-role-fill"
          />
        </div>
      </div>
    </div>

    <!-- Graceful gap note: total present but no per-step breakdown to attribute. -->
    <p
      v-else
      :style="{ color: 'var(--p-text-muted-color)', fontSize: '0.8rem' }"
      data-testid="cost-by-role-empty"
    >
      Per-role breakdown unavailable yet.
    </p>

    <!-- Total this run (real rollup — fix #3). -->
    <div
      class="flex items-center justify-between pt-2"
      :style="{ borderTop: '1px solid var(--p-surface-200)' }"
    >
      <span :style="{ fontSize: '0.82rem', fontWeight: 600 }">Total this run</span>
      <CostTicker :value="breakdown.total" data-testid="cost-by-role-total" />
    </div>
  </section>
</template>
