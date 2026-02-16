<script setup lang="ts">
import { computed } from 'vue'
import Card from 'primevue/card'
import Tag from 'primevue/tag'
import Badge from 'primevue/badge'
import type { Epic } from '@/stores/epics'

const props = defineProps<{
  epic: Epic
}>()

const emit = defineEmits<{
  click: [epic: Epic]
}>()

/** Map story status to PrimeVue severity for Tag color */
type TagSeverity = 'success' | 'info' | 'danger' | 'secondary' | 'warn' | 'contrast' | undefined

interface StatusEntry {
  label: string
  count: number
  severity: TagSeverity
}

const statusEntries = computed<StatusEntry[]>(() => {
  const c = props.epic.story_counts
  return [
    { label: 'Done', count: c.done, severity: 'success' as TagSeverity },
    { label: 'Running', count: c.running, severity: 'info' as TagSeverity },
    { label: 'Backlog', count: c.backlog, severity: 'secondary' as TagSeverity },
    { label: 'Failed', count: c.failed, severity: 'danger' as TagSeverity },
  ].filter((e) => e.count > 0)
})

/** Map epic status to a display label */
const epicStatusLabel = computed(() => {
  const map: Record<string, string> = {
    open: 'Open',
    in_progress: 'In Progress',
    completed: 'Completed',
  }
  return map[props.epic.status] ?? props.epic.status
})

/** Map epic status to PrimeVue Tag severity */
const epicStatusSeverity = computed<TagSeverity>(() => {
  const map: Record<string, TagSeverity> = {
    open: 'secondary',
    in_progress: 'info',
    completed: 'success',
  }
  return map[props.epic.status] ?? 'secondary'
})

/** Progress percentage based on done stories */
const progressPercent = computed(() => {
  const total = props.epic.story_counts.total
  if (total === 0) return 0
  return Math.round((props.epic.story_counts.done / total) * 100)
})
</script>

<template>
  <Card
    class="cursor-pointer transition-shadow hover:shadow-md"
    @click="emit('click', epic)"
  >
    <template #header>
      <div class="flex items-center justify-between px-4 pt-4">
        <Tag :value="epicStatusLabel" :severity="epicStatusSeverity" />
        <Badge :value="String(epic.story_counts.total)" severity="secondary" />
      </div>
    </template>
    <template #title>
      {{ epic.name }}
    </template>
    <template #subtitle>
      <span v-if="epic.description" class="line-clamp-2">{{ epic.description }}</span>
    </template>
    <template #content>
      <div class="flex flex-col gap-3">
        <!-- Progress bar -->
        <div class="flex items-center gap-2">
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-surface-200">
            <div
              class="h-full rounded-full bg-green-500 transition-all"
              :style="{ width: `${progressPercent}%` }"
            />
          </div>
          <span class="text-sm text-surface-500">{{ progressPercent }}%</span>
        </div>

        <!-- Story count tags -->
        <div class="flex flex-wrap gap-2">
          <Tag
            v-for="entry in statusEntries"
            :key="entry.label"
            :value="`${entry.count} ${entry.label}`"
            :severity="entry.severity"
          />
        </div>
      </div>
    </template>
  </Card>
</template>
