import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E configuration for Zist web app.
 *
 * Architecture:
 *   - globalSetup starts a lightweight mock gateway on port 9999
 *   - webServer starts the SvelteKit dev server pointed at the mock gateway
 *   - Tests set a fake zist_session JWT cookie to simulate authenticated users
 *   - API calls from SSR and browser both route through Vite proxy → mock gateway
 */
export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.ts',

  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? 'github' : [['html', { open: 'never' }]],

  use: {
    baseURL: 'http://localhost:5174',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: {
    // Use port 5174 so tests never reuse a dev server pointed at the real gateway.
    command: 'GATEWAY_URL=http://localhost:9999 npm run dev -- --port 5174',
    url: 'http://localhost:5174',
    reuseExistingServer: false,
    timeout: 60_000,
  },
});
