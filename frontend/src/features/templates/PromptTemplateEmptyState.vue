<script setup lang="ts">
import Button from 'primevue/button'

defineProps<{
  isAdmin: boolean
}>()

const emit = defineEmits<{
  createClick: []
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-4 py-16">
    <i class="pi pi-file-edit text-4xl text-surface-400" />
    <h2 class="text-xl font-semibold">No prompt templates found for this project.</h2>
    <p class="text-surface-500">
      Prompt templates define how AI agents generate code for your pipeline steps.
    </p>
    <Button
      v-if="isAdmin"
      label="Create Template"
      icon="pi pi-plus"
      severity="success"
      @click="emit('createClick')"
    />
  </div>
</template>
