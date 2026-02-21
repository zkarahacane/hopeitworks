<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import HITLPendingTable from '@/features/approvals/HITLPendingTable.vue'
import { useHITLStore, type HITLPendingItem } from '@/stores/hitl'

const hitlStore = useHITLStore()
const router = useRouter()

onMounted(() => {
  hitlStore.fetchPending()
})

function handleReview(item: HITLPendingItem) {
  router.push({
    name: 'hitl-approve',
    params: {
      id: item.projectId,
      runId: item.runId,
      stepId: item.stepId,
    },
  })
}
</script>

<template>
  <div class="p-6">
    <h1 class="mb-4 text-2xl font-bold">Approvals</h1>

    <!-- Loading skeleton -->
    <div v-if="hitlStore.isLoading && hitlStore.pendingItems.length === 0" class="flex flex-col gap-3">
      <Skeleton height="2.5rem" />
      <Skeleton height="2.5rem" />
      <Skeleton height="2.5rem" />
    </div>

    <!-- Error state -->
    <Message
      v-else-if="hitlStore.error"
      severity="error"
      :closable="false"
    >
      <div class="flex items-center gap-2">
        <span>{{ hitlStore.error }}</span>
        <Button
          label="Retry"
          icon="pi pi-refresh"
          size="small"
          severity="danger"
          outlined
          @click="hitlStore.fetchPending()"
        />
      </div>
    </Message>

    <!-- Data table -->
    <HITLPendingTable
      v-else
      :items="hitlStore.pendingItems"
      :loading="hitlStore.isLoading"
      @review="handleReview"
    />
  </div>
</template>
