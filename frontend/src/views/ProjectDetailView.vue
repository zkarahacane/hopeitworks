<script setup lang="ts">
import { computed, provide, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import TabMenu from 'primevue/tabmenu'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import { useProject } from '@/composables/useProject'
import { useAuthStore } from '@/stores/auth'
import { useRunsStore } from '@/stores/runs'
import CircuitBreakerBanner from '@/features/projects/CircuitBreakerBanner.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const runsStore = useRunsStore()

const projectId = route.params.id as string
const { project, isLoading, error, retry } = useProject(projectId)

provide('project', project)

const tabs = [
  { label: 'Overview', icon: 'pi pi-home', route: 'project-overview' },
  { label: 'Board', icon: 'pi pi-th-large', route: 'project-board' },
  { label: 'Runs', icon: 'pi pi-play', route: 'project-runs' },
  { label: 'Pipeline', icon: 'pi pi-cog', route: 'project-pipeline' },
  { label: 'Templates', icon: 'pi pi-file', route: 'project-templates' },
  { label: 'Costs', icon: 'pi pi-dollar', route: 'project-costs' },
  { label: 'Notifications', icon: 'pi pi-bell', route: 'project-notifications' },
]

const activeIndex = computed(() => {
  const currentName = route.name as string
  // epic-detail is under board
  if (currentName === 'epic-detail') return 1
  const idx = tabs.findIndex((t) => t.route === currentName)
  return idx >= 0 ? idx : 0
})

/** Whether the circuit breaker banner should be shown */
const showCircuitBreaker = computed(
  () => project.value?.circuit_breaker_active || runsStore.circuitBreakerActive,
)

/** Whether the current user is an admin */
const isAdmin = computed(() => authStore.user?.role === 'admin')

function onTabChange(event: { index: number }) {
  const tab = tabs[event.index]
  if (tab) {
    router.push({ name: tab.route, params: { id: projectId } })
  }
}

/** Sync circuit breaker state from project data into the store */
watch(
  () => project.value?.circuit_breaker_active,
  (active) => {
    if (typeof active === 'boolean') {
      runsStore.circuitBreakerActive = active
    }
  },
  { immediate: true },
)

watch(
  () => route.params.id,
  (newId, oldId) => {
    if (newId !== oldId) {
      router.go(0)
    }
  },
)
</script>

<template>
  <div class="flex flex-col">
    <!-- Header -->
    <div class="flex items-center gap-3 border-b border-surface-200 px-6 py-4">
      <Button
        icon="pi pi-arrow-left"
        text
        rounded
        severity="secondary"
        aria-label="Back to projects"
        data-testid="back-to-projects"
        @click="router.push({ name: 'projects' })"
      />
      <div v-if="isLoading" class="flex items-center gap-3">
        <Skeleton width="12rem" height="1.75rem" />
      </div>
      <h1 v-else-if="project" class="text-2xl font-bold" data-testid="project-name">
        {{ project.name }}
      </h1>
      <h1 v-else class="text-2xl font-bold text-surface-400">Project</h1>
    </div>

    <!-- Circuit breaker banner -->
    <div v-if="showCircuitBreaker" class="px-6 pt-4">
      <CircuitBreakerBanner
        :project-id="projectId"
        :is-admin="isAdmin"
        data-testid="circuit-breaker-banner-wrapper"
        @reset="runsStore.circuitBreakerActive = false"
      />
    </div>

    <!-- Error state -->
    <div v-if="error" class="p-6">
      <Message severity="error" :closable="false" data-testid="project-error">
        <div class="flex items-center gap-3">
          <span>{{ error }}</span>
          <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
        </div>
      </Message>
    </div>

    <!-- Tab menu -->
    <div v-if="!error" class="border-b border-surface-200 px-6">
      <TabMenu
        :model="tabs"
        :active-index="activeIndex"
        data-testid="project-tabs"
        @tab-change="onTabChange"
      />
    </div>

    <!-- Child route content -->
    <router-view v-if="!error" />
  </div>
</template>
