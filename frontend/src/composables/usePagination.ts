import { ref, computed } from 'vue'

export function usePagination(options?: { perPage?: number }) {
  const page = ref(1)
  const perPage = ref(options?.perPage ?? 20)
  const total = ref(0)
  const totalPages = computed(() => Math.ceil(total.value / perPage.value))

  function setTotal(n: number) {
    total.value = n
  }
  function nextPage() {
    if (page.value < totalPages.value) page.value++
  }
  function prevPage() {
    if (page.value > 1) page.value--
  }
  function goToPage(n: number) {
    page.value = Math.max(1, Math.min(n, totalPages.value))
  }
  function reset() {
    page.value = 1
  }

  return { page, perPage, total, totalPages, setTotal, nextPage, prevPage, goToPage, reset }
}
