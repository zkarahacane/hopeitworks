<script setup lang="ts">
import { ref, watch, nextTick, type ComponentPublicInstance } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import { useConfirm } from 'primevue/useconfirm'
import PipelineStepCard from './PipelineStepCard.vue'
import type { PipelineGroup, PipelineStep } from '@/stores/pipelineConfig'
import type { Agent } from '@/stores/agents'

const props = defineProps<{
  group: PipelineGroup
  index: number
  isAdmin: boolean
  isFirst: boolean
  isLast: boolean
  groupCount: number
  agents: Agent[]
}>()

const emit = defineEmits<{
  rename: [groupId: string, name: string]
  remove: [groupId: string]
  'add-step': [groupId: string]
  'update-step': [groupId: string, stepId: string, step: PipelineStep]
  'remove-step': [groupId: string, stepId: string]
  'move-up': [index: number]
  'move-down': [index: number]
  'reorder-step': [groupId: string, fromIndex: number, toIndex: number]
}>()

const confirm = useConfirm()
const collapsed = ref(false)
const isEditing = ref(false)
const localName = ref(props.group.name)
const nameInputRef = ref<ComponentPublicInstance | null>(null)

watch(
  () => props.group.name,
  (name) => {
    localName.value = name
  },
)

function toggleCollapse() {
  collapsed.value = !collapsed.value
}

function startEditing() {
  if (!props.isAdmin) return
  isEditing.value = true
  nextTick(() => {
    const el = nameInputRef.value?.$el as HTMLInputElement | undefined
    el?.focus()
  })
}

function commitRename() {
  isEditing.value = false
  const trimmed = localName.value.trim()
  if (trimmed && trimmed !== props.group.name) {
    emit('rename', props.group.id, trimmed)
  } else {
    localName.value = props.group.name
  }
}

function confirmRemove() {
  confirm.require({
    message: `Remove group "${props.group.name}" and all its steps?`,
    header: 'Confirm Removal',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: () => {
      emit('remove', props.group.id)
    },
  })
}

function handleMoveStepUp(stepIndex: number) {
  if (stepIndex > 0) {
    emit('reorder-step', props.group.id, stepIndex, stepIndex - 1)
  }
}

function handleMoveStepDown(stepIndex: number) {
  if (stepIndex < props.group.steps.length - 1) {
    emit('reorder-step', props.group.id, stepIndex, stepIndex + 1)
  }
}

const expandedStepIndex = ref<number | null>(null)

function toggleStepExpand(index: number) {
  expandedStepIndex.value = expandedStepIndex.value === index ? null : index
}
</script>

<template>
  <div class="flex flex-col gap-1" data-testid="pipeline-group-card">
    <!-- Group header -->
    <div
      class="flex items-center gap-2 px-3 py-2 rounded-lg"
      :style="{ backgroundColor: 'var(--surface-overlay)' }"
    >
      <span class="cursor-grab opacity-40 text-lg select-none" data-testid="group-drag-handle">⠿</span>

      <!-- Collapse toggle -->
      <Button
        :icon="collapsed ? 'pi pi-chevron-right' : 'pi pi-chevron-down'"
        text
        rounded
        size="small"
        aria-label="Toggle collapse"
        data-testid="collapse-toggle"
        @click="toggleCollapse"
      />

      <!-- Editable group name -->
      <InputText
        v-if="isEditing"
        ref="nameInputRef"
        v-model="localName"
        size="small"
        class="flex-1"
        data-testid="group-name-input"
        @blur="commitRename"
        @keydown.enter="commitRename"
      />
      <span
        v-else
        class="flex-1 cursor-pointer font-semibold"
        data-testid="group-name"
        @click="startEditing"
      >
        {{ group.name }}
      </span>

      <!-- Step count badge -->
      <span class="text-sm opacity-60" data-testid="step-count">
        {{ group.steps.length }} {{ group.steps.length === 1 ? 'step' : 'steps' }}
      </span>

      <!-- Admin controls -->
      <template v-if="isAdmin">
        <Button
          icon="pi pi-arrow-up"
          text
          rounded
          size="small"
          :disabled="isFirst"
          aria-label="Move group up"
          data-testid="move-group-up"
          @click="emit('move-up', index)"
        />
        <Button
          icon="pi pi-arrow-down"
          text
          rounded
          size="small"
          :disabled="isLast"
          aria-label="Move group down"
          data-testid="move-group-down"
          @click="emit('move-down', index)"
        />
        <Button
          icon="pi pi-trash"
          text
          rounded
          size="small"
          severity="danger"
          aria-label="Remove group"
          data-testid="remove-group"
          @click="confirmRemove"
        />
      </template>
    </div>

    <!-- Steps area -->
    <div v-show="!collapsed" class="flex flex-col gap-0.5 pl-4 mt-1" data-testid="group-steps">
      <PipelineStepCard
        v-for="(step, stepIndex) in group.steps"
        :key="step.id"
        :step="step"
        :index="stepIndex"
        :is-admin="isAdmin"
        :expanded="expandedStepIndex === stepIndex"
        :is-first="stepIndex === 0"
        :is-last="stepIndex === group.steps.length - 1"
        :agents="agents"
        @toggle="toggleStepExpand(stepIndex)"
        @update="(updatedStep: PipelineStep) => emit('update-step', group.id, step.id, updatedStep)"
        @remove="emit('remove-step', group.id, step.id)"
        @move-up="handleMoveStepUp(stepIndex)"
        @move-down="handleMoveStepDown(stepIndex)"
      />

      <!-- Add step button -->
      <Button
        v-if="isAdmin"
        label="+ Add step"
        text
        size="small"
        data-testid="add-step-to-group"
        @click="emit('add-step', group.id)"
      />
    </div>
  </div>
</template>
