<script setup lang="ts">
import Skeleton from 'primevue/skeleton'
import { formatTokenCount } from '@/utils/formatCost'

const props = defineProps<{
  label: string
  value: string
  isLoading: boolean
  tokensInput?: number
  tokensOutput?: number
}>()
</script>

<template>
  <div
    class="rounded-lg p-4"
    :style="{ background: 'var(--surface-raised)', border: '1px solid var(--surface-border)' }"
  >
    <p class="mb-1 text-sm" :style="{ color: 'var(--p-text-muted-color)' }">{{ label }}</p>
    <Skeleton v-if="isLoading" width="6rem" height="1.75rem" />
    <template v-else>
      <p
        class="font-mono text-2xl font-bold"
        :style="{ color: 'var(--status-done-color)' }"
      >
        {{ value }}
      </p>
      <div
        v-if="props.tokensInput !== undefined || props.tokensOutput !== undefined"
        class="mt-1 text-sm"
        :style="{ color: 'var(--p-text-muted-color)' }"
      >
        <span>In: {{ formatTokenCount(props.tokensInput ?? 0) }}</span>
        <span class="ml-2">Out: {{ formatTokenCount(props.tokensOutput ?? 0) }}</span>
      </div>
    </template>
  </div>
</template>
