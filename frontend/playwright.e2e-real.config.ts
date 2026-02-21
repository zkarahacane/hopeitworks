import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e/real-tests',
  fullyParallel: false,
  forbidOnly: true,
  retries: 0,
  workers: 1,
  reporter: [
    ['html', { outputFolder: './e2e/real-results/html-report' }],
    ['json', { outputFile: './e2e/real-results/results.json' }],
    ['list'],
  ],
  outputDir: './e2e/real-results',
  timeout: 30000,
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on',
    screenshot: 'on',
    video: 'on',
    actionTimeout: 10000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
