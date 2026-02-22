/**
 * Formats a numeric USD value as a currency string.
 * Uses up to 5 decimal places to show small AI usage costs accurately.
 */
export function formatCostUSD(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 5,
  }).format(value)
}

/** Formats a token count with thousands separators (e.g., 100,000). */
export function formatTokenCount(count: number): string {
  return new Intl.NumberFormat('en-US').format(count)
}
