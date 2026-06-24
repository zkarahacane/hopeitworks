<script setup lang="ts">
import { ref, onMounted } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Message from 'primevue/message'
import ConfirmDialog from 'primevue/confirmdialog'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import { useAPIKeys } from '@/composables/useAPIKeys'
import APIKeyDialog from './APIKeyDialog.vue'

const confirm = useConfirm()
const toast = useToast()
const { keys, isLoading, error, fetchKeys, deleteKey } = useAPIKeys()

const dialogVisible = ref(false)

onMounted(() => {
  fetchKeys()
})

function providerSeverity(provider: string): 'info' | 'secondary' {
  return provider === 'claude' ? 'info' : 'secondary'
}

function handleDelete(keyId: string, keyName: string) {
  confirm.require({
    message: `Delete API key "${keyName}"? This cannot be undone.`,
    header: 'Delete API Key',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      // deleteKey coalesces a concurrent duplicate into 'busy' (anti double-fire),
      // so a re-entrant accept never fires a second DELETE nor a duplicate toast.
      const result = await deleteKey(keyId)
      if (result === 'deleted') {
        toast.add({
          severity: 'success',
          summary: 'API key deleted',
          detail: `"${keyName}" was revoked.`,
          life: 3000,
        })
      } else if (result === 'error') {
        toast.add({
          severity: 'error',
          summary: 'Delete failed',
          detail: error.value ?? 'Could not delete the API key.',
          life: 5000,
        })
      }
    },
  })
}
</script>

<template>
  <ConfirmDialog />

  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <span class="text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
        API keys are stored encrypted. Only the last 4 characters are shown.
      </span>
      <Button
        label="Add API Key"
        icon="pi pi-plus"
        size="small"
        @click="dialogVisible = true"
      />
    </div>

    <Message v-if="error" severity="error" :closable="false">{{ error }}</Message>

    <DataTable
      :value="keys"
      :loading="isLoading"
      size="small"
      :rows="20"
    >
      <template #empty>
        <div class="p-4 text-center" :style="{ color: 'var(--p-text-muted-color)' }">No API keys configured yet.</div>
      </template>

      <Column field="provider" header="Provider">
        <template #body="{ data }">
          <Tag :value="data.provider" :severity="providerSeverity(data.provider)" />
        </template>
      </Column>

      <Column field="key_name" header="Key Name" />

      <Column field="key_hint" header="Key Hint">
        <template #body="{ data }">
          <code class="text-xs">...{{ data.key_hint }}</code>
        </template>
      </Column>

      <Column field="created_at" header="Created">
        <template #body="{ data }">
          {{ new Date(data.created_at).toLocaleDateString() }}
        </template>
      </Column>

      <Column header="Actions">
        <template #body="{ data }">
          <Button
            icon="pi pi-trash"
            severity="danger"
            text
            size="small"
            aria-label="Delete API key"
            @click="handleDelete(data.id, data.key_name)"
          />
        </template>
      </Column>
    </DataTable>

    <APIKeyDialog
      v-model:visible="dialogVisible"
      @created="fetchKeys"
    />
  </div>
</template>
