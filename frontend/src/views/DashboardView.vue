<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import ProgressBar from 'primevue/progressbar'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import Card from 'primevue/card'
import { useRecentRuns } from '@/features/runs/composables/useRecentRuns'
import { useHITLStore } from '@/stores/hitl'
import { useProjects } from '@/composables/useProjects'
import { runStatusSeverity } from '@/utils/runStatus'
import { formatRelativeDate } from '@/utils/formatDate'
import type { RunSummary } from '@/features/runs/composables/useRecentRuns'

const router = useRouter()

// Recent runs (cross-project, limit 10)
const { runs, isLoading: runsLoading, error: runsError, refresh: refreshRuns } = useRecentRuns({ limit: 10 })

// Pending approvals
const hitlStore = useHITLStore()
onMounted(() => hitlStore.fetchPending())

// Projects (quick access, limit 5)
const { projects, isLoading: projectsLoading, error: projectsError, fetchProjects } = useProjects()
onMounted(() => fetchProjects({ per_page: 5, page: 1 }))

const activeRunCount = computed(() => runs.value.filter((r) => r.status === 'running').length)
const projectCount = computed(() => projects.value.length)

function onRunRowClick(row: RunSummary) {
  router.push({ name: 'run-detail', params: { id: row.id }, query: { projectId: row.project_id } })
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <!-- Header -->
    <div>
      <h1 class="text-2xl font-bold">Dashboard</h1>
      <p class="text-surface-500 mt-1">Welcome back</p>
    </div>

    <!-- Stat cards row -->
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
      <Card class="shadow-sm">
        <template #content>
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-orange-100 text-orange-600">
              <i class="pi pi-bell text-lg" />
            </div>
            <div>
              <p class="text-sm text-surface-500">Pending Approvals</p>
              <p class="text-2xl font-bold">{{ hitlStore.pendingCount }}</p>
            </div>
          </div>
          <Button
            v-if="hitlStore.pendingCount > 0"
            label="View approvals"
            text
            size="small"
            class="mt-2 !p-0"
            @click="router.push({ name: 'approvals' })"
          />
        </template>
      </Card>

      <Card class="shadow-sm">
        <template #content>
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 text-blue-600">
              <i class="pi pi-play text-lg" />
            </div>
            <div>
              <p class="text-sm text-surface-500">Active Runs</p>
              <p class="text-2xl font-bold">{{ activeRunCount }}</p>
            </div>
          </div>
        </template>
      </Card>

      <Card class="shadow-sm">
        <template #content>
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100 text-green-600">
              <i class="pi pi-folder text-lg" />
            </div>
            <div>
              <p class="text-sm text-surface-500">Projects</p>
              <p class="text-2xl font-bold">{{ projectCount }}</p>
            </div>
          </div>
        </template>
      </Card>
    </div>

    <!-- Main content grid -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <!-- Recent Runs (wide column) -->
      <div class="lg:col-span-2">
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-lg font-semibold">Recent Runs</h2>
          <Button
            v-if="!runsLoading && !runsError"
            icon="pi pi-refresh"
            text
            rounded
            size="small"
            severity="secondary"
            aria-label="Refresh runs"
            @click="refreshRuns"
          />
        </div>

        <!-- Loading -->
        <div v-if="runsLoading" class="flex flex-col gap-2">
          <Skeleton v-for="i in 4" :key="i" width="100%" height="2.5rem" />
        </div>

        <!-- Error -->
        <Message v-else-if="runsError" severity="error" :closable="false">
          <div class="flex items-center gap-3">
            <span>{{ runsError.message }}</span>
            <Button label="Retry" icon="pi pi-refresh" text size="small" @click="refreshRuns" />
          </div>
        </Message>

        <!-- Empty -->
        <div v-else-if="runs.length === 0" class="flex flex-col items-center py-8 text-surface-400">
          <i class="pi pi-play text-3xl mb-2" />
          <p>No runs yet</p>
        </div>

        <!-- Table -->
        <DataTable
          v-else
          :value="runs"
          :rows="10"
          row-hover
          class="cursor-pointer"
          data-testid="dashboard-runs-table"
          @row-click="onRunRowClick($event.data)"
        >
          <Column field="project_name" header="Project" />
          <Column field="story_key" header="Story Key" />
          <Column field="status" header="Status">
            <template #body="{ data }">
              <Tag :value="data.status" :severity="runStatusSeverity[data.status]" />
            </template>
          </Column>
          <Column field="progress" header="Progress">
            <template #body="{ data }">
              <ProgressBar :value="data.progress ?? 0" :show-value="false" style="height: 0.5rem; width: 6rem" />
            </template>
          </Column>
          <Column field="started_at" header="Started">
            <template #body="{ data }">
              {{ data.started_at ? formatRelativeDate(data.started_at) : '-' }}
            </template>
          </Column>
        </DataTable>
      </div>

      <!-- Projects quick-access (narrow column) -->
      <div>
        <h2 class="text-lg font-semibold mb-3">Projects</h2>

        <!-- Loading -->
        <div v-if="projectsLoading" class="flex flex-col gap-2">
          <Skeleton v-for="i in 3" :key="i" width="100%" height="2rem" />
        </div>

        <!-- Error -->
        <Message v-else-if="projectsError" severity="error" :closable="false" class="text-sm">
          Failed to load projects
        </Message>

        <!-- Empty -->
        <div v-else-if="projects.length === 0" class="text-surface-400 text-sm py-4">
          No projects yet
        </div>

        <!-- List -->
        <div v-else class="flex flex-col gap-1">
          <Button
            v-for="project in projects"
            :key="project.id"
            :label="project.name"
            icon="pi pi-folder"
            text
            class="w-full justify-start"
            @click="router.push({ name: 'project-overview', params: { id: project.id } })"
          />
          <Button
            label="View all projects"
            text
            size="small"
            class="mt-2 !p-0 text-primary"
            @click="router.push({ name: 'projects' })"
          />
        </div>
      </div>
    </div>
  </div>
</template>
