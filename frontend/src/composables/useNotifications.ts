import { onMounted, ref } from 'vue'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

export type NotificationConfig = components['schemas']['NotificationConfig']
export type CreateNotificationConfigRequest =
  components['schemas']['CreateNotificationConfigRequest']
export type UpdateNotificationConfigRequest =
  components['schemas']['UpdateNotificationConfigRequest']

/**
 * Composable for managing notification configs for a project.
 * Handles CRUD operations with optimistic updates for toggle.
 */
export function useNotifications(projectId: string) {
  const configs = ref<NotificationConfig[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchConfigs() {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiErr } = await apiClient.GET(
        '/projects/{projectId}/notifications',
        { params: { path: { projectId } } },
      )
      if (apiErr) throw apiErr
      configs.value = data?.data ?? []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load channels'
    } finally {
      isLoading.value = false
    }
  }

  async function createConfig(
    payload: CreateNotificationConfigRequest,
  ): Promise<NotificationConfig | null> {
    const { data, error: apiErr } = await apiClient.POST(
      '/projects/{projectId}/notifications',
      { params: { path: { projectId } }, body: payload },
    )
    if (apiErr || !data) return null
    configs.value.push(data)
    return data
  }

  async function updateConfig(id: string, payload: UpdateNotificationConfigRequest): Promise<void> {
    await apiClient.PUT('/projects/{projectId}/notifications/{notificationId}', {
      params: { path: { projectId, notificationId: id } },
      body: payload,
    })
  }

  async function toggleEnabled(config: NotificationConfig) {
    const original = config.enabled
    const idx = configs.value.findIndex((c) => c.id === config.id)
    // optimistic update
    if (idx >= 0) configs.value[idx] = { ...config, enabled: !original }
    const { error: apiErr } = await apiClient.PUT(
      '/projects/{projectId}/notifications/{notificationId}',
      {
        params: { path: { projectId, notificationId: config.id } },
        body: {
          channel_type: config.channel_type,
          config: config.config,
          events_filter: config.events_filter,
          enabled: !original,
        },
      },
    )
    if (apiErr) {
      // revert on failure
      if (idx >= 0) configs.value[idx] = { ...config, enabled: original }
      throw apiErr
    }
  }

  async function deleteConfig(id: string) {
    await apiClient.DELETE('/projects/{projectId}/notifications/{notificationId}', {
      params: { path: { projectId, notificationId: id } },
    })
    configs.value = configs.value.filter((c) => c.id !== id)
  }

  async function testConfig(id: string) {
    const { error: apiErr } = await apiClient.POST(
      '/projects/{projectId}/notifications/{notificationId}/test',
      { params: { path: { projectId, notificationId: id } } },
    )
    if (apiErr) throw apiErr
  }

  onMounted(fetchConfigs)

  return {
    configs,
    isLoading,
    error,
    fetchConfigs,
    createConfig,
    updateConfig,
    toggleEnabled,
    deleteConfig,
    testConfig,
  }
}
