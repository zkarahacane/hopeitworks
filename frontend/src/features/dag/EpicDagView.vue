<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import Button from 'primevue/button'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import DagGraph from './DagGraph.vue'
import DagInspector from './DagInspector.vue'
import { useDagLayout } from './composables/useDagLayout'
import { useDagLiveStream } from './composables/useDagLiveStream'
import { useEpicLauncher } from './composables/useEpicLauncher'
import { statusFamily } from '@/utils/statusToken'

/**
 * EpicDagView — the Execution Graph hero (DARK flagship).
 *
 * Orchestrates: the live SSE → runtimeStream host (timers + active sets), the
 * layout (rich nodes + marching edges), the side inspector, and the launch
 * action. Logic lives in composables; this component assembles them.
 */
const route = useRoute()
const router = useRouter()
const projectId = route.params.id as string
const epicId = route.params.epicId as string
const toast = useToast()
const confirm = useConfirm()

// Live host: SSE → runtimeStream + tick loop. Returns the connection status.
const { sseStatus } = useDagLiveStream(projectId)

// Layout consumes the live stream for per-node signals. No story→run map is
// available from the DAG endpoint today, so live signals fall back to the
// node's REST status + demo seeds (see useDagLayout); the wiring is in place
// for when an epic-run id can be threaded in.
const { nodes, edges, summary, nodeByKey, isLoading, error, retry } = useDagLayout(
  projectId,
  epicId,
)

const { launch, isLaunching, error: launchError, result } = useEpicLauncher(projectId, epicId)

// ── Selection (inspector target) ──────────────────────────────────────────────
const selectedKey = ref<string | null>(null)
const selectedNode = computed(() => (selectedKey.value ? (nodeByKey.value.get(selectedKey.value) ?? null) : null))

function selectNode(key: string) {
  selectedKey.value = key
}

// ── Dark / light canvas toggle ────────────────────────────────────────────────
const dark = ref(true)
function toggleTheme() {
  dark.value = !dark.value
}

// ── Live header indicator ─────────────────────────────────────────────────────
const runningCount = computed(() => summary.value.running)

function retryNode() {
  // Retry not yet implemented — backend API endpoint pending
}

function handleLaunchClick() {
  confirm.require({
    message: `Launch all ${summary.value.total} stories in this epic? Already-running stories will be skipped.`,
    header: 'Launch Epic Run',
    icon: 'pi pi-play',
    acceptLabel: 'Launch',
    rejectLabel: 'Cancel',
    accept: async () => {
      await launch()
      if (result.value?.epic_run_id) {
        router.push({
          name: 'epic-run-monitor',
          params: { id: projectId, epicRunId: result.value.epic_run_id },
        })
      } else {
        toast.add({
          severity: 'error',
          summary: 'Launch failed',
          detail: launchError.value?.message ?? 'Unexpected error',
          life: 5000,
        })
      }
    },
  })
}

// Legend dots route through the status families (running/done/failed).
const legend = computed(() => [
  { family: statusFamily('running'), label: 'running', count: summary.value.running },
  { family: statusFamily('done'), label: 'done', count: summary.value.done },
  { family: statusFamily('failed'), label: 'failed', count: summary.value.failed },
])
</script>

<template>
  <div class="flex flex-col h-full">
    <Toast />
    <ConfirmDialog />

    <!-- Header: breadcrumb + live running indicator -->
    <header class="flex items-center justify-between gap-3 px-6 pt-5 pb-3">
      <div class="flex items-center gap-2">
        <Button
          icon="pi pi-arrow-left"
          severity="secondary"
          text
          rounded
          aria-label="Back to epic"
          @click="router.push({ name: 'epic-detail', params: { id: projectId, epicId } })"
        />
        <nav
          class="flex items-center gap-1.5 font-mono"
          :style="{ fontSize: '0.78rem', color: 'var(--p-text-muted-color)' }"
          data-testid="dag-breadcrumb"
        >
          <span>hopeitworks</span>
          <span aria-hidden="true">/</span>
          <span>{{ projectId }}</span>
          <span aria-hidden="true">/</span>
          <span :style="{ color: 'var(--p-text-color)' }">epic·{{ epicId }}</span>
        </nav>
      </div>

      <div class="flex items-center gap-3">
        <span
          v-if="runningCount > 0"
          class="inline-flex items-center gap-2"
          :style="{ fontSize: '0.8rem', color: `var(--status-running-color)` }"
          data-testid="dag-running-indicator"
        >
          <span
            class="live-pulse inline-block rounded-full"
            :style="{ width: '0.5rem', height: '0.5rem', backgroundColor: 'var(--status-running-color)' }"
            aria-hidden="true"
          />
          {{ runningCount }} running
        </span>
        <Button
          label="Launch Epic"
          :loading="isLaunching"
          :disabled="isLaunching"
          severity="success"
          icon="pi pi-play"
          @click="handleLaunchClick"
        />
      </div>
    </header>

    <!-- Title + subtitle + legend -->
    <div class="flex items-end justify-between gap-3 px-6 pb-3">
      <div class="flex flex-col gap-1">
        <h1 class="m-0" :style="{ fontSize: '1.5rem', fontWeight: 700 }">Execution Graph</h1>
        <p
          class="m-0"
          :style="{ fontSize: '0.85rem', color: 'var(--p-text-muted-color)' }"
          data-testid="dag-subtitle"
        >
          {{ summary.total }} stories · {{ summary.running }} running in parallel · isolated containers
        </p>
      </div>
      <ul class="flex items-center gap-4 m-0 p-0" data-testid="dag-legend">
        <li v-for="item in legend" :key="item.label" class="flex items-center gap-1.5">
          <span
            class="inline-block rounded-full"
            :style="{ width: '0.5rem', height: '0.5rem', backgroundColor: `var(--status-${item.family}-color)` }"
            aria-hidden="true"
          />
          <span :style="{ fontSize: '0.75rem', color: 'var(--p-text-muted-color)' }">
            {{ item.label }} {{ item.count }}
          </span>
        </li>
      </ul>
    </div>

    <!-- Graph + inspector -->
    <div class="flex flex-1 min-h-0">
      <div class="flex-1 min-w-0">
        <DagGraph
          :nodes="nodes"
          :edges="edges"
          :is-loading="isLoading"
          :error="error"
          :selected-key="selectedKey"
          :dark="dark"
          @retry="retry"
          @select="selectNode"
          @retry-node="retryNode"
          @toggle-theme="toggleTheme"
        />
      </div>
      <DagInspector :node="selectedNode" :sse-status="sseStatus" />
    </div>
  </div>
</template>
