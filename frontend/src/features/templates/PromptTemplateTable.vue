<script setup lang="ts">
import { ref, computed } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Select from 'primevue/select'
import Tag from 'primevue/tag'
import type { PromptTemplate, PromptTemplateType } from '@/stores/promptTemplates'
import { formatRelativeDate } from '@/utils/formatDate'

const props = defineProps<{
  templates: PromptTemplate[]
}>()

const emit = defineEmits<{
  rowClick: [templateId: string]
}>()

const selectedType = ref<PromptTemplateType | null>(null)

const typeFilterOptions = [
  { label: 'All', value: null },
  { label: 'Implement', value: 'implement' },
  { label: 'Retry', value: 'retry' },
  { label: 'Review', value: 'review' },
  { label: 'Merge', value: 'merge' },
  { label: 'Custom', value: 'custom' },
]

const typeSeverityMap: Record<PromptTemplateType, string> = {
  implement: 'info',
  retry: 'warn',
  review: 'secondary',
  merge: 'success',
  custom: 'contrast',
}

const filteredTemplates = computed(() => {
  if (!selectedType.value) return props.templates
  return props.templates.filter((t) => t.type === selectedType.value)
})

function handleRowClick(event: { data: PromptTemplate }) {
  emit('rowClick', event.data.id)
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center gap-2">
      <label for="type-filter" class="text-sm font-medium">Filter by type:</label>
      <Select
        id="type-filter"
        v-model="selectedType"
        :options="typeFilterOptions"
        option-label="label"
        option-value="value"
        placeholder="All"
        class="w-48"
      />
    </div>

    <DataTable
      :value="filteredTemplates"
      :paginator="filteredTemplates.length > 10"
      :rows="10"
      striped-rows
      row-hover
      class="cursor-pointer"
      data-testid="templates-table"
      @row-click="handleRowClick($event as unknown as { data: PromptTemplate })"
    >
      <Column field="name" header="Name" sortable>
        <template #body="{ data }">
          <span class="font-semibold">{{ (data as PromptTemplate).name }}</span>
        </template>
      </Column>
      <Column field="type" header="Type" sortable>
        <template #body="{ data }">
          <Tag
            :value="(data as PromptTemplate).type"
            :severity="typeSeverityMap[(data as PromptTemplate).type]"
          />
        </template>
      </Column>
      <Column field="updated_at" header="Last Updated" sortable>
        <template #body="{ data }">
          {{ formatRelativeDate((data as PromptTemplate).updated_at) }}
        </template>
      </Column>
    </DataTable>
  </div>
</template>
