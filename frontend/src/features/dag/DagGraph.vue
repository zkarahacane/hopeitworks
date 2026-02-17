<script setup lang="ts">
import type { Node, Edge } from '@vue-flow/core'
import { VueFlow } from '@vue-flow/core'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { markRaw } from 'vue'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import DagStoryNode from './DagStoryNode.vue'

defineProps<{
  nodes: Node[]
  edges: Edge[]
  isLoading: boolean
  error: Error | null
}>()

const emit = defineEmits<{
  retry: []
}>()

// markRaw prevents Vue from making the component reactive (vue-flow requirement)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const nodeTypes: Record<string, any> = {
  story: markRaw(DagStoryNode),
}
</script>

<template>
  <div class="dag-graph-container h-full">
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
      fit-view-on-init
      class="h-full"
    >
      <Controls />
      <MiniMap />
    </VueFlow>
  </div>
</template>

<style scoped>
.dag-graph-container {
  min-height: 400px;
}
</style>
