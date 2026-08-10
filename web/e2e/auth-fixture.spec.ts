import { test, expect } from './fixtures/auth'

/**
 * Proves the auth fixture itself, so every spec built on it starts from a
 * checked premise. Without this, a broken fixture would surface as "the
 * calendar has no events" in some unrelated test — the session silently
 * absent, the page silently /login.
 */

test('a seeded session reaches a protected route instead of /login', async ({ authedPage }) => {
  await authedPage.goto('/calendar')
  await authedPage.waitForLoadState('networkidle')

  expect(authedPage.url()).not.toContain('/login')
  expect(new URL(authedPage.url()).pathname).toBe('/calendar')
})

// The negative control. If an unauthenticated visit also stayed on /calendar,
// the test above would prove nothing about the session — it would only prove
// the route exists.
test('without a session the same route redirects to /login', async ({ page }) => {
  await page.goto('/calendar')
  await page.waitForLoadState('networkidle')

  expect(new URL(page.url()).pathname).toBe('/login')
})

test('the app renders as the signed-in user, with no console errors', async ({ authedPage }) => {
  const errors: string[] = []
  authedPage.on('console', (m) => {
    if (m.type() === 'error') errors.push(m.text())
  })

  await authedPage.goto('/calendar')
  await authedPage.waitForLoadState('networkidle')

  await expect(authedPage.locator('body')).toBeVisible()
  expect(errors, `console errors on /calendar:\n${errors.join('\n')}`).toEqual([])
})
