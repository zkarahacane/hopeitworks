<script setup lang="ts">
import Button from 'primevue/button'
import Message from 'primevue/message'

defineProps<{
  isAdmin: boolean
}>()

const emit = defineEmits<{
  createEpic: []
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-4 py-16">
    <Message severity="info" :closable="false">
      No epics found for this project.
    </Message>
    <Button
      v-if="isAdmin"
      label="Create Epic"
      icon="pi pi-plus"
      severity="success"
      @click="emit('createEpic')"
    />
    <p v-else style="color: var(--p-text-muted-color)">
      Contact an administrator to create epics for this project.
    </p>
  </div>
</template>
