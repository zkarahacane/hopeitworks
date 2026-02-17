<script setup lang="ts">
import Badge from 'primevue/badge'
import type { Story } from '@/stores/stories'

defineProps<{
  story: Story | null
}>()

const severityMap: Record<string, 'secondary' | 'info' | 'success' | 'danger'> = {
  backlog: 'secondary',
  running: 'info',
  done: 'success',
  failed: 'danger',
}
</script>

<template>
  <div class="h-full overflow-y-auto">
    <div
      v-if="!story"
      class="flex items-center justify-center h-full"
      style="color: var(--p-text-muted-color)"
    >
      <div class="flex flex-col items-center gap-3">
        <i class="pi pi-book" style="font-size: 2.5rem" />
        <p style="font-size: 1rem">Select a story to view details</p>
      </div>
    </div>

    <div v-else class="flex flex-col gap-4 p-4">
      <div class="flex items-center gap-3">
        <span style="font-family: monospace; font-size: 1rem; color: var(--p-text-muted-color)">
          {{ story.key }}
        </span>
        <Badge :value="story.status" :severity="severityMap[story.status] ?? 'secondary'" />
      </div>

      <h2 class="m-0" style="font-size: 1.25rem; font-weight: 600">{{ story.title }}</h2>

      <div v-if="story.objective" class="flex flex-col gap-1">
        <h3 class="m-0" style="font-size: 0.85rem; font-weight: 600; text-transform: uppercase; color: var(--p-text-muted-color)">
          Objective
        </h3>
        <p class="m-0" style="white-space: pre-wrap">{{ story.objective }}</p>
      </div>

      <div v-if="story.acceptance_criteria" class="flex flex-col gap-1">
        <h3 class="m-0" style="font-size: 0.85rem; font-weight: 600; text-transform: uppercase; color: var(--p-text-muted-color)">
          Acceptance Criteria
        </h3>
        <p class="m-0" style="white-space: pre-wrap">{{ story.acceptance_criteria }}</p>
      </div>

      <div v-if="story.target_files && story.target_files.length > 0" class="flex flex-col gap-1">
        <h3 class="m-0" style="font-size: 0.85rem; font-weight: 600; text-transform: uppercase; color: var(--p-text-muted-color)">
          Target Files
        </h3>
        <ul class="m-0 pl-5">
          <li v-for="file in story.target_files" :key="file" style="font-family: monospace; font-size: 0.85rem">
            {{ file }}
          </li>
        </ul>
      </div>

      <div v-if="story.depends_on && story.depends_on.length > 0" class="flex flex-col gap-1">
        <h3 class="m-0" style="font-size: 0.85rem; font-weight: 600; text-transform: uppercase; color: var(--p-text-muted-color)">
          Dependencies
        </h3>
        <ul class="m-0 pl-5">
          <li v-for="dep in story.depends_on" :key="dep" style="font-family: monospace; font-size: 0.85rem">
            {{ dep }}
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
