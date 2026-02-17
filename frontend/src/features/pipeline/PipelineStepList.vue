<script setup lang="ts">
import { ref } from 'vue'
import PipelineStepCard from './PipelineStepCard.vue'
import type { PipelineStep } from '@/stores/pipelineConfig'

defineProps<{
  steps: PipelineStep[]
  isAdmin: boolean
}>()

const emit = defineEmits<{
  update: [index: number, step: PipelineStep]
  remove: [index: number]
  reorder: [fromIndex: number, toIndex: number]
}>()

const expandedIndex = ref<number | null>(null)

function toggleExpand(index: number) {
  expandedIndex.value = expandedIndex.value === index ? null : index
}

function handleMoveUp(index: number) {
  if (index > 0) {
    emit('reorder', index, index - 1)
    expandedIndex.value = index - 1
  }
}

function handleMoveDown(index: number, stepsLength: number) {
  if (index < stepsLength - 1) {
    emit('reorder', index, index + 1)
    expandedIndex.value = index + 1
  }
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <PipelineStepCard
      v-for="(step, index) in steps"
      :key="step.id"
      :step="step"
      :index="index"
      :is-admin="isAdmin"
      :expanded="expandedIndex === index"
      :is-first="index === 0"
      :is-last="index === steps.length - 1"
      @toggle="toggleExpand(index)"
      @update="(updatedStep: PipelineStep) => emit('update', index, updatedStep)"
      @remove="emit('remove', index)"
      @move-up="handleMoveUp(index)"
      @move-down="handleMoveDown(index, steps.length)"
    />
  </div>
</template>
