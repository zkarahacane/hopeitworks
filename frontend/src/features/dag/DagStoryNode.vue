<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import Button from 'primevue/button'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import ContainerChip from '@/ui/primitives/ContainerChip.vue'
import CostTicker from '@/ui/primitives/CostTicker.vue'
import { statusToken } from '@/utils/statusToken'
import { formatDurationSeconds } from '@/utils/formatDuration'
import type { DagNodeData } from './composables/useDagLayout'

/**
 * DagStoryNode — a rich, status-coloured story node for the Execution Graph.
 *
 * Replaces the old thin node. Consumes the Phase 0 primitives (StatusBadge via
 * statusToken, ContainerChip, CostTicker) and the live runtime data resolved in
 * useDagLayout. Dumb + prop-driven: the live timer/cost/status arrive in
 * `data`; this node only renders. Emits `retry` for failed nodes.
 *
 * VueFlow passes node props as `data`, plus `selected`. We declare both.
 */
const props = withDefaults(
  defineProps<{
    data: DagNodeData
    selected?: boolean
  }>(),
  { selected: false },
)

const emit = defineEmits<{
  retry: [key: string]
}>()

const token = computed(() => statusToken(props.data.status))
const isFailed = computed(() => token.value.family === 'failed')
const isQueued = computed(() => token.value.family === 'queued')
const isDone = computed(() => token.value.family === 'done')

const truncatedTitle = computed(() =>
  props.data.title.length > 46 ? props.data.title.slice(0, 46) + '…' : props.data.title,
)

const elapsedLabel = computed(() =>
  props.data.elapsedSeconds > 0 ? formatDurationSeconds(props.data.elapsedSeconds) : null,
)

/** Left status stripe + selected ring read from the status token color. */
const cardStyle = computed(() => ({
  borderColor: props.selected ? `var(${token.value.colorToken})` : 'var(--surface-border)',
  boxShadow: props.selected ? `0 0 0 1px var(${token.value.colorToken})` : 'none',
}))
</script>

<template>
  <div
    class="dag-story-node flex flex-col gap-2 px-3 py-2.5"
    :class="{ 'is-active': data.active }"
    :data-status="token.family"
    :data-testid="`dag-node-${data.key}`"
    :style="cardStyle"
  >
    <Handle type="target" :position="Position.Top" />

    <!-- status stripe -->
    <span
      class="dag-story-node__stripe"
      :style="{ backgroundColor: `var(${token.colorToken})` }"
      aria-hidden="true"
    />

    <!-- header: key + status -->
    <div class="flex items-center justify-between gap-2">
      <span class="font-mono" :style="{ fontWeight: 700, fontSize: '0.78rem' }">{{ data.key }}</span>
      <StatusBadge :status="data.status" :icon="true" />
    </div>

    <!-- title -->
    <span :title="data.title" :style="{ fontSize: '0.82rem', lineHeight: 1.3 }">
      {{ truncatedTitle }}
    </span>

    <!-- meta row: container · timer · cost -->
    <div class="flex flex-wrap items-center gap-2">
      <ContainerChip
        v-if="data.containerId"
        :container-id="data.containerId"
        :isolation="null"
        :short-length="4"
      />
      <span
        v-if="elapsedLabel"
        class="font-mono"
        :style="{ fontSize: '0.72rem', color: 'var(--p-text-muted-color)' }"
        data-testid="dag-node-timer"
      >
        {{ elapsedLabel }}
      </span>
      <CostTicker
        v-if="data.costUsd > 0 && !isQueued"
        :value="data.costUsd"
        :animated="data.active"
        data-testid="dag-node-cost"
      />
    </div>

    <!-- queued: waiting-on hint -->
    <span
      v-if="isQueued && data.waitingOn.length > 0"
      class="font-mono"
      :style="{ fontSize: '0.7rem', color: 'var(--p-text-muted-color)' }"
      data-testid="dag-node-waiting"
    >
      waiting on {{ data.waitingOn.join(', ') }}
    </span>

    <!-- failed: exit + retry -->
    <div v-if="isFailed" class="flex items-center justify-between gap-2">
      <span
        class="font-mono"
        :style="{ fontSize: '0.72rem', color: `var(${token.colorToken})` }"
        data-testid="dag-node-exit"
      >
        {{ data.exitMessage }}
      </span>
      <Button
        label="retry"
        icon="pi pi-refresh"
        severity="danger"
        text
        size="small"
        data-testid="dag-node-retry"
        @click.stop="emit('retry', data.key)"
      />
    </div>

    <Handle type="source" :position="Position.Bottom" :class="{ 'opacity-50': isDone }" />
  </div>
</template>

<style scoped>
.dag-story-node {
  position: relative;
  min-width: 220px;
  max-width: 260px;
  background: var(--surface-raised);
  border: 1px solid var(--surface-border);
  border-radius: 12px;
  overflow: hidden;
  transition:
    box-shadow 0.15s ease,
    border-color 0.15s ease;
}

.dag-story-node__stripe {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
}

/* Active (running) node breathes a faint glow in its status color. */
.dag-story-node.is-active {
  animation: dag-node-glow 2s ease-in-out infinite;
}

@keyframes dag-node-glow {
  0%,
  100% {
    box-shadow: 0 0 0 0 var(--status-running-surface);
  }
  50% {
    box-shadow: 0 0 0 5px var(--status-running-surface);
  }
}

@media (prefers-reduced-motion: reduce) {
  .dag-story-node.is-active {
    animation: none;
  }
}
</style>
