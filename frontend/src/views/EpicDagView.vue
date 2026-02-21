<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import Button from 'primevue/button'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import DagGraph from '@/features/dag/DagGraph.vue'
import { useDagLayout } from '@/features/dag/composables/useDagLayout'
import { useEpicLauncher } from '@/features/dag/composables/useEpicLauncher'

const route = useRoute()
const router = useRouter()
const projectId = route.params.id as string
const epicId = route.params.epicId as string
const toast = useToast()
const confirm = useConfirm()

const { nodes, edges, isLoading, error, retry } = useDagLayout(projectId, epicId)
const { launch, isLaunching, error: launchError, result } = useEpicLauncher(projectId, epicId)

function handleLaunchClick() {
  confirm.require({
    message: `Launch all ${nodes.value.length} stories in this epic? Already-running stories will be skipped.`,
    header: 'Launch Epic Run',
    icon: 'pi pi-play',
    acceptLabel: 'Launch',
    rejectLabel: 'Cancel',
    accept: async () => {
      await launch()
      if (result.value?.epic_run_id) {
        router.push({
          name: 'epic-run-monitor',
          params: { id: projectId, epicRunId: result.value.epic_run_id },
        })
      } else {
        toast.add({
          severity: 'error',
          summary: 'Launch failed',
          detail: launchError.value?.message ?? 'Unexpected error',
          life: 5000,
        })
      }
    },
  })
}
</script>

<template>
  <div class="flex flex-col h-full p-6">
    <Toast />
    <ConfirmDialog />

    <div class="flex items-center gap-3 mb-4">
      <Button
        icon="pi pi-arrow-left"
        severity="secondary"
        text
        rounded
        aria-label="Back to epic"
        @click="router.push({ name: 'epic-detail', params: { id: projectId, epicId } })"
      />
      <h1 class="m-0 text-2xl font-bold flex-1">Epic DAG</h1>
      <Button
        label="Launch Epic"
        :loading="isLaunching"
        :disabled="isLaunching"
        severity="success"
        icon="pi pi-play"
        @click="handleLaunchClick"
      >
        <template v-if="isLaunching">Launching...</template>
      </Button>
    </div>

    <DagGraph :nodes="nodes" :edges="edges" :is-loading="isLoading" :error="error" @retry="retry" />
  </div>
</template>
