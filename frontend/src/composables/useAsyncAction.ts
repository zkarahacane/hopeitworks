import { ref } from 'vue'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function useAsyncAction<A extends any[], T>(fn: (...args: A) => Promise<T>) {
  const data = ref<T | null>(null) as { value: T | null }
  const error = ref<Error | null>(null)
  const isLoading = ref(false)

  async function execute(...args: A): Promise<T | null> {
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
