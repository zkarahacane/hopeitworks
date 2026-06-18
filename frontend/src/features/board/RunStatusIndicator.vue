<script setup lang="ts">
import { computed } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import { useRelativeTime } from '@/composables/useRelativeTime'

export type RunStatus = 'running' | 'completed' | 'failed' | 'paused' | 'backlog' | null

interface Props {
  status: RunStatus
  completedAt?: string
  errorMessage?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  errorClick: []
}>()

interface StatusConfig {
  icon: string | null
  spinner: boolean
  text: string | null
  colorVar: string
  clickable: boolean
}

const backlogConfig: StatusConfig = {
  icon: 'pi pi-minus-circle',
  spinner: false,
  text: 'Backlog',
  colorVar: '--status-queued-color',
  clickable: false,
}

const statusConfigMap = new Map<string, StatusConfig>([
  ['running', {
    icon: null,
    spinner: true,
    text: 'Running...',
    colorVar: '--status-running-color',
    clickable: false,
  }],
  ['completed', {
    icon: 'pi pi-check-circle',
    spinner: false,
    text: null,
    colorVar: '--status-done-color',
    clickable: false,
  }],
  ['paused', {
    icon: 'pi pi-pause-circle',
    spinner: false,
    text: 'Paused',
    colorVar: '--status-gate-color',
    clickable: false,
  }],
  ['failed', {
    icon: 'pi pi-times-circle',
    spinner: false,
    text: 'Failed',
    colorVar: '--status-failed-color',
    clickable: true,
  }],
  ['backlog', backlogConfig],
])

const config = computed((): StatusConfig => {
  return statusConfigMap.get(props.status ?? 'backlog') ?? backlogConfig
})

const relativeTime = useRelativeTime(computed(() => props.completedAt ?? null))

function handleClick() {
  if (config.value.clickable) {
    emit('errorClick')
  }
}
</script>

<template>
  <div
    class="flex items-center gap-2"
    :class="{ 'cursor-pointer': config.clickable }"
    data-testid="run-status-indicator"
    @click="handleClick"
  >
    <ProgressSpinner
      v-if="config.spinner"
      style="width: 1rem; height: 1rem"
      stroke-width="4"
      data-testid="run-status-spinner"
    />
    <i
      v-else-if="config.icon"
      :class="config.icon"
      :style="`color: var(${config.colorVar})`"
      role="img"
      :aria-label="config.text ?? (status ?? 'status')"
      data-testid="run-status-icon"
    />

    <span
      v-if="status === 'completed'"
      :style="`color: var(${config.colorVar})`"
      data-testid="run-status-text"
    >
      {{ relativeTime }}
    </span>
    <span
      v-else-if="config.text"
      :style="`color: var(${config.colorVar})`"
      data-testid="run-status-text"
    >
      {{ config.text }}
    </span>
  </div>
</template>
