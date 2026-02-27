<script setup lang="ts">
import Tag from 'primevue/tag'
import type { Project } from '@/stores/projects'
import { formatRelativeDate } from '@/utils/formatDate'

defineProps<{
  project: Project
}>()

const emit = defineEmits<{
  click: [projectId: string]
}>()

const providerIcons: Record<string, string> = {
  github: 'pi pi-github',
  gitlab: 'pi pi-gitlab',
  gitea: 'pi pi-server',
  bitbucket: 'pi pi-code',
}

function providerIcon(provider?: string): string {
  return providerIcons[provider ?? ''] ?? 'pi pi-folder'
}

function truncateUrl(url?: string): string {
  if (!url) return ''
  return url
    .replace(/^https?:\/\//, '')
    .replace(/\.git$/, '')
}
</script>

<template>
  <div
    class="flex flex-col cursor-pointer"
    role="button"
    tabindex="0"
    :aria-label="`Project: ${project.name}`"
    @click="emit('click', project.id)"
    @keydown.enter="emit('click', project.id)"
    style="
      border: 1px solid var(--p-surface-200);
      border-radius: var(--p-border-radius);
      background: var(--p-surface-0);
      transition: box-shadow 0.2s;
    "
    @mouseenter="($event.currentTarget as HTMLElement).style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)'"
    @mouseleave="($event.currentTarget as HTMLElement).style.boxShadow = 'none'"
  >
    <!-- Header: icon + name + description -->
    <div class="flex gap-3 p-4">
      <i
        :class="providerIcon(project.git_provider)"
        style="font-size: 1.4rem; color: var(--p-text-muted-color); margin-top: 2px"
      />
      <div class="flex flex-col gap-1" style="min-width: 0">
        <h3 class="m-0" style="font-size: 1.1rem; font-weight: 600">{{ project.name }}</h3>
        <p
          v-if="project.description"
          class="m-0"
          style="
            color: var(--p-text-muted-color);
            font-size: 0.875rem;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
          "
        >
          {{ project.description }}
        </p>
      </div>
    </div>

    <!-- Tags: runtime, provider, model -->
    <div
      class="flex flex-wrap gap-2 px-4 py-2"
      style="border-top: 1px solid var(--p-surface-100)"
    >
      <Tag
        v-if="project.agent_runtime"
        :value="project.agent_runtime"
        severity="info"
      />
      <Tag
        v-if="project.git_provider"
        :value="project.git_provider"
        severity="secondary"
      />
      <Tag
        v-if="project.default_model"
        :value="project.default_model"
        severity="secondary"
      />
    </div>

    <!-- Footer: repo URL + relative date -->
    <div
      class="flex items-center justify-between px-4 py-2"
      style="
        border-top: 1px solid var(--p-surface-100);
        font-size: 0.8rem;
        color: var(--p-text-muted-color);
      "
    >
      <span class="truncate" style="max-width: 60%">{{ truncateUrl(project.repo_url) }}</span>
      <span>Updated {{ formatRelativeDate(project.updated_at) }}</span>
    </div>
  </div>
</template>
