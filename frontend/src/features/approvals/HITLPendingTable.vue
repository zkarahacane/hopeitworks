<script setup lang="ts">
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import type { HITLPendingItem } from '@/stores/hitl'
import { useRelativeTime } from '@/composables/useRelativeTime'
import { computed } from 'vue'

defineProps<{
  items: HITLPendingItem[]
  loading: boolean
}>()

const emit = defineEmits<{
  review: [item: HITLPendingItem]
}>()

/** Wrapper to get relative time for each row's pendingSince */
function RelativeTime(date: string) {
  return useRelativeTime(computed(() => date))
}
</script>

<template>
  <DataTable
    :value="items"
    :loading="loading"
    striped-rows
    responsive-layout="scroll"
    data-testid="hitl-pending-table"
  >
    <template #empty>
      <div class="flex items-center justify-center py-8" :style="{ color: 'var(--p-text-muted-color)' }">
        No pending approvals
      </div>
    </template>

    <Column header="Story" field="storyKey" style="min-width: 8rem">
      <template #body="{ data }">
        <Tag :value="(data as HITLPendingItem).storyKey" severity="info" />
      </template>
    </Column>

    <Column header="Title" field="storyTitle" style="min-width: 14rem">
      <template #body="{ data }">
        {{ (data as HITLPendingItem).storyTitle || '—' }}
      </template>
    </Column>

    <Column header="Project" field="projectName" style="min-width: 10rem">
      <template #body="{ data }">
        {{ (data as HITLPendingItem).projectName || '—' }}
      </template>
    </Column>

    <Column header="PR" style="min-width: 6rem">
      <template #body="{ data }">
        <a
          v-if="(data as HITLPendingItem).prUrl"
          :href="(data as HITLPendingItem).prUrl!"
          target="_blank"
          rel="noopener noreferrer"
          class="underline"
          :style="{ color: 'var(--p-primary-color)' }"
        >
          View PR
        </a>
        <span v-else :style="{ color: 'var(--p-text-muted-color)' }">—</span>
      </template>
    </Column>

    <Column header="Waiting Since" style="min-width: 8rem">
      <template #body="{ data }">
        {{ RelativeTime((data as HITLPendingItem).pendingSince).value || '—' }}
      </template>
    </Column>

    <Column header="Actions" style="min-width: 8rem">
      <template #body="{ data }">
        <Button
          label="Review"
          icon="pi pi-eye"
          size="small"
          severity="warn"
          @click="emit('review', data as HITLPendingItem)"
        />
      </template>
    </Column>
  </DataTable>
</template>
