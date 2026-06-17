<script setup lang="ts">
import { inject } from 'vue'
import type { Ref } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Toast from 'primevue/toast'
import ProjectSettingsForm from '@/features/projects/ProjectSettingsForm.vue'
import { useProjects } from '@/composables/useProjects'
import type { Project, UpdateProjectPayload } from '@/stores/projects'

const route = useRoute()
const toast = useToast()
const projectId = route.params.id as string

const project = inject<Ref<Project | null>>('project')
const { updateProject } = useProjects()

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
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <h2 class="text-xl font-semibold">Settings</h2>

    <ProjectSettingsForm
      v-if="project"
      :project="project"
      :is-saving="updateProject.isLoading.value"
      @save="handleSave"
    />

    <p v-else :style="{ color: 'var(--p-text-muted-color)' }">Loading project settings...</p>

    <Toast />
  </div>
</template>
