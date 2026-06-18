<script setup lang="ts">
import { computed } from 'vue'
import Tag from 'primevue/tag'
import type { components } from '@/api/schema'

type EpicRunStory = components['schemas']['EpicRunStory']

const props = defineProps<{
  stories: EpicRunStory[]
}>()

interface StoryGroup {
  index: number
  stories: EpicRunStory[]
  status: 'secondary' | 'info' | 'success' | 'danger'
}

/** Determine aggregate status severity for a group of stories */
function groupStatus(stories: EpicRunStory[]): 'secondary' | 'info' | 'success' | 'danger' {
  if (stories.some((s) => s.status === 'failed')) return 'danger'
  if (stories.some((s) => s.status === 'running')) return 'info'
  if (stories.every((s) => s.status === 'completed')) return 'success'
  return 'secondary'
}

function groupItemStyle(status: StoryGroup['status']): Record<string, string> {
  if (status === 'info') {
    return {
      background: 'var(--status-accent-surface, color-mix(in srgb, var(--status-accent-color) 12%, transparent))',
      borderColor: 'var(--status-accent-color)',
    }
  }
  return {
    background: 'var(--surface-raised)',
    borderColor: 'var(--surface-border)',
  }
}

const groups = computed<StoryGroup[]>(() => {
  const map = new Map<number, EpicRunStory[]>()
  for (const story of props.stories) {
    const list = map.get(story.group_index) ?? []
    list.push(story)
    map.set(story.group_index, list)
  }
  return Array.from(map.entries())
    .sort(([a], [b]) => a - b)
    .map(([index, stories]) => ({
      index,
      stories,
      status: groupStatus(stories),
    }))
})
</script>

<template>
  <div class="flex flex-col gap-2">
    <h2 class="text-lg font-semibold">Execution Layers</h2>
    <div
      v-for="group in groups"
      :key="group.index"
      class="flex items-center gap-3 p-3 rounded border"
      :style="groupItemStyle(group.status)"
    >
      <span class="font-medium">Layer {{ group.index }}</span>
      <span class="text-sm" :style="{ color: 'var(--p-text-muted-color)' }">{{ group.stories.length }} stories</span>
      <Tag
        :value="
          group.status === 'danger'
            ? 'failed'
            : group.status === 'info'
              ? 'running'
              : group.status === 'success'
                ? 'completed'
                : 'pending'
        "
        :severity="group.status"
        class="text-xs"
      />
    </div>
    <p v-if="groups.length === 0" class="text-sm" :style="{ color: 'var(--p-text-muted-color)' }">No execution layers</p>
  </div>
</template>
