import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import { getApiErrorMessage } from '@/utils/apiError'

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
  latest_run?: LatestRun | null
  created_at: string
  updated_at: string
}

/** Kanban column a story belongs to, derived from its live state. */
export type KanbanColumn = 'backlog' | 'in_progress' | 'blocked' | 'done' | 'failed'

/**
 * Pure function — derives the kanban column for a story.
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
    setSelectedStory,
    setFilters,
    clearError,
    reset,
    handleSSEEvent,
  }
})
