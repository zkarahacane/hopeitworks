<script setup lang="ts">
import Paginator, { type PageState } from 'primevue/paginator'
import Skeleton from 'primevue/skeleton'
import ProjectCard from './ProjectCard.vue'
import type { Project } from '@/stores/projects'

defineProps<{
  projects: Project[]
  totalRecords: number
  rows: number
  first: number
  loading: boolean
}>()

const emit = defineEmits<{
  projectClick: [projectId: string]
  page: [event: PageState]
}>()
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- Skeleton loading state -->
    <div v-if="loading && projects.length === 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="i in 3"
        :key="i"
        class="flex flex-col gap-3 p-4"
        style="
          border: 1px solid var(--surface-border);
          border-radius: var(--p-border-radius);
          background: var(--surface-raised);
        "
      >
        <div class="flex gap-3">
          <Skeleton width="1.4rem" height="1.4rem" />
          <div class="flex flex-col gap-2" style="flex: 1">
            <Skeleton width="60%" height="1.2rem" />
            <Skeleton width="90%" height="0.875rem" />
          </div>
        </div>
        <Skeleton width="100%" height="2rem" />
        <Skeleton width="100%" height="1.5rem" />
      </div>
    </div>

    <!-- Card grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <ProjectCard
        v-for="project in projects"
        :key="project.id"
        :project="project"
        @click="emit('projectClick', $event)"
      />
    </div>

    <!-- Paginator -->
    <Paginator
      v-if="totalRecords > rows"
      :rows="rows"
      :total-records="totalRecords"
      :first="first"
      @page="emit('page', $event)"
    />
  </div>
</template>
