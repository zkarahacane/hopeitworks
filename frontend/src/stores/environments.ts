import { ref } from 'vue'
import { defineStore } from 'pinia'
import {
  getEnvironment,
  putEnvironment,
  deleteEnvironment,
} from '@/api/environment'
import type { Environment, EnvironmentInput } from '@/api/environment'

export type { Environment, EnvironmentInput }

/**
 * Pinia store for project environment state management.
 * One environment per project (upsert via PUT).
 */
export const useEnvironmentsStore = defineStore('environments', () => {
  const environment = ref<Environment | null>(null)
  const isLoading = ref(false)
  const isSaving = ref(false)
  const error = ref<string | null>(null)

  /** Fetch the environment for a project. Sets null on 404 (not yet configured). */
  async function fetchEnvironment(projectId: string) {
    isLoading.value = true
    error.value = null
    try {
      environment.value = await getEnvironment(projectId)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load environment'
    } finally {
      isLoading.value = false
    }
  }

  /** Upsert the environment for a project. Returns the saved environment or null on failure. */
  async function saveEnvironment(
    projectId: string,
    input: EnvironmentInput,
  ): Promise<Environment | null> {
    isSaving.value = true
    error.value = null
    try {
      const saved = await putEnvironment(projectId, input)
      environment.value = saved
      return saved
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to save environment'
      return null
    } finally {
      isSaving.value = false
    }
  }

  /** Delete the environment for a project. Returns true on success. */
  async function removeEnvironment(projectId: string): Promise<boolean> {
    error.value = null
    try {
      await deleteEnvironment(projectId)
      environment.value = null
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete environment'
      return false
    }
  }

  /** Reset store to initial state */
  function reset() {
    environment.value = null
    isLoading.value = false
    isSaving.value = false
    error.value = null
  }

  return {
    environment,
    isLoading,
    isSaving,
    error,
    fetchEnvironment,
    saveEnvironment,
    removeEnvironment,
    reset,
  }
})
