<script setup lang="ts">
import { inject, ref } from 'vue'
import type { Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Toast from 'primevue/toast'
import ProjectSettingsForm from '@/features/projects/ProjectSettingsForm.vue'
import GitConnectionCard from '@/features/projects/GitConnectionCard.vue'
import PlanningConnectorCard from '@/features/projects/PlanningConnectorCard.vue'
import { useProjects } from '@/composables/useProjects'
import type { Project, UpdateProjectPayload } from '@/stores/projects'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const projectId = route.params.id as string

const project = inject<Ref<Project | null>>('project')
const { updateProject, deleteProject } = useProjects()

const isDeleting = ref(false)

async function handleSave(payload: UpdateProjectPayload) {
  const result = await updateProject.execute(projectId, payload)
  if (result && project) {
    project.value = result
    toast.add({
      severity: 'success',
      summary: 'Project settings saved',
      life: 3000,
    })
  }
}

async function handleDelete() {
  if (isDeleting.value) return
  isDeleting.value = true
  try {
    await deleteProject(projectId)
    toast.add({
      severity: 'success',
      summary: 'Project deleted',
      life: 3000,
    })
    await router.push('/projects')
  } catch (e) {
    // RG5: keep the user on Settings, surface the error, project is preserved.
    toast.add({
      severity: 'error',
      summary: 'Delete failed',
      detail: e instanceof Error ? e.message : 'Could not delete the project.',
      life: 5000,
    })
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <h2 class="text-xl font-semibold">Settings</h2>

    <ProjectSettingsForm
      v-if="project"
      :project="project"
      :is-saving="updateProject.isLoading.value"
      :is-deleting="isDeleting"
      @save="handleSave"
      @delete="handleDelete"
    />

    <GitConnectionCard v-if="project" :project="project" />

    <PlanningConnectorCard v-if="project" :project="project" />

    <p v-else :style="{ color: 'var(--p-text-muted-color)' }">Loading project settings...</p>

    <Toast />
  </div>
</template>
