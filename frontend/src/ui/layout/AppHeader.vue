<script setup lang="ts">
import Button from 'primevue/button'

defineProps<{
  showHamburger: boolean
}>()

const emit = defineEmits<{
  'toggle-sidebar': []
}>()
</script>

<template>
  <header
    class="flex h-12 items-center justify-between border-b border-surface-200 bg-surface-0 px-4"
  >
    <div class="flex items-center gap-2">
      <Button
        v-if="showHamburger"
        icon="pi pi-bars"
        text
        rounded
        aria-label="Toggle sidebar"
        @click="emit('toggle-sidebar')"
      />
      <span class="text-lg font-semibold text-surface-700">Hope</span>
    </div>
    <div class="flex items-center gap-2">
      <Button icon="pi pi-user" text rounded aria-label="User menu" />
    </div>
  </header>
</template>
