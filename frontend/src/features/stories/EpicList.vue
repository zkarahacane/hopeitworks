<script setup lang="ts">
import type { Epic } from '@/stores/epics'
import EpicCard from './EpicCard.vue'

defineProps<{
  epics: Epic[]
}>()

const emit = defineEmits<{
  'epic-click': [epic: Epic]
}>()
</script>

<template>
  <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
    <EpicCard
      v-for="epic in epics"
      :key="epic.id"
      :epic="epic"
      @click="emit('epic-click', $event)"
    />
  </div>
</template>
