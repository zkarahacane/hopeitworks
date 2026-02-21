<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import Button from 'primevue/button'
import { useRunsStore } from '@/stores/runs'
import RunStatusIndicator from '@/features/board/RunStatusIndicator.vue'
import type { RunStatus } from '@/features/board/RunStatusIndicator.vue'
import { useToast } from 'primevue/usetoast'

const route = useRoute()
const runsStore = useRunsStore()
const toast = useToast()

const runId = computed(() => route.params.id as string)
const projectId = computed(() => route.query.projectId as string ?? '')

const currentStatus = computed<RunStatus>(() => {
  return (runsStore.current?.status as RunStatus) ?? null
})

const canPause = computed(() => currentStatus.value === 'running')
const canResume = computed(() => currentStatus.value === 'paused')

const pauseError = ref<string | null>(null)

async function handlePause() {
  if (!projectId.value || !runId.value) return
  pauseError.value = null
  try {
    await runsStore.pauseRun(projectId.value, runId.value)
    toast.add({ severity: 'success', summary: 'Run paused', life: 3000 })
  } catch (err) {
    pauseError.value = err instanceof Error ? err.message : 'Failed to pause run'
    toast.add({ severity: 'error', summary: 'Failed to pause run', detail: pauseError.value, life: 5000 })
  }
}

async function handleResume() {
  if (!projectId.value || !runId.value) return
  pauseError.value = null
  try {
    await runsStore.resumeRun(projectId.value, runId.value)
    toast.add({ severity: 'success', summary: 'Run resumed', life: 3000 })
  } catch (err) {
    pauseError.value = err instanceof Error ? err.message : 'Failed to resume run'
    toast.add({ severity: 'error', summary: 'Failed to resume run', detail: pauseError.value, life: 5000 })
  }
}
</script>

<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold">Run Detail</h1>
      <div class="flex items-center gap-3">
        <RunStatusIndicator :status="currentStatus" />
        <Button
          v-if="canPause"
          label="Pause"
          icon="pi pi-pause"
          severity="warn"
          :loading="runsStore.isPausing"
          data-testid="pause-run-btn"
          @click="handlePause"
        />
        <Button
          v-if="canResume"
          label="Resume"
          icon="pi pi-play"
          severity="success"
          :loading="runsStore.isResuming"
          data-testid="resume-run-btn"
          @click="handleResume"
        />
      </div>
    </div>
  </div>
</template>
