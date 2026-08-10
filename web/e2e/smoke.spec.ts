import { test, expect } from '@playwright/test'

/**
 * The baseline the rest of the suite is allowed to trust.
 *
 * Deliberately unauthenticated: it answers "is a real build being served and
 * does it boot without throwing", which is the question that has to be answered
 * before any failure deeper in the app can be blamed on the change under test.
 * When R1's dialog would not open under a synthetic click, the thing that turned
 * a false alarm into a known limitation was trying the same click on a plain
 * event first — this file is that instinct, made routine.
 */

test('the app is served and boots without console errors', async ({ page }) => {
  const errors: string[] = []
  page.on('console', msg => {
    if (msg.type() === 'error') errors.push(msg.text())
  })
  page.on('pageerror', err => errors.push(err.message))

  const response = await page.goto('/')
  expect(response?.status(), 'the origin should serve the SPA, not an error page').toBeLessThan(400)

  // The SPA mounts into #root; a served-but-dead build gives an empty shell,
  // which a status check alone would happily call success.
  await expect(page.locator('#root')).not.toBeEmpty()

  expect(errors, `console errors on first paint:\n${errors.join('\n')}`).toEqual([])
})

test('the login form is reachable and renders its fields', async ({ page }) => {
  // `/` is the marketing landing, not the login — the first version of this test
  // assumed otherwise and failed on both viewports until the failure screenshot
  // showed a hero with "Get Started". Auth lives at /login (web/src/router.tsx:95).
  await page.goto('/login')

  // Located by input type rather than by visible copy: the app is bilingual
  // (en/ru) and language is a user setting, so matching text would make this
  // fail on a language switch instead of on a real regression.
  await expect(page.locator('input[type="password"]')).toBeVisible({ timeout: 15_000 })
  await expect(page.locator('input[type="email"], input[name="email"]').first()).toBeVisible()
})

test('the landing page offers a route into the app', async ({ page }) => {
  await page.goto('/')
  const cta = page.getByRole('link', { name: /get started|начать/i })
    .or(page.getByRole('button', { name: /get started|начать/i }))
  await expect(cta.first()).toBeVisible()
})
