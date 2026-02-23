/**
 * Pipeline stage grouping utilities.
 * Maps live RunStep objects to their respective pipeline groups (stages)
 * using the cumulative step count from the pipeline config snapshot.
 */

/** Minimal shape of a pipeline group from the config snapshot. */
export interface StageGroup {
  id: string
  name: string
  steps: unknown[]
}

/**
 * Groups run steps into their respective pipeline stages based on the
 * cumulative step count in each group from the config snapshot.
 *
 * If groups is empty or undefined, all steps are placed under a single
 * 'default' stage labeled "Pipeline".
 */
export function groupStepsByStage<T extends { step_order: number }>(
  groups: StageGroup[] | undefined,
  steps: T[],
): Map<string, T[]> {
  const result = new Map<string, T[]>()

  if (!groups || groups.length === 0) {
    result.set('default', [...steps])
    return result
  }

  let offset = 0
  for (const group of groups) {
    const groupStepCount = group.steps.length
    const stageSteps = steps.filter(
      (s) => s.step_order >= offset && s.step_order < offset + groupStepCount,
    )
    result.set(group.id, stageSteps)
    offset += groupStepCount
  }

  return result
}
