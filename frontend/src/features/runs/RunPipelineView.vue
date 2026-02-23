<script setup lang="ts">
import { computed, ref } from 'vue'
import Skeleton from 'primevue/skeleton'
import RunStageColumn from './RunStageColumn.vue'
import { groupStepsByStage } from '@/utils/pipelineStageUtils'
import type { RunWithSteps, RunStep } from './composables/useRunDetail'

interface SnapshotGroup {
  id: string
  name: string
  steps: unknown[]
}

const props = defineProps<{
  run: RunWithSteps | null
  steps: RunStep[]
}>()

const emit = defineEmits<{
  'step-selected': [step: RunStep]
}>()

const selectedStepId = ref<string | null>(null)

/** Derive stage definitions from the run's pipeline_config_snapshot. */
const stages = computed(() => {
  const snapshot = props.run?.pipeline_config_snapshot as
    | { groups?: SnapshotGroup[] }
    | undefined
  const groups = snapshot?.groups

  if (!groups || groups.length === 0) {
    return [{ id: 'default', name: 'Pipeline', steps: [] as unknown[] }]
  }

  return groups
})

/** Map live steps into their respective stages. */
const stepsByStage = computed(() => {
  const snapshot = props.run?.pipeline_config_snapshot as
    | { groups?: SnapshotGroup[] }
    | undefined
  const groups = snapshot?.groups
  return groupStepsByStage(groups, props.steps)
})

function handleStepSelected(step: RunStep) {
  selectedStepId.value = step.id
  emit('step-selected', step)
}
</script>

<template>
  <!-- Loading skeleton -->
  <div v-if="!run" class="flex flex-row gap-4 p-4" data-testid="pipeline-loading">
    <div v-for="n in 3" :key="n" class="flex flex-col gap-2 min-w-52">
      <Skeleton width="6rem" height="1.25rem" />
      <Skeleton width="100%" height="2.5rem" />
      <Skeleton width="100%" height="2.5rem" />
    </div>
  </div>

  <!-- Pipeline columns -->
  <div v-else class="flex flex-row overflow-x-auto gap-0 min-h-0 p-2" data-testid="pipeline-view">
    <RunStageColumn
      v-for="(stage, idx) in stages"
      :key="stage.id"
      :stage-name="stage.name"
      :steps="stepsByStage.get(stage.id) ?? []"
      :selected-step-id="selectedStepId"
      :is-last="idx === stages.length - 1"
      @step-selected="handleStepSelected"
    />
  </div>
</template>
