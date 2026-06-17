import { computed } from 'vue'
import type { PipelineGroup } from '@/stores/pipelineConfig'

export function usePipelineValidation(groups: { value: PipelineGroup[] } | (() => PipelineGroup[])) {
  const allGroups = computed(() => typeof groups === 'function' ? groups() : groups.value)

  const hasSteps = computed(() => allGroups.value.some(g => g.steps.length > 0))

  const isEmpty = computed(() => allGroups.value.length === 0 || !hasSteps.value)

  const agentRunsWithoutAgent = computed(() =>
    allGroups.value.flatMap(g => g.steps.filter(s => s.action_type === 'agent_run' && !s.agent_id))
  )

  const isValid = computed(() => !isEmpty.value && agentRunsWithoutAgent.value.length === 0)

  const validationWarnings = computed(() => {
    const warnings: string[] = []
    if (agentRunsWithoutAgent.value.length > 0) {
      warnings.push(`${agentRunsWithoutAgent.value.length} agent_run step(s) have no agent assigned`)
    }
    return warnings
  })

  return { hasSteps, isEmpty, isValid, validationWarnings, agentRunsWithoutAgent }
}
