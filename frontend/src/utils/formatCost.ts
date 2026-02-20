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
