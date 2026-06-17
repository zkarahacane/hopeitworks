import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { usePipelineValidation } from '../composables/usePipelineValidation'
import type { PipelineGroup, PipelineStep } from '@/stores/pipelineConfig'

function makeStep(overrides: Partial<PipelineStep> = {}): PipelineStep {
  return {
    id: crypto.randomUUID(),
    name: 'test-step',
    action_type: 'agent_run',
    auto_approve: false,
    retry_policy: { max_retries: 2, retry_type: 'on-failure' },
    ...overrides,
  }
}

function makeGroup(overrides: Partial<PipelineGroup> = {}): PipelineGroup {
  return {
    id: 'g1',
    name: 'Dev',
    steps: [],
    ...overrides,
  }
}

describe('usePipelineValidation', () => {
  it('isEmpty true when no groups', () => {
    const groups = ref<PipelineGroup[]>([])
    const { isEmpty } = usePipelineValidation(groups)
    expect(isEmpty.value).toBe(true)
  })

  it('isEmpty true when groups have no steps', () => {
    const groups = ref([makeGroup({ steps: [] })])
    const { isEmpty } = usePipelineValidation(groups)
    expect(isEmpty.value).toBe(true)
  })

  it('isEmpty false when at least one step exists', () => {
    const groups = ref([makeGroup({ steps: [makeStep()] })])
    const { isEmpty } = usePipelineValidation(groups)
    expect(isEmpty.value).toBe(false)
  })

  it('isValid false when agent_run step has no agent', () => {
    const groups = ref([makeGroup({ steps: [makeStep({ action_type: 'agent_run', agent_id: undefined })] })])
    const { isValid } = usePipelineValidation(groups)
    expect(isValid.value).toBe(false)
  })

  it('isValid true when agent_run step has agent_id', () => {
    const groups = ref([makeGroup({ steps: [makeStep({ action_type: 'agent_run', agent_id: 'agent-1' })] })])
    const { isValid } = usePipelineValidation(groups)
    expect(isValid.value).toBe(true)
  })

  it('validationWarnings lists agent_run steps without agent', () => {
    const groups = ref([makeGroup({
      steps: [
        makeStep({ action_type: 'agent_run', agent_id: undefined }),
        makeStep({ action_type: 'agent_run', agent_id: undefined }),
      ]
    })])
    const { validationWarnings } = usePipelineValidation(groups)
    expect(validationWarnings.value).toHaveLength(1)
    expect(validationWarnings.value[0]).toContain('2')
  })

  it('works with a getter function backed by a ref', () => {
    const inner = ref<PipelineGroup[]>([])
    const getter = () => inner.value
    const { isEmpty } = usePipelineValidation(getter)
    expect(isEmpty.value).toBe(true)
    inner.value = [makeGroup({ steps: [makeStep()] })]
    expect(isEmpty.value).toBe(false)
  })
})
