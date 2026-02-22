<script setup lang="ts">
import { computed, ref } from 'vue'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import { statusSeverity } from '@/utils/runStatus'
import type { components } from '@/api/schema'

type RunStep = components['schemas']['RunStep']

const props = defineProps<{
  step: RunStep
  /** The parent step whose error_message and log_tail may be shown */
  parentStep?: RunStep
}>()

const MAX_LOG_LINES = 20

const isExpanded = ref(false)

function toggleExpand() {
  isExpanded.value = !isExpanded.value
}

/** Returns "Retry #N (type)" label for display */
function retryLabel(step: RunStep): string {
  const num = step.retry_count ?? 1
  const type = step.retry_type ?? 'incremental'
  return `Retry #${num} (${type})`
}

const errorContext = computed(() => props.step.error_message ?? props.parentStep?.error_message ?? '')

const truncatedLog = computed(() => {
  const source = props.step.log_tail ?? props.parentStep?.log_tail ?? ''
  const lines = source ? source.split('\n') : []
  const isTruncated = lines.length > MAX_LOG_LINES
  return {
    lines: isTruncated ? lines.slice(-MAX_LOG_LINES) : lines,
    isTruncated,
    totalLines: lines.length,
  }
})

const hasExpandableContent = computed(
  () => !!errorContext.value || truncatedLog.value.lines.length > 0,
)
</script>

<template>
  <div class="ml-8 flex flex-col gap-1 border-l-2 border-surface-200 pl-4 py-2">
    <div class="flex items-center gap-2">
      <span class="text-sm text-surface-600 font-medium">{{ retryLabel(step) }}</span>
      <Tag :severity="statusSeverity(step.status)" :value="step.status" class="text-xs" />
      <Button
        v-if="hasExpandableContent"
        :icon="isExpanded ? 'pi pi-chevron-up' : 'pi pi-chevron-down'"
        text
        rounded
        size="small"
        severity="secondary"
        :aria-label="isExpanded ? 'Collapse error context' : 'Expand error context'"
        data-testid="expand-toggle"
        @click="toggleExpand"
      />
    </div>

    <div v-if="isExpanded" class="mt-2 rounded border border-surface-200 bg-surface-50 p-3">
      <div v-if="errorContext" class="mb-2">
        <p class="text-xs font-semibold text-surface-500 uppercase mb-1">Error</p>
        <pre class="text-xs text-red-700 whitespace-pre-wrap break-words">{{ errorContext }}</pre>
      </div>
      <div v-if="truncatedLog.lines.length > 0">
        <p class="text-xs font-semibold text-surface-500 uppercase mb-1">Log tail</p>
        <pre class="text-xs text-surface-800 whitespace-pre-wrap break-words">{{
          truncatedLog.lines.join('\n')
        }}</pre>
        <p v-if="truncatedLog.isTruncated" class="mt-1 text-xs text-surface-500">
          Showing last {{ MAX_LOG_LINES }} of {{ truncatedLog.totalLines }} lines.
          <a href="#" class="text-primary underline" @click.prevent="isExpanded = true"
            >Show more</a
          >
        </p>
      </div>
    </div>
  </div>
</template>
