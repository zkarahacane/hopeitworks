<script setup lang="ts">
import { computed, toRef } from 'vue'
import Drawer from 'primevue/drawer'
import Tag from 'primevue/tag'
import Message from 'primevue/message'
import LogViewer from '@/ui/composed/LogViewer.vue'
import { useStepLogs } from './composables/useStepLogs'
import { statusSeverity } from '@/utils/runStatus'
import { format, parseISO, differenceInSeconds } from 'date-fns'
import type { components } from '@/api/schema'

type RunStep = components['schemas']['RunStep']

const props = defineProps<{
  step: RunStep | null
  runId: string
  projectId: string
  visible: boolean
}>()

const emit = defineEmits<{
  close: []
  'update:visible': [value: boolean]
}>()

const stepId = computed(() => props.step?.id ?? null)
const stepIdRef = toRef(stepId)
const { lines, sseStatus, clearLogs } = useStepLogs(props.projectId, props.runId, stepIdRef)

/** Format an ISO timestamp to HH:mm:ss. */
function formatTime(iso?: string | null): string {
  if (!iso) return ''
  try {
    return format(parseISO(iso), 'HH:mm:ss')
  } catch {
    return ''
  }
}

/** Compute human-readable duration between two ISO timestamps. */
const duration = computed(() => {
  if (!props.step?.started_at) return null
  const start = parseISO(props.step.started_at)
  const end = props.step.completed_at ? parseISO(props.step.completed_at) : new Date()
  const totalSeconds = differenceInSeconds(end, start)
  if (totalSeconds < 60) return `${totalSeconds}s`
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes < 60) return `${minutes}m ${seconds}s`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}h ${remainingMinutes}m`
})

const isRunning = computed(() => props.step?.status === 'running')

function handleVisibleUpdate(value: boolean) {
  emit('update:visible', value)
  if (!value) {
    emit('close')
  }
}
</script>

<template>
  <Drawer
    :visible="visible"
    position="right"
    :modal="true"
    :dismissable="true"
    :close-on-escape="true"
    :pt="{
      root: 'w-full md:w-1/2',
      content: 'flex flex-col flex-1 overflow-hidden p-0',
    }"
    data-testid="step-log-panel"
    @update:visible="handleVisibleUpdate"
  >
    <template #header>
      <div v-if="step" class="flex flex-col gap-2 w-full">
        <div class="flex items-center gap-3 flex-wrap">
          <h3 class="text-lg font-semibold m-0" data-testid="step-name">
            {{ step.step_name }}
          </h3>
          <Tag
            :value="step.status"
            :severity="statusSeverity(step.status)"
            data-testid="step-status"
          />
        </div>
        <div class="flex items-center gap-4 text-sm text-surface-500">
          <span v-if="step.started_at" data-testid="step-started-at">
            Started: {{ formatTime(step.started_at) }}
          </span>
          <span v-if="step.completed_at" data-testid="step-completed-at">
            Completed: {{ formatTime(step.completed_at) }}
          </span>
          <span v-else-if="isRunning" data-testid="step-running-indicator">
            Running...
          </span>
          <span v-if="duration" data-testid="step-duration">
            Duration: {{ duration }}
          </span>
        </div>
      </div>
    </template>

    <div v-if="step" class="flex flex-col flex-1 overflow-hidden p-4 gap-4">
      <!-- Error message -->
      <Message
        v-if="step.error_message"
        severity="error"
        :closable="false"
        data-testid="step-error-message"
      >
        {{ step.error_message }}
      </Message>

      <!-- Log viewer -->
      <div class="flex-1 overflow-hidden">
        <LogViewer :lines="lines" :status="sseStatus" @clear="clearLogs" />
      </div>
    </div>
  </Drawer>
</template>
