import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useRunsStore = defineStore('runs', () => {
  const items = ref<Array<{ id: string; status: string }>>([])
  const current = ref<{ id: string; status: string; steps: Array<unknown> } | null>(null)
  const isLoading = ref(false)
  return { items, current, isLoading }
})
