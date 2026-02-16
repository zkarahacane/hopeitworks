import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useProjectsStore = defineStore('projects', () => {
  const items = ref<Array<{ id: string; name: string }>>([])
  const current = ref<{ id: string; name: string } | null>(null)
  const isLoading = ref(false)
  return { items, current, isLoading }
})
