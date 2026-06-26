import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import { getApiErrorMessage } from '@/utils/apiError'
import { useEpicsStore } from '@/stores/epics'

/** The currently active step of a run, for the live kanban. */
export interface LatestRunStep {
  id: string
  name: string
  action_type: string
  status: string
  index: number
  total: number
}

/** Latest run summary attached to a story — lightweight projection for the kanban. */
export interface LatestRun {
  id: string
  status: string
  current_step?: LatestRunStep | null
  /** Optional fields retained for backwards compat with existing card components */
  completed_at?: string
  error_message?: string
}

/**
 * Story entity type.
 * Aligned with the OpenAPI schema (LatestRun/LatestRunStep for live kanban).
 */
export interface Story {
  id: string
  epic_id: string
  project_id: string
  key: string
  title: string
  status: 'backlog' | 'running' | 'done' | 'failed'
  objective?: string
  acceptance_criteria?: string
  target_files?: string[]
  depends_on?: string[]
  scope?: 'backend' | 'frontend' | 'shared'
  /**
   * Name of the stage the card currently sits in, advanced by the executor at
   * stage boundaries. Null/undefined when the story has no live stage (backlog
   * before its first run, or after completion). Matches a PipelineGroup `name`.
   */
  current_stage?: string | null
  latest_run?: LatestRun | null
  /**
   * Planning provenance (read-only, stamped by the import connector).
   * `manual` = created in-app/seed; `markdown`/`github_projects` = imported.
   * Drives the SourceBadge + the source_url deep-link.
   */
  source?: 'manual' | 'markdown' | 'github_projects'
  external_id?: string | null
  source_url?: string | null
  synced_at?: string | null
  created_at: string
  updated_at: string
}

/** Kanban column a story belongs to, derived from its live state (macro / lifecycle view). */
export type KanbanColumn = 'backlog' | 'in_progress' | 'blocked' | 'done' | 'failed'

/** Sentinel column keys for the stage (détail) view's terminal lanes. */
export const STAGE_DONE_COLUMN = '__done__'
export const STAGE_FAILED_COLUMN = '__failed__'
export const STAGE_BACKLOG_COLUMN = '__backlog__'
/**
 * Sentinel for a running story with no live `current_stage` (the executor has not
 * stamped a stage yet). The détail board has no generic "running" lane, so the
 * board resolves this sentinel to the pipeline's entry (first) stage. A running
 * story must never land in the Backlog lane.
 */
export const STAGE_RUNNING_ENTRY = '__running_entry__'

/**
 * Pure function — derives the macro (lifecycle) kanban column for a story.
 * - done  → "done"
 * - failed → "failed"
 * - running + current_step.status === "waiting_approval" → "blocked"
 * - running → "in_progress"
 * - everything else → "backlog"
 */
export function boardColumn(story: Story): KanbanColumn {
  if (story.status === 'done') return 'done'
  if (story.status === 'failed') return 'failed'
  if (story.status === 'running') {
    if (story.latest_run?.current_step?.status === 'waiting_approval') return 'blocked'
    return 'in_progress'
  }
  return 'backlog'
}

/**
 * Pure function — derives the stage (détail) column key for a story.
 *
 * Terminal lifecycle states win over the live stage: a done/failed story sits in
 * its terminal lane regardless of any stale `current_stage`. Otherwise the story
 * is placed in its `current_stage` (a PipelineGroup name). A running story with no
 * live stage yet (executor has not stamped one) returns STAGE_RUNNING_ENTRY,
 * which the board resolves to the pipeline entry stage — never the Backlog lane.
 * Any other stageless story (backlog before its first run) falls back to Backlog.
 *
 * The returned key is either a stage name (matched against the pipeline's stage
 * names by the board) or one of the STAGE_* sentinels.
 */
export function stageColumn(story: Story): string {
  if (story.status === 'done') return STAGE_DONE_COLUMN
  if (story.status === 'failed') return STAGE_FAILED_COLUMN
  if (story.current_stage) return story.current_stage
  if (story.status === 'running') return STAGE_RUNNING_ENTRY
  return STAGE_BACKLOG_COLUMN
}

/** Fields for updating an existing story */
export interface UpdateStoryFields {
  title?: string
  objective?: string
  acceptance_criteria?: string
  target_files?: string[]
  depends_on?: string[]
  scope?: 'backend' | 'frontend' | 'shared'
  status?: 'backlog' | 'running' | 'done' | 'failed'
}

/** Fields for creating a new story */
export interface CreateStoryFields {
  key: string
  title: string
  objective?: string
  acceptance_criteria?: string
  target_files?: string[]
  depends_on?: string[]
  scope?: 'backend' | 'frontend' | 'shared'
  epic_id?: string
}

/** Filter state for the story list */
export interface StoryFilters {
  status: string | null
  search: string
}

// ── SSE payload shapes ─────────────────────────────────────────────────────

interface StoryStatusUpdatedPayload {
  story_id: string
  run_id?: string
  status: 'backlog' | 'running' | 'done' | 'failed'
}

interface RunEventPayload {
  story_id?: string
  run_id?: string
  status?: string
}

interface StepEventPayload {
  story_id?: string
  run_id?: string
  step_id?: string
  name?: string
  action_type?: string
  status?: string
  index?: number
  total?: number
}

interface HitlGatePayload {
  story_id?: string
  run_id?: string
}

interface StageEventPayload {
  story_id?: string
  run_id?: string
  stage_id?: string
  stage_name?: string
}

/**
 * Pinia store for story state management.
 * Handles fetching, filtering, and selecting stories within an epic.
 */
export const useStoriesStore = defineStore('stories', () => {
  const items = ref<Story[]>([])
  const selectedStoryId = ref<string | null>(null)
  const filters = ref<StoryFilters>({ status: null, search: '' })
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  /** Stories filtered by current filter state */
  const filteredStories = computed(() => {
    let result = items.value

    if (filters.value.status && filters.value.status !== 'all') {
      result = result.filter((s) => s.status === filters.value.status)
    }

    if (filters.value.search) {
      const term = filters.value.search.toLowerCase()
      result = result.filter(
        (s) => s.key.toLowerCase().includes(term) || s.title.toLowerCase().includes(term),
      )
    }

    return result
  })

  /** Currently selected story */
  const selectedStory = computed(() =>
    items.value.find((s) => s.id === selectedStoryId.value) ?? null,
  )

  /** Helper: fetch all pages for a given query until all results are collected */
  async function fetchAllPages(
    projectId: string,
    queryParams: Record<string, string | undefined>,
  ): Promise<Story[]> {
    const allStories: Story[] = []
    let page = 1
    const perPage = 50

    while (true) {
      const { data, error: apiError } = await apiClient.GET(
        '/projects/{projectId}/stories',
        {
          params: {
            path: { projectId },
            query: {
              ...queryParams,
              page,
              per_page: perPage,
            },
          },
        },
      )

      if (apiError) {
        throw new Error('Failed to load stories')
      }

      if (!data) break

      const responseData = data as unknown as { data: Story[]; pagination: { total: number } }
      const stories = responseData?.data ?? []
      allStories.push(...stories)

      const pagination = responseData?.pagination
      if (!pagination || allStories.length >= pagination.total) break

      page += 1
    }

    return allStories
  }

  /** Fetch stories for an epic from the API with server-side filtering */
  async function fetchStoriesByEpic(projectId: string, epicId: string) {
    isLoading.value = true
    error.value = null
    try {
      const allStories = await fetchAllPages(projectId, { epic_id: epicId })
      items.value = allStories
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load stories'
    } finally {
      isLoading.value = false
    }
  }

  /** Fetch all stories for a project (across all epics) from the API, fetching all pages */
  async function fetchAllStories(projectId: string) {
    isLoading.value = true
    error.value = null
    try {
      const allStories = await fetchAllPages(projectId, {})
      items.value = allStories
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load stories'
    } finally {
      isLoading.value = false
    }
  }

  /** Set the currently selected story */
  function setSelectedStory(storyId: string | null) {
    selectedStoryId.value = storyId
  }

  /** Update filter state */
  function setFilters(newFilters: Partial<StoryFilters>) {
    filters.value = { ...filters.value, ...newFilters }
  }

  /** Clear current error state */
  function clearError() {
    error.value = null
  }

  /** Update an existing story via PUT API */
  async function updateStory(
    projectId: string,
    storyId: string,
    fields: UpdateStoryFields,
  ): Promise<Story | null> {
    try {
      const { data, error: apiError } = await apiClient.PUT(
        '/projects/{projectId}/stories/{storyId}',
        {
          params: { path: { projectId, storyId } },
          body: fields,
        },
      )
      if (apiError) {
        error.value = getApiErrorMessage(apiError, 'Failed to update story')
        return null
      }
      const updated = data as unknown as Story
      const index = items.value.findIndex((s) => s.id === storyId)
      if (index !== -1) {
        items.value[index] = updated
      }
      return updated
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update story'
      return null
    }
  }

  /** Create a new story via POST API */
  async function createStory(
    projectId: string,
    fields: CreateStoryFields,
  ): Promise<Story | null> {
    try {
      const { data, error: apiError } = await apiClient.POST(
        '/projects/{projectId}/stories',
        {
          params: { path: { projectId } },
          body: fields,
        },
      )
      if (apiError) {
        error.value = getApiErrorMessage(apiError, 'Failed to create story')
        return null
      }
      const created = data as unknown as Story
      items.value.push(created)
      return created
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create story'
      return null
    }
  }

  /**
   * Refresh the board after a planning import committed changes. An import can create
   * epics (markdown `epic:` field / GitHub epic issues), so re-fetch epics first, then
   * re-fetch stories — scoped to the selected epic, or all stories when none is selected.
   * The actual import POST is owned by `usePlanningImport`; this only re-hydrates state.
   */
  async function runPlanningImport(projectId: string, epicId?: string | null) {
    const epicsStore = useEpicsStore()
    await epicsStore.fetchEpics(projectId)
    if (epicId) {
      await fetchStoriesByEpic(projectId, epicId)
    } else {
      await fetchAllStories(projectId)
    }
  }

  /** Reset store state to initial values */
  function reset() {
    items.value = []
    selectedStoryId.value = null
    filters.value = { status: null, search: '' }
    error.value = null
    isLoading.value = false
  }

  /**
   * Handles incoming SSE events and mutates the items array reactively.
   * Silently ignores events for stories not currently in the store.
   */
  function handleSSEEvent(name: string, data: unknown): void {
    switch (name) {
      case 'story.status_updated': {
        const payload = data as StoryStatusUpdatedPayload
        const idx = items.value.findIndex((s) => s.id === payload.story_id)
        if (idx === -1) return
        items.value[idx] = { ...items.value[idx]!, status: payload.status }
        break
      }

      case 'run.started':
      case 'run.completed':
      case 'run.failed':
      case 'run.paused':
      case 'run.resumed':
      case 'run.cancelled': {
        const payload = data as RunEventPayload
        if (!payload.story_id) return
        const idx = items.value.findIndex((s) => s.id === payload.story_id)
        if (idx === -1) return
        const story = items.value[idx]!
        const runStatus = name.split('.')[1]! // e.g. "started", "completed"
        const currentRun = story.latest_run
        items.value[idx] = {
          ...story,
          latest_run: {
            id: payload.run_id ?? currentRun?.id ?? '',
            status: runStatus,
            current_step: name === 'run.started' ? null : currentRun?.current_step,
          },
        }
        break
      }

      case 'step.started':
      case 'step.completed':
      case 'step.failed':
      case 'step.cancelled': {
        const payload = data as StepEventPayload
        if (!payload.story_id) return
        const idx = items.value.findIndex((s) => s.id === payload.story_id)
        if (idx === -1) return
        const story = items.value[idx]!
        const existingStep = story.latest_run?.current_step
        const stepStatus = name.split('.')[1]! // "started", "completed", etc.

        const updatedStep: LatestRunStep | null =
          name === 'step.started' || name === 'step.completed' || name === 'step.failed'
            ? {
                id: payload.step_id ?? existingStep?.id ?? '',
                name: payload.name ?? existingStep?.name ?? '',
                action_type: payload.action_type ?? existingStep?.action_type ?? '',
                status: payload.status ?? stepStatus,
                index: payload.index ?? existingStep?.index ?? 0,
                total: payload.total ?? existingStep?.total ?? 0,
              }
            : null // step.cancelled → clear current step

        items.value[idx] = {
          ...story,
          latest_run: story.latest_run
            ? { ...story.latest_run, current_step: updatedStep }
            : { id: payload.run_id ?? '', status: 'running', current_step: updatedStep },
        }
        break
      }

      case 'stage.entered': {
        // The executor advances stories.current_stage to the stage name at each
        // stage boundary. Mirror it onto the card so the stage (détail) board
        // moves the card into its new stage column live.
        const payload = data as StageEventPayload
        if (!payload.story_id || !payload.stage_name) return
        const idx = items.value.findIndex((s) => s.id === payload.story_id)
        if (idx === -1) return
        items.value[idx] = { ...items.value[idx]!, current_stage: payload.stage_name }
        break
      }

      case 'stage.awaiting_start': {
        // The card has parked idle at the entry of a not-yet-started manual stage:
        // advance current_stage and mark the run paused so the board surfaces the
        // "Go · start stage" affordance (the executor pauses without a HITL gate).
        const payload = data as StageEventPayload
        if (!payload.story_id || !payload.stage_name) return
        const idx = items.value.findIndex((s) => s.id === payload.story_id)
        if (idx === -1) return
        const story = items.value[idx]!
        const currentRun = story.latest_run
        items.value[idx] = {
          ...story,
          current_stage: payload.stage_name,
          latest_run: {
            id: payload.run_id ?? currentRun?.id ?? '',
            status: 'paused',
            current_step: null,
          },
        }
        break
      }

      case 'hitl_gate.pending': {
        // Mark the current step as waiting_approval so boardColumn() routes to "blocked"
        const payload = data as HitlGatePayload
        if (!payload.story_id) return
        const idx = items.value.findIndex((s) => s.id === payload.story_id)
        if (idx === -1) return
        const story = items.value[idx]!
        if (!story.latest_run) return
        items.value[idx] = {
          ...story,
          latest_run: {
            ...story.latest_run,
            current_step: story.latest_run.current_step
              ? { ...story.latest_run.current_step, status: 'waiting_approval' }
              : null,
          },
        }
        break
      }

      default:
        // Unrecognised event — ignore
        break
    }
  }

  return {
    items,
    selectedStoryId,
    filters,
    isLoading,
    error,
    filteredStories,
    selectedStory,
    fetchStoriesByEpic,
    fetchAllStories,
    updateStory,
    createStory,
    runPlanningImport,
    setSelectedStory,
    setFilters,
    clearError,
    reset,
    handleSSEEvent,
  }
})
