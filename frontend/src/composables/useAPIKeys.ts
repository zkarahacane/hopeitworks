import { ref } from 'vue'
import { apiClient } from '@/api/client'

/** A user API key (hint only — never the raw key) */
export interface APIKey {
  id: string
  provider: string
  key_name: string
  key_hint: string
  created_at: string
}

/**
 * Composable for managing the current user's API keys.
 * Provides fetch, create, and delete operations.
 */
export function useAPIKeys() {
  const keys = ref<APIKey[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  /** Fetch the current user's API keys */
  async function fetchKeys() {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET('/users/me/api-keys')
      if (apiError) {
        error.value = 'Failed to load API keys'
        return
      }
      keys.value = (data as APIKey[]) ?? []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load API keys'
    } finally {
      isLoading.value = false
    }
  }

  /** Create a new API key for the given provider */
  async function createKey(provider: string, keyName: string, apiKey: string): Promise<boolean> {
    error.value = null
    try {
      const { error: apiError } = await apiClient.POST('/users/me/api-keys', {
        body: { provider, key_name: keyName, api_key: apiKey },
      })
      if (apiError) {
        error.value = 'Failed to create API key'
        return false
      }
      await fetchKeys()
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create API key'
      return false
    }
  }

  /** Delete an API key by ID */
  async function deleteKey(keyId: string): Promise<boolean> {
    error.value = null
    try {
      const { error: apiError } = await apiClient.DELETE('/users/me/api-keys/{keyId}', {
        params: { path: { keyId } },
      })
      if (apiError) {
        error.value = 'Failed to delete API key'
        return false
      }
      keys.value = keys.value.filter((k) => k.id !== keyId)
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete API key'
      return false
    }
  }

  return { keys, isLoading, error, fetchKeys, createKey, deleteKey }
}
