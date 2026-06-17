<script setup lang="ts">
import type { Node, Edge } from '@vue-flow/core'
import { VueFlow, useVueFlow, Panel } from '@vue-flow/core'
import { MiniMap } from '@vue-flow/minimap'
import { markRaw } from 'vue'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/minimap/dist/style.css'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import DagStoryNode from './DagStoryNode.vue'
import DagEdge from './DagEdge.vue'
import type { DagNodeData } from './composables/useDagLayout'

const props = withDefaults(
  defineProps<{
    nodes: Node<DagNodeData>[]
    edges: Edge[]
    isLoading: boolean
    error: Error | null
    /** Currently selected story key (highlights the node). */
    selectedKey?: string | null
    /** Dark canvas (the flagship default). */
    dark?: boolean
  }>(),
  { selectedKey: null, dark: true },
)

const emit = defineEmits<{
  retry: []
  select: [key: string]
  'retry-node': [key: string]
  'toggle-theme': []
}>()

const { zoomIn, zoomOut, fitView } = useVueFlow()

// markRaw prevents Vue from making the components reactive (vue-flow requirement)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const nodeTypes: Record<string, any> = {
  story: markRaw(DagStoryNode),
}
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const edgeTypes: Record<string, any> = {
  dag: markRaw(DagEdge),
}

function onNodeClick(key: string) {
  emit('select', key)
}
</script>

<template>
  <div
    class="dag-graph-container flex h-full flex-col"
    :class="{ dark: props.dark }"
    data-testid="dag-graph"
  >
    <div v-if="isLoading" class="flex items-center justify-center h-full">
      <Skeleton width="100%" height="100%" />
    </div>

    <Message v-else-if="error" severity="error" :closable="false" class="m-4">
      <div class="flex items-center gap-3">
        <span>{{ error.message }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="emit('retry')" />
      </div>
    </Message>

    <VueFlow
      v-else
      :nodes="nodes"
      :edges="edges"
      :node-types="nodeTypes"
      :edge-types="edgeTypes"
      :default-edge-options="{ type: 'dag' }"
      :min-zoom="0.3"
      :max-zoom="1.6"
      fit-view-on-init
      class="flex-1 dag-flow"
      @node-click="(e) => onNodeClick((e.node as Node<DagNodeData>).data!.key)"
    >
      <template #node-story="storyProps">
        <DagStoryNode
          v-bind="storyProps"
          :selected="storyProps.data.key === props.selectedKey"
          @retry="(key) => emit('retry-node', key)"
        />
      </template>

      <!-- Zoom controls (bottom-left) -->
      <Panel position="bottom-left" class="dag-controls">
        <div class="flex flex-col gap-1">
          <Button
            icon="pi pi-plus"
            severity="secondary"
            text
            rounded
            size="small"
            aria-label="Zoom in"
            data-testid="dag-zoom-in"
            @click="() => zoomIn()"
          />
          <Button
            icon="pi pi-minus"
            severity="secondary"
            text
            rounded
            size="small"
            aria-label="Zoom out"
            data-testid="dag-zoom-out"
            @click="() => zoomOut()"
          />
          <Button
            icon="pi pi-expand"
            severity="secondary"
            text
            rounded
            size="small"
            aria-label="Fit view"
            data-testid="dag-fit-view"
            @click="() => fitView()"
          />
          <Button
            :icon="props.dark ? 'pi pi-sun' : 'pi pi-moon'"
            severity="secondary"
            text
            rounded
            size="small"
            :aria-label="props.dark ? 'Switch to light' : 'Switch to dark'"
            data-testid="dag-theme-toggle"
            @click="emit('toggle-theme')"
          />
        </div>
      </Panel>

      <!-- Mini-map (bottom-right) -->
      <MiniMap pannable zoomable position="bottom-right" />
    </VueFlow>
  </div>
</template>

<style scoped>
.dag-graph-container {
  min-height: 400px;
  height: 100%;
}

/* Dark canvas — the flagship surface. Tints VueFlow's background + minimap. */
.dag-graph-container.dark :deep(.dag-flow) {
  background: var(--surface-base);
}

.dag-graph-container :deep(.vue-flow__minimap) {
  background-color: var(--surface-raised);
  border: 1px solid var(--surface-border);
  border-radius: 8px;
}

.dag-controls {
  background-color: var(--surface-raised);
  border: 1px solid var(--surface-border);
  border-radius: 10px;
  padding: 0.25rem;
}
</style>
