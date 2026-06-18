<script setup lang="ts">
import { computed } from 'vue'

/**
 * AgentChip — identifies the agent doing the work: role + model.
 *
 * Machine voice (mono) for the model id; role reads as UI chrome. Dumb +
 * prop-driven. Used wherever a step/run shows which agent ran it.
 */
const props = withDefaults(
  defineProps<{
    /** Agent role, e.g. "dev", "review", "test". */
    role: string
    /** Model id, e.g. "claude-opus-4-8". Rendered in mono. */
    model?: string | null
    /** Optional provider, e.g. "anthropic". */
    provider?: string | null
  }>(),
  { model: null, provider: null },
)

const hasModel = computed(() => !!props.model)
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md"
    data-testid="agent-chip"
    :style="{
      backgroundColor: 'var(--surface-overlay)',
      border: '1px solid var(--surface-border)',
    }"
  >
    <i class="pi pi-microchip-ai" :style="{ fontSize: '0.75rem', color: 'var(--p-text-muted-color)' }" aria-hidden="true" />
    <span :style="{ fontSize: '0.8rem', fontWeight: 500 }" data-testid="agent-chip-role">{{ role }}</span>
    <span
      v-if="hasModel"
      class="font-mono"
      :style="{ fontSize: '0.72rem', color: 'var(--p-text-muted-color)' }"
      data-testid="agent-chip-model"
    >
      {{ model }}
    </span>
    <span
      v-if="provider"
      :style="{ fontSize: '0.7rem', color: 'var(--p-text-muted-color)', opacity: 0.7 }"
      data-testid="agent-chip-provider"
    >
      · {{ provider }}
    </span>
  </span>
</template>
