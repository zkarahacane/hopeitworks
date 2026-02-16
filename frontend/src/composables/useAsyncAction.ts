import { ref } from 'vue'

export function useAsyncAction<T>(fn: (...args: unknown[]) => Promise<T>) {
  const data = ref<T | null>(null) as { value: T | null }
  const error = ref<Error | null>(null)
  const isLoading = ref(false)

  async function execute(...args: unknown[]): Promise<T | null> {
    isLoading.value = true
    error.value = null
    try {
      data.value = await fn(...args)
      return data.value
    } catch (e) {
      error.value = e instanceof Error ? e : new Error(String(e))
      return null
    } finally {
      isLoading.value = false
    }
  }

  return { data, error, isLoading, execute }
}
