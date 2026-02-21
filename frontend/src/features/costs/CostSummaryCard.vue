<script setup lang="ts">
import Skeleton from 'primevue/skeleton'

defineProps<{
  label: string
  value: string
  isLoading: boolean
}>()
</script>

<template>
  <div class="rounded-lg border border-surface-200 bg-surface-0 p-4">
    <p class="mb-1 text-sm text-surface-500">{{ label }}</p>
    <Skeleton v-if="isLoading" width="6rem" height="1.75rem" />
    <p v-else class="text-2xl font-bold">{{ value }}</p>
  </div>
</template>
