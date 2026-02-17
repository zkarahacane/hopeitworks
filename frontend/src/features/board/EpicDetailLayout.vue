<script setup lang="ts">
import type { Story, StoryFilters } from '@/stores/stories'
import StoryListPanel from './StoryListPanel.vue'
import StoryDetailPanel from './StoryDetailPanel.vue'

defineProps<{
  stories: Story[]
  allStories: Story[]
  selectedStory: Story | null
  selectedStoryId: string | null
  filters: StoryFilters
  projectId: string
}>()

const emit = defineEmits<{
  select: [storyId: string]
  'update:filters': [filters: StoryFilters]
  'launch-click': []
}>()
</script>

<template>
  <div class="flex h-full gap-4">
    <div class="w-[300px] shrink-0 overflow-y-auto">
      <StoryListPanel
        :stories="stories"
        :selected-id="selectedStoryId"
        :filters="filters"
        @select="emit('select', $event)"
        @update:filters="emit('update:filters', $event)"
      />
    </div>
    <div class="flex-1 overflow-y-auto border-l border-surface-200">
      <StoryDetailPanel
        :story="selectedStory"
        :all-stories="allStories"
        :project-id="projectId"
        :show-launch-button="true"
        @select-dependency="emit('select', $event)"
        @launch-click="emit('launch-click')"
      />
    </div>
  </div>
</template>
