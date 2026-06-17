import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useEpicsStore } from '@/stores/epics'
import { useStoriesStore } from '@/stores/stories'
import { useRuntimeStream } from '@/stores/runtimeStream'
import { useSSE } from '@/composables/useSSE'

/**
 * Board composable — wires epics, stories, SSE, and the runtimeStream for the
 * Story Board hero screen.
 *
 * The host view calls this once and gets back everything needed to render the
 * kanban: epics for the epic selector, stories for the columns, an epic-picker
 * function, and the live stream store for per-card signals.
 */
export function useBoard(projectId: string) {
  const epicsStore = useEpicsStore()
  const storiesStore = useStoriesStore()
  const stream = useRuntimeStream()

  // ── Epic selector ───────────────────────────────────────────────────────────
  const selectedEpicId = ref<string | null>(null)

  const epics = computed(() => epicsStore.items)
  const isLoadingEpics = computed(() => epicsStore.isLoading)
  const epicsError = computed(() => epicsStore.error)

  function setEpicId(epicId: string | null) {
    selectedEpicId.value = epicId
    if (epicId) {
      storiesStore.fetchStoriesByEpic(projectId, epicId)
    } else {
      storiesStore.reset()
    }
  }

  // ── Stories ─────────────────────────────────────────────────────────────────
  const stories = computed(() => storiesStore.items)
  const isLoadingStories = computed(() => storiesStore.isLoading)
  const storiesError = computed(() => storiesStore.error)

  // ── SSE + runtimeStream wiring ───────────────────────────────────────────────
  // Both stores ingest every event independently — storiesStore for column
  // placement mutations, runtimeStream for live cost/elapsed/gate signals.
  useSSE(projectId, (name, data) => {
    storiesStore.handleSSEEvent(name, data)
    stream.ingest(name, data)
  })

  // 1-second tick drives elapsed timers in runtimeStream getters.
  let tickInterval: ReturnType<typeof setInterval> | null = null

  onMounted(async () => {
    // Load epics and auto-select the first one to show the board immediately.
    await epicsStore.fetchEpics(projectId)
    if (epics.value.length > 0 && !selectedEpicId.value) {
      const first = epics.value[0]
      if (first) {
        selectedEpicId.value = first.id
        storiesStore.fetchStoriesByEpic(projectId, first.id)
      }
    }
    tickInterval = setInterval(() => stream.tick(), 1000)
  })

  onBeforeUnmount(() => {
    if (tickInterval !== null) clearInterval(tickInterval)
  })

  return {
    // Epics
    epics,
    isLoadingEpics,
    epicsError,
    selectedEpicId,
    setEpicId,
    // Stories
    stories,
    isLoadingStories,
    storiesError,
    // Live signals
    stream,
  }
}
