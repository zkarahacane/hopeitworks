<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Button from 'primevue/button'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import Toast from 'primevue/toast'
import StoryDetailPanel from '@/features/board/StoryDetailPanel.vue'
import RunLaunchConfirmDialog from '@/features/runs/RunLaunchConfirmDialog.vue'
import { useStoryDetail } from '@/composables/useStoryDetail'
import { useRunLauncher, ALREADY_RUNNING_ERROR } from '@/composables/useRunLauncher'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const projectId = computed(() => route.params.projectId as string)
const storyId = computed(() => route.params.storyId as string)

const { story: storyRef, isLoading, error, retry } = useStoryDetail(projectId.value, storyId.value)
const story = computed(() => storyRef.value)

const dialogVisible = ref(false)
const { isLoading: launchLoading, error: launchError, launchRun } = useRunLauncher()

function handleLaunchClick() {
  dialogVisible.value = true
}

async function handleConfirm() {
  if (!story.value) return

  const result = await launchRun(projectId.value, storyId.value)

  if (result !== null) {
    toast.add({
      severity: 'success',
      summary: 'Run launched',
      detail: `Run started for ${story.value.key}`,
      life: 3000,
    })
    dialogVisible.value = false
    return
  }

  if (launchError.value?.message === ALREADY_RUNNING_ERROR) {
    toast.add({
      severity: 'warn',
      summary: 'Already running',
      detail: 'This story already has a run in progress',
      life: 5000,
    })
    return
  }

  toast.add({
    severity: 'error',
    summary: 'Launch failed',
    detail: launchError.value?.message ?? 'An unexpected error occurred',
    life: 5000,
  })
  dialogVisible.value = false
}

function navigateToEpic() {
  if (story.value?.epic_id) {
    router.push({
      name: 'epic-detail',
      params: { id: projectId.value, epicId: story.value.epic_id },
    })
  }
}
</script>

<template>
  <div class="flex flex-col h-full p-6">
    <Toast />

    <div class="flex items-center gap-3 mb-4">
      <Button
        icon="pi pi-arrow-left"
        severity="secondary"
        text
        rounded
        aria-label="Back to epic"
        @click="navigateToEpic"
      />
      <h1 class="m-0 text-2xl font-bold">Story Detail</h1>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="flex flex-col gap-4 p-6">
      <div class="flex items-center justify-between">
        <Skeleton width="8rem" height="1.25rem" />
        <Skeleton width="6rem" height="2rem" />
      </div>
      <Skeleton width="60%" height="1.75rem" />
      <Skeleton width="100%" height="6rem" />
      <Skeleton width="100%" height="8rem" />
    </div>

    <!-- Error -->
    <Message v-else-if="error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error.message }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
      </div>
    </Message>

    <!-- Story -->
    <StoryDetailPanel
      v-else-if="story"
      :story="story"
      :project-id="projectId"
      :all-stories="[]"
      :show-launch-button="true"
      @launch-click="handleLaunchClick"
    />

    <RunLaunchConfirmDialog
      v-if="story"
      v-model:visible="dialogVisible"
      :story-key="story.key"
      :story-title="story.title"
      :loading="launchLoading"
      @confirm="handleConfirm"
      @cancel="dialogVisible = false"
    />
  </div>
</template>
