<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Toast from 'primevue/toast'
import RunLaunchButton from '@/features/runs/RunLaunchButton.vue'
import RunLaunchConfirmDialog from '@/features/runs/RunLaunchConfirmDialog.vue'
import { useRunLauncher, ALREADY_RUNNING_ERROR } from '@/composables/useRunLauncher'

const route = useRoute()
const toast = useToast()

const projectId = computed(() => route.params.projectId as string)
const storyId = computed(() => route.params.storyId as string)

/** Placeholder story data — will be replaced when story detail API is available. */
const storyKey = ref('S-01')
const storyTitle = ref('Placeholder Story')
const storyStatus = ref<'backlog' | 'running' | 'done' | 'failed'>('backlog')

const dialogVisible = ref(false)
const { isLoading, error, launchRun } = useRunLauncher()

function handleLaunchClick() {
  dialogVisible.value = true
}

async function handleConfirm() {
  const result = await launchRun(projectId.value, storyId.value)

  if (result !== null) {
    toast.add({
      severity: 'success',
      summary: 'Run launched',
      detail: `Run started for ${storyKey.value}`,
      life: 3000,
    })
    dialogVisible.value = false
    storyStatus.value = 'running'
    return
  }

  if (error.value?.message === ALREADY_RUNNING_ERROR) {
    toast.add({
      severity: 'warn',
      summary: 'Already running',
      detail: 'This story already has a run in progress',
      life: 5000,
    })
    // Dialog stays open on 409
    return
  }

  toast.add({
    severity: 'error',
    summary: 'Launch failed',
    detail: error.value?.message ?? 'An unexpected error occurred',
    life: 5000,
  })
  dialogVisible.value = false
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold">{{ storyKey }}: {{ storyTitle }}</h1>
      </div>
      <RunLaunchButton
        :story-id="storyId"
        :story-key="storyKey"
        :story-title="storyTitle"
        :status="storyStatus"
        @launch-click="handleLaunchClick"
      />
    </div>

    <p>Story detail content will be implemented in a future story.</p>

    <RunLaunchConfirmDialog
      v-model:visible="dialogVisible"
      :story-key="storyKey"
      :story-title="storyTitle"
      :loading="isLoading"
      @confirm="handleConfirm"
      @cancel="dialogVisible = false"
    />

    <Toast />
  </div>
</template>
