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
          <div v-if="project.repo_url" class="sm:col-span-2">
            <dt class="text-sm font-medium text-surface-500">Repository URL</dt>
            <dd class="mt-1">
              <a
                :href="project.repo_url"
                target="_blank"
                rel="noopener noreferrer"
                class="underline"
              >{{ project.repo_url }}</a>
            </dd>
          </div>
          <div>
            <dt class="text-sm font-medium text-surface-500">Git Provider</dt>
            <dd class="mt-1">{{ project.git_provider || '-' }}</dd>
          </div>
          <div>
            <dt class="text-sm font-medium text-surface-500">Agent Runtime</dt>
            <dd class="mt-1">{{ project.agent_runtime || '-' }}</dd>
          </div>
          <div>
            <dt class="text-sm font-medium text-surface-500">Default Model</dt>
            <dd class="mt-1">{{ project.default_model || '-' }}</dd>
          </div>
        </dl>
      </template>
    </Card>

    <p v-else class="text-surface-500">Loading project details...</p>
  </div>
</template>
