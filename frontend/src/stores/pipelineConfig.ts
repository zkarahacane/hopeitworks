import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

export type PipelineConfig = components['schemas']['PipelineConfig']
export type PipelineStep = components['schemas']['PipelineStep']
export type RetryPolicy = components['schemas']['RetryPolicy']

/**
 * Pinia store for pipeline configuration state management.
 * Handles fetching, local editing, and saving pipeline config for a project.
 */
export const usePipelineConfigStore = defineStore('pipelineConfig', () => {
  const config = ref<PipelineConfig | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const isDirty = ref(false)
  const isSaving = ref(false)

  const steps = computed(() => config.value?.steps ?? [])

  /** Fetch pipeline configuration from the API */
  async function fetchConfig(projectId: string) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET(
        '/projects/{projectId}/pipeline',
        {
          params: { path: { projectId } },
        },
      )
      if (apiError) {
        error.value = 'Failed to load pipeline configuration'
        return
      }
      config.value = data as PipelineConfig
      isDirty.value = false
    } catch (e) {
      error.value =
        e instanceof Error ? e.message : 'Failed to load pipeline configuration'
    } finally {
      isLoading.value = false
    }
  }

  /** Update the local config steps (marks as dirty) */
  function updateSteps(newSteps: PipelineStep[]) {
    if (config.value) {
      config.value = { ...config.value, steps: newSteps }
      isDirty.value = true
    }
  }

  /** Add a step to the local config */
  function addStep(step: PipelineStep) {
    if (config.value) {
      config.value = { ...config.value, steps: [...config.value.steps, step] }
      isDirty.value = true
    }
  }

  /** Remove a step by index */
  function removeStep(index: number) {
    if (config.value) {
      const newSteps = config.value.steps.filter((_, i) => i !== index)
      config.value = { ...config.value, steps: newSteps }
      isDirty.value = true
    }
  }

  /** Reorder steps by swapping fromIndex and toIndex */
  function reorderSteps(fromIndex: number, toIndex: number) {
    if (!config.value) return
    const newSteps = [...config.value.steps]
    const removed = newSteps.splice(fromIndex, 1)
    const movedStep = removed[0]
    if (!movedStep) return
    newSteps.splice(toIndex, 0, movedStep)
    config.value = { ...config.value, steps: newSteps }
    isDirty.value = true
  }

  /** Update a single step at the given index */
  function updateStep(index: number, step: PipelineStep) {
    if (!config.value) return
    const newSteps = [...config.value.steps]
    newSteps[index] = step
    config.value = { ...config.value, steps: newSteps }
    isDirty.value = true
  }

  /** Save pipeline configuration to the API */
  async function saveConfig(projectId: string): Promise<boolean> {
    if (!config.value) return false
    isSaving.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.PUT(
        '/projects/{projectId}/pipeline',
        {
          params: { path: { projectId } },
          body: { steps: config.value.steps },
        },
      )
      if (apiError) {
        const message =
          (apiError as { error?: { message?: string } })?.error?.message ??
          'Failed to save pipeline configuration'
        throw new Error(message)
      }
      config.value = data as PipelineConfig
      isDirty.value = false
      return true
    } catch (e) {
      error.value =
        e instanceof Error ? e.message : 'Failed to save pipeline configuration'
      return false
    } finally {
      isSaving.value = false
    }
  }

  /** Reset store state */
  function reset() {
    config.value = null
    isLoading.value = false
    error.value = null
    isDirty.value = false
    isSaving.value = false
  }

  return {
    config,
    steps,
    isLoading,
    error,
    isDirty,
    isSaving,
    fetchConfig,
    updateSteps,
    addStep,
    removeStep,
    reorderSteps,
    updateStep,
    saveConfig,
    reset,
  }
})
