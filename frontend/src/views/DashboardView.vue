<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import { useRecentRuns } from '@/features/runs/composables/useRecentRuns'
import type { RunSummary } from '@/features/runs/composables/useRecentRuns'
import { useHITLStore } from '@/stores/hitl'
import { useAuthStore } from '@/stores/auth'
import { useRuntimeStream } from '@/stores/runtimeStream'
import { useProjects } from '@/composables/useProjects'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import { formatRelativeDate } from '@/utils/formatDate'

const router = useRouter()
const authStore = useAuthStore()
const hitlStore = useHITLStore()
const stream = useRuntimeStream()

// Recent runs (cross-project, limit 20 so dedup still gives enough)
const { runs, isLoading: runsLoading, error: runsError, refresh: refreshRuns } = useRecentRuns({ limit: 20 })

// Seed the runtime stream with REST timing so elapsed renders for runs that were
// already running before this view opened (no `run.started` SSE captured). The
// stream stays authoritative: hydration never overwrites a live `startedAt`.
watch(
  runs,
  (list) => {
    for (const run of list) {
      stream.hydrateRunStartedAt(run.id, run.started_at, run.completed_at, run.status)
    }
  },
  { immediate: true },
)

// Projects (prefetch to prime the store; projects ref not used directly in this view)
const { fetchProjects } = useProjects()

onMounted(() => {
  hitlStore.fetchPending()
  fetchProjects({ per_page: 5, page: 1 })
})

// Clock tick for live elapsed timers
let tickInterval: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  tickInterval = setInterval(() => stream.tick(), 1000)
})
onBeforeUnmount(() => {
  if (tickInterval) clearInterval(tickInterval)
})

// ── Dedup runs by story_id, keep most recent per story ───────────────────────
const dedupedRuns = computed(() => {
  const byStory = new Map<string, RunSummary>()
  for (const run of runs.value) {
    const existing = byStory.get(run.story_id)
    if (!existing || new Date(run.created_at) > new Date(existing.created_at)) {
      byStory.set(run.story_id, run)
    }
  }
  return [...byStory.values()].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  )
})

// ── KPI counts ────────────────────────────────────────────────────────────────
const activeRunCount = computed(() => dedupedRuns.value.filter((r) => r.status === 'running').length)
const gatesWaiting = computed(() => hitlStore.pendingCount)

// ── Helpers ───────────────────────────────────────────────────────────────────
function formatElapsed(s: number): string {
  const m = Math.floor(s / 60)
  const sec = s % 60
  return `${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
}

/**
 * Live elapsed string for a running run. Returns the placeholder when the run
 * has no known start (pending / not yet hydrated) so we never show "00:00" for
 * a run whose duration is simply unknown.
 */
function displayElapsed(run: RunSummary): string {
  if (!run.started_at) return '—'
  return formatElapsed(stream.runElapsedSeconds(run.id))
}

function navigateToRun(run: RunSummary) {
  router.push({ name: 'run-detail', params: { id: run.id }, query: { projectId: run.project_id } })
}

function navigateToApproval(item: typeof hitlStore.pendingItems[0]) {
  router.push(
    '/projects/' + item.projectId + '/runs/' + item.runId + '/approve/' + item.stepId,
  )
}

</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <!-- Header -->
    <div>
      <h1 class="text-2xl font-bold">Welcome back, {{ authStore.user?.name || 'there' }}</h1>
      <p class="mt-1" style="color: var(--p-text-muted-color)">Here's what's happening right now.</p>
    </div>

    <!-- KPI cards row -->
    <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <!-- Active Runs -->
      <div
        class="flex flex-col gap-3 rounded-xl p-4"
        style="background: var(--surface-raised); border: 1px solid var(--surface-border)"
      >
        <div class="flex items-center justify-between">
          <span class="text-sm" style="color: var(--p-text-muted-color)">Active runs</span>
          <i class="pi pi-play text-lg" style="color: var(--status-running-color)" />
        </div>
        <span class="text-3xl font-bold" style="color: var(--status-running-color)">{{ activeRunCount }}</span>
      </div>

      <!-- Gates waiting -->
      <div
        class="flex flex-col gap-3 rounded-xl p-4"
        style="background: var(--surface-raised); border: 1px solid var(--surface-border)"
      >
        <div class="flex items-center justify-between">
          <span class="text-sm" style="color: var(--p-text-muted-color)">Gates waiting</span>
          <i
            class="pi pi-pause-circle text-lg"
            :class="{ 'amber-breathe': gatesWaiting > 0 }"
            style="color: var(--status-gate-color)"
          />
        </div>
        <span class="text-3xl font-bold" style="color: var(--status-gate-color)">{{ gatesWaiting }}</span>
      </div>

      <!-- Stories done today -->
      <div
        class="flex flex-col gap-3 rounded-xl p-4"
        style="background: var(--surface-raised); border: 1px solid var(--surface-border)"
      >
        <div class="flex items-center justify-between">
          <span class="text-sm" style="color: var(--p-text-muted-color)">Stories done today</span>
          <i class="pi pi-check-circle text-lg" style="color: var(--status-done-color)" />
        </div>
        <span class="text-3xl font-bold" style="color: var(--status-done-color)">0</span>
      </div>

      <!-- Spend today -->
      <div
        class="flex flex-col gap-3 rounded-xl p-4"
        style="background: var(--surface-raised); border: 1px solid var(--surface-border)"
      >
        <div class="flex items-center justify-between">
          <span class="text-sm" style="color: var(--p-text-muted-color)">Spend today</span>
          <i class="pi pi-dollar text-lg" style="color: var(--status-done-color)" />
        </div>
        <span class="text-3xl font-bold font-mono" style="color: var(--status-done-color)">$0.00</span>
      </div>
    </div>

    <!-- Main 2-col grid -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <!-- Live Runs (2/3) -->
      <div class="lg:col-span-2 flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">Live Runs</h2>
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
          <Skeleton v-for="i in 4" :key="i" width="100%" height="3rem" />
        </div>

        <!-- Error -->
        <Message v-else-if="runsError" severity="error" :closable="false">
          <div class="flex items-center gap-3">
            <span>{{ runsError.message }}</span>
            <Button label="Retry" icon="pi pi-refresh" text size="small" @click="refreshRuns" />
          </div>
        </Message>

        <!-- Empty -->
        <div
          v-else-if="dedupedRuns.length === 0"
          class="flex flex-col items-center py-10 gap-2"
          style="color: var(--p-text-muted-color)"
        >
          <i class="pi pi-play text-3xl" />
          <p>No runs yet</p>
        </div>

        <!-- Run rows -->
        <div v-else class="flex flex-col gap-2">
          <div
            v-for="run in dedupedRuns"
            :key="run.id"
            class="flex items-center gap-3 rounded-lg px-4 py-3 cursor-pointer transition-colors"
            style="background: var(--surface-raised); border: 1px solid var(--surface-border)"
            @click="navigateToRun(run)"
          >
            <StatusBadge :status="run.status" />

            <div class="flex flex-col min-w-0 flex-1 gap-0.5">
              <span class="font-mono text-sm truncate">{{ run.story_key || run.story_id }}</span>
              <span class="text-xs truncate" style="color: var(--p-text-muted-color)">{{ run.project_name || run.project_id }}</span>
            </div>

            <span class="text-xs shrink-0 font-mono" style="color: var(--p-text-muted-color)">
              <template v-if="run.status === 'running'">
                {{ displayElapsed(run) }}
              </template>
              <template v-else>
                {{ formatRelativeDate(run.started_at || run.created_at) }}
              </template>
            </span>
          </div>
        </div>
      </div>

      <!-- Needs you panel (1/3) -->
      <div class="flex flex-col gap-3">
        <h2 class="text-lg font-semibold">
          Needs you
          <span
            v-if="gatesWaiting > 0"
            class="ml-2 rounded-full px-2 py-0.5 text-xs font-bold"
            style="background: var(--status-gate-color); color: var(--surface-base)"
          >{{ gatesWaiting }}</span>
        </h2>

        <!-- Empty -->
        <div
          v-if="hitlStore.pendingItems.length === 0"
          class="rounded-lg px-4 py-6 text-center text-sm"
          style="color: var(--p-text-muted-color); background: var(--surface-raised); border: 1px solid var(--surface-border)"
        >
          No pending approvals
        </div>

        <!-- Items (up to 3) -->
        <div
          v-for="item in hitlStore.pendingItems.slice(0, 3)"
          :key="item.hitlRequestId"
          class="flex flex-col gap-2 rounded-lg px-4 py-3"
          style="border-left: 3px solid var(--status-gate-color); background: var(--surface-raised)"
        >
          <div class="flex items-center gap-2">
            <span class="font-mono text-sm">{{ item.storyKey }}</span>
            <span
              class="rounded px-1.5 py-0.5 text-xs font-semibold"
              style="background: color-mix(in srgb, var(--status-gate-color) 20%, transparent); color: var(--status-gate-color)"
            >awaiting</span>
          </div>
          <p class="text-sm truncate" style="color: var(--p-text-muted-color)">{{ item.storyTitle || '—' }}</p>
          <Button
            label="Approve"
            severity="warn"
            size="small"
            @click.stop="navigateToApproval(item)"
          />
        </div>

        <!-- More link if > 3 -->
        <Button
          v-if="hitlStore.pendingItems.length > 3"
          :label="`View all ${hitlStore.pendingItems.length} approvals`"
          text
          size="small"
          severity="warn"
          class="w-full"
          @click="router.push({ name: 'approvals' })"
        />
      </div>
    </div>
  </div>
</template>
