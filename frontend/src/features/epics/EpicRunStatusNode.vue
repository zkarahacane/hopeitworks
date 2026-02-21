<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import Tag from 'primevue/tag'

const props = defineProps<{
  data: { key: string; title: string; status: string; runId: string | null }
}>()

const emit = defineEmits<{ 'node-click': [runId: string] }>()

const truncatedTitle = computed(() =>
  props.data.title.length > 40 ? props.data.title.slice(0, 40) + '\u2026' : props.data.title,
)

const nodeClass = computed(() => ({
  'bg-surface-200': props.data.status === 'pending',
  'bg-blue-500 animate-pulse': props.data.status === 'running',
  'bg-green-500': props.data.status === 'completed',
  'bg-red-500': props.data.status === 'failed',
}))

const statusSeverity = computed(() => {
  const map: Record<string, 'secondary' | 'info' | 'success' | 'danger'> = {
    pending: 'secondary',
    running: 'info',
    completed: 'success',
    failed: 'danger',
  }
  return map[props.data.status] ?? 'secondary'
})

function handleClick() {
  if (props.data.runId) emit('node-click', props.data.runId)
}
</script>

<template>
  <div
    :class="[
      'epic-run-node rounded p-2',
      nodeClass,
      data.runId ? 'cursor-pointer' : 'cursor-default',
    ]"
    @click="handleClick"
  >
    <Handle type="target" :position="Position.Top" />
    <div class="flex flex-col gap-1">
      <span class="font-mono font-bold text-sm">{{ data.key }}</span>
      <span :title="data.title" class="text-xs">{{ truncatedTitle }}</span>
      <Tag :value="data.status" :severity="statusSeverity" class="text-xs" />
    </div>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<style scoped>
.epic-run-node {
  border: 1px solid var(--p-surface-300);
  min-width: 160px;
}
</style>
