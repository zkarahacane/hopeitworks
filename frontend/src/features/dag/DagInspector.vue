<script setup lang="ts">
import { computed } from 'vue'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import ContainerChip from '@/ui/primitives/ContainerChip.vue'
import CostTicker from '@/ui/primitives/CostTicker.vue'
import StepTimeline from '@/ui/composed/StepTimeline.vue'
import LogStreamPanel from '@/ui/composed/LogStreamPanel.vue'
import { useDagInspector } from './composables/useDagInspector'
import { formatDurationSeconds } from '@/utils/formatDuration'
import type { SSEStatus } from '@/composables/useSSE'
import type { DagNodeData } from './composables/useDagLayout'

/**
 * DagInspector — the right-hand panel for the selected story node.
 *
 * Sections: header (key/title/container/status/timer/cost), PIPELINE
 * (StepTimeline of the four phases), LIVE LOG (LogStreamPanel, mono, blinking
 * caret while streaming). All data is derived live via useDagInspector; this
 * component is a visual assembler only.
 */
const props = defineProps<{
  node: DagNodeData | null
  sseStatus: SSEStatus
}>()

const selected = computed(() => props.node)
const sseStatus = computed(() => props.sseStatus)

const { isActive, containerId, pipelineSteps, logLines } = useDagInspector(selected, sseStatus)

const elapsedLabel = computed(() =>
  props.node && props.node.elapsedSeconds > 0
    ? formatDurationSeconds(props.node.elapsedSeconds)
    : null,
)
</script>

<template>
  <aside
    class="dag-inspector flex flex-col gap-5 p-4 overflow-y-auto"
    data-testid="dag-inspector"
  >
    <!-- Empty state -->
    <div
      v-if="!node"
      class="flex flex-1 items-center justify-center text-center"
      :style="{ color: 'var(--p-text-muted-color)', fontSize: '0.85rem' }"
      data-testid="dag-inspector-empty"
    >
      Select a story node to inspect its pipeline and live log
    </div>

    <template v-else>
      <!-- Header -->
      <header class="flex flex-col gap-2">
        <div class="flex items-center justify-between gap-2">
          <span class="font-mono" :style="{ fontWeight: 700, fontSize: '0.85rem' }">
            {{ node.key }}
          </span>
          <StatusBadge :status="node.status" :icon="true" />
        </div>
        <h2 class="m-0" :style="{ fontSize: '1rem', fontWeight: 600, lineHeight: 1.3 }">
          {{ node.title }}
        </h2>
        <div class="flex flex-wrap items-center gap-2">
          <ContainerChip v-if="containerId" :container-id="containerId" :short-length="4" />
          <span
            v-if="elapsedLabel"
            class="font-mono"
            :style="{ fontSize: '0.72rem', color: 'var(--p-text-muted-color)' }"
          >
            {{ elapsedLabel }}
          </span>
          <CostTicker v-if="node.costUsd > 0" :value="node.costUsd" :animated="isActive" />
        </div>
      </header>

      <!-- Pipeline -->
      <section class="flex flex-col gap-2">
        <h3
          :style="{ fontSize: '0.7rem', fontWeight: 700, letterSpacing: '0.06em', color: 'var(--p-text-muted-color)' }"
        >
          PIPELINE
        </h3>
        <StepTimeline :steps="pipelineSteps" />
      </section>

      <!-- Live log -->
      <section class="flex flex-col gap-2">
        <h3
          :style="{ fontSize: '0.7rem', fontWeight: 700, letterSpacing: '0.06em', color: 'var(--p-text-muted-color)' }"
        >
          LIVE LOG
        </h3>
        <LogStreamPanel :lines="logLines" :status="sseStatus" :active="isActive" />
      </section>
    </template>
  </aside>
</template>

<style scoped>
.dag-inspector {
  width: 22rem;
  min-width: 22rem;
  border-left: 1px solid var(--p-surface-700);
  background: var(--p-surface-900);
}
</style>
