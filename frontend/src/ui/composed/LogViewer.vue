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
  <div
    class="flex flex-col rounded-lg overflow-hidden"
    :style="{ border: '1px solid var(--surface-border)' }"
  >
    <!-- Toolbar -->
    <div
      class="flex items-center justify-between px-3 py-2"
      :style="{ backgroundColor: 'var(--surface-overlay)', borderBottom: '1px solid var(--surface-border)' }"
    >
      <div class="flex items-center gap-2">
        <Tag :value="statusLabel[status]" :severity="statusSeverity[status]" />
        <span
          v-if="!autoScroll"
          class="text-xs cursor-pointer"
          :style="{ color: 'var(--p-text-muted-color)' }"
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
      class="overflow-y-auto font-mono text-sm p-3 min-h-48 max-h-96"
      :style="{ backgroundColor: 'var(--surface-base)', color: 'var(--p-text-color)' }"
      @scroll="onScroll"
    >
      <div
        v-for="(line, index) in lines"
        :key="index"
        class="log-line whitespace-pre-wrap leading-relaxed"
        v-html="formatLogLine(line.text, line.timestamp)"
      />
      <div
        v-if="lines.length === 0"
        class="text-center py-8"
        :style="{ color: 'var(--p-text-muted-color)' }"
      >
        No log output yet
      </div>
    </div>
  </div>
</template>

<style scoped>
:deep(.log-ts) {
  color: var(--p-text-muted-color);
  opacity: 0.7;
  font-size: 0.8em;
}
</style>
