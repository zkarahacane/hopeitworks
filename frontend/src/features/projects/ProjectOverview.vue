<script setup lang="ts">
import { inject } from 'vue'
import type { Ref } from 'vue'
import Card from 'primevue/card'
import { formatDate } from '@/utils/formatDate'
import type { Project } from '@/stores/projects'

const project = inject<Ref<Project | null>>('project')
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <h2 class="text-xl font-semibold">Overview</h2>

    <Card v-if="project" data-testid="project-overview-card">
      <template #content>
        <dl class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <dt class="text-sm font-medium text-surface-500">Name</dt>
            <dd class="mt-1 text-lg">{{ project.name }}</dd>
          </div>
          <div>
            <dt class="text-sm font-medium text-surface-500">Created</dt>
            <dd class="mt-1 text-lg">{{ formatDate(project.created_at) }}</dd>
          </div>
          <div v-if="project.description" class="sm:col-span-2">
            <dt class="text-sm font-medium text-surface-500">Description</dt>
            <dd class="mt-1">{{ project.description }}</dd>
          </div>
        </dl>
      </template>
    </Card>

    <p v-else class="text-surface-500">Loading project details...</p>
  </div>
</template>
