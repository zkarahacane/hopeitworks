<script setup lang="ts">
import { computed } from 'vue'
import ProgressBar from 'primevue/progressbar'

/**
 * LiveProgress — thin wrapper over PrimeVue ProgressBar.
 *
 * Determinate when `value` (0–100) is provided; indeterminate otherwise (e.g. a
 * running step with unknown duration). Dumb + prop-driven.
 */
const props = withDefaults(
  defineProps<{
    /** Percent complete (0–100). When null/undefined → indeterminate. */
    value?: number | null
    /** Force indeterminate regardless of value. */
    indeterminate?: boolean
    /** Show the % label (determinate only). */
    showValue?: boolean
  }>(),
  { value: null, indeterminate: false, showValue: false },
)

const isIndeterminate = computed(
  () => props.indeterminate || props.value === null || props.value === undefined,
)

const clamped = computed(() => {
  const v = props.value ?? 0
  return Math.min(100, Math.max(0, v))
})
</script>

<template>
  <ProgressBar
    v-if="isIndeterminate"
    mode="indeterminate"
    :style="{ height: '0.375rem' }"
    data-testid="live-progress-indeterminate"
  />
  <ProgressBar
    v-else
    :value="clamped"
    :show-value="showValue"
    :style="{ height: showValue ? '1.25rem' : '0.375rem' }"
    data-testid="live-progress-determinate"
  />
</template>
