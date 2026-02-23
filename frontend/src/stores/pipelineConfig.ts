import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'
import { getApiErrorMessage } from '@/utils/apiError'

export type PipelineConfig = components['schemas']['PipelineConfig']
export type PipelineStep = components['schemas']['PipelineStep']
export type PipelineGroup = components['schemas']['PipelineGroup']
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

  const groups = computed(() => config.value?.groups ?? [])

  /** Flattened steps across all groups for backward-compatible views */
  const steps = computed(() =>
    groups.value.flatMap((g) => g.steps),
  )

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

  /** Update the local config groups (marks as dirty) */
  function updateGroups(newGroups: PipelineGroup[]) {
    if (config.value) {
      config.value = { ...config.value, groups: newGroups }
      isDirty.value = true
    }
  }

  /** Add a new empty group to the pipeline config */
  function addGroup(name = 'New Group') {
    if (!config.value) return
    const newGroup: PipelineGroup = {
      id: crypto.randomUUID(),
      name,
      steps: [],
    }
    config.value = {
      ...config.value,
      groups: [...config.value.groups, newGroup],
    }
    isDirty.value = true
  }

  /** Remove a group by id (and all its steps) */
  function removeGroup(groupId: string) {
    if (!config.value) return
    config.value = {
      ...config.value,
      groups: config.value.groups.filter((g: PipelineGroup) => g.id !== groupId),
    }
    isDirty.value = true
  }

  /** Update a group name in place */
  function renameGroup(groupId: string, name: string) {
    if (!config.value) return
    config.value = {
      ...config.value,
      groups: config.value.groups.map((g: PipelineGroup) =>
        g.id === groupId ? { ...g, name } : g,
      ),
    }
    isDirty.value = true
  }

  /** Add a step to a specific group by group id */
  function addStepToGroup(groupId: string, step: PipelineStep) {
    if (!config.value) return
    config.value = {
      ...config.value,
      groups: config.value.groups.map((g: PipelineGroup) =>
        g.id === groupId ? { ...g, steps: [...g.steps, step] } : g,
      ),
    }
    isDirty.value = true
  }

  /** Remove a step from a specific group */
  function removeStepFromGroup(groupId: string, stepId: string) {
    if (!config.value) return
    config.value = {
      ...config.value,
      groups: config.value.groups.map((g: PipelineGroup) =>
        g.id === groupId
          ? { ...g, steps: g.steps.filter((s: PipelineStep) => s.id !== stepId) }
          : g,
      ),
    }
    isDirty.value = true
  }

  /** Update a step within a specific group by step id */
  function updateStepInGroup(groupId: string, stepId: string, step: PipelineStep) {
    if (!config.value) return
    config.value = {
      ...config.value,
      groups: config.value.groups.map((g: PipelineGroup) =>
        g.id === groupId
          ? { ...g, steps: g.steps.map((s: PipelineStep) => (s.id === stepId ? step : s)) }
          : g,
      ),
    }
    isDirty.value = true
  }

  /** Reorder steps within a specific group */
  function reorderStepsInGroup(groupId: string, fromIndex: number, toIndex: number) {
    if (!config.value) return
    config.value = {
      ...config.value,
      groups: config.value.groups.map((g: PipelineGroup) => {
        if (g.id !== groupId) return g
        const newSteps = [...g.steps]
        const [moved] = newSteps.splice(fromIndex, 1)
        if (!moved) return g
        newSteps.splice(toIndex, 0, moved)
        return { ...g, steps: newSteps }
      }),
    }
    isDirty.value = true
  }

  /** Reorder groups by moving from one index to another */
  function reorderGroups(fromIndex: number, toIndex: number) {
    if (!config.value) return
    const newGroups = [...config.value.groups]
    const [moved] = newGroups.splice(fromIndex, 1)
    if (!moved) return
    newGroups.splice(toIndex, 0, moved)
    config.value = { ...config.value, groups: newGroups }
    isDirty.value = true
  }

  /** Add a step to the last group (or creates a default group if none exist) */
  function addStep(step: PipelineStep) {
    if (!config.value) return
    const currentGroups = [...config.value.groups]
    if (currentGroups.length === 0) {
      currentGroups.push({ id: 'default', name: 'Default', steps: [] })
    }
    const lastGroup = currentGroups[currentGroups.length - 1]!
    currentGroups[currentGroups.length - 1] = {
      ...lastGroup,
      steps: [...lastGroup.steps, step],
    }
    config.value = { ...config.value, groups: currentGroups }
    isDirty.value = true
  }

  /** Remove a step by flat index across all groups */
  function removeStep(index: number) {
    if (!config.value) return
    let remaining = index
    const newGroups = config.value.groups.map((g) => {
      if (remaining >= g.steps.length) {
        remaining -= g.steps.length
        return g
      }
      const newSteps = g.steps.filter((_: PipelineStep, i: number) => i !== remaining)
      remaining = -1
      return { ...g, steps: newSteps }
    })
    config.value = { ...config.value, groups: newGroups }
    isDirty.value = true
  }

  /** Reorder steps within the flat step list */
  function reorderSteps(fromIndex: number, toIndex: number) {
    if (!config.value) return
    const allSteps = groups.value.flatMap((g) => g.steps)
    const removed = allSteps.splice(fromIndex, 1)
    const movedStep = removed[0]
    if (!movedStep) return
    allSteps.splice(toIndex, 0, movedStep)
    // Re-distribute steps into existing groups proportionally
    let offset = 0
    const newGroups = config.value.groups.map((g) => {
      const groupSteps = allSteps.slice(offset, offset + g.steps.length)
      offset += g.steps.length
      return { ...g, steps: groupSteps }
    })
    config.value = { ...config.value, groups: newGroups }
    isDirty.value = true
  }

  /** Update a single step at the given flat index */
  function updateStep(index: number, step: PipelineStep) {
    if (!config.value) return
    let remaining = index
    const newGroups = config.value.groups.map((g) => {
      if (remaining >= g.steps.length) {
        remaining -= g.steps.length
        return g
      }
      const newSteps = [...g.steps]
      newSteps[remaining] = step
      remaining = -1
      return { ...g, steps: newSteps }
    })
    config.value = { ...config.value, groups: newGroups }
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
          body: { groups: config.value.groups },
        },
      )
      if (apiError) {
        throw new Error(getApiErrorMessage(apiError, 'Failed to save pipeline configuration'))
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
    groups,
    steps,
    isLoading,
    error,
    isDirty,
    isSaving,
    fetchConfig,
    updateGroups,
    addGroup,
    removeGroup,
    renameGroup,
    addStepToGroup,
    removeStepFromGroup,
    updateStepInGroup,
    reorderStepsInGroup,
    reorderGroups,
    addStep,
    removeStep,
    reorderSteps,
    updateStep,
    saveConfig,
    reset,
  }
})
