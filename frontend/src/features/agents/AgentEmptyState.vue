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
    <i class="pi pi-microchip text-4xl" :style="{ color: 'var(--p-text-muted-color)' }" />
    <h2 class="text-xl font-semibold">No agents found for this project.</h2>
    <p :style="{ color: 'var(--p-text-muted-color)' }">
      Agents define how AI models generate code for your pipeline steps.
    </p>
    <Button
      v-if="isAdmin"
      label="New Agent"
      icon="pi pi-plus"
      severity="success"
      @click="emit('createClick')"
    />
  </div>
</template>
