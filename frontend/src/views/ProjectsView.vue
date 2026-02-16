<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { DataTablePageEvent } from 'primevue/datatable'
import Button from 'primevue/button'
import Message from 'primevue/message'
import ProgressSpinner from 'primevue/progressspinner'
import ProjectListTable from '@/features/projects/ProjectListTable.vue'
import ProjectEmptyState from '@/features/projects/ProjectEmptyState.vue'
import { useProjects } from '@/composables/useProjects'
import type { Project } from '@/stores/projects'

const router = useRouter()
const { projects, pagination, isLoading, error, fetchProjects, retry } = useProjects()

const perPage = 20
const first = ref(0)

onMounted(() => {
  fetchProjects({ page: 1, per_page: perPage })
})

function handlePage(event: DataTablePageEvent) {
  const newPage = Math.floor(event.first / event.rows) + 1
  first.value = event.first
  fetchProjects({ page: newPage, per_page: event.rows })
}

function handleRowClick(project: Project) {
  router.push({ name: 'project-detail', params: { id: project.id } })
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Projects</h1>
      <Button label="New Project" icon="pi pi-plus" severity="success" disabled />
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
      @create="router.push({ name: 'project-detail', params: { id: 'new' } })"
    />

    <ProjectListTable
      v-else
      :projects="projects"
      :total-records="pagination?.total ?? 0"
      :rows="perPage"
      :loading="isLoading"
      :first="first"
      @page="handlePage"
      @row-click="handleRowClick"
    />
  </div>
</template>
