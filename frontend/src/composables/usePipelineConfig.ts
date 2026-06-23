import { computed, onMounted, watch } from 'vue'
import {
  usePipelineConfigStore,
  type PipelineStep,
  type Guard,
  type TransitionPolicy,
} from '@/stores/pipelineConfig'
import type { Ref } from 'vue'

/**
 * Composable for pipeline configuration operations.
 * Wraps the pipeline config store with reactive computed properties.
 * Auto-fetches config on mount when projectId is provided.
 */
export function usePipelineConfig(projectId: Ref<string>) {
  const store = usePipelineConfigStore()

  onMounted(() => {
    if (projectId.value) {
      store.fetchConfig(projectId.value)
    }
  })

  watch(projectId, (newId) => {
    if (newId) {
      store.fetchConfig(newId)
    }
  })

  async function retry() {
    if (projectId.value) {
      await store.fetchConfig(projectId.value)
    }
  }

  async function saveConfig(): Promise<boolean> {
    if (projectId.value) {
      return await store.saveConfig(projectId.value)
    }
    return false
  }

  function addStep(step: PipelineStep) {
    store.addStep(step)
  }

  function removeStep(index: number) {
    store.removeStep(index)
  }

  function reorderSteps(fromIndex: number, toIndex: number) {
    store.reorderSteps(fromIndex, toIndex)
  }

  function updateStep(index: number, step: PipelineStep) {
    store.updateStep(index, step)
  }

  function addGroup(name?: string) {
    store.addGroup(name)
  }

  function removeGroup(groupId: string) {
    store.removeGroup(groupId)
  }

  function renameGroup(groupId: string, name: string) {
    store.renameGroup(groupId, name)
  }

  function updateGroupTransition(groupId: string, transition: TransitionPolicy) {
    store.updateGroupTransition(groupId, transition)
  }

  function addGuard(groupId: string, stepId?: string) {
    store.addGuard(groupId, stepId)
  }

  function removeGuard(groupId: string, guardIndex: number, stepId?: string) {
    store.removeGuard(groupId, guardIndex, stepId)
  }

  function updateGuard(groupId: string, guardIndex: number, guard: Guard, stepId?: string) {
    store.updateGuard(groupId, guardIndex, guard, stepId)
  }

  function addStepToGroup(groupId: string, step: PipelineStep) {
    store.addStepToGroup(groupId, step)
  }

  function removeStepFromGroup(groupId: string, stepId: string) {
    store.removeStepFromGroup(groupId, stepId)
  }

  function updateStepInGroup(groupId: string, stepId: string, step: PipelineStep) {
    store.updateStepInGroup(groupId, stepId, step)
  }

  function reorderStepsInGroup(groupId: string, fromIndex: number, toIndex: number) {
    store.reorderStepsInGroup(groupId, fromIndex, toIndex)
  }

  function reorderGroups(fromIndex: number, toIndex: number) {
    store.reorderGroups(fromIndex, toIndex)
  }

  return {
    config: computed(() => store.config),
    groups: computed(() => store.groups),
    steps: computed(() => store.steps),
    isLoading: computed(() => store.isLoading),
    isSaving: computed(() => store.isSaving),
    error: computed(() => store.error),
    isDirty: computed(() => store.isDirty),
    fetchConfig: retry,
    saveConfig,
    retry,
    addStep,
    removeStep,
    reorderSteps,
    updateStep,
    addGroup,
    removeGroup,
    renameGroup,
    updateGroupTransition,
    addGuard,
    removeGuard,
    updateGuard,
    addStepToGroup,
    removeStepFromGroup,
    updateStepInGroup,
    reorderStepsInGroup,
    reorderGroups,
  }
}
