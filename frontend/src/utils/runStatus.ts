type TagSeverity = 'info' | 'success' | 'warn' | 'danger' | 'secondary'

/** Severity mapping for run/step status badges (PrimeVue Tag component). */
export const runStatusSeverity: Record<string, TagSeverity> = {
  pending: 'secondary',
  running: 'info',
  paused: 'warn',
  completed: 'success',
  failed: 'danger',
  cancelled: 'warn',
  waiting_approval: 'warn',
}

/** Returns the PrimeVue Tag severity for a given run or step status string. */
export function statusSeverity(status: string): TagSeverity {
  return runStatusSeverity[status] ?? 'secondary'
}
