import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useLayoutStore = defineStore('layout', () => {
  const sidebarCollapsed = ref(
    typeof window !== 'undefined' &&
      localStorage.getItem('layout-sidebar-collapsed') === 'true',
  )

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
    if (typeof window !== 'undefined') {
      localStorage.setItem(
        'layout-sidebar-collapsed',
        String(sidebarCollapsed.value),
      )
    }
  }

  return { sidebarCollapsed, toggleSidebar }
})
