/**
 * Formats the duration between two ISO 8601 timestamps as mm:ss.
 * Returns '--' if startedAt is not provided (pending step).
 * Uses Date.now() as the end time for running steps (no completedAt).
 */
export function formatDuration(
  startedAt?: string | null,
  completedAt?: string | null,
): string {
  if (!startedAt) return '--'
  const start = new Date(startedAt).getTime()
  const end = completedAt ? new Date(completedAt).getTime() : Date.now()
  const secs = Math.floor((end - start) / 1000)
  return formatDurationSeconds(secs)
}

/**
 * Formats an elapsed number of seconds as mm:ss (zero-padded).
 * Negative inputs are clamped to 0. Used by live tickers that already hold a
 * seconds count (e.g. runtimeStream elapsed getters).
 */
export function formatDurationSeconds(seconds: number): string {
  const secs = Math.max(0, Math.floor(seconds))
  const m = Math.floor(secs / 60)
    .toString()
    .padStart(2, '0')
  const s = (secs % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}
