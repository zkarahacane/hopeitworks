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

const nodeStyle = computed(() => {
  const statusBg: Record<string, string> = {
    pending: 'var(--status-queued-surface)',
    running: 'var(--status-accent-color)',
    completed: 'var(--status-done-surface)',
    failed: 'var(--status-failed-color)',
  }
  return {
    background: statusBg[props.data.status] ?? 'var(--surface-overlay)',
    borderColor: 'var(--surface-border)',
  }
})

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
      data.status === 'running' ? 'animate-pulse' : '',
      data.runId ? 'cursor-pointer' : 'cursor-default',
    ]"
    :style="nodeStyle"
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
  border: 1px solid;
  min-width: 160px;
}
</style>
