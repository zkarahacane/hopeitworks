<script setup lang="ts">
import EpicCard from './EpicCard.vue'
import type { Epic } from '@/stores/epics'

defineProps<{
  epics: Epic[]
}>()

const emit = defineEmits<{
  epicClick: [epicId: string]
}>()
</script>

<template>
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    <EpicCard
      v-for="epic in epics"
      :key="epic.id"
      :epic="epic"
      @click="emit('epicClick', $event)"
    />
  </div>
</template>
