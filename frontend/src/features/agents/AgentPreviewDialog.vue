<script setup lang="ts">
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'

interface Props {
  visible: boolean
  renderedContent: string
  loading: boolean
  error: string | null
}

defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
}>()
</script>

<template>
  <Dialog
    :visible="visible"
    header="Agent Preview"
    modal
    :style="{ width: '50rem' }"
    :closable="true"
    @update:visible="emit('update:visible', $event)"
  >
    <!-- Loading state -->
    <div v-if="loading" class="flex flex-col gap-3">
      <Skeleton height="1rem" />
      <Skeleton height="1rem" />
      <Skeleton height="1rem" width="75%" />
    </div>

    <!-- Error state -->
    <Message v-else-if="error" severity="error" :closable="false">
      {{ error }}
    </Message>

    <!-- Rendered content -->
    <pre
      v-else
      class="max-h-[60vh] overflow-auto rounded-md p-4 text-sm whitespace-pre-wrap"
      :style="{ backgroundColor: 'var(--surface-base)', color: 'var(--p-text-color)' }"
    >{{ renderedContent }}</pre>

    <template #footer>
      <Button
        label="Close"
        severity="secondary"
        @click="emit('update:visible', false)"
      />
    </template>
  </Dialog>
</template>
