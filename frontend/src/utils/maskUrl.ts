/**
 * Masks a URL by showing only the last 6 characters, prefixed with asterisks.
 * Used to display partially hidden webhook URLs in the UI.
 */
export function maskUrl(url: string): string {
  if (url.length <= 6) return url
  return `****${url.slice(-6)}`
}
