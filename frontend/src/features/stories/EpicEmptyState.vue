<script setup lang="ts">
import Button from 'primevue/button'

const emit = defineEmits<{
  import: []
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-4 py-16">
    <i class="pi pi-th-large text-4xl text-surface-400" />
    <h2 class="text-xl font-semibold">No epics yet</h2>
    <p class="text-surface-500">Import stories from your repository to get started</p>
    <Button
      label="Import Stories"
      icon="pi pi-upload"
      severity="success"
      @click="emit('import')"
    />
  </div>
</template>
