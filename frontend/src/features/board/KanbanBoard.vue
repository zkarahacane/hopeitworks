<script setup lang="ts">
import { computed } from 'vue'
import Badge from 'primevue/badge'
import ProgressSpinner from 'primevue/progressspinner'
import Tag from 'primevue/tag'
import type { Story, KanbanColumn } from '@/stores/stories'
import { boardColumn } from '@/stores/stories'

const props = defineProps<{
  stories: Story[]
  selectedId: string | null
}>()

const emit = defineEmits<{
  select: [storyId: string]
}>()

// ── Column definitions ──────────────────────────────────────────────────────

interface ColumnDef {
  key: KanbanColumn
  label: string
  severity: 'secondary' | 'info' | 'warn' | 'success' | 'danger'
}

const COLUMNS: ColumnDef[] = [
  { key: 'backlog', label: 'Backlog', severity: 'secondary' },
  { key: 'in_progress', label: 'In Progress', severity: 'info' },
  { key: 'blocked', label: 'Blocked', severity: 'warn' },
  { key: 'done', label: 'Done', severity: 'success' },
  { key: 'failed', label: 'Failed', severity: 'danger' },
]

// ── Grouping ────────────────────────────────────────────────────────────────

const grouped = computed((): Record<KanbanColumn, Story[]> => {
  const result: Record<KanbanColumn, Story[]> = {
    backlog: [],
    in_progress: [],
    blocked: [],
    done: [],
    failed: [],
  }
  for (const story of props.stories) {
    result[boardColumn(story)].push(story)
  }
  return result
})

// ── Step progress display ───────────────────────────────────────────────────

function stepLabel(story: Story): string | null {
  const step = story.latest_run?.current_step
  if (!step) return null
  return step.name
}

function stepProgress(story: Story): string | null {
  const step = story.latest_run?.current_step
  if (!step || step.total === 0) return null
  // index is zero-based, display as 1-based
  return `${step.index + 1}/${step.total}`
}
</script>

<template>
  <div class="flex gap-3 h-full overflow-x-auto">
    <div
      v-for="col in COLUMNS"
      :key="col.key"
      class="flex flex-col gap-2 min-w-[220px] w-[220px] shrink-0"
    >
      <!-- Column header -->
      <div class="flex items-center gap-2 px-1 pb-1" style="border-bottom: 1px solid var(--p-surface-200)">
        <span style="font-weight: 600; font-size: 0.85rem">{{ col.label }}</span>
        <Tag
          :value="String(grouped[col.key].length)"
          :severity="col.severity"
          rounded
          style="font-size: 0.7rem; padding: 0.1rem 0.4rem"
        />
      </div>

      <!-- Story cards -->
      <div class="flex flex-col gap-2 overflow-y-auto flex-1">
        <div
          v-for="story in grouped[col.key]"
          :key="story.id"
          class="kanban-card flex flex-col gap-2 p-3 cursor-pointer"
          :class="story.id === selectedId ? 'kanban-card--selected' : 'kanban-card--default'"
          role="button"
          tabindex="0"
          :aria-label="`Story: ${story.key} - ${story.title}`"
          :aria-selected="story.id === selectedId"
          @click="emit('select', story.id)"
          @keydown.enter="emit('select', story.id)"
        >
          <!-- Key + status badge -->
          <div class="flex items-center justify-between gap-2">
            <span style="font-family: monospace; font-size: 0.75rem; color: var(--p-text-muted-color)">
              {{ story.key }}
            </span>
            <Badge
              :value="story.status"
              :severity="
                story.status === 'done' ? 'success'
                : story.status === 'failed' ? 'danger'
                : story.status === 'running' ? 'info'
                : 'secondary'
              "
            />
          </div>

          <!-- Title -->
          <span
            style="
              font-size: 0.85rem;
              font-weight: 500;
              display: -webkit-box;
              -webkit-line-clamp: 2;
              -webkit-box-orient: vertical;
              overflow: hidden;
            "
          >
            {{ story.title }}
          </span>

          <!-- Live step progress (only for in_progress / blocked) -->
          <div
            v-if="(col.key === 'in_progress' || col.key === 'blocked') && stepLabel(story)"
            class="flex items-center gap-2"
          >
            <ProgressSpinner
              v-if="col.key === 'in_progress'"
              style="width: 0.85rem; height: 0.85rem"
              stroke-width="4"
              aria-hidden="true"
            />
            <i
              v-else
              class="pi pi-clock"
              style="font-size: 0.85rem; color: var(--p-yellow-500)"
              aria-hidden="true"
            />
            <span style="font-size: 0.75rem; color: var(--p-text-muted-color)">
              {{ stepLabel(story) }}
              <span v-if="stepProgress(story)" style="color: var(--p-text-secondary-color)">
                ({{ stepProgress(story) }})
              </span>
            </span>
          </div>

          <!-- Blocked label when no step info yet -->
          <div
            v-else-if="col.key === 'blocked'"
            class="flex items-center gap-2"
          >
            <i class="pi pi-clock" style="font-size: 0.85rem; color: var(--p-yellow-500)" aria-hidden="true" />
            <span style="font-size: 0.75rem; color: var(--p-text-muted-color)">Waiting for approval</span>
          </div>
        </div>

        <!-- Empty column state -->
        <p
          v-if="grouped[col.key].length === 0"
          class="p-3 text-center"
          style="color: var(--p-text-muted-color); font-size: 0.8rem"
        >
          No stories
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.kanban-card {
  border-radius: var(--p-border-radius);
  transition: border-color 0.2s, background-color 0.2s;
}
.kanban-card--selected {
  border: 2px solid var(--p-primary-color);
  background: var(--p-primary-50);
}
.kanban-card--default {
  border: 1px solid var(--p-surface-200);
  background: var(--p-surface-0);
}
.kanban-card--default:hover {
  background: var(--p-surface-50);
}
</style>
