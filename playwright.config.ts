import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './test',
  timeout: 30000,
  expect: {
    timeout: 5000,
  },
  fullyParallel: true,
  workers: 4,
  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report' }],
  ],
  webServer: {
    command: 'PORT=8092 EXPERIMENT_ROLLOUT=100 DATABASE_URL="file:/tmp/transport-playwright.db?mode=rwc&cache=shared&_foreign_keys=on&_journal_mode=WAL" go run ./cmd/server/',
    port: 8092,
    reuseExistingServer: true,
    timeout: 60000,
  },
  projects: [
    {
      name: 'Chromium',
      use: {
        ...devices['Desktop Chrome'],
        headless: true,
        viewport: { width: 1280, height: 720 },
        baseURL: 'http://localhost:8092',
        actionTimeout: 10000,
        navigationTimeout: 20000,
        trace: 'on-first-retry',
      },
    },
  ],
});
