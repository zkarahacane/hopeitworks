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
/** Outcome of a delete: a 204, a failure, or coalesced because already in flight. */
export type DeleteKeyResult = 'deleted' | 'error' | 'busy'

export function useAPIKeys() {
  const keys = ref<APIKey[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  // Ids whose DELETE is in flight, to coalesce concurrent calls (anti double-fire).
  const deleting = new Set<string>()

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

  /**
   * Delete an API key by ID. Concurrent calls for the same key are coalesced: a
   * call issued while a delete of that key is still in flight returns 'busy'
   * without firing a second DELETE (bug #288 RG3, anti double-fire). Returns
   * 'deleted' on a 204 (the key is removed from the list, no reload), or 'error'
   * on any failure (the key is kept — no optimistic removal).
   */
  async function deleteKey(keyId: string): Promise<DeleteKeyResult> {
    if (deleting.has(keyId)) {
      return 'busy'
    }
    deleting.add(keyId)
    error.value = null
    try {
      const { error: apiError } = await apiClient.DELETE('/users/me/api-keys/{keyId}', {
        params: { path: { keyId } },
      })
      if (apiError) {
        error.value = 'Failed to delete API key'
        return 'error'
      }
      keys.value = keys.value.filter((k) => k.id !== keyId)
      return 'deleted'
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete API key'
      return 'error'
    } finally {
      deleting.delete(keyId)
    }
  }

  return { keys, isLoading, error, fetchKeys, createKey, deleteKey }
}
