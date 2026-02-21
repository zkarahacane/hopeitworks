import { type Page } from '@playwright/test'

export interface LogEntry {
  type: 'console-error' | 'console-warning' | 'js-error' | 'network-error'
  message: string
  url?: string
  timestamp: number
}

export interface LogReport {
  errors: LogEntry[]
  warnings: LogEntry[]
  networkErrors: LogEntry[]
  summary: {
    totalErrors: number
    totalWarnings: number
    totalNetworkErrors: number
  }
}

export class LogCollector {
  private entries: LogEntry[] = []

  /**
   * Attach listeners to page for console errors, JS errors, and network errors.
   */
  attach(page: Page): void {
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        this.entries.push({
          type: 'console-error',
          message: msg.text(),
          url: page.url(),
          timestamp: Date.now(),
        })
      } else if (msg.type() === 'warning') {
        this.entries.push({
          type: 'console-warning',
          message: msg.text(),
          url: page.url(),
          timestamp: Date.now(),
        })
      }
    })

    page.on('pageerror', (error) => {
      this.entries.push({
        type: 'js-error',
        message: error.message,
        url: page.url(),
        timestamp: Date.now(),
      })
    })

    page.on('response', (response) => {
      if (response.status() >= 400) {
        this.entries.push({
          type: 'network-error',
          message: `${response.status()} ${response.statusText()} - ${response.url()}`,
          url: page.url(),
          timestamp: Date.now(),
        })
      }
    })
  }

  getReport(): LogReport {
    const errors = this.entries.filter((e) => e.type === 'console-error' || e.type === 'js-error')
    const warnings = this.entries.filter((e) => e.type === 'console-warning')
    const networkErrors = this.entries.filter((e) => e.type === 'network-error')

    return {
      errors,
      warnings,
      networkErrors,
      summary: {
        totalErrors: errors.length,
        totalWarnings: warnings.length,
        totalNetworkErrors: networkErrors.length,
      },
    }
  }

  clear(): void {
    this.entries = []
  }
}
