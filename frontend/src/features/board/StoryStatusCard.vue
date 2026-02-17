<script setup lang="ts">
import { computed } from 'vue'
import Badge from 'primevue/badge'
import { useToast } from 'primevue/usetoast'
import type { Story } from '@/stores/stories'
import RunStatusIndicator from './RunStatusIndicator.vue'
import type { RunStatus } from './RunStatusIndicator.vue'

const props = defineProps<{
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

const toast = useToast()

/** Derive run status from the latest_run field */
const runStatus = computed<RunStatus>(() => {
  if (!props.story.latest_run) return 'backlog'
  return props.story.latest_run.status === 'cancelled'
    ? 'failed'
    : (props.story.latest_run.status as RunStatus)
})

function handleErrorClick() {
  if (props.story.latest_run?.error_message) {
    toast.add({
      severity: 'error',
      summary: 'Run Failed',
      detail: props.story.latest_run.error_message,
      life: 5000,
    })
  }
}
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
    <RunStatusIndicator
      :status="runStatus"
      :completed-at="story.latest_run?.completed_at"
      :error-message="story.latest_run?.error_message"
      @error-click="handleErrorClick"
    />
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
