import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useLayoutStore = defineStore('layout', () => {
  const sidebarCollapsed = ref(
    localStorage.getItem('layout-sidebar-collapsed') === 'true',
  )

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
    localStorage.setItem(
      'layout-sidebar-collapsed',
      String(sidebarCollapsed.value),
    )
  }

  return { sidebarCollapsed, toggleSidebar }
})
