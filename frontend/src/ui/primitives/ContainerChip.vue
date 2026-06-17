<script setup lang="ts">
import { computed } from 'vue'

/**
 * ContainerChip — renders a container identity as `ctr·a3f9 · isolated`.
 *
 * Full machine voice (mono). The id is shortened to a short hash for density;
 * pass the full id and an optional isolation label. Dumb + prop-driven.
 */
const props = withDefaults(
  defineProps<{
    /** Full container id (e.g. docker id or name). */
    containerId: string
    /** Isolation descriptor, e.g. "isolated", "shared". Default "isolated". */
    isolation?: string | null
    /** Characters of the id to show (from the end). Default 4. */
    shortLength?: number
  }>(),
  { isolation: 'isolated', shortLength: 4 },
)

/** Short, stable suffix of the container id (machine convention `ctr·<hash>`). */
const shortId = computed(() => {
  const id = props.containerId ?? ''
  if (id.length <= props.shortLength) return id
  return id.slice(-props.shortLength)
})
</script>

<template>
  <span
    class="font-mono inline-flex items-center gap-1 px-2 py-0.5 rounded-md"
    data-testid="container-chip"
    :title="containerId"
    :style="{
      fontSize: '0.72rem',
      backgroundColor: 'var(--surface-overlay)',
      border: '1px solid var(--surface-border)',
      color: 'var(--p-text-muted-color)',
    }"
  >
    <i class="pi pi-box" :style="{ fontSize: '0.7rem' }" aria-hidden="true" />
    <span data-testid="container-chip-id">ctr·{{ shortId }}</span>
    <span v-if="isolation" data-testid="container-chip-isolation">· {{ isolation }}</span>
  </span>
</template>
