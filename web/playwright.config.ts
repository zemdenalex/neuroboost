import { defineConfig, devices } from '@playwright/test'

/**
 * Visual verification harness.
 *
 * Exists because green unit tests cannot see the screen: the R1 recurring-scope
 * dialog was "verified" by reading JSON and had never been clicked in a browser,
 * and v0.4.5 shipped as done while four mobile calendar views were never built.
 *
 * Target defaults to staging rather than a dev server: what matters is whether
 * the deployed build works, and the loop that uses this runs unattended, where
 * a dev server that silently died reads as a broken app.
 */
const baseURL = process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

export default defineConfig({
  testDir: './e2e',
  // Outside `src`, so tsconfig's include:["src"] leaves these files to
  // Playwright's own transform and `pnpm typecheck` keeps its current meaning.
  outputDir: './e2e-results',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? 'github' : [['list'], ['html', { open: 'never' }]],

  use: {
    baseURL,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
    video: 'off',
  },

  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } },
    // 375px is the width the project's own responsive standard names first, and
    // the one where the header, the event editor and the FAB actually collide.
    //
    // Spelled out rather than using devices['iPhone SE'], which selects WebKit
    // and fails with "Executable doesn't exist" unless a second ~100MB browser
    // is installed. The viewport is what this project's bugs live in, not the
    // engine, so mobile runs on the Chromium already downloaded.
    {
      name: 'mobile',
      use: {
        browserName: 'chromium',
        viewport: { width: 375, height: 667 },
        deviceScaleFactor: 2,
        isMobile: true,
        hasTouch: true,
      },
    },
  ],
})
