<script setup lang="ts">
import { computed, toRef } from 'vue'
import { useCountUp } from '@/composables/useCountUp'
import { formatCostUSD } from '@/utils/formatCost'

/**
 * CostTicker — a live, count-up USD figure in machine voice (mono).
 *
 * Pass a reactive `value` (USD); it smoothly counts up to each new target via
 * useCountUp. Dumb + prop-driven — hero screens feed it the running cost from
 * `useRuntimeStream.runCostUsd(runId)` (or a REST total). No data access here.
 */
const props = withDefaults(
  defineProps<{
    /** Current USD value to display. */
    value: number
    /** Animate the count-up. Disable for static/historical figures. */
    animated?: boolean
    /** Count-up duration in ms. */
    durationMs?: number
  }>(),
  { animated: true, durationMs: 600 },
)

const target = toRef(props, 'value')
const { current } = useCountUp(target, {
  durationMs: props.animated ? props.durationMs : 0,
})

const displayValue = computed(() => (props.animated ? current.value : props.value))
const formatted = computed(() => formatCostUSD(displayValue.value))
</script>

<template>
  <span
    class="font-mono inline-flex items-center gap-1"
    data-testid="cost-ticker"
    :style="{ fontVariantNumeric: 'tabular-nums' }"
  >
    <i class="pi pi-dollar" :style="{ fontSize: '0.7rem', color: 'var(--p-text-muted-color)' }" aria-hidden="true" />
    <span data-testid="cost-ticker-value">{{ formatted }}</span>
  </span>
</template>
