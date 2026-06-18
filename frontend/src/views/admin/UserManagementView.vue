<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import ConfirmDialog from 'primevue/confirmdialog'
import Toast from 'primevue/toast'
import Button from 'primevue/button'
import UserTable from '@/features/admin/UserTable.vue'
import CreateUserDialog from '@/features/admin/CreateUserDialog.vue'
import EditUserDialog from '@/features/admin/EditUserDialog.vue'
import { useUsers } from '@/composables/useUsers'
import type { User } from '@/stores/auth'

const { users, pagination, isLoading, fetchUsers, deleteUser } = useUsers()

const confirm = useConfirm()
const toast = useToast()

const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const selectedUser = ref<User | null>(null)

onMounted(() => {
  fetchUsers.execute()
})

function onEdit(user: User) {
  selectedUser.value = user
  showEditDialog.value = true
}

function onDelete(user: User) {
  confirm.require({
    message: `Are you sure you want to delete ${user.email}?`,
    header: 'Delete User',
    icon: 'pi pi-exclamation-triangle',
    rejectLabel: 'Cancel',
    acceptLabel: 'Delete',
    acceptClass: 'p-button-danger',
    accept: async () => {
      await deleteUser.execute(user.id)
      if (!deleteUser.error.value) {
        toast.add({ severity: 'success', summary: 'Deleted', detail: `${user.email} has been deleted`, life: 3000 })
      } else {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to delete user', life: 3000 })
      }
    },
  })
}

function onPageChange(page: number) {
  fetchUsers.execute({ page })
}

function onUserCreated() {
  toast.add({ severity: 'success', summary: 'Created', detail: 'User created successfully', life: 3000 })
}

function onUserUpdated() {
  toast.add({ severity: 'success', summary: 'Updated', detail: 'User updated successfully', life: 3000 })
}
</script>

<template>
  <div class="flex flex-col gap-4 p-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold" style="font-family: var(--font-sans)">User management</h1>
        <p class="mt-1 text-sm" style="color: var(--p-text-muted-color)">Manage workspace members, roles, and access.</p>
      </div>
      <Button label="Create User" icon="pi pi-plus" @click="showCreateDialog = true" />
    </div>

    <UserTable
      :users="users"
      :loading="isLoading"
      :pagination="pagination"
      @edit="onEdit"
      @delete="onDelete"
      @page-change="onPageChange"
    />

    <CreateUserDialog v-model:visible="showCreateDialog" @created="onUserCreated" />

    <EditUserDialog v-model:visible="showEditDialog" :user="selectedUser" @updated="onUserUpdated" />

    <ConfirmDialog />
    <Toast />
  </div>
</template>
