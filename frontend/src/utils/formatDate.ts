import { formatDistanceToNow, format, parseISO } from 'date-fns'

/**
 * Format an ISO 8601 date string as a relative time (e.g., "3 days ago").
 * Returns "Invalid date" for unparseable inputs.
 */
export function formatRelativeDate(dateStr: string): string {
  try {
    return formatDistanceToNow(parseISO(dateStr), { addSuffix: true })
  } catch {
    return 'Invalid date'
  }
}

/**
 * Format an ISO 8601 date string as a human-readable date (e.g., "Feb 15, 2026").
 * Returns "Invalid date" for unparseable inputs.
 */
export function formatDate(dateStr: string): string {
  try {
    return format(parseISO(dateStr), 'MMM d, yyyy')
  } catch {
    return 'Invalid date'
  }
}
