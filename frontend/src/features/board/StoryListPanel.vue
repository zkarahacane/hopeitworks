<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Story, StoryFilters } from '@/stores/stories'
import StoryStatusCard from './StoryStatusCard.vue'
import StoryFilterBar from './StoryFilterBar.vue'
import { useKeyboard } from '@/composables/useKeyboard'

const props = defineProps<{
  stories: Story[]
  selectedId: string | null
  filters: StoryFilters
}>()

const emit = defineEmits<{
  select: [storyId: string]
  'update:filters': [filters: StoryFilters]
}>()

const selectedIndex = ref(0)

/** Sync selection index when selectedId changes externally */
watch(
  () => props.selectedId,
  (newId) => {
    if (newId) {
      const idx = props.stories.findIndex((s) => s.id === newId)
      if (idx >= 0) {
        selectedIndex.value = idx
      }
    }
  },
)

/** Reset index when stories list changes */
watch(
  () => props.stories.length,
  () => {
    if (selectedIndex.value >= props.stories.length) {
      selectedIndex.value = Math.max(0, props.stories.length - 1)
    }
  },
)

useKeyboard({
  j: () => {
    if (props.stories.length === 0) return
    selectedIndex.value = Math.min(selectedIndex.value + 1, props.stories.length - 1)
  },
  k: () => {
    if (props.stories.length === 0) return
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
  },
  Enter: () => {
    if (props.stories.length === 0) return
    const story = props.stories[selectedIndex.value]
    if (story) {
      emit('select', story.id)
    }
  },
})

function handleCardClick(storyId: string) {
  const idx = props.stories.findIndex((s) => s.id === storyId)
  if (idx >= 0) {
    selectedIndex.value = idx
  }
  emit('select', storyId)
}

function handleFilterUpdate(newFilters: StoryFilters) {
  emit('update:filters', newFilters)
}
</script>

<template>
  <div class="flex flex-col gap-3 h-full">
    <StoryFilterBar :model-value="filters" @update:model-value="handleFilterUpdate" />
    <div
      class="flex flex-col gap-1 overflow-y-auto"
      style="flex: 1; min-height: 0"
      role="listbox"
      aria-label="Story list"
    >
      <p
        v-if="stories.length === 0"
        class="p-4 text-center"
        style="color: var(--p-text-muted-color)"
      >
        No stories match the current filters
      </p>
      <StoryStatusCard
        v-for="(story, index) in stories"
        :key="story.id"
        :story="story"
        :is-selected="story.id === selectedId || (selectedId === null && index === selectedIndex)"
        @click="handleCardClick"
      />
    </div>
  </div>
</template>
