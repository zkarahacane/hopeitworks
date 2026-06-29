import { apiClient } from '@/api/client'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { getApiErrorMessage } from '@/utils/apiError'
import type { components } from '@/api/schema'

/** Types straight from the frozen OpenAPI contract. */
export type PlanningConnector = components['schemas']['PlanningConnector']
export type SetPlanningConnectorRequest = components['schemas']['SetPlanningConnectorRequest']
export type PlanningStatusMapping = components['schemas']['PlanningStatusMapping']
export type PlanningStatusOptions = components['schemas']['PlanningStatusOptions']
export type PlanningStatusOption = components['schemas']['PlanningStatusOption']

/** Documented error codes returned on 422 from PUT /projects/{id}/planning/connector. */
const CONNECTOR_ERROR_MESSAGES: Record<string, string> = {
  PLANNING_CONNECTOR_NO_GIT_CONNECTION:
    'Write-back requires a configured git connection for this project. Go to Settings → Git connection to add one.',
  PLANNING_CONNECTOR_INVALID_MAPPING:
    'Write-back is enabled but no usable status mapping is configured. Map at least one internal status to an external option, or disable write-back.',
}

/** Extract the `error.code` from an OpenAPI-fetch error body, if present. */
function errorCode(errBody: unknown): string | undefined {
  return (errBody as { error?: { code?: string } } | undefined)?.error?.code
}

/** Turn a failed PUT response into a friendly, code-aware message. */
function friendlyError(status: number | undefined, errBody: unknown, fallback: string): string {
  if (status === 403) {
    return 'You must be the project owner or a global admin to configure the planning connector.'
  }
  const code = errorCode(errBody)
  if (code && CONNECTOR_ERROR_MESSAGES[code]) {
    return CONNECTOR_ERROR_MESSAGES[code]!
  }
  return getApiErrorMessage(errBody, fallback)
}

/**
 * usePlanningConnector — wraps the three `/projects/{id}/planning/connector` endpoints
 * with the `useAsyncAction` (loading / error / data) pattern over the typed `apiClient`.
 *
 * - `fetchConnector(projectId)`  → GET; returns null on 404 (no connector yet)
 * - `saveConnector(projectId, payload)` → PUT; maps documented 422 codes to friendly messages
 * - `fetchStatusOptions(projectId, overrides?)` → GET status-options (live probe)
 */
export function usePlanningConnector() {
  /**
   * GET — fetch the persisted connector config.
   * Returns null (not an error) when no connector has been configured yet (404).
   */
  const fetchConnector = useAsyncAction(
    async (projectId: string): Promise<PlanningConnector | null> => {
      const res = await apiClient.GET('/projects/{projectId}/planning/connector', {
        params: { path: { projectId } },
      })
      if (res?.response?.status === 404) {
        return null
      }
      if (res?.error || !res?.data) {
        throw new Error(
          friendlyError(res?.response?.status, res?.error, 'Failed to load planning connector.'),
        )
      }
      return res.data
    },
  )

  /**
   * PUT — create or replace the connector config.
   * Maps 422 PLANNING_CONNECTOR_NO_GIT_CONNECTION / PLANNING_CONNECTOR_INVALID_MAPPING
   * to user-friendly messages.
   */
  const saveConnector = useAsyncAction(
    async (
      projectId: string,
      payload: SetPlanningConnectorRequest,
    ): Promise<PlanningConnector> => {
      const res = await apiClient.PUT('/projects/{projectId}/planning/connector', {
        params: { path: { projectId } },
        body: payload,
      })
      if (res?.error || !res?.data) {
        throw new Error(
          friendlyError(res?.response?.status, res?.error, 'Failed to save planning connector.'),
        )
      }
      return res.data
    },
  )

  /**
   * GET status-options — live-probes the tracker to list the single-select field options.
   * Pass `project_url` / `status_field` to override the persisted connector (useful before
   * the connector is saved for the first time).
   */
  const fetchStatusOptions = useAsyncAction(
    async (
      projectId: string,
      overrides?: { project_url?: string; status_field?: string },
    ): Promise<PlanningStatusOptions> => {
      const res = await apiClient.GET(
        '/projects/{projectId}/planning/connector/status-options',
        {
          params: {
            path: { projectId },
            query: {
              project_url: overrides?.project_url,
              status_field: overrides?.status_field,
            },
          },
        },
      )
      if (res?.error || !res?.data) {
        throw new Error(
          friendlyError(
            res?.response?.status,
            res?.error,
            'Failed to load tracker status options. Check the project URL and git connection.',
          ),
        )
      }
      return res.data
    },
  )

  return { fetchConnector, saveConnector, fetchStatusOptions }
}
