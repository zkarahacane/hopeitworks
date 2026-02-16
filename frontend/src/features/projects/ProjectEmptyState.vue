<script setup lang="ts">
import Button from 'primevue/button'

const emit = defineEmits<{
  create: []
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-4 py-16">
    <i class="pi pi-folder-open text-4xl text-surface-400" />
    <h2 class="text-xl font-semibold">No projects yet</h2>
    <p class="text-surface-500">Get started by creating your first project</p>
    <Button
      label="Create your first project"
      icon="pi pi-plus"
      severity="success"
      @click="emit('create')"
    />
  </div>
</template>
