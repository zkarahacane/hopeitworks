<script setup lang="ts">
import { computed } from 'vue'
import Divider from 'primevue/divider'
import AgentChip from '@/ui/primitives/AgentChip.vue'
import type { Agent } from '@/stores/agents'

const props = defineProps<{
  agents: Agent[]
}>()

const emit = defineEmits<{
  'add-step': [actionType: string]
}>()

const stepTypes = [
  { type: 'git_branch', icon: 'pi pi-code-branch', label: 'git_branch', description: 'Create git branch' },
  { type: 'agent_run', icon: 'pi pi-microchip-ai', label: 'agent_run', description: 'Run an AI agent' },
  { type: 'human', icon: 'pi pi-user', label: 'human', description: 'Human gate', isGate: true },
  { type: 'git_pr', icon: 'pi pi-arrow-right-arrow-left', label: 'git_pr', description: 'Open pull request' },
  { type: 'ci_poll', icon: 'pi pi-sync', label: 'ci_poll', description: 'Wait for CI' },
  { type: 'notification', icon: 'pi pi-bell', label: 'notification', description: 'Send notification' },
  { type: 'hitl_gate', icon: 'pi pi-shield', label: 'hitl_gate', description: 'HITL gate' },
]

const hasAgents = computed(() => props.agents.length > 0)

function onDragStart(event: DragEvent, actionType: string) {
  event.dataTransfer?.setData('text/plain', actionType)
}
</script>

<template>
  <div
    class="flex flex-col gap-4 p-4 rounded-lg h-full"
    :style="{ backgroundColor: 'var(--surface-overlay)' }"
    data-testid="pipeline-step-palette"
  >
    <!-- Step types section -->
    <div class="flex flex-col gap-2">
      <span
        class="text-xs font-semibold tracking-widest uppercase"
        style="color: var(--p-text-muted-color)"
      >
        Step Types
      </span>
      <div class="grid grid-cols-2 gap-1.5">
        <div
          v-for="type in stepTypes"
          :key="type.type"
          draggable="true"
          class="flex flex-col gap-1 p-2 rounded-md cursor-grab select-none"
          :class="type.isGate ? 'amber-breathe' : ''"
          :style="type.isGate
            ? 'background-color: var(--status-gate-surface); border: 1px solid var(--status-gate-color)'
            : 'background-color: var(--surface-overlay); border: 1px solid var(--surface-border)'"
          :data-testid="`step-type-tile-${type.type}`"
          @dragstart="onDragStart($event, type.type)"
          @click="emit('add-step', type.type)"
        >
          <i :class="type.icon" class="text-sm" />
          <span class="font-mono text-xs">{{ type.label }}</span>
          <span class="text-xs opacity-60">{{ type.description }}</span>
        </div>
      </div>
    </div>

    <Divider />

    <!-- Agents section -->
    <div class="flex flex-col gap-2" data-testid="palette-agents-section">
      <span
        class="text-xs font-semibold tracking-widest uppercase"
        style="color: var(--p-text-muted-color)"
      >
        Agents
      </span>
      <div class="flex flex-col gap-1">
        <div
          v-for="agent in agents"
          :key="agent.id"
          class="flex items-center"
          data-testid="palette-agent-item"
        >
          <AgentChip
            :role="agent.name"
            :model="agent.model"
            :provider="agent.provider ?? null"
          />
        </div>
        <span
          v-if="!hasAgents"
          class="text-xs"
          style="color: var(--p-text-muted-color)"
        >
          No agents configured
        </span>
      </div>
    </div>
  </div>
</template>
