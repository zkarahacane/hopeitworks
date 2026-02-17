<script setup lang="ts">
import Badge from 'primevue/badge'
import type { Story } from '@/stores/stories'

defineProps<{
  story: Story
  isSelected: boolean
}>()

const emit = defineEmits<{
  click: [storyId: string]
}>()

const severityMap: Record<string, 'secondary' | 'info' | 'success' | 'danger'> = {
  backlog: 'secondary',
  running: 'info',
  done: 'success',
  failed: 'danger',
}
</script>

<template>
  <div
    class="flex flex-col gap-2 p-3 cursor-pointer"
    role="button"
    tabindex="0"
    :aria-label="`Story: ${story.key} - ${story.title}`"
    :aria-selected="isSelected"
    :style="{
      border: isSelected ? '2px solid var(--p-primary-color)' : '1px solid var(--p-surface-200)',
      borderRadius: 'var(--p-border-radius)',
      background: isSelected ? 'var(--p-primary-50)' : 'var(--p-surface-0)',
      transition: 'border-color 0.2s, background-color 0.2s',
    }"
    @click="emit('click', story.id)"
    @keydown.enter="emit('click', story.id)"
    @mouseenter="!isSelected && (($event.currentTarget as HTMLElement).style.background = 'var(--p-surface-50)')"
    @mouseleave="!isSelected && (($event.currentTarget as HTMLElement).style.background = 'var(--p-surface-0)')"
  >
    <div class="flex items-center justify-between gap-2">
      <span style="font-family: monospace; font-size: 0.8rem; color: var(--p-text-muted-color)">
        {{ story.key }}
      </span>
      <Badge :value="story.status" :severity="severityMap[story.status] ?? 'secondary'" />
    </div>
    <span
      style="
        font-size: 0.9rem;
        font-weight: 500;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
      "
    >
      {{ story.title }}
    </span>
  </div>
</template>
