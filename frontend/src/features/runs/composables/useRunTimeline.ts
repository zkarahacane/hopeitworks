import { computed, type Ref } from 'vue'
import type { components } from '@/api/schema'

type RunStep = components['schemas']['RunStep']

/** A root step grouped with its retry attempts. */
export interface StepGroup {
  root: RunStep
  retries: RunStep[]
}

/**
 * Groups a flat list of run steps into root steps and their associated retries.
 * Root steps have no `parent_step_id`; retry steps reference a root via `parent_step_id`.
 */
export function useRunTimeline(steps: Ref<RunStep[]>) {
  const groupedSteps = computed<StepGroup[]>(() => {
    const rootSteps = steps.value
      .filter((s) => !s.parent_step_id)
      .sort((a, b) => a.step_order - b.step_order)

    return rootSteps.map((root) => ({
      root,
      retries: steps.value
        .filter((s) => s.parent_step_id === root.id)
        .sort((a, b) => (a.retry_count ?? 0) - (b.retry_count ?? 0)),
    }))
  })

  return { groupedSteps }
}
