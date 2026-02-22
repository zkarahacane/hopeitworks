import { ref, onMounted } from 'vue'
import { apiClient } from '@/api/client'

/** Lightweight run shape for list views (matches OpenAPI Run schema fields). */
export interface RunSummary {
  id: string
  project_id: string
  story_id: string
  status: string
  progress: number
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
  project_name?: string
  story_key?: string
}

/**
 * Composable for fetching recent runs.
 *
 * When `projectId` is provided, fetches runs for that single project.
 * When omitted, fetches the user's projects and fans out per-project run
 * fetches, then merges and sorts by created_at descending.
 */
export function useRecentRuns(options?: { projectId?: string; limit?: number }) {
  const projectId = options?.projectId
  const limit = options?.limit ?? 10

  const runs = ref<RunSummary[]>([])
  const isLoading = ref(false)
  const error = ref<Error | null>(null)

  async function fetchRuns() {
    isLoading.value = true
    error.value = null
    try {
      if (projectId) {
        await fetchProjectRuns(projectId)
      } else {
        await fetchGlobalRuns()
      }
    } catch (e) {
      error.value = e instanceof Error ? e : new Error(String(e))
    } finally {
      isLoading.value = false
    }
  }

  async function fetchProjectRuns(pid: string) {
    const { data, error: apiError } = await apiClient.GET(
      '/projects/{projectId}/runs',
      {
        params: {
          path: { projectId: pid },
          query: { per_page: limit, page: 1 },
        },
      },
    )
    if (apiError) throw new Error('Failed to load runs')
    runs.value = ((data?.data ?? []) as RunSummary[]).map((r) => ({ ...r, project_id: r.project_id ?? pid }))
  }

  async function fetchGlobalRuns() {
    // Fetch up to 5 projects to fan out run fetches
    const { data: projectsData, error: projError } = await apiClient.GET('/projects', {
      params: { query: { per_page: 5, page: 1 } },
    })
    if (projError) throw new Error('Failed to load projects')

    interface ProjectItem { id: string; name: string }
    const projects = (projectsData?.data ?? []) as ProjectItem[]
    if (projects.length === 0) {
      runs.value = []
      return
    }

    // Fan out: fetch runs from each project concurrently
    const results = await Promise.allSettled(
      projects.map(async (proj) => {
        const { data } = await apiClient.GET('/projects/{projectId}/runs', {
          params: {
            path: { projectId: proj.id },
            query: { per_page: limit, page: 1 },
          },
        })
        return ((data?.data ?? []) as RunSummary[]).map((r) => ({
          ...r,
          project_id: r.project_id ?? proj.id,
          project_name: proj.name,
        }))
      }),
    )

    // Merge successful results, sort by created_at desc, take top N
    const allRuns: RunSummary[] = []
    for (const result of results) {
      if (result.status === 'fulfilled') {
        allRuns.push(...result.value)
      }
    }
    allRuns.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    runs.value = allRuns.slice(0, limit)
  }

  onMounted(fetchRuns)

  return { runs, isLoading, error, refresh: fetchRuns }
}
