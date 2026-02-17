import { computed, onMounted, watch } from 'vue'
import { usePipelineConfigStore, type PipelineStep } from '@/stores/pipelineConfig'
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

  return {
    config: computed(() => store.config),
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
  }
}
