<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useLocalStorage } from '@vueuse/core'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import SelectButton from 'primevue/selectbutton'
import type { Story, KanbanColumn } from '@/stores/stories'
import {
  boardColumn,
  stageColumn,
  STAGE_BACKLOG_COLUMN,
  STAGE_DONE_COLUMN,
  STAGE_FAILED_COLUMN,
} from '@/stores/stories'
import { statusTokenSeverity } from '@/utils/statusToken'
import { useRuntimeStream } from '@/stores/runtimeStream'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import ContainerChip from '@/ui/primitives/ContainerChip.vue'
import CostTicker from '@/ui/primitives/CostTicker.vue'

/** A pipeline stage projected onto the board (id + name, in pipeline order). */
export interface BoardStage {
  id: string
  name: string
  transition?: 'auto' | 'manual' | 'gate'
}

const props = defineProps<{
  stories: Story[]
  selectedId: string | null
  projectId: string
  /**
   * Ordered pipeline stages (groups) for the détail view's dynamic columns.
   * When empty/omitted the détail view shows only the entry + terminal lanes.
   */
  stages?: BoardStage[]
}>()

const emit = defineEmits<{
  select: [storyId: string]
  /**
   * Request to "Go" a story. The card already knows whether this launches a fresh
   * run (Backlog) or starts the parked manual stage (idle in a manual stage), so it
   * passes an explicit action and the parent just dispatches it.
   */
  go: [payload: { story: Story; action: 'launch' | 'start-stage' }]
}>()

const router = useRouter()
const stream = useRuntimeStream()

// ── View mode (macro lifecycle vs détail stages) ──────────────────────────────
type ViewMode = 'macro' | 'detail'
const VIEW_OPTIONS: { label: string; value: ViewMode }[] = [
  { label: 'Macro', value: 'macro' },
  { label: 'Détail', value: 'detail' },
]
const viewMode = useLocalStorage<ViewMode>('board.viewMode', 'macro')

const stages = computed<BoardStage[]>(() => props.stages ?? [])

// ── Card visual variant ───────────────────────────────────────────────────────
// Independent of column layout: derived from the story's live lifecycle so a card
// looks the same in macro and détail views.
type CardVariant = 'backlog' | 'manual_idle' | 'running' | 'gate' | 'done' | 'failed'

/** Transition policy of the stage whose name matches the card's current_stage. */
function currentStageTransition(story: Story): 'auto' | 'manual' | 'gate' | undefined {
  if (!story.current_stage) return undefined
  return stages.value.find((s) => s.name === story.current_stage)?.transition
}

/**
 * True when the card sits idle in a manual stage not yet started: its run is paused
 * and the current stage's policy is manual. The executor parks the run on entering a
 * not-yet-started manual stage (no waiting_approval step), so it is distinct from a
 * gate (which suspends a step pending review).
 */
function isManualIdle(story: Story): boolean {
  return (
    story.latest_run?.status === 'paused' &&
    story.latest_run?.current_step?.status !== 'waiting_approval' &&
    currentStageTransition(story) === 'manual'
  )
}

function cardVariant(story: Story): CardVariant {
  if (isManualIdle(story)) return 'manual_idle'
  const col = boardColumn(story)
  if (col === 'in_progress') return 'running'
  if (col === 'blocked') return 'gate'
  if (col === 'done') return 'done'
  if (col === 'failed') return 'failed'
  return 'backlog'
}

/**
 * Whether the "Go" affordance is shown: a card in Backlog (→ launches its run) or a
 * card idle in a manual stage (→ starts that stage). Never while a segment runs nor
 * at a gate (those have their own CTAs).
 */
function showGo(story: Story): boolean {
  const v = cardVariant(story)
  return v === 'backlog' || v === 'manual_idle'
}

/** Label for the Go button: launch from backlog, or start the parked manual stage. */
function goLabel(story: Story): string {
  return cardVariant(story) === 'manual_idle' ? 'Go · start stage' : 'Go'
}

/** Emit the Go request with the action the card's variant implies. */
function handleGo(story: Story) {
  emit('go', {
    story,
    action: cardVariant(story) === 'manual_idle' ? 'start-stage' : 'launch',
  })
}

// ── Card style helpers ───────────────────────────────────────────────────────

function cardStyle(story: Story): Record<string, string> {
  const isSelected = story.id === props.selectedId
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
  const variant = cardVariant(story)
  if (variant === 'running') {
    return { ...base, border: '1px solid var(--surface-border)', borderLeft: '3px solid var(--status-running-color)' }
  }
  if (variant === 'gate') {
    return { ...base, border: '1px solid var(--surface-border)', borderLeft: '3px solid var(--status-gate-color)' }
  }
  return { ...base, border: '1px solid var(--surface-border)' }
}

/** Extra class for the gate (awaiting-review) breathe animation. */
function cardClass(story: Story): string {
  return cardVariant(story) === 'gate' ? 'amber-breathe' : ''
}

// ── Column definitions ──────────────────────────────────────────────────────

interface ColumnDef {
  /** Stable key used for grouping/keying; a KanbanColumn (macro) or stage/sentinel key (détail). */
  key: string
  label: string
  /** Status key driving the count-tag severity. */
  severityKey: string
}

const MACRO_COLUMNS: ColumnDef[] = [
  { key: 'backlog', label: 'Backlog', severityKey: 'backlog' },
  { key: 'in_progress', label: 'Running', severityKey: 'in_progress' },
  { key: 'blocked', label: 'In Review', severityKey: 'blocked' },
  { key: 'done', label: 'Done', severityKey: 'done' },
  { key: 'failed', label: 'Failed', severityKey: 'failed' },
]

/**
 * Détail columns = a Backlog entry lane + one lane per pipeline stage (in order)
 * + terminal Done/Failed lanes. Stage lanes are keyed by stage name, matching the
 * card's `current_stage`.
 */
const detailColumns = computed<ColumnDef[]>(() => {
  const cols: ColumnDef[] = [
    { key: STAGE_BACKLOG_COLUMN, label: 'Backlog', severityKey: 'backlog' },
  ]
  for (const stage of stages.value) {
    cols.push({ key: stage.name, label: stage.name, severityKey: 'in_progress' })
  }
  cols.push({ key: STAGE_DONE_COLUMN, label: 'Done', severityKey: 'done' })
  cols.push({ key: STAGE_FAILED_COLUMN, label: 'Failed', severityKey: 'failed' })
  return cols
})

const columns = computed<ColumnDef[]>(() =>
  viewMode.value === 'detail' ? detailColumns.value : MACRO_COLUMNS,
)

// ── Grouping ────────────────────────────────────────────────────────────────

const grouped = computed<Record<string, Story[]>>(() => {
  const result: Record<string, Story[]> = {}
  for (const col of columns.value) result[col.key] = []

  const place = viewMode.value === 'detail'
    ? stageColumn
    : (boardColumn as (s: Story) => string)
  const fallback = viewMode.value === 'detail' ? STAGE_BACKLOG_COLUMN : 'backlog'

  for (const story of props.stories) {
    let key = place(story)
    // In détail mode a card whose stage name is not a known column (e.g. the
    // pipeline changed) falls back to the entry lane so it never vanishes.
    if (!(key in result)) key = fallback
    result[key]!.push(story)
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

/** StatusBadge status string for a card's variant. */
function badgeStatus(story: Story): string {
  const variant = cardVariant(story)
  return variant === 'running' ? 'running'
    : variant === 'gate' ? 'blocked'
    : variant === 'done' ? 'done'
    : variant === 'failed' ? 'failed'
    : variant === 'manual_idle' ? 'pending'
    : 'backlog'
}

/** Whether the card badge should animate (running/gate are live). */
function badgeAnimated(story: Story): boolean {
  const variant = cardVariant(story)
  return variant === 'running' || variant === 'gate'
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

/** Count-tag severity for a column header. */
function columnSeverity(severityKey: string) {
  return statusTokenSeverity(severityKey as KanbanColumn)
}
</script>

<template>
  <div class="flex flex-col h-full gap-2">
    <!-- ── View toggle (macro lifecycle / détail stages) ───────────────────── -->
    <div class="flex items-center justify-end shrink-0">
      <SelectButton
        v-model="viewMode"
        :options="VIEW_OPTIONS"
        option-label="label"
        option-value="value"
        :allow-empty="false"
        aria-label="Board view mode"
        :pt="{ root: { style: 'font-size: 0.75rem' } }"
      />
    </div>

    <!-- ── Columns ─────────────────────────────────────────────────────────── -->
    <div class="flex gap-3 flex-1 min-h-0 overflow-x-auto">
      <div
        v-for="col in columns"
        :key="col.key"
        class="flex flex-col gap-2 min-w-[240px] w-[240px] shrink-0"
      >
        <!-- Column header -->
        <div class="flex items-center gap-2 px-1 pb-2" style="border-bottom: 1px solid var(--surface-border)">
          <span style="font-weight: 600; font-size: 0.85rem; font-family: var(--font-sans)">{{ col.label }}</span>
          <Tag
            :value="String(grouped[col.key]?.length ?? 0)"
            :severity="columnSeverity(col.severityKey)"
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
            :class="cardClass(story)"
            :style="cardStyle(story)"
            role="button"
            tabindex="0"
            :aria-label="`Story: ${story.key} - ${story.title}`"
            :aria-selected="story.id === selectedId"
            @click="emit('select', story.id)"
            @keydown.enter="emit('select', story.id)"
          >
            <div class="flex items-center justify-between gap-2">
              <StatusBadge :status="badgeStatus(story)" :animated="badgeAnimated(story)" />
              <span style="font-family: monospace; font-size: 0.75rem; color: var(--p-text-muted-color)">
                {{ story.key }}
              </span>
            </div>
            <span class="card-title">{{ story.title }}</span>

            <!-- ── BACKLOG: dependency hint ────────────────────────────────── -->
            <div
              v-if="cardVariant(story) === 'backlog' && story.depends_on && story.depends_on.length > 0"
              class="flex items-center gap-1"
            >
              <i class="pi pi-link" style="font-size: 0.7rem; color: var(--p-text-muted-color)" aria-hidden="true" />
              <span style="font-size: 0.72rem; color: var(--p-text-muted-color)">
                waiting on {{ story.depends_on.slice(0, 2).join(', ') }}
                <span v-if="story.depends_on.length > 2">+{{ story.depends_on.length - 2 }}</span>
              </span>
            </div>

            <!-- ── RUNNING: container chip + elapsed timer ─────────────────── -->
            <div
              v-else-if="cardVariant(story) === 'running'"
              class="flex items-center justify-between gap-2"
            >
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

            <!-- ── GATE (in review): review CTA ─────────────────────────────── -->
            <div v-else-if="cardVariant(story) === 'gate'" class="flex items-start">
              <Button
                label="Needs you · review →"
                severity="warn"
                size="small"
                text
                style="padding: 0; font-size: 0.75rem; color: var(--status-gate-color)"
                @click.stop="navigateToApproval(story)"
              />
            </div>

            <!-- ── DONE: elapsed + cost ─────────────────────────────────────── -->
            <div
              v-else-if="cardVariant(story) === 'done' && runId(story)"
              class="flex items-center justify-between gap-2"
            >
              <span style="font-size: 0.72rem; color: var(--p-text-muted-color)">
                {{ formatTimer(stream.runElapsedSeconds(runId(story))) }}
              </span>
              <CostTicker :value="stream.runCostUsd(runId(story))" :animated="false" />
            </div>

            <!-- ── FAILED: error message ────────────────────────────────────── -->
            <span
              v-else-if="cardVariant(story) === 'failed'"
              style="font-family: monospace; font-size: 0.72rem; color: var(--status-failed-color)"
            >
              {{ story.latest_run?.error_message ?? 'exit 1' }}
            </span>

            <!-- ── GO: launch (Backlog) or start the manual stage (idle) ───────── -->
            <div v-if="showGo(story)" class="flex items-start">
              <Button
                :label="goLabel(story)"
                icon="pi pi-play"
                severity="success"
                size="small"
                :aria-label="`Go: ${story.key}`"
                data-testid="board-go-button"
                style="font-size: 0.75rem"
                @click.stop="handleGo(story)"
              />
            </div>
          </div>

          <!-- Empty column state -->
          <p
            v-if="(grouped[col.key]?.length ?? 0) === 0"
            class="p-3 text-center"
            style="color: var(--p-text-muted-color); font-size: 0.8rem"
          >
            No stories
          </p>
        </div>
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
