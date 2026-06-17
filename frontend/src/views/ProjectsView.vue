<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import type { PageState } from 'primevue/paginator'
import Button from 'primevue/button'
import Message from 'primevue/message'
import ProgressSpinner from 'primevue/progressspinner'
import Toast from 'primevue/toast'
import ProjectCardGrid from '@/features/projects/ProjectCardGrid.vue'
import ProjectEmptyState from '@/features/projects/ProjectEmptyState.vue'
import CreateProjectDialog from '@/features/projects/CreateProjectDialog.vue'
import { useProjects } from '@/composables/useProjects'
import type { Project } from '@/stores/projects'

const router = useRouter()
const toast = useToast()
const { projects, pagination, isLoading, error, fetchProjects, retry } = useProjects()

const perPage = 20
const first = ref(0)
const showCreateDialog = ref(false)

onMounted(() => {
  fetchProjects({ page: 1, per_page: perPage })
})

function handlePage(event: PageState) {
  const newPage = Math.floor(event.first / event.rows) + 1
  first.value = event.first
  fetchProjects({ page: newPage, per_page: event.rows })
}

function handleCardClick(projectId: string) {
  router.push({ name: 'project-overview', params: { id: projectId } })
}

function handleCreated(project: Project) {
  toast.add({
    severity: 'success',
    summary: 'Project created',
    detail: `"${project.name}" has been created successfully`,
    life: 3000,
  })
  router.push({ name: 'project-overview', params: { id: project.id } })
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Projects</h1>
      <Button
        label="New Project"
        icon="pi pi-plus"
        severity="success"
        @click="showCreateDialog = true"
      />
    </div>

    <ProgressSpinner
      v-if="isLoading && projects.length === 0"
      class="flex justify-center"
    />

    <Message v-else-if="error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
      </div>
    </Message>

    <ProjectEmptyState
      v-else-if="!isLoading && !error && projects.length === 0"
      @create="showCreateDialog = true"
    />

    <template v-else>
      <ProjectCardGrid
        :projects="projects"
        :total-records="pagination?.total ?? 0"
        :rows="perPage"
        :loading="isLoading"
        :first="first"
        @page="handlePage"
        @project-click="handleCardClick"
      />

      <!-- Connect a repo card -->
      <div
        v-if="!isLoading && !error"
        class="flex flex-col items-center justify-center gap-2"
        role="button"
        tabindex="0"
        aria-label="Connect a repository"
        style="
          border: 2px dashed var(--p-surface-300);
          border-radius: 0.5rem;
          background: transparent;
          cursor: pointer;
          min-height: 8rem;
          transition: border-color 0.15s, background 0.15s;
        "
        @click="showCreateDialog = true"
        @keydown.enter="showCreateDialog = true"
        @mouseenter="($event.currentTarget as HTMLElement).style.borderColor = 'var(--p-surface-400)'"
        @mouseleave="($event.currentTarget as HTMLElement).style.borderColor = 'var(--p-surface-300)'"
      >
        <i
          class="pi pi-plus-circle"
          style="font-size: 1.5rem; color: var(--p-text-muted-color)"
        />
        <span style="color: var(--p-text-muted-color); font-size: 0.875rem">
          Connect a repo
        </span>
      </div>
    </template>

    <CreateProjectDialog
      v-model:visible="showCreateDialog"
      @created="handleCreated"
    />

    <Toast />
  </div>
</template>
