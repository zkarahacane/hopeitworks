/** Severity mapping for run status badges (PrimeVue Tag component). */
export const runStatusSeverity: Record<string, 'info' | 'success' | 'warn' | 'danger' | 'secondary'> = {
  pending: 'secondary',
  running: 'info',
  paused: 'warn',
  completed: 'success',
  failed: 'danger',
  cancelled: 'warn',
}
