<script setup lang="ts">
import { computed } from 'vue'
import Timeline from 'primevue/timeline'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import PhaseGroup from '@/ui/primitives/PhaseGroup.vue'
import { statusToken } from '@/utils/statusToken'
import { PHASES, phaseForStep, type Phase } from '@/utils/phaseGroup'

/** A single step rendered on the timeline. Prop-driven — no data access. */
export interface TimelineStep {
  id: string
  name: string
  status: string
  actionType?: string | null
  /** Explicit phase override; otherwise derived from actionType/name. */
  phase?: Phase
  /** Optional duration label (e.g. "01:23") rendered in mono. */
  duration?: string | null
}

const props = withDefaults(
  defineProps<{
    steps: TimelineStep[]
    /** Currently selected/active step id (highlighted). */
    selectedId?: string | null
    /** Animate live statuses (pulse). */
    animated?: boolean
  }>(),
  { selectedId: null, animated: true },
)

const emit = defineEmits<{
  select: [stepId: string]
}>()

/** Group steps into the four phases, preserving input order within a phase. */
const grouped = computed(() => {
  const map: Record<Phase, TimelineStep[]> = { setup: [], dev: [], review: [], delivery: [] }
  for (const step of props.steps) {
    const phase = step.phase ?? phaseForStep({ actionType: step.actionType, name: step.name })
    map[phase].push(step)
  }
  return map
})

/** Phases that actually have steps, in canonical order. */
const visiblePhases = computed(() => PHASES.filter((p) => grouped.value[p.key].length > 0))

/** Roll-up status for a phase: the most "significant" child family. */
const FAMILY_RANK: Record<string, number> = {
  failed: 5,
  gate: 4,
  running: 3,
  done: 2,
  queued: 1,
}
function rollup(steps: TimelineStep[]): string | null {
  let best: { status: string; rank: number } | null = null
  for (const s of steps) {
    const fam = statusToken(s.status).family
    const rank = FAMILY_RANK[fam] ?? 0
    if (!best || rank > best.rank) best = { status: s.status, rank }
  }
  return best?.status ?? null
}

/** Marker color for a step (the status token color). */
function markerColor(status: string): string {
  return `var(${statusToken(status).colorToken})`
}
</script>

<template>
  <div class="flex flex-col gap-5" data-testid="step-timeline">
    <p
      v-if="steps.length === 0"
      :style="{ color: 'var(--p-text-muted-color)', fontSize: '0.85rem' }"
      data-testid="step-timeline-empty"
    >
      No steps yet
    </p>

    <PhaseGroup
      v-for="phase in visiblePhases"
      :key="phase.key"
      :phase="phase.key"
      :count="grouped[phase.key].length"
      :rollup-status="rollup(grouped[phase.key])"
    >
      <Timeline :value="grouped[phase.key]" data-testid="phase-timeline">
        <template #marker="{ item }">
          <span
            class="inline-flex items-center justify-center rounded-full"
            :class="{ 'live-pulse': animated && statusToken(item.status).pulse && statusToken(item.status).family === 'running', 'amber-breathe': animated && statusToken(item.status).pulse && statusToken(item.status).family === 'gate' }"
            :style="{ width: '0.75rem', height: '0.75rem', backgroundColor: markerColor(item.status) }"
            aria-hidden="true"
          />
        </template>
        <template #content="{ item }">
          <button
            type="button"
            class="flex items-center gap-2 w-full text-left rounded-md px-2 py-1"
            :class="{ 'cursor-pointer': true }"
            :style="{
              backgroundColor: item.id === selectedId ? 'var(--surface-overlay)' : 'transparent',
            }"
            :data-selected="item.id === selectedId"
            data-testid="step-timeline-item"
            @click="emit('select', item.id)"
          >
            <span :style="{ fontSize: '0.85rem', fontWeight: 500 }">{{ item.name }}</span>
            <StatusBadge :status="item.status" :icon="false" :animated="animated" />
            <span
              v-if="item.duration"
              class="font-mono ml-auto"
              :style="{ fontSize: '0.72rem', color: 'var(--p-text-muted-color)' }"
            >
              {{ item.duration }}
            </span>
          </button>
        </template>
      </Timeline>
    </PhaseGroup>
  </div>
</template>
