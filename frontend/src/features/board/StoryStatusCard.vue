<script setup lang="ts">
import type { Story } from '@/stores/stories'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'

defineProps<{
  story: Story
  isSelected: boolean
}>()

const emit = defineEmits<{
  click: [storyId: string]
}>()
</script>

<template>
  <div
    class="story-card flex flex-col gap-2 p-3 cursor-pointer"
    :class="isSelected ? 'story-card--selected' : 'story-card--default'"
    role="button"
    tabindex="0"
    :aria-label="`Story: ${story.key} - ${story.title}`"
    :aria-selected="isSelected"
    @click="emit('click', story.id)"
    @keydown.enter="emit('click', story.id)"
  >
    <div class="flex items-center justify-between gap-2">
      <span class="font-mono text-xs" style="color: var(--p-text-muted-color)">
        {{ story.key }}
      </span>
      <StatusBadge
        :status="story.status"
        :animated="story.status === 'running'"
        :icon="false"
      />
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

<style scoped>
.story-card {
  border-radius: var(--p-border-radius);
  transition: border-color 0.2s, background-color 0.2s;
}
.story-card--selected {
  border: 2px solid var(--p-primary-color);
  background: var(--p-primary-50);
}
.story-card--default {
  border: 1px solid var(--p-surface-200);
  background: var(--p-surface-0);
}
.story-card--default:hover {
  background: var(--p-surface-50);
}
</style>
