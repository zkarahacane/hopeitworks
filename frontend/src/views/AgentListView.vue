<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import ConfirmDialog from 'primevue/confirmdialog'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import { useAgents } from '@/composables/useAgents'
import { useAuth } from '@/composables/useAuth'
import { useAgentsStore } from '@/stores/agents'
import AgentTable from '@/features/agents/AgentTable.vue'
import AgentEmptyState from '@/features/agents/AgentEmptyState.vue'

const route = useRoute()
const router = useRouter()
const confirm = useConfirm()
const toast = useToast()
const projectId = route.params.id as string

const { user } = useAuth()
const isAdmin = computed(() => user.value?.role === 'admin')
const agentsStore = useAgentsStore()

const { agents, isLoading, error, fetchAgents, retry } = useAgents(projectId)

onMounted(() => {
  fetchAgents()
})

function handleRowClick(agentId: string) {
  router.push({ name: 'agent-editor', params: { id: projectId, agentId } })
}

function handleCreateClick() {
  router.push({ name: 'agent-create', params: { id: projectId } })
}

function handleDelete(agentId: string) {
  confirm.require({
    message: 'Are you sure you want to delete this agent?',
    header: 'Confirm Delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      const success = await agentsStore.deleteAgent(projectId, agentId)
      if (success) {
        toast.add({
          severity: 'success',
          summary: 'Deleted',
          detail: 'Agent deleted successfully',
          life: 3000,
        })
      } else {
        toast.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to delete agent',
          life: 5000,
        })
      }
    },
  })
}
</script>

<template>
  <ConfirmDialog />
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Agents</h1>
      <Button
        v-if="isAdmin && agents.length > 0"
        label="New Agent"
        icon="pi pi-plus"
        severity="success"
        @click="handleCreateClick"
      />
    </div>

    <!-- Loading state -->
    <div v-if="isLoading && agents.length === 0" class="flex flex-col gap-4">
      <Skeleton height="2rem" />
      <Skeleton height="2rem" />
      <Skeleton height="2rem" />
      <Skeleton height="2rem" />
    </div>

    <!-- Error state -->
    <Message v-else-if="error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
      </div>
    </Message>

    <!-- Empty state -->
    <AgentEmptyState
      v-else-if="!isLoading && !error && agents.length === 0"
      :is-admin="isAdmin"
      @create-click="handleCreateClick"
    />

    <!-- Data state -->
    <AgentTable
      v-else
      :agents="agents"
      :is-admin="isAdmin"
      @row-click="handleRowClick"
      @delete="handleDelete"
    />
  </div>
</template>
