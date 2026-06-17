<script setup lang="ts">
import { computed } from 'vue'
import { BaseEdge, getBezierPath, Position } from '@vue-flow/core'

/**
 * DagEdge — a dependency link between two story nodes.
 *
 * When the source story is active (running), the edge stroke "marches" using
 * the shared `.marching-edge` primitive (animated stroke-dasharray, respects
 * prefers-reduced-motion). Idle edges render as a static muted line. The active
 * flag is driven from runtimeStream via useDagLayout (edge.data.active).
 *
 * VueFlow injects the geometry props (source/target x/y + positions) and the
 * edge `data`. Dumb: it only computes the path and picks the class/colour.
 */
const props = withDefaults(
  defineProps<{
    id: string
    sourceX: number
    sourceY: number
    targetX: number
    targetY: number
    sourcePosition?: Position
    targetPosition?: Position
    data?: { active?: boolean }
  }>(),
  {
    sourcePosition: Position.Bottom,
    targetPosition: Position.Top,
    data: () => ({ active: false }),
  },
)

const active = computed(() => props.data?.active ?? false)

const path = computed(
  () =>
    getBezierPath({
      sourceX: props.sourceX,
      sourceY: props.sourceY,
      sourcePosition: props.sourcePosition,
      targetX: props.targetX,
      targetY: props.targetY,
      targetPosition: props.targetPosition,
    })[0],
)

/** Active edges march in running-green; idle edges are a muted static line. */
const edgeStyle = computed(() => ({
  stroke: active.value ? 'var(--status-running-color)' : 'var(--p-surface-600)',
  strokeWidth: active.value ? 2 : 1.5,
}))
</script>

<template>
  <BaseEdge
    :id="id"
    :path="path"
    :class="{ 'marching-edge': active }"
    :style="edgeStyle"
    :data-testid="`dag-edge-${id}`"
    :data-active="active"
  />
</template>
