<script setup lang="ts">
import { ref, watch } from 'vue'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import type { StoryFilters } from '@/stores/stories'

const props = defineProps<{
  modelValue: StoryFilters
}>()

const emit = defineEmits<{
  'update:modelValue': [filters: StoryFilters]
}>()

const statusOptions = [
  { label: 'All statuses', value: 'all' },
  { label: 'Backlog', value: 'backlog' },
  { label: 'Running', value: 'running' },
  { label: 'Done', value: 'done' },
  { label: 'Failed', value: 'failed' },
]

const localSearch = ref(props.modelValue.search)
let debounceTimer: ReturnType<typeof setTimeout> | null = null

watch(localSearch, (newValue) => {
  if (debounceTimer) {
    clearTimeout(debounceTimer)
  }
  debounceTimer = setTimeout(() => {
    emit('update:modelValue', { ...props.modelValue, search: newValue })
  }, 200)
})

watch(
  () => props.modelValue.search,
  (newValue) => {
    if (newValue !== localSearch.value) {
      localSearch.value = newValue
    }
  },
)

function handleStatusChange(value: string) {
  emit('update:modelValue', { ...props.modelValue, status: value })
}
</script>

<template>
  <div class="flex flex-col gap-2">
    <Select
      :model-value="modelValue.status ?? 'all'"
      :options="statusOptions"
      option-label="label"
      option-value="value"
      placeholder="Filter by status"
      class="w-full"
      @update:model-value="handleStatusChange"
    />
    <div class="relative">
      <i
        class="pi pi-search absolute"
        style="left: 0.75rem; top: 50%; transform: translateY(-50%); color: var(--p-text-muted-color)"
      />
      <InputText
        v-model="localSearch"
        placeholder="Search stories..."
        class="w-full"
        style="padding-left: 2.25rem"
      />
    </div>
  </div>
</template>
