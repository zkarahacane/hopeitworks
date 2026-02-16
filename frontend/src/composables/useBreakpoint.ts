import { ref, onMounted, onUnmounted } from 'vue'

export function useBreakpoint() {
  const query = window.matchMedia('(max-width: 1023px)')
  const isMobile = ref(query.matches)

  const onChange = (event: MediaQueryListEvent) => {
    isMobile.value = event.matches
  }

  onMounted(() => {
    query.addEventListener('change', onChange)
  })

  onUnmounted(() => {
    query.removeEventListener('change', onChange)
  })

  return { isMobile }
}
