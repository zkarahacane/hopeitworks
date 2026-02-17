import { computed, ref, type MaybeRef, unref } from 'vue'
import { useIntervalFn } from '@vueuse/core'

/**
 * Composable that returns a reactive relative time string for a given date.
 * Updates automatically every minute.
 */
export function useRelativeTime(date: MaybeRef<string | Date | null>) {
  const now = ref(Date.now())

  useIntervalFn(() => {
    now.value = Date.now()
  }, 60_000)

  const relativeTime = computed(() => {
    const dateValue = unref(date)
    if (!dateValue) return null

    const d = typeof dateValue === 'string' ? new Date(dateValue) : dateValue
    if (isNaN(d.getTime())) return null

    const diff = now.value - d.getTime()
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)
    const weeks = Math.floor(days / 7)

    if (seconds < 60) return 'just now'
    if (minutes < 60) return `${minutes}m ago`
    if (hours < 24) return `${hours}h ago`
    if (days < 7) return `${days}d ago`
    return `${weeks}w ago`
  })

  return relativeTime
}
