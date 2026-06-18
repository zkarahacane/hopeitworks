<script setup lang="ts">
import { computed } from 'vue'
import Tag from 'primevue/tag'
import { statusToken } from '@/utils/statusToken'

/**
 * StatusBadge — the canonical product-status pill.
 *
 * Consumes the unified `statusToken` system: pass ANY run/step/story/epic/hitl
 * status string and it renders a PrimeVue Tag with the family color token, icon,
 * and (optionally) a live pulse. Dumb + prop-driven; no data access.
 */
const props = withDefaults(
  defineProps<{
    /** Raw status string from any domain enum. */
    status: string | null | undefined
    /** Override the displayed label (defaults to the raw status). */
    label?: string
    /** Show the family icon. */
    icon?: boolean
    /** Animate (pulse/breathe) when the family is live. Set false for history rows. */
    animated?: boolean
    /** Treat gate/running as resolved (no pulse) — e.g. historical rows. */
    resolved?: boolean
  }>(),
  { icon: true, animated: true, resolved: false },
)

const token = computed(() => statusToken(props.status, { resolved: props.resolved }))
const displayLabel = computed(() => props.label ?? props.status ?? token.value.label)
const showPulse = computed(() => props.animated && token.value.pulse)

// Family color applied via the design token CSS variable (not a hardcoded hex).
const colorStyle = computed(() => ({
  color: `var(${token.value.colorToken})`,
  backgroundColor: `var(${token.value.surfaceToken})`,
}))

// Pulse class depends on family: running uses the dot pulse, gate breathes.
const pulseClass = computed(() =>
  token.value.family === 'gate' ? 'amber-breathe' : 'live-pulse',
)
</script>

<template>
  <Tag
    :value="displayLabel"
    rounded
    :data-family="token.family"
    data-testid="status-badge"
    :style="colorStyle"
  >
    <template #default>
      <span class="inline-flex items-center gap-1.5">
        <span
          v-if="showPulse"
          class="inline-block rounded-full"
          :class="pulseClass"
          :style="{ width: '0.5rem', height: '0.5rem', backgroundColor: `var(${token.colorToken})` }"
          data-testid="status-badge-pulse"
          aria-hidden="true"
        />
        <i
          v-else-if="icon"
          :class="token.icon"
          :style="{ fontSize: '0.75rem' }"
          data-testid="status-badge-icon"
          aria-hidden="true"
        />
        <span>{{ displayLabel }}</span>
      </span>
    </template>
  </Tag>
</template>
