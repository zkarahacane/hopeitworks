<script setup lang="ts">
import DataTable, { type DataTablePageEvent } from 'primevue/datatable'
import Column from 'primevue/column'
import type { Project } from '@/stores/projects'
import { formatRelativeDate } from '@/utils/formatDate'

defineProps<{
  projects: Project[]
  totalRecords: number
  rows: number
  loading: boolean
  first: number
}>()

const emit = defineEmits<{
  page: [event: DataTablePageEvent]
  'row-click': [project: Project]
}>()
</script>

<template>
  <DataTable
    :value="projects"
    :lazy="true"
    :paginator="true"
    :rows="rows"
    :total-records="totalRecords"
    :loading="loading"
    :first="first"
    striped-rows
    row-hover
    class="cursor-pointer"
    @page="emit('page', $event)"
    @row-click="emit('row-click', $event.data as Project)"
  >
    <Column field="name" header="Name">
      <template #body="{ data }">
        <span class="font-semibold">{{ (data as Project).name }}</span>
      </template>
    </Column>
    <Column field="description" header="Description">
      <template #body="{ data }">
        <span class="block max-w-md truncate">{{ (data as Project).description || '-' }}</span>
      </template>
    </Column>
    <Column field="created_at" header="Created">
      <template #body="{ data }">
        {{ formatRelativeDate((data as Project).created_at) }}
      </template>
    </Column>
  </DataTable>
</template>
