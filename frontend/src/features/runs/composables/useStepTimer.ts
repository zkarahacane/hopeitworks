import { ref, computed, onBeforeUnmount } from 'vue'

/**
 * Composable that provides a live elapsed timer for a running step.
 * Updates every second and formats as 'Xs elapsed' or 'Xm Ys elapsed'.
 * Clears the interval on component unmount.
 */
export function useStepTimer(startedAt: string | undefined) {
  const now = ref(Date.now())

  if (!startedAt) {
    return { elapsed: computed(() => '') }
  }

  const intervalId = setInterval(() => {
    now.value = Date.now()
  }, 1000)

  onBeforeUnmount(() => clearInterval(intervalId))

  const elapsed = computed(() => {
    const startMs = new Date(startedAt).getTime()
    const totalSeconds = Math.floor((now.value - startMs) / 1000)
    if (totalSeconds < 60) return `${totalSeconds}s elapsed`
    const m = Math.floor(totalSeconds / 60)
    const s = totalSeconds % 60
    return `${m}m ${s}s elapsed`
  })

  return { elapsed }
}
