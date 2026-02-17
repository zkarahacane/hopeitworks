<script setup lang="ts">
import Badge from 'primevue/badge'
import Tag from 'primevue/tag'
import type { Story } from '@/stores/stories'
import { renderMarkdown } from '@/utils/renderMarkdown'
import RunLaunchButton from '@/features/runs/RunLaunchButton.vue'

const props = defineProps<{
  story: Story | null
  allStories?: Story[]
  projectId?: string
  showLaunchButton?: boolean
}>()

const emit = defineEmits<{
  'select-dependency': [storyId: string]
  'launch-click': []
}>()

const severityMap: Record<string, 'secondary' | 'info' | 'success' | 'danger'> = {
  backlog: 'secondary',
  running: 'info',
  done: 'success',
  failed: 'danger',
}

const scopeSeverityMap: Record<string, 'info' | 'warn' | 'secondary'> = {
  backend: 'info',
  frontend: 'warn',
  shared: 'secondary',
}

function handleDependencyClick(key: string) {
  const match = props.allStories?.find((s) => s.key === key)
  if (match) emit('select-dependency', match.id)
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
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <span
            style="font-family: monospace; font-size: 1rem; color: var(--p-text-muted-color)"
          >
            {{ story.key }}
          </span>
          <Badge
            :value="story.status"
            :severity="severityMap[story.status] ?? 'secondary'"
          />
          <Tag
            v-if="story.scope"
            :value="story.scope"
            :severity="scopeSeverityMap[story.scope] ?? 'secondary'"
          />
        </div>
        <RunLaunchButton
          v-if="showLaunchButton"
          :story-id="story.id"
          :story-key="story.key"
          :story-title="story.title"
          :status="story.status"
          @launch-click="emit('launch-click')"
        />
      </div>

      <h2 class="m-0" style="font-size: 1.25rem; font-weight: 600">{{ story.title }}</h2>

      <div v-if="story.objective" class="flex flex-col gap-1">
        <h3
          class="m-0"
          style="
            font-size: 0.85rem;
            font-weight: 600;
            text-transform: uppercase;
            color: var(--p-text-muted-color);
          "
        >
          Objective
        </h3>
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div class="prose-content" v-html="renderMarkdown(story.objective)" />
      </div>

      <div v-if="story.acceptance_criteria" class="flex flex-col gap-1">
        <h3
          class="m-0"
          style="
            font-size: 0.85rem;
            font-weight: 600;
            text-transform: uppercase;
            color: var(--p-text-muted-color);
          "
        >
          Acceptance Criteria
        </h3>
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div class="prose-content" v-html="renderMarkdown(story.acceptance_criteria)" />
      </div>

      <div
        v-if="story.target_files && story.target_files.length > 0"
        class="flex flex-col gap-1"
      >
        <h3
          class="m-0"
          style="
            font-size: 0.85rem;
            font-weight: 600;
            text-transform: uppercase;
            color: var(--p-text-muted-color);
          "
        >
          Target Files
        </h3>
        <ul class="m-0 pl-5">
          <li
            v-for="file in story.target_files"
            :key="file"
            style="font-family: monospace; font-size: 0.85rem"
          >
            {{ file }}
          </li>
        </ul>
      </div>

      <div
        v-if="story.depends_on && story.depends_on.length > 0"
        class="flex flex-col gap-1"
      >
        <h3
          class="m-0"
          style="
            font-size: 0.85rem;
            font-weight: 600;
            text-transform: uppercase;
            color: var(--p-text-muted-color);
          "
        >
          Dependencies
        </h3>
        <ul class="m-0 pl-5">
          <li
            v-for="dep in story.depends_on"
            :key="dep"
            style="font-family: monospace; font-size: 0.85rem"
          >
            <button
              type="button"
              style="
                background: none;
                border: none;
                padding: 0;
                font-family: monospace;
                font-size: 0.85rem;
                color: var(--p-primary-color);
                cursor: pointer;
                text-decoration: underline;
              "
              @click="handleDependencyClick(dep)"
            >
              {{ dep }}
            </button>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.prose-content :deep(p) {
  margin: 0.25rem 0;
}

.prose-content :deep(ul),
.prose-content :deep(ol) {
  margin: 0.25rem 0;
  padding-left: 1.25rem;
}

.prose-content :deep(code) {
  font-size: 0.85rem;
  padding: 0.1rem 0.3rem;
  border-radius: 0.25rem;
  background-color: var(--p-surface-100);
}

.prose-content :deep(pre) {
  margin: 0.5rem 0;
  padding: 0.75rem;
  border-radius: 0.375rem;
  background-color: var(--p-surface-100);
  overflow-x: auto;
}

.prose-content :deep(pre code) {
  padding: 0;
  background: none;
}

.prose-content :deep(h1),
.prose-content :deep(h2),
.prose-content :deep(h3),
.prose-content :deep(h4) {
  margin: 0.5rem 0 0.25rem;
  font-weight: 600;
}

.prose-content :deep(strong) {
  font-weight: 600;
}
</style>
