<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import type { Story, KanbanColumn } from '@/stores/stories'
import { boardColumn } from '@/stores/stories'
import { statusTokenSeverity } from '@/utils/statusToken'
import { useRuntimeStream } from '@/stores/runtimeStream'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import ContainerChip from '@/ui/primitives/ContainerChip.vue'
import CostTicker from '@/ui/primitives/CostTicker.vue'

const props = defineProps<{
  stories: Story[]
  selectedId: string | null
  projectId: string
}>()

const emit = defineEmits<{
  select: [storyId: string]
}>()

const router = useRouter()
const stream = useRuntimeStream()

// ── Card style helpers ───────────────────────────────────────────────────────

function cardStyle(variant: 'default' | 'running' | 'gate', isSelected: boolean): Record<string, string> {
  if (isSelected) {
    return {
      border: '2px solid var(--p-primary-color)',
      background: 'var(--p-primary-50)',
      borderRadius: 'var(--p-border-radius)',
    }
  }
  const base = {
    borderRadius: 'var(--p-border-radius)',
    background: 'var(--surface-raised)',
  }
  if (variant === 'running') {
    return { ...base, border: '1px solid var(--surface-border)', borderLeft: '3px solid var(--status-running-color)' }
  }
  if (variant === 'gate') {
    return { ...base, border: '1px solid var(--surface-border)', borderLeft: '3px solid var(--status-gate-color)' }
  }
  return { ...base, border: '1px solid var(--surface-border)' }
}

// ── Column definitions ──────────────────────────────────────────────────────

interface ColumnDef {
  key: KanbanColumn
  label: string
}

const COLUMNS: ColumnDef[] = [
  { key: 'backlog', label: 'Backlog' },
  { key: 'in_progress', label: 'Running' },
  { key: 'blocked', label: 'In Review' },
  { key: 'done', label: 'Done' },
  { key: 'failed', label: 'Failed' },
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

// ── Timer formatting ─────────────────────────────────────────────────────────

function formatTimer(seconds: number): string {
  const m = Math.floor(seconds / 60).toString().padStart(2, '0')
  const s = (seconds % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

// ── Card helpers ─────────────────────────────────────────────────────────────

/** Latest run id for a story (used with runtimeStream). */
function runId(story: Story): string {
  return story.latest_run?.id ?? ''
}

/**
 * Pseudo-container id: we use the run id as a proxy since LatestRunStep
 * does not carry a container_id field (backend gap — container id is not
 * projected onto the kanban summary).
 */
function pseudoContainerId(story: Story): string {
  return story.latest_run?.id ?? story.id
}

/** Navigate to HITL approval for a blocked (in-review) story. */
function navigateToApproval(story: Story) {
  // hitl-approve route needs runId + stepId; use latest_run.current_step when available.
  const run = story.latest_run
  const step = run?.current_step
  if (run && step) {
    router.push({
      name: 'hitl-approve',
      params: { id: props.projectId, runId: run.id, stepId: step.id },
    })
  } else {
    // Fallback to story detail when step info not yet available
    router.push({
      name: 'story-detail',
      params: { projectId: props.projectId, storyId: story.id },
    })
  }
}

</script>

<template>
  <div class="flex gap-3 h-full overflow-x-auto">
    <div
      v-for="col in COLUMNS"
      :key="col.key"
      class="flex flex-col gap-2 min-w-[240px] w-[240px] shrink-0"
    >
      <!-- Column header -->
      <div class="flex items-center gap-2 px-1 pb-2" style="border-bottom: 1px solid var(--surface-border)">
        <span style="font-weight: 600; font-size: 0.85rem; font-family: var(--font-sans)">{{ col.label }}</span>
        <Tag
          :value="String(grouped[col.key].length)"
          :severity="statusTokenSeverity(col.key)"
          rounded
          style="font-size: 0.7rem; padding: 0.1rem 0.4rem"
        />
      </div>

      <!-- Story cards -->
      <div class="flex flex-col gap-2 overflow-y-auto flex-1">
        <!-- ── BACKLOG cards ───────────────────────────────────────────────── -->
        <template v-if="col.key === 'backlog'">
          <div
            v-for="story in grouped['backlog']"
            :key="story.id"
            class="kanban-card flex flex-col gap-2 p-3 cursor-pointer"
            :style="cardStyle('default', story.id === selectedId)"
            role="button"
            tabindex="0"
            :aria-label="`Story: ${story.key} - ${story.title}`"
            :aria-selected="story.id === selectedId"
            @click="emit('select', story.id)"
            @keydown.enter="emit('select', story.id)"
          >
            <div class="flex items-center justify-between gap-2">
              <StatusBadge status="backlog" :animated="false" />
              <span style="font-family: monospace; font-size: 0.75rem; color: var(--p-text-muted-color)">
                {{ story.key }}
              </span>
            </div>
            <span class="card-title">{{ story.title }}</span>
            <div
              v-if="story.depends_on && story.depends_on.length > 0"
              class="flex items-center gap-1"
            >
              <i class="pi pi-link" style="font-size: 0.7rem; color: var(--p-text-muted-color)" aria-hidden="true" />
              <span style="font-size: 0.72rem; color: var(--p-text-muted-color)">
                waiting on {{ story.depends_on.slice(0, 2).join(', ') }}
                <span v-if="story.depends_on.length > 2">+{{ story.depends_on.length - 2 }}</span>
              </span>
            </div>
          </div>
        </template>

        <!-- ── RUNNING (in_progress) cards ───────────────────────────────── -->
        <template v-else-if="col.key === 'in_progress'">
          <div
            v-for="story in grouped['in_progress']"
            :key="story.id"
            class="kanban-card flex flex-col gap-2 p-3 cursor-pointer"
            :style="cardStyle('running', story.id === selectedId)"
            role="button"
            tabindex="0"
            :aria-label="`Story: ${story.key} - ${story.title}`"
            :aria-selected="story.id === selectedId"
            @click="emit('select', story.id)"
            @keydown.enter="emit('select', story.id)"
          >
            <div class="flex items-center justify-between gap-2">
              <StatusBadge status="running" :animated="true" />
              <span style="font-family: monospace; font-size: 0.75rem; color: var(--p-text-muted-color)">
                {{ story.key }}
              </span>
            </div>
            <span class="card-title">{{ story.title }}</span>
            <div class="flex items-center justify-between gap-2">
              <ContainerChip
                :container-id="pseudoContainerId(story)"
                isolation="isolated"
                :short-length="4"
              />
              <span
                v-if="runId(story)"
                style="font-family: monospace; font-size: 0.72rem; color: var(--status-running-color)"
              >
                {{ formatTimer(stream.runElapsedSeconds(runId(story))) }}
              </span>
            </div>
          </div>
        </template>

        <!-- ── IN REVIEW (blocked) cards ─────────────────────────────────── -->
        <template v-else-if="col.key === 'blocked'">
          <div
            v-for="story in grouped['blocked']"
            :key="story.id"
            class="kanban-card amber-breathe flex flex-col gap-2 p-3 cursor-pointer"
            :style="cardStyle('gate', story.id === selectedId)"
            role="button"
            tabindex="0"
            :aria-label="`Story: ${story.key} - ${story.title}`"
            :aria-selected="story.id === selectedId"
            @click="emit('select', story.id)"
            @keydown.enter="emit('select', story.id)"
          >
            <div class="flex items-center justify-between gap-2">
              <StatusBadge status="blocked" :animated="true" />
              <span style="font-family: monospace; font-size: 0.75rem; color: var(--p-text-muted-color)">
                {{ story.key }}
              </span>
            </div>
            <span class="card-title">{{ story.title }}</span>
            <div class="flex items-start">
              <Button
                label="Needs you · review →"
                severity="warn"
                size="small"
                text
                style="padding: 0; font-size: 0.75rem; color: var(--status-gate-color)"
                @click.stop="navigateToApproval(story)"
              />
            </div>
          </div>
        </template>

        <!-- ── DONE cards ─────────────────────────────────────────────────── -->
        <template v-else-if="col.key === 'done'">
          <div
            v-for="story in grouped['done']"
            :key="story.id"
            class="kanban-card flex flex-col gap-2 p-3 cursor-pointer"
            :style="cardStyle('default', story.id === selectedId)"
            role="button"
            tabindex="0"
            :aria-label="`Story: ${story.key} - ${story.title}`"
            :aria-selected="story.id === selectedId"
            @click="emit('select', story.id)"
            @keydown.enter="emit('select', story.id)"
          >
            <div class="flex items-center justify-between gap-2">
              <StatusBadge status="done" :animated="false" />
              <span style="font-family: monospace; font-size: 0.75rem; color: var(--p-text-muted-color)">
                {{ story.key }}
              </span>
            </div>
            <span class="card-title">{{ story.title }}</span>
            <div v-if="runId(story)" class="flex items-center justify-between gap-2">
              <span style="font-size: 0.72rem; color: var(--p-text-muted-color)">
                {{ formatTimer(stream.runElapsedSeconds(runId(story))) }}
              </span>
              <CostTicker
                :value="stream.runCostUsd(runId(story))"
                :animated="false"
              />
            </div>
          </div>
        </template>

        <!-- ── FAILED cards ───────────────────────────────────────────────── -->
        <template v-else-if="col.key === 'failed'">
          <div
            v-for="story in grouped['failed']"
            :key="story.id"
            class="kanban-card flex flex-col gap-2 p-3 cursor-pointer"
            :style="cardStyle('default', story.id === selectedId)"
            role="button"
            tabindex="0"
            :aria-label="`Story: ${story.key} - ${story.title}`"
            :aria-selected="story.id === selectedId"
            @click="emit('select', story.id)"
            @keydown.enter="emit('select', story.id)"
          >
            <div class="flex items-center justify-between gap-2">
              <StatusBadge status="failed" :animated="false" />
              <span style="font-family: monospace; font-size: 0.75rem; color: var(--p-text-muted-color)">
                {{ story.key }}
              </span>
            </div>
            <span class="card-title">{{ story.title }}</span>
            <span style="font-family: monospace; font-size: 0.72rem; color: var(--status-failed-color)">
              {{ story.latest_run?.error_message ?? 'exit 1' }}
            </span>
          </div>
        </template>

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
.card-title {
  font-size: 0.85rem;
  font-weight: 500;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.kanban-card {
  transition: border-color 0.2s, background-color 0.2s;
}
</style>
