/**
 * Extracts a human-readable error message from an OpenAPI-fetch error response.
 * Falls back to the provided default message if extraction fails.
 */
export function getApiErrorMessage(error: unknown, fallback: string): string {
  return (error as { error?: { message?: string } })?.error?.message ?? fallback
}
