<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import Tag from 'primevue/tag'

const props = defineProps<{
  data: { key: string; title: string; status: string }
}>()

const truncatedTitle = computed(() =>
  props.data.title.length > 40 ? props.data.title.slice(0, 40) + '\u2026' : props.data.title,
)

const statusSeverity = computed(() => {
  const map: Record<string, 'secondary' | 'info' | 'success' | 'danger'> = {
    backlog: 'secondary',
    running: 'info',
    done: 'success',
    failed: 'danger',
  }
  return map[props.data.status] ?? 'secondary'
})

const isDone = computed(() => props.data.status === 'done')
</script>

<template>
  <div :class="['dag-story-node', { 'opacity-40': isDone }]">
    <Handle type="target" :position="Position.Top" />
    <div class="flex flex-col gap-1 p-2">
      <span class="font-mono font-bold text-sm">{{ data.key }}</span>
      <span :title="data.title" class="text-xs">{{ truncatedTitle }}</span>
      <Tag :value="data.status" :severity="statusSeverity" class="text-xs" />
    </div>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<style scoped>
.dag-story-node {
  background: var(--p-surface-0);
  border: 1px solid var(--p-surface-300);
  border-radius: 8px;
  min-width: 160px;
}

.opacity-40 {
  opacity: 0.4;
}
</style>
