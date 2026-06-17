<script setup lang="ts">
import { ref, watch, nextTick, useTemplateRef } from 'vue'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import { formatLogLine } from '@/utils/formatLogLine'
import type { SSEStatus } from '@/composables/useSSE'

/** A single log line with raw text and timestamp. */
export interface LogLine {
  text: string
  timestamp: Date
}

const props = defineProps<{
  lines: LogLine[]
  status: SSEStatus
}>()

const emit = defineEmits<{
  clear: []
}>()

const scrollContainer = useTemplateRef<HTMLElement>('scrollContainer')
const autoScroll = ref(true)

watch(
  () => props.lines.length,
  async () => {
    if (!autoScroll.value) return
    await nextTick()
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
    }
  },
)

function onScroll() {
  if (!scrollContainer.value) return
  const { scrollTop, scrollHeight, clientHeight } = scrollContainer.value
  const distFromBottom = scrollHeight - scrollTop - clientHeight
  autoScroll.value = distFromBottom < 50
}

// SSE *connection* state — deliberately NOT routed through the product
// statusToken system (that's for run/step/story/epic status). "connecting" uses
// the blue informational accent, which is allowed for non-status signals.
const statusSeverity: Record<SSEStatus, 'info' | 'success' | 'warn' | 'danger'> = {
  connecting: 'info',
  open: 'success',
  closed: 'warn',
  error: 'danger',
}

const statusLabel: Record<SSEStatus, string> = {
  connecting: 'Connecting...',
  open: 'Live',
  closed: 'Disconnected',
  error: 'Error',
}
</script>

<template>
  <div class="flex flex-col border border-surface rounded-lg overflow-hidden">
    <!-- Toolbar -->
    <div class="flex items-center justify-between px-3 py-2 bg-surface-50 dark:bg-surface-800 border-b border-surface">
      <div class="flex items-center gap-2">
        <Tag :value="statusLabel[status]" :severity="statusSeverity[status]" />
        <span
          v-if="!autoScroll"
          class="text-xs text-surface-500 cursor-pointer hover:text-primary"
          @click="autoScroll = true"
        >
          Auto-scroll paused — click to resume
        </span>
      </div>
      <Button
        label="Clear"
        icon="pi pi-trash"
        severity="secondary"
        text
        size="small"
        @click="emit('clear')"
      />
    </div>

    <!-- Log lines -->
    <div
      ref="scrollContainer"
      class="overflow-y-auto font-mono text-sm p-3 bg-surface-900 text-surface-100 min-h-48 max-h-96"
      @scroll="onScroll"
    >
      <div
        v-for="(line, index) in lines"
        :key="index"
        class="log-line whitespace-pre-wrap leading-relaxed"
        v-html="formatLogLine(line.text, line.timestamp)"
      />
      <div v-if="lines.length === 0" class="text-surface-500 text-center py-8">
        No log output yet
      </div>
    </div>
  </div>
</template>

<style scoped>
:deep(.log-ts) {
  color: var(--p-surface-400);
  opacity: 0.7;
  font-size: 0.8em;
}
</style>
