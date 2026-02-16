import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useStoriesStore = defineStore('stories', () => {
  const items = ref<Array<{ id: string; summary: string; status: string }>>([])
  const isLoading = ref(false)
  return { items, isLoading }
})
