<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Breadcrumb from 'primevue/breadcrumb'
import Message from 'primevue/message'
import Button from 'primevue/button'
import Skeleton from 'primevue/skeleton'
import Toast from 'primevue/toast'
import ProjectSettingsForm from '@/features/projects/ProjectSettingsForm.vue'
import { useProjects } from '@/composables/useProjects'

const route = useRoute()
const toast = useToast()
const projectId = route.params.id as string

const { currentProject, getProject, updateProject, clearCurrentProject } = useProjects()

const breadcrumbItems = computed(() => [
  { label: 'Projects', route: '/projects' },
  {
    label: currentProject.value?.name ?? 'Project',
    route: `/projects/${projectId}`,
  },
  { label: 'Settings' },
])

const breadcrumbHome = { icon: 'pi pi-home', route: '/' }

onMounted(() => {
  getProject.execute(projectId)
})

onUnmounted(() => {
  clearCurrentProject()
})

async function handleSave(payload: { name: string; description: string }) {
  try {
    await updateProject.execute(projectId, payload)
    toast.add({
      severity: 'success',
      summary: 'Saved',
      detail: 'Project settings saved',
      life: 3000,
    })
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to save project settings',
      life: 5000,
    })
  }
}

function handleRetry() {
  getProject.execute(projectId)
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <Toast />

    <Breadcrumb :model="breadcrumbItems" :home="breadcrumbHome" />

    <h1 class="text-2xl font-bold">Project Settings</h1>

    <!-- Loading state -->
    <div v-if="getProject.isLoading.value && !currentProject" class="flex flex-col gap-4 max-w-xl">
      <Skeleton height="2.5rem" />
      <Skeleton height="6rem" />
      <Skeleton width="6rem" height="2.5rem" />
    </div>

    <!-- Error state -->
    <div v-else-if="getProject.error.value" class="flex flex-col gap-4 max-w-xl">
      <Message severity="error" :closable="false">
        Failed to load project. Please try again.
      </Message>
      <Button label="Retry" severity="secondary" icon="pi pi-refresh" @click="handleRetry" />
    </div>

    <!-- Settings form -->
    <ProjectSettingsForm
      v-else-if="currentProject"
      :project="currentProject"
      :is-saving="updateProject.isLoading.value"
      @save="handleSave"
    />
  </div>
</template>
