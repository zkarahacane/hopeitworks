<script setup lang="ts">
import { ref, computed } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Select from 'primevue/select'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import type { Agent, AgentScope } from '@/stores/agents'
import AgentChip from '@/ui/primitives/AgentChip.vue'

const props = withDefaults(
  defineProps<{
    agents: Agent[]
    isAdmin: boolean
    /** Returns true while the DELETE for an agent id is in flight (#295). */
    isDeleting?: (agentId: string) => boolean
  }>(),
  {
    isDeleting: () => false,
  },
)

const emit = defineEmits<{
  rowClick: [agentId: string]
  delete: [agentId: string]
}>()

const selectedScope = ref<AgentScope | null>(null)

const scopeFilterOptions = [
  { label: 'All', value: null },
  { label: 'Project', value: 'project' },
  { label: 'Global', value: 'global' },
]

const filteredAgents = computed(() => {
  if (!selectedScope.value) return props.agents
  return props.agents.filter((a) => a.scope === selectedScope.value)
})

/** Whether the edit action should be enabled for this agent */
function canEdit(agent: Agent): boolean {
  return agent.scope !== 'global' || props.isAdmin
}

function handleRowClick(event: { data: Agent }) {
  emit('rowClick', event.data.id)
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center gap-2">
      <label for="scope-filter" class="text-sm font-medium">Filter by scope:</label>
      <Select
        id="scope-filter"
        v-model="selectedScope"
        :options="scopeFilterOptions"
        option-label="label"
        option-value="value"
        placeholder="All"
        class="w-48"
      />
    </div>

    <DataTable
      :value="filteredAgents"
      :paginator="filteredAgents.length > 10"
      :rows="10"
      striped-rows
      row-hover
      class="cursor-pointer"
      data-testid="agents-table"
      @row-click="handleRowClick($event as unknown as { data: Agent })"
    >
      <Column field="name" header="Agent" sortable>
        <template #body="{ data }">
          <AgentChip
            :role="(data as Agent).name"
            :model="(data as Agent).model"
            :provider="(data as Agent).provider ?? undefined"
          />
        </template>
      </Column>
      <Column field="scope" header="Scope" sortable>
        <template #body="{ data }">
          <Tag
            :value="(data as Agent).scope"
            :severity="(data as Agent).scope === 'global' ? 'info' : 'secondary'"
            data-testid="scope-badge"
          />
        </template>
      </Column>
      <Column field="image" header="Image" sortable>
        <template #body="{ data }">
          <code class="text-sm">{{ (data as Agent).image }}</code>
        </template>
      </Column>
      <Column header="Actions" :style="{ width: '8rem' }">
        <template #body="{ data }">
          <div class="flex items-center gap-1">
            <Button
              icon="pi pi-pencil"
              text
              rounded
              size="small"
              severity="secondary"
              :disabled="!canEdit(data as Agent)"
              :title="canEdit(data as Agent) ? 'Edit agent' : 'Global agents can only be edited by administrators'"
              data-testid="edit-agent-button"
              @click.stop="emit('rowClick', (data as Agent).id)"
            />
            <Button
              v-if="canEdit(data as Agent)"
              icon="pi pi-trash"
              text
              rounded
              size="small"
              severity="danger"
              title="Delete agent"
              data-testid="delete-agent-button"
              :loading="props.isDeleting((data as Agent).id)"
              :disabled="props.isDeleting((data as Agent).id)"
              @click.stop="emit('delete', (data as Agent).id)"
            />
          </div>
        </template>
      </Column>
    </DataTable>
  </div>
</template>
