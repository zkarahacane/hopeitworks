<script setup lang="ts">
import DataTable, { type DataTablePageEvent } from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import type { User } from '@/stores/auth'
import type { Pagination } from '@/stores/users'
import { formatDate } from '@/utils/formatDate'

defineProps<{
  users: User[]
  loading: boolean
  pagination: Pagination
}>()

const emit = defineEmits<{
  edit: [user: User]
  delete: [user: User]
  'page-change': [page: number]
}>()

function onPage(event: DataTablePageEvent) {
  emit('page-change', event.page + 1)
}
</script>

<template>
  <DataTable
    :value="users"
    :lazy="true"
    :paginator="true"
    :rows="pagination.per_page"
    :total-records="pagination.total"
    :loading="loading"
    :first="(pagination.page - 1) * pagination.per_page"
    striped-rows
    @page="onPage"
  >
    <Column field="email" header="Email" />
    <Column field="name" header="Name" />
    <Column field="role" header="Role">
      <template #body="{ data }">
        <Tag
          :value="(data as User).role"
          :severity="(data as User).role === 'admin' ? 'danger' : 'info'"
        />
      </template>
    </Column>
    <Column field="created_at" header="Created">
      <template #body="{ data }">
        {{ formatDate((data as User).created_at ?? '') }}
      </template>
    </Column>
    <Column header="Actions" :exportable="false" style="width: 8rem">
      <template #body="{ data }">
        <div class="flex gap-2">
          <Button
            icon="pi pi-pencil"
            severity="info"
            text
            rounded
            aria-label="Edit user"
            @click="emit('edit', data as User)"
          />
          <Button
            icon="pi pi-trash"
            severity="danger"
            text
            rounded
            aria-label="Delete user"
            @click="emit('delete', data as User)"
          />
        </div>
      </template>
    </Column>
  </DataTable>
</template>
