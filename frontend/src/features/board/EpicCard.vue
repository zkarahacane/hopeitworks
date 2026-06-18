<script setup lang="ts">
import Badge from 'primevue/badge'
import type { Epic } from '@/stores/epics'
import { statusTokenSeverity } from '@/utils/statusToken'

defineProps<{
  epic: Epic
}>()

const emit = defineEmits<{
  click: [epicId: string]
}>()

// Severities route through the unified statusToken system.
const statusConfig = [
  { key: 'backlog', label: 'Backlog' },
  { key: 'running', label: 'Running' },
  { key: 'done', label: 'Done' },
  { key: 'failed', label: 'Failed' },
] as const
</script>

<template>
  <div
    class="flex flex-col gap-3 p-4 cursor-pointer"
    role="button"
    tabindex="0"
    :aria-label="`Epic: ${epic.name}`"
    @click="emit('click', epic.id)"
    @keydown.enter="emit('click', epic.id)"
    style="
      border: 1px solid var(--surface-border);
      border-radius: var(--p-border-radius);
      background: var(--surface-raised);
      transition: box-shadow 0.2s;
    "
    @mouseenter="($event.currentTarget as HTMLElement).style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)'"
    @mouseleave="($event.currentTarget as HTMLElement).style.boxShadow = 'none'"
  >
    <h3 class="m-0" style="font-size: 1.1rem; font-weight: 600">{{ epic.name }}</h3>
    <p
      v-if="epic.description"
      class="m-0"
      style="
        color: var(--p-text-muted-color);
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
      "
    >
      {{ epic.description }}
    </p>
    <div class="flex flex-wrap gap-2">
      <span
        v-for="status in statusConfig"
        :key="status.key"
        class="flex items-center gap-1"
      >
        <Badge
          :value="String(epic.story_counts[status.key])"
          :severity="statusTokenSeverity(status.key)"
        />
        <span style="font-size: 0.8rem; color: var(--p-text-muted-color)">{{ status.label }}</span>
      </span>
    </div>
  </div>
</template>
