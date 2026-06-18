<script setup lang="ts">
import { computed } from 'vue'
import { statusFamily } from '@/utils/statusToken'
import type { Phase } from '@/utils/phaseGroup'
import { phaseLabel, PHASES } from '@/utils/phaseGroup'

/**
 * PhaseGroup — a labelled section header for one timeline phase
 * (Setup / Dev / Review / Delivery). Shows the phase icon, label, a count, and
 * a roll-up tint derived from the worst child status. Dumb + prop-driven;
 * renders its steps via the default slot.
 */
const props = withDefaults(
  defineProps<{
    phase: Phase
    /** Number of steps in this phase. */
    count: number
    /**
     * Roll-up status string for the phase (e.g. the most-significant child
     * status). Drives the header tint via statusFamily. Optional.
     */
    rollupStatus?: string | null
  }>(),
  { rollupStatus: null },
)

const def = computed(() => PHASES.find((p) => p.key === props.phase))
const label = computed(() => phaseLabel(props.phase))
const family = computed(() => statusFamily(props.rollupStatus))
const tintStyle = computed(() =>
  props.rollupStatus
    ? { color: `var(--status-${family.value}-color)` }
    : { color: 'var(--p-text-muted-color)' },
)
</script>

<template>
  <section class="flex flex-col gap-2" :data-phase="phase" data-testid="phase-group">
    <header class="flex items-center gap-2">
      <i :class="def?.icon" :style="tintStyle" aria-hidden="true" />
      <span :style="{ fontWeight: 600, fontSize: '0.8rem' }" data-testid="phase-group-label">
        {{ label }}
      </span>
      <span
        class="font-mono"
        :style="{ fontSize: '0.7rem', color: 'var(--p-text-muted-color)' }"
        data-testid="phase-group-count"
      >
        {{ count }}
      </span>
    </header>
    <div class="flex flex-col gap-1">
      <slot />
    </div>
  </section>
</template>
