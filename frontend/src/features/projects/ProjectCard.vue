<script setup lang="ts">
import Tag from 'primevue/tag'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import type { Project } from '@/stores/projects'
import { formatRelativeDate } from '@/utils/formatDate'

defineProps<{
  project: Project
  activeRunCount?: number
  gateCount?: number
  storyCount?: number
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

function onMouseEnter(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  el.style.boxShadow = '0 4px 12px rgba(0,0,0,0.08)'
  el.style.borderColor = 'var(--p-text-muted-color)'
}

function onMouseLeave(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  el.style.boxShadow = 'none'
  el.style.borderColor = 'var(--surface-border)'
}
</script>

<template>
  <div
    class="flex flex-col"
    role="button"
    tabindex="0"
    :aria-label="`Project: ${project.name}`"
    style="
      background: var(--surface-raised);
      border: 1px solid var(--surface-border);
      border-radius: 0.5rem;
      transition: box-shadow 0.15s, border-color 0.15s;
      cursor: pointer;
    "
    @click="emit('click', project.id)"
    @keydown.enter="emit('click', project.id)"
    @mouseenter="onMouseEnter"
    @mouseleave="onMouseLeave"
  >
    <!-- Header row: provider icon + project name + status indicator -->
    <div class="flex items-center justify-between gap-3 px-4 pt-4 pb-3">
      <div class="flex items-center gap-2" style="min-width: 0">
        <i
          :class="providerIcon(project.git_provider)"
          style="font-size: 1.25rem; color: var(--p-text-muted-color); flex-shrink: 0"
        />
        <h3
          class="m-0 truncate"
          style="font-size: 1rem; font-weight: 600; color: var(--p-text-color)"
        >
          {{ project.name }}
        </h3>
      </div>

      <!-- Status indicator -->
      <div class="flex items-center gap-1.5 flex-shrink-0">
        <template v-if="(activeRunCount ?? 0) > 0">
          <span
            class="inline-block rounded-full live-pulse"
            style="
              width: 0.5rem;
              height: 0.5rem;
              background: var(--status-running-color);
              flex-shrink: 0;
            "
            aria-hidden="true"
          />
          <span style="font-size: 0.75rem; color: var(--status-running-color)">
            {{ activeRunCount }} running
          </span>
        </template>
        <span v-else style="font-size: 0.75rem; color: var(--p-text-muted-color)">idle</span>
      </div>
    </div>

    <!-- Chips row: runtime, provider, model, gate -->
    <div
      class="flex flex-wrap items-center gap-1.5 px-4 pb-3"
      :style="{ borderTop: '1px solid var(--surface-border)', paddingTop: '0.625rem' }"
    >
      <Tag
        v-if="project.agent_runtime"
        :value="project.agent_runtime"
        severity="secondary"
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
      <StatusBadge
        v-if="(gateCount ?? 0) > 0"
        status="paused"
        :label="'◑ ' + gateCount + ' gate'"
        :animated="false"
      />
    </div>

    <!-- Footer: story count + updated date -->
    <div
      class="flex items-center justify-between px-4 py-2"
      style="
        border-top: 1px solid var(--surface-border);
        font-size: 0.75rem;
        font-family: var(--p-font-family-mono, monospace);
        color: var(--p-text-muted-color);
      "
    >
      <span v-if="(storyCount ?? 0) > 0">{{ storyCount }} stories</span>
      <span v-else style="opacity: 0.5">no stories</span>
      <span>updated {{ formatRelativeDate(project.updated_at) }}</span>
    </div>
  </div>
</template>
