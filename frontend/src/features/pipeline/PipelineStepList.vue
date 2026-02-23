<script setup lang="ts">
import PipelineGroupCard from './PipelineGroupCard.vue'
import type { PipelineGroup, PipelineStep } from '@/stores/pipelineConfig'

defineProps<{
  groups: PipelineGroup[]
  isAdmin: boolean
}>()

const emit = defineEmits<{
  'rename-group': [groupId: string, name: string]
  'remove-group': [groupId: string]
  'add-step': [groupId: string]
  'update-step': [groupId: string, stepId: string, step: PipelineStep]
  'remove-step': [groupId: string, stepId: string]
  'reorder-groups': [fromIndex: number, toIndex: number]
  'reorder-step': [groupId: string, fromIndex: number, toIndex: number]
}>()

function handleMoveGroupUp(index: number) {
  if (index > 0) {
    emit('reorder-groups', index, index - 1)
  }
}

function handleMoveGroupDown(index: number, groupCount: number) {
  if (index < groupCount - 1) {
    emit('reorder-groups', index, index + 1)
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <PipelineGroupCard
      v-for="(group, index) in groups"
      :key="group.id"
      :group="group"
      :index="index"
      :is-admin="isAdmin"
      :is-first="index === 0"
      :is-last="index === groups.length - 1"
      :group-count="groups.length"
      @rename="(gId: string, name: string) => emit('rename-group', gId, name)"
      @remove="emit('remove-group', $event)"
      @add-step="emit('add-step', $event)"
      @update-step="(gId: string, sId: string, step: PipelineStep) => emit('update-step', gId, sId, step)"
      @remove-step="(gId: string, sId: string) => emit('remove-step', gId, sId)"
      @move-up="handleMoveGroupUp(index)"
      @move-down="handleMoveGroupDown(index, groups.length)"
      @reorder-step="(gId: string, from: number, to: number) => emit('reorder-step', gId, from, to)"
    />
  </div>
</template>
