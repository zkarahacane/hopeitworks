import { differenceInSeconds } from 'date-fns'

/** Formats the duration between two ISO timestamps as 'Xs' or 'Xm Ys'. */
export function formatStepDuration(startedAt: string, completedAt: string): string {
  const total = differenceInSeconds(new Date(completedAt), new Date(startedAt))
  if (total < 60) return `${total}s`
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}m ${s}s`
}
